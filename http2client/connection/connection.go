package connection

import (
	"crypto/tls"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/util"
	"go.googlesource.com/go/src/io"
	"net"
	"os"
)

type Connection struct {
	in              chan frames.Frame
	out             chan *writeFrameRequest
	shutdown        chan bool
	info            *info
	settings        *settings
	streams         map[uint32]*Stream // StreamID -> *stream
	conn            net.Conn
	dump            bool
	isShutdown      bool
	encodingContext *frames.EncodingContext
	decodingContext *frames.DecodingContext
}

type info struct {
	host string
	port int
}

type settings struct {
	serverFrameSize                uint32
	initialWindowSizeForNewStreams uint32
}

type writeFrameRequest struct {
	frame frames.Frame
	task  *util.AsyncTask
}

func Start(host string, port int, dump bool) (*Connection, error) {
	hostAndPort := fmt.Sprintf("%v:%v", host, port)
	conn, err := tls.Dial("tcp", hostAndPort, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %v: %v", hostAndPort, err.Error())
	}
	_, err = conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")) // client preface
	if err != nil {
		return nil, fmt.Errorf("Failed to write client preface to %v: %v", hostAndPort, err.Error())
	}
	c := newConnection(conn, host, port, dump)
	go c.runFrameHandlerLoop()
	go c.runIncomingFrameReader()
	task := util.NewAsyncTask()
	c.out <- &writeFrameRequest{
		task:  task,
		frame: frames.NewSettingsFrame(0),
	}
	err = task.WaitForCompletion(10)
	if err != nil {
		c.Shutdown()
		return nil, err
	}
	return c, nil
}

func (c *Connection) Shutdown() {
	c.shutdown <- true
}

func newConnection(conn net.Conn, host string, port int, dump bool) *Connection {
	return &Connection{
		in:       make(chan frames.Frame),
		out:      make(chan *writeFrameRequest),
		shutdown: make(chan bool),
		info: &info{
			host: host,
			port: port,
		},
		settings: &settings{
			serverFrameSize:                2 << 13,   // Minimum size that must be supported by all server implementations.
			initialWindowSizeForNewStreams: 2<<15 - 1, // Initial flow-control window size for new streams is 65,535 octets.
		},
		streams:         make(map[uint32]*Stream),
		isShutdown:      false,
		conn:            conn,
		dump:            dump,
		encodingContext: frames.NewEncodingContext(),
		decodingContext: frames.NewDecodingContext(),
	}
}

// Frame processing loop. This makes sure that all frames are handled sequentially in a single thread.
func (c *Connection) runFrameHandlerLoop() {
	for {
		select {
		case incomingFrame := <-c.in:
			c.handleIncomingFrame(incomingFrame)
		case req := <-c.out:
			c.handleOutgoingFrame(req)
		case <-c.shutdown:
			c.isShutdown = true
			c.conn.Close()
			return
		}
	}
}

// Read frames from network socket and provide them to c.in channel
func (c *Connection) runIncomingFrameReader() {
	for {
		frame, err := c.readNextFrame()
		if c.isShutdown {
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading next frame: %v", err.Error()) // TODO: Error handling
		}
		c.in <- frame
	}
}

func (c *Connection) handleIncomingFrame(frame frames.Frame) {
	switch frame := frame.(type) {
	case *frames.SettingsFrame:
		c.settings.apply(frame)
	case *frames.HeadersFrame:
		stream := c.getOrCreateStream(frame.GetStreamId())
		stream.addReceivedHeaders(frame.Headers...)
		// TODO: continuations
		// TODO: error handling
		if frame.EndStream {
			stream.endStream()
		}
	case *frames.DataFrame:
		stream, exists := c.streams[frame.GetStreamId()]
		if !exists {
			// TODO: error handling
			fmt.Fprintf(os.Stderr, "Received data for unknown stream %v. Ignoring this frame.", frame.GetStreamId())
			return
		}
		stream.appendReceivedData(frame.Data)
		if frame.EndStream {
			stream.endStream()
		}
	case *frames.RstStreamFrame:
		stream, exists := c.streams[frame.GetStreamId()]
		if !exists {
			// TODO: error handling
			fmt.Fprintf(os.Stderr, "Received data for unknown stream %v. Ignoring this frame.", frame.GetStreamId())
			return
		}
		stream.setError(fmt.Errorf("ERROR: Server sent RST_STREAM with error code %v.", frame.ErrorCode.String()))
		stream.endStream()
	case *frames.WindowUpdateFrame:
		// TODO: implement flow control
	case *frames.GoAwayFrame:
		// TODO: error handling
		fmt.Fprintf(os.Stderr, "Connection closed: Server sent GOAWAY with error code %v", frame.ErrorCode.String())
		c.shutdown <- true
	default:
		// TODO: error handling
		fmt.Fprintf(os.Stderr, "Received unknown frame type %v", frame.Type())
	}
}

func (s *settings) apply(frame *frames.SettingsFrame) {
	if frames.SETTINGS_MAX_FRAME_SIZE.IsSet(frame) {
		s.serverFrameSize = (frames.SETTINGS_MAX_FRAME_SIZE.Get(frame))
	}
	if frames.SETTINGS_INITIAL_WINDOW_SIZE.IsSet(frame) {
		// TODO: This only covers the INITIAL_WINDOW_SIZE setting in the connection preface phase.
		// TODO: Once the connection is established, the following needs to be implemented:
		// TODO: When the value of SETTINGS_INITIAL_WINDOW_SIZE changes,
		// TODO: a receiver MUST adjust the size of all stream flow-control windows that it maintains
		// TODO: by the difference between the new value and the old value.
		// TODO: See Section 6.9.2 in the spec.
		s.initialWindowSizeForNewStreams = frames.SETTINGS_INITIAL_WINDOW_SIZE.Get(frame)
	}
	// TODO: Implement other settings, like HEADER_TABLE_SIZE.
}

func (c *Connection) handleOutgoingFrame(req *writeFrameRequest) {
	data, err := req.frame.Encode(c.encodingContext)
	if err != nil {
		req.task.CompleteWithError(fmt.Errorf("Failed to write frame: %v", err.Error()))
	}
	if c.dump {
		frames.DumpOutgoing(req.frame)
	}
	_, err = c.conn.Write(data)
	if err != nil {
		req.task.CompleteWithError(fmt.Errorf("Failed to write frame: %v", err.Error()))
	}
	req.task.CompleteSuccessfully()
}

func (c *Connection) getOrCreateStream(streamId uint32) *Stream {
	stream, ok := c.streams[streamId]
	if !ok {
		stream = newStream(streamId, nil, c.settings.initialWindowSizeForNewStreams, c.out)
		c.streams[streamId] = stream
	}
	return stream
}

func (c *Connection) GetStreamIfExists(streamId uint32) (*Stream, bool) {
	stream, exists := c.streams[streamId]
	return stream, exists
}

func (c *Connection) InitNewStream(onClosed *util.AsyncTask) *Stream {
	// Streams initiated by the client must use odd-numbered stream identifiers.
	streamIdsInUse := make([]uint32, len(c.streams))
	for id, _ := range c.streams {
		if id%2 == 1 {
			streamIdsInUse = append(streamIdsInUse, id)
		}
	}
	nextStreamId := uint32(1)
	if len(streamIdsInUse) > 0 {
		nextStreamId = max(streamIdsInUse) + 2
	}
	c.streams[nextStreamId] = newStream(nextStreamId, onClosed, c.settings.initialWindowSizeForNewStreams, c.out)
	return c.streams[nextStreamId]
}

func max(numbers []uint32) uint32 {
	if numbers == nil || len(numbers) == 0 {
		return 0
	}
	result := numbers[0]
	for _, n := range numbers {
		if n > result {
			result = n
		}
	}
	return result
}

func (c *Connection) ServerFrameSize() uint32 {
	return c.settings.serverFrameSize
}

func (c *Connection) SetServerFrameSize(size uint32) {
	c.settings.serverFrameSize = size
}

func (c *Connection) SetInitialWindowSizeForNewStreams(size uint32) {
	c.settings.initialWindowSizeForNewStreams = size
}

func (c *Connection) Host() string {
	return c.info.host
}

func (c *Connection) Port() int {
	return c.info.port
}

func (c *Connection) readNextFrame() (frames.Frame, error) {
	headerData := make([]byte, 9) // Frame starts with a 9 Bytes header
	_, err := io.ReadFull(c.conn, headerData)
	if err != nil {
		return nil, err
	}
	header := frames.DecodeHeader(headerData)
	payload := make([]byte, header.Length)
	_, err = io.ReadFull(c.conn, payload)
	if err != nil {
		return nil, err
	}
	decodeFunc := frames.FindDecoder(frames.Type(header.HeaderType))
	if decodeFunc == nil {
		return nil, fmt.Errorf("%v: Unknown frame type.", header.HeaderType)
	}
	frame, err := decodeFunc(header.Flags, header.StreamId, payload, c.decodingContext)
	if c.dump {
		frames.DumpIncoming(frame)
	}
	return frame, err
}
