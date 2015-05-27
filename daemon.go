package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fstab/h2c/http2client"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

// Daemon process that keeps an h2c client instance, reads commands from /tmp/h2c.sock, and uses the h2c client to execute these commands.

func start(socketFilePath string) error {
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
		go func() {
			if err := executeCommandAndCloseConnection(h2c, conn, sock, socketFilePath); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				close(sock, socketFilePath)
				os.Exit(-1)
			}
		}()
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

func execute(h2c *http2client.Http2Client, cmd *Command) string {
	var (
		result string
		err    error
	)
	switch cmd.Name {
	case "connect":
		result, err = h2c.Connect(cmd.Params["url"])
	case "pid":
		result = strconv.Itoa(os.Getpid())
	default:
		err = fmt.Errorf("%v: unknown command", cmd.Name)
	}
	if err != nil {
		return err.Error()
	}
	return result
}

func executeCommandAndCloseConnection(h2c *http2client.Http2Client, conn net.Conn, sock io.Closer, socketFilePath string) error {
	reader := bufio.NewReader(conn)
	base64data, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Error reading command from UNIX socket: %v", err.Error())
	}
	jsonData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(base64data))
	cmd := &Command{}
	if err != nil {
		return fmt.Errorf("Error reading command from UNIX socket: %v", err.Error())
	}
	err = json.Unmarshal(jsonData, cmd)
	if err != nil {
		return fmt.Errorf("Error reading command from UNIX socket: %v", err.Error())
	}
	var response string
	if cmd.Name == "stop" {
		defer stop(sock, socketFilePath)
		response = ""
	} else {
		response = execute(h2c, cmd)
	}
	_, err = conn.Write([]byte(response))
	if err != nil {
		return fmt.Errorf("Error writing response to UNIX socket: %v", err.Error())
	}
	err = conn.Close()
	if err != nil {
		return fmt.Errorf("Error closing connection to UNIX socket: %v", err.Error())
	}
	return nil
}
