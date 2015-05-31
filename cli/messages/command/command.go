package command

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// The resulting string is a single line, it does not contain newlines.
func (cmd *Command) Encode() (string, error) {
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

func Decode(encodedCmd string) (*Command, error) {
	jsonData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedCmd))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode base64 data: %v", err.Error())
	}
	cmd := &Command{}
	err = json.Unmarshal(jsonData, cmd)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode json data: %v %v", err.Error())
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
