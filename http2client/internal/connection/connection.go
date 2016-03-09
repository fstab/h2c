package connection

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/eventloop/commands"
	"github.com/fstab/h2c/http2client/internal/stream"
	"github.com/fstab/h2c/http2client/internal/streamstate"
	"github.com/fstab/h2c/http2client/internal/util"
	"golang.org/x/net/http2/hpack"
	"io"
	"net"
	"os"
)

const CLIENT_PREFACE = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

// Some of these methods may no longer be needed after the last refactoring. Need to clean up.
type Connection interface {
	HandleIncomingFrame(frame frames.Frame)
	ExecuteHttpCommand(cmd *commands.HttpCommand)
	ExecuteMonitoringCommand(cmd *commands.MonitoringCommand)
	ExecutePingCommand(cmd *commands.PingCommand)
	ReadNextFrame() (frames.Frame, error)
	Shutdown()
	IsShutdown() bool
}

type connection struct {
	info                       *info
	settings                   *settings
	streams                    map[uint32]stream.Stream // StreamID -> *stream
	promisedStreamCache        map[uint32]stream.Stream // StreamID -> *stream
	nextPingId                 uint64
	pendingPingCommands        map[uint64]*commands.PingCommand
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
	supportedProtocols := []string{"h2", "h2-16"} // The netty server still uses h2-16, treat it as if it was h2.
	conn, err := tls.Dial("tcp", hostAndPort, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         supportedProtocols,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %v: %v", hostAndPort, err.Error())
	}
	if !util.SliceContainsString(supportedProtocols, conn.ConnectionState().NegotiatedProtocol) {
		return nil, fmt.Errorf("Server does not support HTTP/2 protocol.")
	}
	_, err = conn.Write([]byte(CLIENT_PREFACE))
	if err != nil {
		return nil, fmt.Errorf("Failed to write client preface to %v: %v", hostAndPort, err.Error())
	}
	c := newConnection(conn, host, port, incomingFrameFilters, outgoingFrameFilters)
	c.Write(frames.NewSettingsFrame(0))
	return c, nil
}

func (conn *connection) ExecuteHttpCommand(cmd *commands.HttpCommand) {
	if conn.error() != nil {
		cmd.CompleteWithError(conn.error())
	}
	switch cmd.Request.GetHeader(":method") {
	case "GET":
		conn.executeGetCommand(cmd)
	case "PUT":
		conn.executePutCommand(cmd)
	case "POST":
		conn.executePostCommand(cmd)
	case "":
		cmd.CompleteWithError(errors.New("Received HttpCommand without ':method' header. This is a bug."))
	default:
		cmd.CompleteWithError(fmt.Errorf("Request method '%v' not supported.", cmd.Request.GetHeader(":method")))
	}
}

func (conn *connection) executeGetCommand(cmd *commands.HttpCommand) {
	stream := conn.findStreamCreatedWithPushPromise(cmd.Request.GetHeader(":path"))
	if stream != nil {
		// Remove from cache -> Push Promises only used once
		delete(conn.promisedStreamCache, stream.StreamId())
		// Don't need to send request, because PUSH_PROMISE for this request already arrived.
		err := stream.AssociateWithCommand(cmd)
		if err != nil {
			cmd.CompleteWithError(err)
		}
	} else {
		conn.doRequest(cmd)
	}
}

func (conn *connection) executePutCommand(cmd *commands.HttpCommand) {
	conn.doRequest(cmd)
}

func (conn *connection) executePostCommand(cmd *commands.HttpCommand) {
	conn.doRequest(cmd)
}

func (conn *connection) doRequest(cmd *commands.HttpCommand) {
	stream := conn.newStream(cmd)
	headersFrame := frames.NewHeadersFrame(stream.StreamId(), cmd.Request.GetHeaders())
	headersFrame.EndStream = len(cmd.Request.GetBody()) == 0
	stream.SendFrame(headersFrame)
	if len(cmd.Request.GetBody()) > 0 {
		conn.sendDataFrames(cmd.Request.GetBody(), stream)
	}
}

func (conn *connection) sendDataFrames(data []byte, stream stream.Stream) {
	// chunkSize := uint32(len(data)) // use this to provoke GOAWAY frame with FRAME_SIZE_ERROR
	chunkSize := conn.serverFrameSize() // TODO: Query chunk size with each iteration -> allow changes during loop
	nChunksSent := uint32(0)
	total := uint32(len(data))
	for nChunksSent*chunkSize < total {
		nextChunk := data[nChunksSent*chunkSize : min((nChunksSent+1)*chunkSize, total)]
		nChunksSent = nChunksSent + 1
		isLast := nChunksSent*chunkSize >= total
		dataFrame := frames.NewDataFrame(stream.StreamId(), nextChunk, isLast)
		stream.SendFrame(dataFrame)
	}
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func (c *connection) ExecuteMonitoringCommand(cmd *commands.MonitoringCommand) {
	for _, s := range c.streams {
		_, isCachedPushPromise := c.promisedStreamCache[s.StreamId()]
		cmd.Result.AddStreamInfo(s.StreamId(), findHeader(":method", s.RequestHeaders()), findHeader(":path", s.RequestHeaders()), s.GetState(), isCachedPushPromise)
	}
	cmd.CompleteSuccessfully()
}

func (c *connection) findStreamCreatedWithPushPromise(path string) stream.Stream {
	var result stream.Stream = nil
	for _, stream := range c.promisedStreamCache {
		if findHeader(":method", stream.RequestHeaders()) == "GET" &&
			findHeader(":path", stream.RequestHeaders()) == path {
			result = stream
		}
	}
	return result
}

func (c *connection) ExecutePingCommand(cmd *commands.PingCommand) {
	pingFrame := frames.NewPingFrame(0, c.nextPingId, false)
	c.nextPingId = c.nextPingId + 1
	c.pendingPingCommands[pingFrame.Payload] = cmd
	c.Write(pingFrame)
}

func newConnection(conn net.Conn, host string, port int, incomingFrameFilters []func(frames.Frame) frames.Frame, outgoingFrameFilters []func(frames.Frame) frames.Frame) *connection {
	return &connection{
		info: &info{
			host: host,
			port: port,
		},
		settings: &settings{
			serverFrameSize:                       2 << 13,   // Minimum size that must be supported by all server implementations.
			initialSendWindowSizeForNewStreams:    2<<15 - 1, // Initial flow-control window size for new streams is 65,535 octets.
			initialReceiveWindowSizeForNewStreams: 2<<15 - 1,
		},
		streams:                    make(map[uint32]stream.Stream),
		promisedStreamCache:        make(map[uint32]stream.Stream),
		pendingPingCommands:        make(map[uint64]*commands.PingCommand),
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

func (c *connection) IsShutdown() bool {
	return c.isShutdown
}

func (c *connection) HandleIncomingFrame(frame frames.Frame) {
	streamId := frame.GetStreamId()
	if streamId == 0 {
		c.handleFrameForConnection(frame)
	} else {
		c.handleFrameForStream(frame)
	}
}

func (c *connection) handleFrameForConnection(frame frames.Frame) {
	switch frame := frame.(type) {
	case *frames.SettingsFrame:
		c.settings.handleSettingsFrame(frame)
	case *frames.PingFrame:
		if frame.Ack {
			pendingPingCommand, exists := c.pendingPingCommands[frame.Payload]
			if exists {
				delete(c.pendingPingCommands, frame.Payload)
				pendingPingCommand.CompleteSuccessfully()
			}
		} else {
			pingFrame := frames.NewPingFrame(0, frame.Payload, true)
			c.Write(pingFrame)
		}
	case *frames.WindowUpdateFrame:
		c.handleWindowUpdateFrame(frame)
	case *frames.GoAwayFrame:
		c.Shutdown()
	default:
		msg := fmt.Sprintf("Received %v frame with stream identifier 0x00.", frame.Type())
		c.connectionError(frames.PROTOCOL_ERROR, msg)
	}
}

func (c *connection) connectionError(errorCode frames.ErrorCode, msg string) {
	// TODO:
	//   * Find highest stream id that was successfully processed
	//   * Send GO_AWAY frame with error code (maybe msg as additional debug data)
	//   * Shut down connection
	fmt.Fprintf(os.Stderr, "%v Should send GOAWAY frame with error code %v, but this is not implemented yet.\n", msg, errorCode)
}

func (c *connection) handleFrameForStream(frame frames.Frame) {
	switch frame := frame.(type) {
	case *frames.PushPromiseFrame:
		c.handleIncomingPushPromiseFrame(frame)
	case *frames.DataFrame:
		c.handleIncomingDataFrame(frame)
	case *frames.RstStreamFrame:
		c.handleIncomingRstStreamFrame(frame)
	default:
		c.getOrCreateStream(frame.GetStreamId()).ReceiveFrame(frame)
	}
}

func (c *connection) handleIncomingDataFrame(frame *frames.DataFrame) {
	c.flowControlForIncomingDataFrame(frame)
	c.getOrCreateStream(frame.StreamId).ReceiveFrame(frame)
}

func (c *connection) handleIncomingRstStreamFrame(frame *frames.RstStreamFrame) {
	stream := c.getOrCreateStream(frame.GetStreamId())
	if stream.GetState().In(streamstate.IDLE) {
		c.connectionError(frames.PROTOCOL_ERROR, fmt.Sprintf("Received %v for strem in IDLE state.", frame.Type()))
	} else {
		stream.ReceiveFrame(frame)
	}
}

func (c *connection) handleIncomingPushPromiseFrame(frame *frames.PushPromiseFrame) {
	associatedStream, exists := c.getStreamIfExists(frame.StreamId)
	if !exists {
		c.connectionError(frames.PROTOCOL_ERROR, fmt.Sprintf("Received %v frame for non-existing associated stream %v.", frame.Type(), frame.StreamId))
		return
	}
	if !associatedStream.GetState().In(streamstate.OPEN, streamstate.HALF_CLOSED_LOCAL) {
		c.connectionError(frames.PROTOCOL_ERROR, fmt.Sprintf("Received %v frame for associated stream in state %v.", frame.Type(), associatedStream.GetState()))
		return
	}
	promisedStream := c.getOrCreateStream(frame.PromisedStreamId)
	promisedStream.ReceiveFrame(frame)
	method := findHeader(":method", frame.Headers)
	if method != "GET" {
		promisedStream.CloseWithError(frames.REFUSED_STREAM, fmt.Sprintf("%v with method %v not supported.", frame.Type(), method))
		return
	}
	c.promisedStreamCache[promisedStream.StreamId()] = promisedStream
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
func (c *connection) flowControlForIncomingDataFrame(frame *frames.DataFrame) {
	threshold := int64(2 << 13) // size of one frame
	c.remainingReceiveWindowSize -= int64(len(frame.Data))
	if c.remainingReceiveWindowSize < threshold {
		diff := int64(2<<15-1) - c.remainingReceiveWindowSize
		c.remainingReceiveWindowSize += diff
		c.Write(frames.NewWindowUpdateFrame(0, uint32(diff)))
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
	// TODO: Send ACK
	// TODO: Send PROTOCOL_ERROR if ACK is set but length > 0
}

func (c *connection) handleWindowUpdateFrame(frame *frames.WindowUpdateFrame) {
	c.increaseFlowControlWindow(int64(frame.WindowSizeIncrement))
	for _, s := range c.streams {
		s.ProcessPendingDataFrames()
	}
}

func (c *connection) RemainingFlowControlWindowIsEnough(nBytesToWrite int64) bool {
	return c.remainingReceiveWindowSize > nBytesToWrite
}

func (c *connection) DecreaseFlowControlWindow(nBytesToWrite int64) {
	c.remainingSendWindowSize -= nBytesToWrite
}

func (c *connection) increaseFlowControlWindow(nBytes int64) {
	c.remainingSendWindowSize += nBytes
}

func (c *connection) Write(frame frames.Frame) {
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

func (c *connection) getOrCreateStream(streamId uint32) stream.Stream {
	result, exists := c.getStreamIfExists(streamId)
	if !exists {
		result = stream.New(streamId, nil, c.settings.initialSendWindowSizeForNewStreams, c.settings.initialReceiveWindowSizeForNewStreams, c)
		c.streams[streamId] = result
	}
	return result
}

func (c *connection) getStreamIfExists(streamId uint32) (stream.Stream, bool) {
	stream, exists := c.streams[streamId]
	return stream, exists
}

func (c *connection) newStream(cmd *commands.HttpCommand) stream.Stream {
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
	c.streams[nextStreamId] = stream.New(nextStreamId, cmd, c.settings.initialSendWindowSizeForNewStreams, c.settings.initialReceiveWindowSizeForNewStreams, c)
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

func (c *connection) serverFrameSize() uint32 {
	return c.settings.serverFrameSize
}

func (c *connection) setServerFrameSize(size uint32) {
	c.settings.serverFrameSize = size
}

func (c *connection) host() string {
	return c.info.host
}

func (c *connection) port() int {
	return c.info.port
}

func (c *connection) error() error {
	return c.err
}

// TODO: This is called in another thread, which is confusing. Should have a different Handler for things that are not called from the event loop.
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
