package http2client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/bradfitz/http2"
	"github.com/bradfitz/http2/hpack"
)

type Http2Client struct {
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Connect(host string, port int) (string, error) {
	hostAndPort := fmt.Sprintf("%v:%v", host, port)
	fmt.Printf("Connecting to %v:%v\n", hostAndPort)
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
	if err != nil {
		return "", fmt.Errorf("Failed to read initial settings frame from %v: %v", hostAndPort, err.Error())
	}
	fmt.Printf("Received frame %v\n", frame)
	err = framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      1,
		BlockFragment: makeGet(host),
		EndStream:     true,
		EndHeaders:    true,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to write HEADERS frame to %v: %v", hostAndPort, err.Error())
	}
	frame, err = framer.ReadFrame()
	if err != nil {
		return "", fmt.Errorf("Failed to read initial settings frame from %v: %v", hostAndPort, err.Error())
	}
	fmt.Printf("Received frame %v\n", frame)
	frame, err = framer.ReadFrame()
	if err != nil {
		return "", fmt.Errorf("Failed to read initial settings frame from %v: %v", hostAndPort, err.Error())
	}
	fmt.Printf("Received frame %v\n", frame)
	frame, err = framer.ReadFrame()
	if err != nil {
		return "", fmt.Errorf("Failed to read initial settings frame from %v: %v", hostAndPort, err.Error())
	}
	fmt.Printf("Received frame %v\n", frame)
	conn.Close()
	return fmt.Sprintf("Received frame %v", frame), nil
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
