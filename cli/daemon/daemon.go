// Package daemon implements the h2c process, i.e, the process started with 'h2c start'.
package daemon

import (
	"bufio"
	"fmt"
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
func Run(sock net.Listener) error {
	var conn net.Conn
	var err error
	var h2c = http2client.New()
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
	case "connect":
		return executeConnect(h2c, cmd)
	case "pid":
		return strconv.Itoa(os.Getpid()), nil
	case "get":
		return executeGet(h2c, cmd)
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

func executeGet(h2c *http2client.Http2Client, cmd *rpc.Command) (string, error) {
	_, includeHeaders := cmd.Options["--include"]
	timeoutString, exists := cmd.Options["--timeout"]
	var timeout int
	var err error
	if exists {
		timeout, err = strconv.Atoi(timeoutString)
		if err != nil {
			return "", fmt.Errorf("%v: invalid timeout", timeoutString)
		}
	} else {
		timeout = 10
	}
	return h2c.Get(cmd.Args[0], includeHeaders, timeout)
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
	if cmd.Name == "stop" {
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
