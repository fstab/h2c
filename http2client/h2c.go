package http2client

import (
	"fmt"
)

type Http2Client struct {
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Connect(url string) (string, error) {
	fmt.Printf("Connecting to %v\n", url)
	return "", nil
}
