package connection

import (
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"io"
	"net"
	"os"
)

type Connection struct {
	host            string
	port            int
	conn            net.Conn
	dump            io.Writer
	encodingContext *frames.EncodingContext
	decodingContext *frames.DecodingContext
}

func NewConnection(conn net.Conn, host string, port int, dump bool) *Connection {
	var writer io.Writer
	if dump {
		writer = os.Stdout
	}
	return &Connection{
		host:            host,
		port:            port,
		conn:            conn,
		dump:            writer,
		encodingContext: frames.NewEncodingContext(),
		decodingContext: frames.NewDecodingContext(),
	}
}

func (c *Connection) Host() string {
	return c.host
}

func (c *Connection) Port() int {
	return c.port
}

func (c *Connection) ReadNext() (frames.Frame, error) {
	headerData := make([]byte, 9) // Frame starts with a 9 Bytes header
	_, err := io.ReadFull(c.conn, headerData)
	if err != nil {
		return nil, err
	}
	header := frames.DecodeHeader(headerData)
	payload := make([]byte, header.Length)
	_, err = io.ReadFull(c.conn, payload)
	if err != nil {
		return nil, err
	}
	decodeFunc := frames.FindDecoder(frames.Type(header.HeaderType))
	if decodeFunc == nil {
		return nil, fmt.Errorf("%v: Unknown frame type.", header.HeaderType)
	}
	frame, err := decodeFunc(header.Flags, header.StreamId, payload, c.decodingContext)
	if c.dump != nil {
		io.WriteString(c.dump, fmt.Sprintf("%v\n", frame.String()))
	}
	return frame, err
}

func (c *Connection) Write(frame frames.Frame) error {
	if c.dump != nil {
		io.WriteString(c.dump, fmt.Sprintf("%v\n", frame.String()))
	}
	data, err := frame.Encode(c.encodingContext)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}
