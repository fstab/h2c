package http2client

import "fmt"

type Http2Client struct {
}

func New() *Http2Client {
	return &Http2Client{}
}

func (h2c *Http2Client) Connect(host, port string) (string, error) {
	fmt.Printf("Connecting to https://%v:%v\n", host, port)
	return "", nil
}
