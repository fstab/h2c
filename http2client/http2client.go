// Package http2client is a HTTP/2 client library.
package http2client

import (
	"errors"
	"fmt"
	"github.com/fstab/h2c/http2client/connection"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/util"
	"github.com/fstab/http2/hpack"
	"sort"
	"strconv"
	"strings"
)

type Http2Client struct {
	conn          *connection.Connection
	customHeaders []hpack.HeaderField // filled with 'h2c set'
	err           error               // if != nil, the Http2Client becomes unusable
	dump          bool                // h2c start --dump
}

func New(dump bool) *Http2Client {
	return &Http2Client{
		dump: dump,
	}
}

func (h2c *Http2Client) Connect(host string, port int) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if h2c.isConnected() {
		return "", fmt.Errorf("Already connected to %v:%v.", h2c.conn.Host(), h2c.conn.Port())
	}
	conn, err := connection.Start(host, port, h2c.dump)
	if err != nil {
		return "", err
	}
	h2c.conn = conn
	return "", nil
}

func (h2c *Http2Client) isConnected() bool {
	return h2c.conn != nil && !h2c.conn.IsShutdown()
}

func (h2c *Http2Client) Disconnect() (string, error) {
	if h2c.isConnected() {
		// TODO: Send goaway to server.
		h2c.conn.Shutdown()
		h2c.conn = nil
	}
	return "", nil
}

func (h2c *Http2Client) Get(path string, includeHeaders bool, timeoutInSeconds int) (string, error) {
	return h2c.doRequest("GET", path, nil, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) Post(path string, data []byte, includeHeaders bool, timeoutInSeconds int) (string, error) {
	return h2c.doRequest("POST", path, data, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) doRequest(method string, path string, data []byte, includeHeaders bool, timeoutInSeconds int) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if !h2c.isConnected() {
		return "", fmt.Errorf("Not connected.")
	}
	task := util.NewAsyncTask()
	stream := h2c.conn.InitNewStream(task)
	requestHeaders := makeHeaders(h2c.conn.Host(), method, path, "http2", h2c.customHeaders, data)
	headersFrame := frames.NewHeadersFrame(stream.StreamId(), requestHeaders)
	headersFrame.EndStream = data == nil
	err := stream.Write(headersFrame, timeoutInSeconds) // use same timeout for writing single frame and entire request
	if err != nil {
		return "", fmt.Errorf("Failed to write HEADERS frame to %v: %v", h2c.conn.Host(), err.Error())
	}
	if data != nil {
		err = h2c.sendDataFrames(data, stream, timeoutInSeconds) // use same timeout for writing single frame and entire request
		if err != nil {
			return "", err
		}
	}
	err = task.WaitForCompletion(timeoutInSeconds)
	if err != nil {
		return "", fmt.Errorf("Error while waiting for response: %v", err.Error())
	}
	// TODO: Make sure stream is closed. If not closed, this would be a bug.
	if stream.Error() != nil {
		return "", stream.Error()
	}
	// TODO: Check for errors in received headers
	responseHeaders := ""
	if includeHeaders {
		sortedHeaderNames := make([]string, len(stream.ReceivedHeaders()))
		i := 0
		for name, _ := range stream.ReceivedHeaders() {
			sortedHeaderNames[i] = name
			i++
		}
		sort.Strings(sortedHeaderNames)
		for _, name := range sortedHeaderNames {
			responseHeaders += name + ": " + stream.ReceivedHeaders()[name] + "\n"
		}
		responseHeaders += "\n"
	}
	return responseHeaders + string(stream.ReceivedData()), nil
}

func makeHeaders(authority, method, path, scheme string, customHeaders []hpack.HeaderField, data []byte) []hpack.HeaderField {
	headers := []hpack.HeaderField{
		hpack.HeaderField{Name: ":authority", Value: authority},
		hpack.HeaderField{Name: ":method", Value: method},
		hpack.HeaderField{Name: ":path", Value: path},
		hpack.HeaderField{Name: ":scheme", Value: scheme},
	}
	headers = append(headers, customHeaders...)
	if data != nil {
		headers = append(headers, hpack.HeaderField{
			Name:  "content-length",
			Value: strconv.Itoa(len(data)),
		})
	}
	return headers
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func (h2c *Http2Client) sendDataFrames(data []byte, stream *connection.Stream, timeoutPerFrameInSeconds int) error {
	// chunkSize := uint32(len(data)) // use this to provoke GOAWAY frame with FRAME_SIZE_ERROR
	chunkSize := h2c.conn.ServerFrameSize() // TODO: Query chunk size with each iteration -> allow changes during loop
	nChunksSent := uint32(0)
	total := uint32(len(data))
	for nChunksSent*chunkSize < total {
		nextChunk := data[nChunksSent*chunkSize : min((nChunksSent+1)*chunkSize, total)]
		nChunksSent = nChunksSent + 1
		isLast := nChunksSent*chunkSize >= total
		dataFrame := frames.NewDataFrame(stream.StreamId(), nextChunk, isLast)
		err := stream.Write(dataFrame, timeoutPerFrameInSeconds)
		if err != nil {
			return fmt.Errorf("Failed to write DATA frame to %v: %v", h2c.conn.Host(), err.Error())
		}
	}
	return nil
}

func (h2c *Http2Client) SetHeader(name, value string) (string, error) {
	h2c.customHeaders = append(h2c.customHeaders, hpack.HeaderField{
		Name:  normalizeHeaderName(name),
		Value: value,
	})
	return "", nil
}

// "Content-Type:" -> "content-type"
func normalizeHeaderName(name string) string {
	for name[len(name)-1] == ':' {
		name = name[:len(name)-1]
	}
	// return name // Use this and set header "Content-Type" to provoke RST_STREAM with error CANCEL.
	return strings.ToLower(name)
}

func (h2c *Http2Client) UnsetHeader(nameValue []string) (string, error) {
	if len(nameValue) != 1 && len(nameValue) != 2 {
		return "", errors.New("Syntax error.")
	}
	remainingHeaders := make([]hpack.HeaderField, 0, len(h2c.customHeaders))
	matches := func(field hpack.HeaderField) bool {
		if len(nameValue) == 1 {
			return field.Name == normalizeHeaderName(nameValue[0])
		} else {
			return field.Name == normalizeHeaderName(nameValue[0]) && field.Value == nameValue[1]
		}
	}
	for _, field := range h2c.customHeaders {
		if !matches(field) {
			remainingHeaders = append(remainingHeaders, field)
		}
	}
	h2c.customHeaders = remainingHeaders
	return "", nil
}

func (h2c *Http2Client) die(err error) {
	// TODO: disconnect
	if h2c.err == nil {
		h2c.err = err
	}
}
