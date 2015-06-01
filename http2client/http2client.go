// Package http2client is a HTTP/2 client library.
package http2client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/bradfitz/http2"
	"github.com/bradfitz/http2/hpack"
	"net"
)

type Http2Client struct {
	host   string
	port   int
	conn   net.Conn
	framer *http2.Framer
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Connect(host string, port int) (string, error) {
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
	return "", nil
}

func (h2c *Http2Client) Get(path string) (string, error) {
	if !h2c.isConnected() {
		return "", fmt.Errorf("Not connected.")
	}
	err := h2c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      1,
		BlockFragment: makeGet(h2c.host),
		EndStream:     true,
		EndHeaders:    true,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to write HEADERS frame to %v: %v", h2c.host, err.Error())
	}

	frame, err := h2c.framer.ReadFrame()
	if err != nil {
		return "", fmt.Errorf("Failed to read next frame: %v", err.Error())
	}
	fmt.Printf("Received frame %v\n", frame)

	return "", nil
}

func (h2c *Http2Client) isConnected() bool {
	return h2c.conn != nil || h2c.framer != nil || h2c.host != "" || h2c.port != 0
}

func makeGet(host string) []byte {

	var buf bytes.Buffer
	encoder := hpack.NewEncoder(&buf)

	encoder.WriteField(hpack.HeaderField{Name: ":authority", Value: host})
	encoder.WriteField(hpack.HeaderField{Name: ":method", Value: "GET"})
	encoder.WriteField(hpack.HeaderField{Name: ":path", Value: "/index.html"})
	encoder.WriteField(hpack.HeaderField{Name: ":scheme", Value: "https"})

	return buf.Bytes()
}
