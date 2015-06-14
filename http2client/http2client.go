// Package http2client is a HTTP/2 client library.
package http2client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/bradfitz/http2"
	"github.com/bradfitz/http2/hpack"
	"net"
	"sort"
)

type Http2Client struct {
	host    string
	port    int
	conn    net.Conn
	streams map[uint32]*stream // StreamID -> *stream
	framer  *http2.Framer
	err     error
}

type stream struct {
	receivedHeaders map[string]string
	receivedData    bytes.Buffer
	onClosed        chan *stream
}

func (s *stream) headerCallback(f hpack.HeaderField) {
	s.receivedHeaders[f.Name] = f.Value
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Connect(host string, port int) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if h2c.isConnected() {
		return "", fmt.Errorf("Already connected to %v:%v.", h2c.host, h2c.port)
	}
	hostAndPort := fmt.Sprintf("%v:%v", host, port)
	conn, err := tls.Dial("tcp", hostAndPort, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{http2.NextProtoTLS},
	})
	if err != nil {
		return "", fmt.Errorf("Failed to connect to %v: %v", hostAndPort, err.Error())
	}
	_, err = conn.Write([]byte(http2.ClientPreface))
	if err != nil {
		return "", fmt.Errorf("Failed to write client preface to %v: %v", hostAndPort, err.Error())
	}
	framer := http2.NewFramer(conn, conn)
	err = framer.WriteSettings()
	if err != nil {
		return "", fmt.Errorf("Failed to write initial settings frame to %v: %v", hostAndPort, err.Error())
	}
	frame, err := framer.ReadFrame()
	if err != nil || frame.Header().Type != http2.FrameSettings {
		return "", fmt.Errorf("Failed to read initial settings frame from %v: %v", hostAndPort, err.Error())
	}
	h2c.conn = conn
	h2c.framer = framer
	h2c.host = host
	h2c.port = port
	h2c.streams = make(map[uint32]*stream)
	go h2c.handleIncomingFrames()
	return "", nil
}

func (h2c *Http2Client) handleIncomingFrames() {
	for {
		frame, err := h2c.framer.ReadFrame()
		if err != nil {
			h2c.die(fmt.Errorf("Failed to read next frame: %v", err.Error()))
			return
		}
		if frame.Header().Type == http2.FrameHeaders {
			headersFrame, ok := frame.(*http2.HeadersFrame)
			if !ok {
				h2c.die(fmt.Errorf("ERROR: Incompatable version of github.com/bradfitz/http2"))
				return
			}
			s, exists := h2c.streams[headersFrame.StreamID]
			if !exists {
				h2c.streams[headersFrame.StreamID] = &stream{
					receivedHeaders: make(map[string]string),
				}
				s = h2c.streams[headersFrame.StreamID]
			}
			headerCallback := func(f hpack.HeaderField) {
				s.receivedHeaders[f.Name] = f.Value
			}
			decoder := hpack.NewDecoder(4096, headerCallback)
			blockFragment := headersFrame.HeaderBlockFragment()
			decoder.Write(blockFragment)
			// TODO: Handler continuations
			// TODO: error handling
			decoder.Close()
			if headersFrame.StreamEnded() && s.onClosed != nil {
				s.onClosed <- s
			}
		}
		if frame.Header().Type == http2.FrameData {
			dataFrame, ok := frame.(*http2.DataFrame)
			if !ok {
				h2c.die(fmt.Errorf("ERROR: Incompatable version of github.com/bradfitz/http2"))
				return
			}
			s, exists := h2c.streams[dataFrame.StreamID]
			if !exists {
				h2c.die(fmt.Errorf("Received data for unknown stream %v", dataFrame.StreamID))
				return
			}
			s.receivedData.Write(dataFrame.Data())
			if dataFrame.StreamEnded() && s.onClosed != nil {
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

func (h2c *Http2Client) Get(path string, includeHeaders bool) (string, error) {
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
	err := h2c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamId,
		BlockFragment: makeGet(h2c.host, path),
		EndStream:     true,
		EndHeaders:    true,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to write HEADERS frame to %v: %v", h2c.host, err.Error())
	}
	received := <-h2c.streams[streamId].onClosed
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
}

func (h2c *Http2Client) isConnected() bool {
	return h2c.conn != nil || h2c.framer != nil || h2c.host != "" || h2c.port != 0
}

func makeGet(host, path string) []byte {

	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)

	encoder.WriteField(hpack.HeaderField{Name: ":authority", Value: host})
	encoder.WriteField(hpack.HeaderField{Name: ":method", Value: "GET"})
	encoder.WriteField(hpack.HeaderField{Name: ":path", Value: path})
	encoder.WriteField(hpack.HeaderField{Name: ":scheme", Value: "https"})

	return buf.Bytes()
}

func (h2c *Http2Client) die(err error) {
	// TODO: disconnect
	if h2c.err == nil {
		h2c.err = err
	}
}
