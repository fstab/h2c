package daemon

import (
	"bufio"
	"fmt"
	"github.com/fstab/h2c/cli/messages/command"
	"github.com/fstab/h2c/cli/messages/result"
	"github.com/fstab/h2c/http2client"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
)

// The process started with 'h2c start'.
// It keeps an h2c client instance, reads commands from the socket file, and uses the h2c client to execute these commands.
func Start(socketFilePath string) error {
	var err error
	var conn net.Conn
	var h2c = http2client.New()
	sock, err := net.Listen("unix", socketFilePath)
	if err != nil {
		return fmt.Errorf("Error creating %v: %v", socketFilePath, err.Error())
	}
	defer close(sock, socketFilePath)
	stopOnSigterm(sock, socketFilePath)
	for {
		if conn, err = sock.Accept(); err != nil {
			return fmt.Errorf("Error reading from %v: %v", socketFilePath, err.Error())
			stop(sock, socketFilePath)
		}
		go executeCommandAndCloseConnection(h2c, conn, sock, socketFilePath)
	}
	return nil
}

func close(sock io.Closer, socketFilePath string) {
	if err := sock.Close(); err != nil {
		fmt.Printf("Error removing %v: %v", socketFilePath, err.Error())
	}
}

func stop(sock io.Closer, socketFilePath string) {
	close(sock, socketFilePath)
	os.Exit(0)
}

func stopOnSigterm(sock io.Closer, socketFilePath string) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	go func(c chan os.Signal) {
		<-c // Wait for a SIGINT
		stop(sock, socketFilePath)
	}(sigc)
}

func execute(h2c *http2client.Http2Client, cmd *command.Command) (string, error) {
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
	default:
		return "", fmt.Errorf("%v: unknown command", cmd.Name)
	}
}

func executeCommandAndCloseConnection(h2c *http2client.Http2Client, conn net.Conn, sock io.Closer, socketFilePath string) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	encodedCmd, err := reader.ReadString('\n')
	cmd, err := command.Decode(encodedCmd)
	if err != nil {
		handleCommunicationError("Failed to decode command: %v", err.Error())
		return
	}
	if cmd.Name == "stop" {
		writeResult(conn, "", nil)
		stop(sock, socketFilePath)
	} else {
		msg, err := execute(h2c, cmd)
		writeResult(conn, msg, err)
	}
}

func writeResult(conn io.Writer, msg string, err error) {
	encodedResult, err := result.New(msg, err).Encode()
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
