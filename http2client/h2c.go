package http2client

import (
	"crypto/tls"
	"fmt"
	"strings"
)

type Http2Client struct {
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Connect(url string) (string, error) {
	if url[0:8] != "https://" {
		return "", fmt.Errorf("Only https:// urls are currently supported.")
	}
	host := strings.Split(url[8:], "/")[0]
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}
	fmt.Printf("Connecting to %v\n", host)
	conn, err := tls.Dial("tcp", host, &tls.Config{})
	if err != nil {
		return "", fmt.Errorf("Failed to connect to %v: %v", host, err.Error())
	}
	conn.Close()
	return "", nil
}
