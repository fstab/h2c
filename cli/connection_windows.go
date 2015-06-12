// +build windows

package cli

import (
	"fmt"
	"net"
)

type TcpSocketConnection struct {
	port int
}

func NewIpcManager() *TcpSocketConnection {
	return &TcpSocketConnection{
		port: 27888,
	}
}

func (c *TcpSocketConnection) IsListening() bool {
	sock, err := c.Listen()
	if err != nil {
		return true
	} else {
		sock.Close()
		return false
	}
}

func (c *TcpSocketConnection) Listen() (net.Listener, error) {
	return net.Listen("tcp", c.connectString())
}

func (c *TcpSocketConnection) Dial() (net.Conn, error) {
	return net.Dial("tcp", c.connectString())
}

func (c *TcpSocketConnection) InUseErrorMessage() string {
	return fmt.Sprintf("TCP port %v is already in use.", c.port)
}

func (c *TcpSocketConnection) connectString() string {
	return fmt.Sprintf("localhost:%v", c.port)
}
