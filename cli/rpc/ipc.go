package rpc

import "net"

// IpcManager maintains the socket for communication between the cli and the h2c process.
type IpcManager interface {
	IsListening() bool
	Listen() (net.Listener, error)
	Dial() (net.Conn, error)
	InUseErrorMessage() string
}
