// Package http2client is a HTTP/2 client library.
package http2client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/bradfitz/http2/hpack"
	"github.com/fstab/h2c/http2client/connection"
	"github.com/fstab/h2c/http2client/frames"
	"net"
	"sort"
	"time"
)

type Http2Client struct {
	conn          *connection.Connection
	streams       map[uint32]*stream  // StreamID -> *stream
	customHeaders []hpack.HeaderField // filled with 'h2c set'
	err           error               // if != nil, the Http2Client becomes unusable
	dump          bool                // h2c start --dump
}

func (h2c *Http2Client) initConnection(conn net.Conn, host string, port int) {
	h2c.conn = connection.NewConnection(conn, host, port, h2c.dump)
	h2c.streams = make(map[uint32]*stream)
	h2c.customHeaders = make([]hpack.HeaderField, 0)
}

func (h2c *Http2Client) isConnected() bool {
	return h2c.conn != nil
}

type stream struct {
	receivedHeaders map[string]string // TODO: Does not allow multiple headers with same name
	receivedData    bytes.Buffer
	onClosed        chan *stream
}

func (s *stream) headerCallback(f hpack.HeaderField) {
	s.receivedHeaders[f.Name] = f.Value
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
	hostAndPort := fmt.Sprintf("%v:%v", host, port)
	conn, err := tls.Dial("tcp", hostAndPort, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	})
	if err != nil {
		return "", fmt.Errorf("Failed to connect to %v: %v", hostAndPort, err.Error())
	}
	_, err = conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")) // client preface
	if err != nil {
		return "", fmt.Errorf("Failed to write client preface to %v: %v", hostAndPort, err.Error())
	}
	h2c.initConnection(conn, host, port)
	err = h2c.conn.Write(frames.NewSettingsFrame(0))
	if err != nil {
		return "", fmt.Errorf("Failed to write initial settings frame to %v: %v", hostAndPort, err.Error())
	}
	frame, err := h2c.conn.ReadNext()
	if err != nil || frame.Type() != frames.SETTINGS_TYPE {
		return "", fmt.Errorf("Failed to read initial settings frame from %v: %v", hostAndPort, err.Error())
	}

	go h2c.handleIncomingFrames()
	return "", nil
}

func (h2c *Http2Client) handleIncomingFrames() {
	for {
		frame, err := h2c.conn.ReadNext()
		if err != nil {
			h2c.die(fmt.Errorf("Failed to read next frame: %v", err.Error()))
			return
		}
		if frame.Type() == frames.HEADERS_TYPE {
			headersFrame, ok := frame.(*frames.HeadersFrame)
			if !ok {
				h2c.die(fmt.Errorf("ERROR: frames.ReadNext() returned frame with inconsisten type."))
				return
			}
			s, exists := h2c.streams[headersFrame.StreamId]
			if !exists {
				h2c.streams[headersFrame.StreamId] = &stream{
					receivedHeaders: make(map[string]string),
				}
				s = h2c.streams[headersFrame.StreamId]
			}
			for _, header := range headersFrame.Headers {
				s.receivedHeaders[header.Name] = header.Value
			}
			// TODO: continuations
			// TODO: error handling
			if headersFrame.EndStream && s.onClosed != nil {
				s.onClosed <- s
			}
		}
		if frame.Type() == frames.DATA_TYPE {
			dataFrame, ok := frame.(*frames.DataFrame)
			if !ok {
				h2c.die(fmt.Errorf("ERROR: frames.ReadNext() returned frame with inconsisten type."))
				return
			}
			s, exists := h2c.streams[dataFrame.StreamId]
			if !exists {
				h2c.die(fmt.Errorf("Received data for unknown stream %v", dataFrame.StreamId))
				return
			}
			s.receivedData.Write(dataFrame.Data)
			if dataFrame.EndStream && s.onClosed != nil {
				s.onClosed <- s
			}
		}
	}
}

func (h2c *Http2Client) nextAvailableStreamId() uint32 {
	// Streams initiated by the client must use odd-numbered stream identifiers.
	var result uint32 = 1
	for used, _ := range h2c.streams {
		if used%2 == 1 && used >= result {
			result = used + 2
		}
	}
	return result
}

func (h2c *Http2Client) Get(path string, includeHeaders bool, timeoutInSeconds int) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if !h2c.isConnected() {
		return "", fmt.Errorf("Not connected.")
	}
	streamId := h2c.nextAvailableStreamId()
	h2c.streams[streamId] = &stream{
		receivedHeaders: make(map[string]string),
		//		receivedData:    bytes.Buffer,
		onClosed: make(chan *stream),
	}
	headers := []hpack.HeaderField{
		hpack.HeaderField{Name: ":authority", Value: h2c.conn.Host()},
		hpack.HeaderField{Name: ":method", Value: "GET"},
		hpack.HeaderField{Name: ":path", Value: path},
		hpack.HeaderField{Name: ":scheme", Value: "https"},
	}
	for _, header := range h2c.customHeaders {
		headers = append(headers, header)
	}
	err := h2c.conn.Write(frames.NewHeadersFrame(streamId, headers))
	if err != nil {
		return "", fmt.Errorf("Failed to write HEADERS frame to %v: %v", h2c.conn.Host(), err.Error())
	}
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(timeoutInSeconds) * time.Second)
		timeout <- true
	}()
	select {
	case received := <-h2c.streams[streamId].onClosed:
		// TODO: Check for errors in received headers
		headers := ""
		if includeHeaders {
			sortedHeaderNames := make([]string, len(received.receivedHeaders))
			i := 0
			for name, _ := range received.receivedHeaders {
				sortedHeaderNames[i] = name
				i++
			}
			sort.Strings(sortedHeaderNames)
			for _, name := range sortedHeaderNames {
				headers += name + ": " + received.receivedHeaders[name] + "\n"
			}
			headers += "\n"
		}
		return headers + string(received.receivedData.Bytes()), nil
	case <-timeout:
		// TODO: Send RST_STREAM frame
		return "", errors.New("Timeout while waiting for response.")
	}
}

func (h2c *Http2Client) SetHeader(name, value string) (string, error) {
	for name[len(name)-1] == ':' {
		name = name[:len(name)-1]
	}
	h2c.customHeaders = append(h2c.customHeaders, hpack.HeaderField{Name: name, Value: value})
	return "", nil
}

func (h2c *Http2Client) UnsetHeader(nameValue []string) (string, error) {
	if len(nameValue) != 1 && len(nameValue) != 2 {
		return "", errors.New("Syntax error.")
	}
	remainingHeaders := make([]hpack.HeaderField, 0, len(h2c.customHeaders))
	matches := func(field hpack.HeaderField) bool {
		if len(nameValue) == 1 {
			return field.Name == nameValue[0]
		} else {
			return field.Name == nameValue[0] && field.Value == nameValue[1]
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
