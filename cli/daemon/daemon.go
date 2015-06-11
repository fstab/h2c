// Package daemon implements the h2c process, i.e, the process started with 'h2c start'.
package daemon

import (
	"bufio"
	"fmt"
	"github.com/fstab/h2c/cli/commands"
	"github.com/fstab/h2c/http2client"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
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

func execute(h2c *http2client.Http2Client, cmd *commands.Command) (string, error) {
	switch cmd.Name {
	case "connect":
		port, err := strconv.Atoi(cmd.Params["port"])
		if err == nil {
			return h2c.Connect(cmd.Params["host"], port)
		} else {
			return "", fmt.Errorf("%v: Illegal port.", cmd.Params["port"])
		}
	case "pid":
		return strconv.Itoa(os.Getpid()), nil
	case "get":
		return h2c.Get(cmd.Params["path"])
	default:
		return "", fmt.Errorf("%v: unknown command", cmd.Name)
	}
}

func executeCommandAndCloseConnection(h2c *http2client.Http2Client, conn net.Conn, sock net.Listener) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	encodedCmd, err := reader.ReadString('\n')
	cmd, err := commands.UnmarshalCommand(encodedCmd)
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
	encodedResult, err := commands.NewResult(msg, err).Marshal()
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
