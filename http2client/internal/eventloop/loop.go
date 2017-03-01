package eventloop

import (
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/connection"
	"github.com/fstab/h2c/http2client/internal/eventloop/commands"
	"os"
	"crypto/tls"
)

type Loop struct {
	HttpCommands       chan (*commands.HttpCommand)
	MonitoringCommands chan (*commands.MonitoringCommand)
	PingCommands       chan (*commands.PingCommand)
	IncomingFrames     chan (frames.Frame)
	Shutdown           chan (bool)
	Host               string
	Port               int
	terminated         bool
}

// Start starts the event loop managing the HTTP/2 communication with a server.
//
// A lot of HTTP/2 features are hard to implement in a thread safe way:
//  * Push promises arriving while the client sends a request at the same time.
//  * Window updates increase the flow control window while the client sends data
//    and decreases the flow control window at the same time.
//  * etc.
//
// Therefore, each HTTP/2 connection is handled single thread in h2c
// (that is, h2c avoids concurrency problems by being single-threaded per connection).
//
// The eventloop takes all events, and executes them sequentially in a single thread.
// The implementation in github.com/fstab/h2c/http2client/connection does not need
// to care about thread safety.
//
// There are two sources of events:
//
// 1. Command line: A user types a comand in order to send a GET, POST, ... request.
// 2. Network Socket: Frames received from the server.
func Start(host string, port int, incomingFrameFilters []func(frames.Frame) frames.Frame, outgoingFrameFilters []func(frames.Frame) frames.Frame, tlsConfig *tls.Config) (*Loop, error) {
	l := &Loop{
		HttpCommands:       make(chan (*commands.HttpCommand)),
		MonitoringCommands: make(chan (*commands.MonitoringCommand)),
		PingCommands:       make(chan (*commands.PingCommand)),
		IncomingFrames:     make(chan (frames.Frame)),
		Shutdown:           make(chan (bool)),
		Host:               host,
		Port:               port,
		terminated:         false,
	}
	conn, err := connection.Start(host, port, incomingFrameFilters, outgoingFrameFilters, tlsConfig)
	stopFrameReader := false
	if err != nil {
		return nil, err
	}
	// Start event loop
	go func() {
		for {
			select {
			case frame := <-l.IncomingFrames:
				conn.HandleIncomingFrame(frame)
			case cmd := <-l.HttpCommands:
				conn.ExecuteHttpCommand(cmd)
			case cmd := <-l.PingCommands:
				conn.ExecutePingCommand(cmd)
			case cmd := <-l.MonitoringCommands:
				conn.ExecuteMonitoringCommand(cmd)
			case <-l.Shutdown:
				conn.Shutdown()
			}
			if conn.IsShutdown() {
				stopFrameReader = true // Would this change be visible in the other go function?
				l.terminated = true
				return
			}
		}
	}()
	// Read frames from network socket and provide them to the IncomingFrames channel
	go func() {
		for {
			frame, err := conn.ReadNextFrame()
			if stopFrameReader {
				return
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while reading next frame: %v\n", err.Error()) // TODO: Error handling
				conn.Shutdown()
				l.terminated = true
				return
			} else {
				l.IncomingFrames <- frame
			}
		}
	}()
	return l, nil
}

func (l *Loop) IsTerminated() bool {
	return l.terminated
}
