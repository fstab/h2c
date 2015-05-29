package http2client

import (
	"crypto/tls"
	"fmt"
	"github.com/bradfitz/http2"
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
	conn.Close()
	return fmt.Sprintf("Received frame %v", frame), nil
}
