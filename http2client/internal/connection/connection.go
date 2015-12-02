package connection

import (
	"crypto/tls"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/message"
	"github.com/fstab/h2c/http2client/internal/util"
	"golang.org/x/net/http2/hpack"
	"io"
	"net"
	"os"
)

const CLIENT_PREFACE = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

// Some of these methods may no longer be needed after the last refactoring. Need to clean up.
type Connection interface {
	Host() string
	Port() int
	Error() error
	Shutdown()
	NewStream(request message.HttpRequest) Stream
	ServerFrameSize() uint32
	FetchPromisedStream(path string) Stream
	Write(stream Stream, frame frames.Frame)
	HandleIncomingFrame(frame frames.Frame)
	ReadNextFrame() (frames.Frame, error)
	HandleHttpRequest(request message.HttpRequest)
	HandleMonitoringRequest(request message.MonitoringRequest)
}

type connection struct {
	in                         chan frames.Frame
	shutdown                   chan bool
	info                       *info
	settings                   *settings
	streams                    map[uint32]*stream // StreamID -> *stream
	promisedStreamIDs          map[string]uint32  // Push Promise Path -> StreamID
	conn                       net.Conn
	isShutdown                 bool
	encodingContext            *frames.EncodingContext
	decodingContext            *frames.DecodingContext
	remainingSendWindowSize    int64
	remainingReceiveWindowSize int64
	incomingFrameFilters       []func(frames.Frame) frames.Frame
	outgoingFrameFilters       []func(frames.Frame) frames.Frame
	err                        error // TODO: not used
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

func Start(host string, port int, incomingFrameFilters []func(frames.Frame) frames.Frame, outgoingFrameFilters []func(frames.Frame) frames.Frame) (Connection, error) {
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
	c := newConnection(conn, host, port, incomingFrameFilters, outgoingFrameFilters)
	c.Write(nil, frames.NewSettingsFrame(0))
	return c, nil
}

func (conn *connection) HandleHttpRequest(request message.HttpRequest) {
	if conn.Error() != nil {
		request.CompleteWithError(conn.Error())
	}
	switch request.GetHeader(":method") {
	case "GET":
		conn.handleGetRequest(request)
	case "PUT":
		conn.handlePutRequest(request)
	case "POST":
		conn.handlePostRequest(request)
	default:
		request.CompleteWithError(fmt.Errorf("Request method '%v' not supported.", request.GetHeader(":method")))
	}
}

func (conn *connection) handleGetRequest(request message.HttpRequest) {
	stream := conn.FetchPromisedStream(request.GetHeader(":path"))
	if stream != nil {
		// Don't need to send request, because PUSH_PROMISE for this request already arrived.
		err := stream.AssociateWithRequest(request)
		if err != nil {
			request.CompleteWithError(err)
		}
	} else {
		conn.doRequest(request)
	}
}

func (conn *connection) handlePutRequest(request message.HttpRequest) {
	conn.doRequest(request)
}

func (conn *connection) handlePostRequest(request message.HttpRequest) {
	conn.doRequest(request)
}

func (conn *connection) doRequest(request message.HttpRequest) {
	stream := conn.NewStream(request)
	headersFrame := frames.NewHeadersFrame(stream.StreamId(), request.GetHeaders())
	headersFrame.EndStream = request.GetData() == nil
	conn.Write(stream, headersFrame)
	if request.GetData() != nil {
		conn.sendDataFrames(request.GetData(), stream)
	}
}

func (conn *connection) sendDataFrames(data []byte, stream Stream) {
	// chunkSize := uint32(len(data)) // use this to provoke GOAWAY frame with FRAME_SIZE_ERROR
	chunkSize := conn.ServerFrameSize() // TODO: Query chunk size with each iteration -> allow changes during loop
	nChunksSent := uint32(0)
	total := uint32(len(data))
	for nChunksSent*chunkSize < total {
		nextChunk := data[nChunksSent*chunkSize : min((nChunksSent+1)*chunkSize, total)]
		nChunksSent = nChunksSent + 1
		isLast := nChunksSent*chunkSize >= total
		dataFrame := frames.NewDataFrame(stream.StreamId(), nextChunk, isLast)
		conn.Write(stream, dataFrame)
	}
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func (c *connection) HandleMonitoringRequest(request message.MonitoringRequest) {
	response := message.NewMonitoringResponse()
	for path := range c.promisedStreamIDs {
		response.AddPromisedPath(path)
	}
	request.CompleteSuccessfully(response)
}

func (c *connection) FetchPromisedStream(path string) Stream {
	streamId, exists := c.promisedStreamIDs[path]
	if exists {
		delete(c.promisedStreamIDs, path)
		return c.streams[streamId]
	} else {
		return nil
	}
}

func newConnection(conn net.Conn, host string, port int, incomingFrameFilters []func(frames.Frame) frames.Frame, outgoingFrameFilters []func(frames.Frame) frames.Frame) *connection {
	return &connection{
		in:       make(chan frames.Frame),
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
		promisedStreamIDs:          make(map[string]uint32),
		isShutdown:                 false,
		conn:                       conn,
		encodingContext:            frames.NewEncodingContext(),
		decodingContext:            frames.NewDecodingContext(),
		remainingSendWindowSize:    2<<15 - 1,
		remainingReceiveWindowSize: 2<<15 - 1,
		incomingFrameFilters:       incomingFrameFilters,
		outgoingFrameFilters:       outgoingFrameFilters,
	}
}

func (c *connection) Shutdown() {
	c.isShutdown = true
	c.conn.Close()
}

func (c *connection) HandleIncomingFrame(frame frames.Frame) {
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
		c.promisedStreamIDs[path] = frame.PromisedStreamId
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
		diff := int64(c.settings.initialReceiveWindowSizeForNewStreams) - stream.remainingReceiveWindowSize
		stream.remainingReceiveWindowSize += diff
		c.Write(stream, frames.NewWindowUpdateFrame(stream.streamId, uint32(diff)))
	}
	c.remainingReceiveWindowSize -= int64(len(frame.Data))
	if c.remainingReceiveWindowSize < threshold {
		diff := int64(2<<15-1) - c.remainingReceiveWindowSize
		c.remainingReceiveWindowSize += diff
		c.Write(nil, frames.NewWindowUpdateFrame(0, uint32(diff)))
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

func (c *connection) Write(stream Stream, frame frames.Frame) {
	_, isDataFrame := frame.(*frames.DataFrame)
	if isDataFrame {
		stream.scheduleDataFrameWrite(frame.(*frames.DataFrame))
		c.processPendingDataFrames()
	} else {
		c.writeImmediately(frame)
	}
}

func (c *connection) writeImmediately(frame frames.Frame) {
	encodedFrame, err := frame.Encode(c.encodingContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode frame: %v", err.Error())
		os.Exit(-1)
	}
	if c.outgoingFrameFilters != nil {
		for _, filter := range c.outgoingFrameFilters {
			frame = filter(frame)
		}
	}
	_, err = c.conn.Write(encodedFrame)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write frame: %v", err.Error())
	}
}

// processPendingDataFrames is called when:
// a) a new data frame is scheduled for writing
// b) the flow control window size has changed
func (c *connection) processPendingDataFrames() {
	frameSent := true
	for frameSent { // As long as this loop sends out a frame, we loop again. Only if no frame was sent, we stop.
		frameSent = false
		for _, s := range c.streams {
			frame := s.firstPendingDataFrameWrite()
			if frame != nil {
				nBytes := int64(len(frame.Data))
				if c.remainingSendWindowSize >= nBytes && s.remainingSendWindowSize >= nBytes {
					c.remainingSendWindowSize -= nBytes
					s.remainingSendWindowSize -= nBytes
					s.popFirstPendingDataFrameWrite()
					c.writeImmediately(frame)
					frameSent = true
				}
			}
		}
	}
}

func (c *connection) getOrCreateStream(streamId uint32) *stream {
	stream, ok := c.streams[streamId]
	if !ok {
		stream = newStream(streamId, nil, c.settings.initialSendWindowSizeForNewStreams, c.settings.initialReceiveWindowSizeForNewStreams)
		c.streams[streamId] = stream
	}
	return stream
}

func (c *connection) GetStreamIfExists(streamId uint32) (*stream, bool) {
	stream, exists := c.streams[streamId]
	return stream, exists
}

func (c *connection) NewStream(request message.HttpRequest) Stream {
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
	c.streams[nextStreamId] = newStream(nextStreamId, request, c.settings.initialSendWindowSizeForNewStreams, c.settings.initialReceiveWindowSizeForNewStreams)
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

func (c *connection) Error() error {
	return c.err
}

func (c *connection) ReadNextFrame() (frames.Frame, error) {
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
	if c.incomingFrameFilters != nil {
		for _, filter := range c.incomingFrameFilters {
			frame = filter(frame)
		}
	}
	return frame, err
}
