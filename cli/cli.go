// Package cli implements the h2c command line interface.
package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fstab/h2c/cli/commands"
	"github.com/fstab/h2c/cli/daemon"
	"io"
	"net"
	"os"
	"regexp"
)

// IpcManager maintains the socket for communication between the cli and the h2c process.
type IpcManager interface {
	IsListening() bool
	Listen() (net.Listener, error)
	Dial() (net.Conn, error)
	InUseErrorMessage() string
}

// Run executes the command, as provided in os.Args.
func Run() (string, error) {
	ipc := NewIpcManager()
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "start":
			return "", startDaemon(ipc)
		default:
			cmd, syntaxError := commands.NewCommand(os.Args[1:])
			if syntaxError != nil {
				return "", syntaxError
			}
			if !ipc.IsListening() {
				if os.Args[1] == "stop" {
					return "", fmt.Errorf("h2c is not running.")
				} else {
					return "", fmt.Errorf("Please start h2c first. In order to start h2c as a background process, run '%v'.", startCmd)
				}
			}
			res := sendCommand(cmd, ipc)
			if res.Error != nil {
				return res.Message, fmt.Errorf("%v", *res.Error)
			} else {
				return res.Message, nil
			}
		}
	} else {
		return "", fmt.Errorf(usage())
	}
}

func startDaemon(ipc IpcManager) error {
	if ipc.IsListening() {
		pidCmd, _ := commands.NewCommand([]string{"pid"})
		res := sendCommand(pidCmd, ipc)
		if res.Error != nil || !isNumber(res.Message) {
			return fmt.Errorf(ipc.InUseErrorMessage())
		} else {
			return fmt.Errorf("h2c already running with PID %v\n", res.Message)
		}
	}
	sock, err := ipc.Listen()
	if err != nil {
		return err
	}
	return daemon.Run(sock)
}

func sendCommand(cmd *commands.Command, ipc IpcManager) *commands.Result {
	conn, err := ipc.Dial()
	if err != nil {
		return communicationError(err)
	}
	writer := bufio.NewWriter(conn)
	base64cmd, err := cmd.Marshal()
	if err != nil {
		return communicationError(err)
	}
	_, err = writer.WriteString(base64cmd + "\n")
	if err != nil {
		return communicationError(err)
	}
	err = writer.Flush()
	if err != nil {
		return communicationError(err)
	}
	responseBuffer := bytes.NewBuffer(nil)
	_, err = io.Copy(responseBuffer, conn)
	if err != nil {
		return communicationError(err)
	}
	res, err := commands.UnmarshalResult(string(responseBuffer.Bytes()))
	if err != nil {
		return communicationError(err)
	}
	return res
}

func communicationError(err error) *commands.Result {
	return commands.NewResult("", fmt.Errorf("Failed to communicate with h2c process: %v", err.Error()))
}

func isNumber(s string) bool {
	return regexp.MustCompile("^[0-9]+$").MatchString(s)
}

func usage() string {
	return "Usage:\n" +
		startCmd + "\n" +
		"h2c connect <host>:<port>\n" +
		"h2c get <path>\n" +
		"h2c stop"
}
