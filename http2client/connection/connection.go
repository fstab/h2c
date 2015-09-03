package connection

import (
	"crypto/tls"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/util"
	"github.com/fstab/http2/hpack"
	"io"
	"net"
	"os"
)

const CLIENT_PREFACE = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

type Connection interface {
	Host() string
	Port() int
	IsShutdown() bool
	Shutdown()
	InitNewStream(onClosed *util.AsyncTask) Stream
	ServerFrameSize() uint32
	GetPromisedPaths() []string
	FetchPromisedStream(method, path string) Stream
}

type connection struct {
	in                         chan frames.Frame
	out                        chan *writeFrameRequest
	shutdown                   chan bool
	info                       *info
	settings                   *settings
	streams                    map[uint32]*stream         // StreamID -> *stream
	promisedStreamIDs          map[promisedRequest]uint32 // Push Promise -> StreamID
	conn                       net.Conn
	dump                       bool
	isShutdown                 bool
	encodingContext            *frames.EncodingContext
	decodingContext            *frames.DecodingContext
	remainingSendWindowSize    int64
	remainingReceiveWindowSize int64
}

type promisedRequest struct {
	method string // "GET"
	path   string // "/index.html"
}

type info struct {
	host string
	port int
}

type settings struct {
	serverFrameSize                       uint32
	initialSendWindowSizeForNewStreams    uint32
	initialReceiveWindowSizeForNewStreams uint32
}

type writeFrameRequest struct {
	frame frames.Frame
	task  *util.AsyncTask
}

func Start(host string, port int, dump bool) (Connection, error) {
	hostAndPort := fmt.Sprintf("%v:%v", host, port)
	conn, err := tls.Dial("tcp", hostAndPort, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %v: %v", hostAndPort, err.Error())
	}
	if conn.ConnectionState().NegotiatedProtocol != "h2" {
		return nil, fmt.Errorf("Server does not support HTTP/2 protocol.")
	}
	_, err = conn.Write([]byte(CLIENT_PREFACE))
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

func (c *connection) Shutdown() {
	c.shutdown <- true
}

func (c *connection) IsShutdown() bool {
	return c.isShutdown
}

func (c *connection) GetPromisedPaths() []string {
	result := make([]string, len(c.promisedStreamIDs))
	i := 0
	for k := range c.promisedStreamIDs {
		result[i] = k.path
		i = i + 1
	}
	return result
}

func (c *connection) FetchPromisedStream(method, path string) Stream {
	streamId, exists := c.promisedStreamIDs[promisedRequest{method, path}]
	if exists {
		delete(c.promisedStreamIDs, promisedRequest{method, path})
		return c.streams[streamId]
	} else {
		return nil
	}
}

func newConnection(conn net.Conn, host string, port int, dump bool) *connection {
	return &connection{
		in:       make(chan frames.Frame),
		out:      make(chan *writeFrameRequest),
		shutdown: make(chan bool),
		info: &info{
			host: host,
			port: port,
		},
		settings: &settings{
			serverFrameSize:                       2 << 13,   // Minimum size that must be supported by all server implementations.
			initialSendWindowSizeForNewStreams:    2<<15 - 1, // Initial flow-control window size for new streams is 65,535 octets.
			initialReceiveWindowSizeForNewStreams: 2<<15 - 1,
		},
		streams:                    make(map[uint32]*stream),
		promisedStreamIDs:          make(map[promisedRequest]uint32),
		isShutdown:                 false,
		conn:                       conn,
		dump:                       dump,
		encodingContext:            frames.NewEncodingContext(),
		decodingContext:            frames.NewDecodingContext(),
		remainingSendWindowSize:    2<<15 - 1,
		remainingReceiveWindowSize: 2<<15 - 1,
	}
}

// Frame processing loop. This makes sure that all frames are handled sequentially in a single thread.
func (c *connection) runFrameHandlerLoop() {
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
func (c *connection) runIncomingFrameReader() {
	for {
		frame, err := c.readNextFrame()
		if c.isShutdown {
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading next frame: %v\n", err.Error()) // TODO: Error handling
			fmt.Fprintf(os.Stderr, "Closing connection to %v:%v\n", c.info.host, c.info.port)
			c.Shutdown()
			return
		} else {
			c.in <- frame
		}
	}
}

func (c *connection) handleIncomingFrame(frame frames.Frame) {
	switch frame := frame.(type) {
	case *frames.SettingsFrame:
		c.settings.handleSettingsFrame(frame)
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
		c.flowControlForIncomingDataFrame(frame, stream)
		stream.appendReceivedData(frame.Data)
		if frame.EndStream {
			stream.endStream()
		}
	case *frames.PushPromiseFrame:
		method := findHeader(":method", frame.Headers)
		path := findHeader(":path", frame.Headers)
		if method != "GET" {
			fmt.Fprintf(os.Stderr, "ERROR: PUSH_PROMISE with method %v not supported.", method)
			return
		}
		_, exists := c.streams[frame.PromisedStreamId]
		if exists {
			fmt.Fprintf(os.Stderr, "ERROR: Received PUSH_PROMISE for existing stream %v", frame.PromisedStreamId)
			return
		}
		if !frame.EndHeaders {
			fmt.Fprintf(os.Stderr, "ERROR: Push promise with multiple header frames not supported.")
			return
		}
		c.getOrCreateStream(frame.PromisedStreamId)
		c.promisedStreamIDs[promisedRequest{method, path}] = frame.PromisedStreamId
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
		c.handleWindowUpdateFrame(frame)
	case *frames.GoAwayFrame:
		// TODO: error handling
		fmt.Fprintf(os.Stderr, "Connection closed: Server sent GOAWAY with error code %v", frame.ErrorCode.String())
		c.shutdown <- true
	default:
		// TODO: error handling
		fmt.Fprintf(os.Stderr, "Received unknown frame type %v", frame.Type())
	}
}

func findHeader(name string, headers []hpack.HeaderField) string {
	for _, header := range headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}

// Just a quick implementation to make large downloads work.
// Should be replaced with a more sophisticated flow control strategy
func (c *connection) flowControlForIncomingDataFrame(frame *frames.DataFrame, stream *stream) {
	threshold := int64(2 << 13) // size of one frame
	stream.remainingReceiveWindowSize -= int64(len(frame.Data))
	if stream.remainingReceiveWindowSize < threshold {
		go func() {
			diff := int64(c.settings.initialReceiveWindowSizeForNewStreams) - stream.remainingReceiveWindowSize
			stream.remainingReceiveWindowSize += diff
			c.out <- &writeFrameRequest{
				frame: frames.NewWindowUpdateFrame(stream.streamId, uint32(diff)),
				task:  util.NewAsyncTask(),
			}
		}()
	}
	c.remainingReceiveWindowSize -= int64(len(frame.Data))
	if c.remainingReceiveWindowSize < threshold {
		go func() {
			diff := int64(2<<15-1) - c.remainingReceiveWindowSize
			c.remainingReceiveWindowSize += diff
			c.out <- &writeFrameRequest{
				frame: frames.NewWindowUpdateFrame(0, uint32(diff)),
				task:  util.NewAsyncTask(),
			}
		}()
	}
}

func (s *settings) handleSettingsFrame(frame *frames.SettingsFrame) {
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
		s.initialSendWindowSizeForNewStreams = frames.SETTINGS_INITIAL_WINDOW_SIZE.Get(frame)
	}
	// TODO: Implement other settings, like HEADER_TABLE_SIZE.
}

func (c *connection) handleWindowUpdateFrame(frame *frames.WindowUpdateFrame) {
	if frame.StreamId == 0 {
		c.remainingSendWindowSize += int64(frame.WindowSizeIncrement)
	} else {
		stream, exists := c.streams[frame.GetStreamId()]
		if exists {
			stream.remainingSendWindowSize += int64(frame.WindowSizeIncrement)
		}
	}
	c.processPendingDataFrames()
}

func (c *connection) handleOutgoingFrame(req *writeFrameRequest) {
	_, isDataFrame := req.frame.(*frames.DataFrame)
	if isDataFrame {
		// This is called through stream.Write() so we know that the stream with that id exists.
		c.streams[req.frame.GetStreamId()].scheduleDataFrameWrite(req)
		c.processPendingDataFrames()
	} else {
		c.writeImmediately(req)
	}
}

func (c *connection) writeImmediately(req *writeFrameRequest) {
	encodedFrame, err := req.frame.Encode(c.encodingContext)
	if err != nil {
		req.task.CompleteWithError(fmt.Errorf("Failed to write frame: %v", err.Error()))
	}
	if c.dump {
		frames.DumpOutgoing(req.frame)
	}
	_, err = c.conn.Write(encodedFrame)
	if err != nil {
		req.task.CompleteWithError(fmt.Errorf("Failed to write frame: %v", err.Error()))
	}
	req.task.CompleteSuccessfully()
}

// onFlowControlEvent is called when:
// a) a new data frame is scheduled for writing
// b) the flow control window size has changed
func (c *connection) processPendingDataFrames() {
	for _, s := range c.streams {
		req := s.firstPendingDataFrameWrite()
		if req != nil {
			nBytes := int64(len(req.frame.(*frames.DataFrame).Data))
			if c.remainingSendWindowSize >= nBytes && s.remainingSendWindowSize >= nBytes {
				c.remainingSendWindowSize -= nBytes
				s.remainingSendWindowSize -= nBytes
				s.popFirstPendingDataFrameWrite()
				c.writeImmediately(req)
			}
		}
	}
}

func (c *connection) getOrCreateStream(streamId uint32) *stream {
	stream, ok := c.streams[streamId]
	if !ok {
		stream = newStream(streamId, nil, c.settings.initialSendWindowSizeForNewStreams, c.settings.initialReceiveWindowSizeForNewStreams, c.out)
		c.streams[streamId] = stream
	}
	return stream
}

func (c *connection) GetStreamIfExists(streamId uint32) (*stream, bool) {
	stream, exists := c.streams[streamId]
	return stream, exists
}

func (c *connection) InitNewStream(onClosed *util.AsyncTask) Stream {
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
	c.streams[nextStreamId] = newStream(nextStreamId, onClosed, c.settings.initialSendWindowSizeForNewStreams, c.settings.initialReceiveWindowSizeForNewStreams, c.out)
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

func (c *connection) ServerFrameSize() uint32 {
	return c.settings.serverFrameSize
}

func (c *connection) SetServerFrameSize(size uint32) {
	c.settings.serverFrameSize = size
}

func (c *connection) Host() string {
	return c.info.host
}

func (c *connection) Port() int {
	return c.info.port
}

func (c *connection) readNextFrame() (frames.Frame, error) {
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
