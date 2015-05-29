package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
)

var (
	number = regexp.MustCompile("^[0-9]+$")
)

type Command struct {
	Name   string
	Params map[string]string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func newConnectCmd(args []string) (*Command, error) {
	if len(args) != 2 || args[0] != "connect" {
		return nil, fmt.Errorf("Syntax error: h2c connect host:port")
	}
	hostAndPort := strings.Split(args[1], ":")
	if len(hostAndPort) == 1 {
		hostAndPort = append(hostAndPort, "443")
	}
	if len(hostAndPort) != 2 {
		return nil, fmt.Errorf("Syntax error: h2c connect host:port")
	}
	host := hostAndPort[0]
	if len(host) < 1 {
		return nil, fmt.Errorf("Syntax error: h2c connect host:port")
	}
	port := hostAndPort[1]
	if !number.MatchString(port) {
		return nil, fmt.Errorf("Syntax error: h2c connect host:port")
	}
	return &Command{
		Name: args[0],
		Params: map[string]string{
			"host": host,
			"port": port,
		},
	}, nil
}

func newPidCmd(args []string) (*Command, error) {
	if len(args) != 1 || args[0] != "pid" {
		return nil, fmt.Errorf("Syntax error: h2c pid")
	}
	return &Command{
		Name:   args[0],
		Params: make(map[string]string),
	}, nil
}

func newCommand(args []string) (*Command, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("Syntax error: h2c <cmd>")
	}
	switch args[0] {
	case "connect":
		return newConnectCmd(args)
	case "pid":
		return newPidCmd(args)
	default:
		return nil, fmt.Errorf("Syntax error: Command %v unknown", args[0])
	}
}

func main() {
	socketFilePath := "/tmp/h2c.sock"
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "start":
			if fileExists(socketFilePath) {
				cmd, _ := newPidCmd([]string{"pid"})
				resp, err := sendCommand(cmd, socketFilePath)
				if err != nil || !number.MatchString(resp) {
					fmt.Fprintf(os.Stderr, "The file %v already exists. Make sure h2c is not running and remove the file.\n", socketFilePath)
					os.Exit(-1)
				} else {
					fmt.Fprintf(os.Stderr, "h2c already running with PID %v\n", resp)
					os.Exit(-1)
				}
			}
			if err := start(socketFilePath); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(-1)
			}
		default:
			if !fileExists(socketFilePath) {
				fmt.Fprint(os.Stderr, "Please run 'h2c start' first.\n")
				os.Exit(-1)
			}
			cmd, err := newCommand(os.Args[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(-1)
			}
			resp, err := sendCommand(cmd, socketFilePath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(-1)
			} else {
				fmt.Println(resp)
			}
		}
	} else {
		usage()
	}
}

// The resulting string is a single line, it does not contain newlines.
func (cmd *Command) encode() (string, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", fmt.Errorf("Marshalling error: %v", err.Error())
	}
	result := base64.StdEncoding.EncodeToString(data)
	if strings.Contains(result, "\n") {
		return "", fmt.Errorf("Base64 encoding error: Received newline in base64 string.")
	}
	return result, nil
}

func sendCommand(cmd *Command, socketFilePath string) (string, error) {
	conn, err := net.Dial("unix", socketFilePath)
	if err != nil {
		return "", fmt.Errorf("Failed to connect UNIX socket %v: %v", socketFilePath, err.Error())
	}
	writer := bufio.NewWriter(conn)
	base64cmd, err := cmd.encode()
	if err != nil {
		return "", fmt.Errorf("Failed to send command to h2c process: %v", err)
	}
	_, err = writer.WriteString(base64cmd + "\n")
	if err != nil {
		return "", fmt.Errorf("Failed to send command to h2c process: %v", err.Error())
	}
	err = writer.Flush()
	if err != nil {
		return "", fmt.Errorf("Failed to send command to h2c process: %v", err.Error())
	}
	responseBuffer := bytes.NewBuffer(nil)
	_, err = io.Copy(responseBuffer, conn)
	if err != nil {
		return "", fmt.Errorf("Failed to read response from h2c process: %v", err.Error())
	}
	return string(responseBuffer.Bytes()), nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "h2s start")
	fmt.Fprintln(os.Stderr, "h2s echo")
	os.Exit(-1)
}
