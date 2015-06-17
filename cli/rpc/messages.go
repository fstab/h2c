// Package rpc implements the communication between the h2c command line interface and the h2c process.
//
// The command line interface uses a simple request/response protocol to communicate with the h2c process:
//
// The cli sends a Command struct to the h2c process, and receives a Result struct as result.
package rpc

// Command struct is sent from the command line interface to the h2c process.
type Command struct {
	Name    string
	Args    []string
	Options map[string]string
}

func NewCommand(name string, args []string, options map[string]string) (*Command, error) {
	return &Command{
		Name:    name,
		Args:    args,
		Options: options,
	}, nil
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

// Result is sent from the h2c process to the command line interface.
type Result struct {
	Message string
	Error   *string // Should be type error, but this doesn't seem to work well with JSON marshalling.
}

func NewResult(msg string, err error) *Result {
	if err == nil {
		return &Result{msg, nil}
	} else {
		errString := err.Error()
		return &Result{msg, &errString}
	}
}

// Marshal returns the base64 encoding of Result.
func (res *Result) Marshal() (string, error) {
	return marshal(res)
}

// Used by the command line interface when receiving a Result from the h2c process.
func UnmarshalResult(encodedResult string) (*Result, error) {
	res := &Result{}
	err := unmarshal(encodedResult, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
