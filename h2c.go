package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
)

var sock net.Listener

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "start":
		start()
	case "stop":
		sendCommand("stop")
	case "echo":
		sendCommand("echo")
	default:
		usage()
	}
}

func start() {
	var err error
	var conn net.Conn
	if sock, err = net.Listen("unix", "/tmp/h2c.sock"); err != nil {
		log.Fatal("Listen error:", err)
	}
	stopOnSigterm()
	for {
		if conn, err = sock.Accept(); err != nil {
			log.Fatal("accept error:", err)
		}
		go executeCommandAndCloseConnection(conn)
	}
}

func stop() {
	sock.Close()
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

func executeCommandAndCloseConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	cmd, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Error reading command from socket:", err)
	}
	switch strings.TrimSpace(cmd) {
	case "stop":
		defer stop()
	default:
		fmt.Printf("Server got: %v", cmd)
		_, err = conn.Write([]byte(cmd))
		if err != nil {
			log.Fatal("Write:", err)
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
