// Package cli implements the user interaction on the command line.
package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/fstab/h2c/cli/daemon"
	"github.com/fstab/h2c/cli/messages/command"
	"github.com/fstab/h2c/cli/messages/result"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
)

// Run executes the command, as provided in os.Args.
func Run() (string, error) {
	socketFilePath := filepath.Join(os.TempDir(), "h2c.sock")
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "start":
			return "", startDaemon(socketFilePath)
		default:
			if !fileExists(socketFilePath) {
				return "", fmt.Errorf("Please start h2c first. In order to start h2c as a background process, run '%v'.", startCmd)
			}
			cmd, err := command.New(os.Args[1:])
			if err != nil {
				return "", err
			}
			res := sendCommand(cmd, socketFilePath)
			return res.Message, res.Error
		}
	} else {
		return "", fmt.Errorf(usage())
	}
}

func startDaemon(socketFilePath string) error {
	if fileExists(socketFilePath) {
		pidCmd, _ := command.New([]string{"pid"})
		res := sendCommand(pidCmd, socketFilePath)
		if res.Error != nil || !isNumber(res.Message) {
			return fmt.Errorf("The file %v already exists. Make sure h2c is not running and remove the file.\n", socketFilePath)
		} else {
			return fmt.Errorf("h2c already running with PID %v\n", res.Error)
		}
	}
	return daemon.Start(socketFilePath)
}

func sendCommand(cmd *command.Command, socketFilePath string) *result.Result {
	conn, err := net.Dial("unix", socketFilePath)
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
	res, err := result.Unmarshal(string(responseBuffer.Bytes()))
	if err != nil {
		return communicationError(err)
	}
	return res
}

func communicationError(err error) *result.Result {
	return result.New("", fmt.Errorf("Failed to communicate with h2c process: %v", err.Error()))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isNumber(s string) bool {
	return regexp.MustCompile("^[0-9]+$").MatchString(s)
}

func usage() string {
	return "Usage:\n" +
		startCmd + "\n" +
		"h2s connect <host>:<port>\n" +
		"h2s get <path>"
}
