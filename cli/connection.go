// +build !windows

package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

type UnixSocketConnection struct {
	socketFilePath string
}

func NewIpcManager() *UnixSocketConnection {
	return &UnixSocketConnection{
		socketFilePath: filepath.Join(os.TempDir(), "h2c.sock"),
	}
}

func (s *UnixSocketConnection) IsListening() bool {
	_, err := os.Stat(s.socketFilePath)
	return err == nil
}

func (s *UnixSocketConnection) Listen() (net.Listener, error) {
	sock, err := net.Listen("unix", s.socketFilePath)
	if err != nil {
		return nil, fmt.Errorf("Error creating %v: %v", s.socketFilePath, err.Error())
	}
	return sock, nil
}

func (s *UnixSocketConnection) Dial() (net.Conn, error) {
	return net.Dial("unix", s.socketFilePath)
}

func (s *UnixSocketConnection) InUseErrorMessage() string {
	return fmt.Sprintf("The file %v already exists. Make sure h2c is not running and remove the file.", s.socketFilePath)
}
