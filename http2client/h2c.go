package http2client

import "fmt"

type Http2Client struct {
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Execute(cmd string) (string, error) {
	fmt.Printf("Executing %v", cmd)
	return "", nil
}
