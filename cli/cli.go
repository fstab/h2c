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
	"io/ioutil"
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
	cmd, err = applySpecialConventions(cmd)
	if err != nil {
		return "", err
	}
	if cmdline.START_COMMAND.Name() == cmd.Name {
		return "", startDaemon(ipc, cmdline.DUMP_OPTION.IsSet(cmd.Options))
	} else {
		if !ipc.IsListening() {
			if cmdline.STOP_COMMAND.Name() == cmd.Name {
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

func applySpecialConventions(cmd *rpc.Command) (*rpc.Command, error) {
	var err error
	if cmd.Name == cmdline.POST_COMMAND.Name() {
		if cmdline.DATA_OPTION.IsSet(cmd.Options) && cmdline.FILE_OPTION.IsSet(cmd.Options) {
			return nil, fmt.Errorf("Syntax error: --data and --file cannot be used together.")
		} else if cmdline.FILE_OPTION.IsSet(cmd.Options) {
			cmd, err = mapFile2Data(cmd)
			if err != nil {
				return nil, err
			}
		}
	}
	return cmd, nil
}

func mapFile2Data(cmd *rpc.Command) (*rpc.Command, error) {
	var (
		filename string
		data     []byte
		err      error
	)
	filename = cmdline.FILE_OPTION.Get(cmd.Options)
	if filename == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("Failed to read data from stdin: %v", err.Error())
		}
	} else {
		data, err = ioutil.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("Failed to read %v: %v", filename, err.Error())
		}
	}
	cmdline.FILE_OPTION.Delete(cmd.Options)
	cmdline.DATA_OPTION.Set(string(data), cmd.Options)
	return cmd, nil
}

func startDaemon(ipc rpc.IpcManager, dump bool) error {
	if ipc.IsListening() {
		pidCmd, _ := rpc.NewCommand(cmdline.PID_COMMAND.Name(), make([]string, 0), make(map[string]string))
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
	return daemon.Run(sock, dump)
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
		if cmd.Name == cmdline.STOP_COMMAND.Name() && len(responseBuffer.Bytes()) > 0 {
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
