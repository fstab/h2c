package main

import (
	"bufio"
	"fmt"
	"github.com/fstab/h2c/http2client"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
)

var sock net.Listener

const socketFile = "/tmp/h2c.sock"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "start":
			if err := start(); err != nil {
				log.Fatal(err)
			}
		default:
			sendCommand(strings.Join(os.Args[1:], " "))
		}
	} else {
		usage()
	}
}

func start() error {
	var err error
	var conn net.Conn
	var h2c = http2client.New()
	if sock, err = net.Listen("unix", socketFile); err != nil {
		return fmt.Errorf("Error creating %v: %v", socketFile, err.Error())
	}
	defer close()
	stopOnSigterm()
	for {
		if conn, err = sock.Accept(); err != nil {
			return fmt.Errorf("Error reading from %v: %v", socketFile, err.Error())
			stop()
		}
		go executeCommandAndCloseConnection(h2c, conn)
	}
	return nil
}

func close() {
	if err := sock.Close(); err != nil {
		fmt.Printf("Error removing %v: %v", socketFile, err.Error())
	}
}

func stop() {
	close()
	os.Exit(0)
}

func stopOnSigterm() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	go func(c chan os.Signal) {
		<-c // Wait for a SIGINT
		stop()
	}(sigc)
}

func sendCommand(cmd string) {
	conn, err := net.Dial("unix", "/tmp/h2c.sock")
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(cmd + "\n")
	if err != nil {
		log.Fatal("Write error:", err)
	}
	writer.Flush()
	if err != nil {
		log.Fatal("Write error:", err)
	}
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			return
		}
		fmt.Printf("Client got: %v", string(buf[0:n]))
	}
}

func executeCommandAndCloseConnection(h2c *http2client.Http2Client, conn net.Conn) {
	reader := bufio.NewReader(conn)
	cmd, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Error reading command from socket:", err)
	}
	var (
		result string
	)
	cmd = strings.TrimSpace(cmd)
	args := strings.Split(cmd, " ")
	if len(args) < 1 {
		err = fmt.Errorf("Received empty command.")
	} else {
		switch args[0] {
		case "stop":
			defer stop()
		case "connect":
			if len(args) == 2 {
				result, err = h2c.Connect(args[1], "443")
			} else if len(args) == 3 {
				result, err = h2c.Connect(args[1], args[2])
			} else {
				result = "Usage: connect <host> <port>"
			}
		default:
			err = fmt.Errorf("%v: Unknown command.")
		}
		if err != nil {
			result = "ERROR: " + err.Error()
		}
		_, err = conn.Write([]byte(result))
		if err != nil {
			log.Fatal("Write Error:", err)
		}
	}
	err = conn.Close()
	if err != nil {
		log.Fatal("Close error???")
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "h2s start")
	fmt.Fprintln(os.Stderr, "h2s echo")
	os.Exit(-1)
}
