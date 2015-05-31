package command

import (
	"fmt"
	"github.com/fstab/h2c/cli/messages/marshaller"
	"regexp"
	"strings"
)

type Command struct {
	Name   string
	Params map[string]string
}

type Result struct {
	Message string
	Error   error
}

func New(args []string) (*Command, error) {
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

// Marshal returns the base64 encoding of cmd.
//
// The resulting string does not contain newlines,
// so newlines can be used as separators between multiple commands.
func (cmd *Command) Marshal() (string, error) {
	return marshaller.Marshal(cmd)
}

// Unmarshal is the inverse function of Marshal().
func Unmarshal(encodedCmd string) (*Command, error) {
	cmd := &Command{}
	err := marshaller.Unmarshal(encodedCmd, cmd)
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
