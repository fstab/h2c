// Package cli implements the h2c command line interface.
package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fstab/h2c/cli/cmdline"
	"github.com/fstab/h2c/cli/daemon"
	"github.com/fstab/h2c/cli/rpc"
	"io"
	"os"
	"regexp"
)

// Run executes the command, as provided in os.Args.
func Run() (string, error) {
	ipc := rpc.NewIpcManager()
	cmd, err := cmdline.Parse(os.Args[1:])
	if err != nil {
		return "", err
	}
	switch cmd.Name {
	case "start":
		return "", startDaemon(ipc)
	default:
		if !ipc.IsListening() {
			if cmd.Name == "stop" {
				return "", fmt.Errorf("h2c is not running.")
			} else {
				return "", fmt.Errorf("Please start h2c first. In order to start h2c as a background process, run '%v'.", cmdline.StartCmd)
			}
		}
		res := sendCommand(cmd, ipc)
		if res.Error != nil {
			return res.Message, fmt.Errorf("%v", *res.Error)
		} else {
			return res.Message, nil
		}
	}
}

func startDaemon(ipc rpc.IpcManager) error {
	if ipc.IsListening() {
		pidCmd, _ := rpc.NewCommand("pid", make([]string, 0), make(map[string]string))
		res := sendCommand(pidCmd, ipc)
		if res.Error != nil || !isNumber(res.Message) {
			return fmt.Errorf(ipc.InUseErrorMessage())
		} else {
			return fmt.Errorf("h2c already running with PID %v", res.Message)
		}
	}
	sock, err := ipc.Listen()
	if err != nil {
		return err
	}
	return daemon.Run(sock)
}

func sendCommand(cmd *rpc.Command, ipc rpc.IpcManager) *rpc.Result {
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
		if cmd.Name == "stop" && len(responseBuffer.Bytes()) > 0 {
			// Ignore. This seems to happen on windows when the connection is closed because of the 'stop' command.
		} else {
			return communicationError(err)
		}
	}
	res, err := rpc.UnmarshalResult(string(responseBuffer.Bytes()))
	if err != nil {
		return communicationError(err)
	}
	return res
}

func communicationError(err error) *rpc.Result {
	return rpc.NewResult("", fmt.Errorf("Failed to communicate with h2c process: %v", err.Error()))
}

func isNumber(s string) bool {
	return regexp.MustCompile("^[0-9]+$").MatchString(s)
}
