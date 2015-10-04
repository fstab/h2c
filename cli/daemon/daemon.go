// Package daemon implements the h2c process, i.e, the process started with 'h2c start'.
package daemon

import (
	"bufio"
	"fmt"
	"github.com/fstab/h2c/cli/cmdline"
	"github.com/fstab/h2c/cli/rpc"
	"github.com/fstab/h2c/http2client"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

// Run the h2c process, i.e, the process started with 'h2c start'.
//
// The h2c process keeps an Http2Client instance, reads Commands from the socket file,
// and uses the Http2Client to execute these commands.
//
// The socket will be closed when the h2c process is terminated.
func Run(sock net.Listener, dump bool) error {
	var conn net.Conn
	var err error
	var h2c = http2client.New(dump)
	stopOnSigterm(sock)
	for {
		if conn, err = sock.Accept(); err != nil {
			return fmt.Errorf("Error while waiting for commands: %v", err.Error())
			stop(sock)
		}
		go executeCommandAndCloseConnection(h2c, conn, sock)
	}
}

func close(sock io.Closer) {
	if err := sock.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error terminating the h2c process: %v", err.Error())
	}
}

func stop(sock net.Listener) {
	close(sock)
	os.Exit(0)
}

func stopOnSigterm(sock net.Listener) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	go func(c chan os.Signal) {
		<-c // Wait for a SIGINT
		stop(sock)
	}(sigc)
}

func execute(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	switch cmd.Name {
	case cmdline.CONNECT_COMMAND.Name():
		return executeConnect(h2c, cmd)
	case cmdline.DISCONNECT_COMMAND.Name():
		return executeDisconnect(h2c, cmd)
	case cmdline.PID_COMMAND.Name():
		return strconv.Itoa(os.Getpid()), nil
	case cmdline.GET_COMMAND.Name():
		return executeGet(h2c, cmd)
	case cmdline.PUT_COMMAND.Name():
		return executePut(h2c, cmd)
	case cmdline.POST_COMMAND.Name():
		return executePost(h2c, cmd)
	case cmdline.PUSH_LIST_COMMAND.Name():
		return executePushList(h2c, cmd)
	case cmdline.SET_COMMAND.Name():
		return h2c.SetHeader(cmd.Args[0], cmd.Args[1])
	case cmdline.UNSET_COMMAND.Name():
		return h2c.UnsetHeader(cmd.Args)
	default:
		return "", fmt.Errorf("%v: unknown command", cmd.Name)
	}
}

func executeConnect(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	hostAndPort := strings.Split(cmd.Args[0], ":")
	if len(hostAndPort) == 1 {
		hostAndPort = append(hostAndPort, "443")
	}
	if len(hostAndPort) != 2 {
		return "", fmt.Errorf("%v: Invalid hostanme", cmd.Args[0])
	}
	host := hostAndPort[0]
	if len(host) < 1 {
		return "", fmt.Errorf("%v: Invalid hostname", cmd.Args[0])
	}
	port, err := strconv.Atoi(hostAndPort[1])
	if err != nil {
		return "", fmt.Errorf("%v: Invalid port", hostAndPort[1])
	}
	return h2c.Connect(host, port)
}

func executeDisconnect(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	return h2c.Disconnect()
}

func executeGet(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	includeHeaders := cmdline.INCLUDE_OPTION.IsSet(cmd.Options)
	var timeout int
	var err error
	if cmdline.TIMEOUT_OPTION.IsSet(cmd.Options) {
		timeout, err = strconv.Atoi(cmdline.TIMEOUT_OPTION.Get(cmd.Options))
		if err != nil {
			return "", fmt.Errorf("%v: invalid timeout", cmdline.TIMEOUT_OPTION.Get(cmd.Options))
		}
	} else {
		timeout = 10
	}
	return h2c.Get(cmd.Args[0], includeHeaders, timeout)
}

func executePushList(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	return h2c.PushList()
}

func executePut(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	return executePutOrPost(h2c, cmd, h2c.Put)
}

func executePost(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	return executePutOrPost(h2c, cmd, h2c.Post)
}

func executePutOrPost(h2c *http2client.Http2Client, cmd *rpc.Command, putOrPost func(path string, data []byte, includeHeaders bool, timeoutInSeconds int) (string, error)) (string, error) {
	includeHeaders := cmdline.INCLUDE_OPTION.IsSet(cmd.Options)
	var timeout int
	var err error
	if cmdline.TIMEOUT_OPTION.IsSet(cmd.Options) {
		timeout, err = strconv.Atoi(cmdline.TIMEOUT_OPTION.Get(cmd.Options))
		if err != nil {
			return "", fmt.Errorf("%v: invalid timeout", cmdline.TIMEOUT_OPTION.Get(cmd.Options))
		}
	} else {
		timeout = 10
	}
	var data []byte
	if cmdline.DATA_OPTION.IsSet(cmd.Options) {
		data = []byte(cmdline.DATA_OPTION.Get(cmd.Options))
	}
	return putOrPost(cmd.Args[0], data, includeHeaders, timeout)
}

func executeCommandAndCloseConnection(h2c *http2client.Http2Client, conn net.Conn, sock net.Listener) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	encodedCmd, err := reader.ReadString('\n')
	cmd, err := rpc.UnmarshalCommand(encodedCmd)
	if err != nil {
		handleCommunicationError("Failed to decode command: %v", err.Error())
		return
	}
	if cmd.Name == cmdline.STOP_COMMAND.Name() {
		writeResult(conn, "", nil)
		stop(sock)
	} else {
		msg, err := execute(h2c, cmd)
		writeResult(conn, msg, err)
	}
}

func writeResult(conn io.Writer, msg string, err error) {
	encodedResult, err := rpc.NewResult(msg, err).Marshal()
	if err != nil {
		handleCommunicationError("Failed to encode result: %v", err)
		return
	}
	_, err = conn.Write([]byte(encodedResult))
	if err != nil {
		handleCommunicationError("Error writing result to socket: %v", err.Error())
		return
	}
}

func handleCommunicationError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error communicating with the h2c command line: %v", fmt.Sprintf(format, a))
}
