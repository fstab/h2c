// Package commands implements the protocol between the h2c command line interface and the h2c process.
//
// The command line interface uses a simple request/response protocol to communicate with the h2c process:
//
// The cli sends a Command struct to the h2c process, and receives a Result struct as result.
package commands

import (
	"fmt"
	"regexp"
	"strings"
)

// Command struct is sent from the command line interface to the h2c process.
type Command struct {
	Name   string
	Params map[string]string
}

func NewCommand(args []string) (*Command, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("Syntax error: h2c <cmd>")
	}
	switch args[0] {
	case "connect":
		return newConnectCmd(args)
	case "get":
		return newGetCmd(args)
	case "pid":
		return newPidCmd(args)
	case "stop":
		return newStopCmd(args)
	default:
		return nil, fmt.Errorf("Syntax error: Command %v unknown", args[0])
	}
}

// Marshal returns the base64 encoding of a Command.
//
// The resulting string does not contain newlines,
// so newlines can be used as separators between multiple commands.
func (cmd *Command) Marshal() (string, error) {
	return marshal(cmd)
}

// Used by the h2c process when receiving a command from the command line interface.
func UnmarshalCommand(encodedCmd string) (*Command, error) {
	cmd := &Command{}
	err := unmarshal(encodedCmd, cmd)
	if err != nil {
		return nil, err
	}
	return cmd, nil
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
	if !regexp.MustCompile("^[0-9]+$").MatchString(port) {
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

func newGetCmd(args []string) (*Command, error) {
	syntaxError := fmt.Errorf(
		`Syntax error: h2c get [options] <path>
Available options are:
  -i, --include: Show response headers in the output.`)
	result := &Command{
		Name:   "get",
		Params: make(map[string]string),
	}
	if len(args) < 2 || args[0] != "get" {
		return nil, syntaxError
	}
	if args[1] == "-i" || args[1] == "--include" {
		result.Params["include-headers"] = "true"
		if len(args) != 3 {
			return nil, syntaxError
		}
		result.Params["path"] = args[2]
	} else {
		result.Params["include-headers"] = "false"
		if len(args) != 2 {
			return nil, syntaxError
		}
		result.Params["path"] = args[1]
	}
	return result, nil
}

func newStopCmd(args []string) (*Command, error) {
	if len(args) != 1 || args[0] != "stop" {
		return nil, fmt.Errorf("Syntax error: h2c stop")
	}
	return &Command{
		Name:   args[0],
		Params: make(map[string]string),
	}, nil
}
