// Package http2client is a HTTP/2 client library.
package http2client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	neturl "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/eventloop"
	"github.com/fstab/h2c/http2client/internal/eventloop/commands"
	"github.com/fstab/h2c/http2client/internal/util"
	"golang.org/x/net/http2/hpack"
)

type Http2Client struct {
	loop                 *eventloop.Loop
	pingTask             util.RepeatedTask   // Set when PingRepeatedly is called.
	customHeaders        []hpack.HeaderField // filled with 'h2c set'
	err                  error               // if != nil, the Http2Client becomes unusable
	incomingFrameFilters []func(frames.Frame) frames.Frame
	outgoingFrameFilters []func(frames.Frame) frames.Frame
	tlsConfig            *tls.Config
}

func (h2c *Http2Client) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	var err error

	if req.Body != nil {
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("Unable to extract body from request %v", req.URL.String())
		}

	}
	s, err := h2c.putOrPostOrGet(req, body, true, 5)
	if err != nil {
		return nil, fmt.Errorf("Tried to connect to %v %v got %v", req.Method, req.URL.String(), err)
	}
	for _, v := range h2c.customHeaders {
		fmt.Println("Header:", v.Name)
	}
	return s, nil
}

func Client(tlsConfig *tls.Config) *http.Client {
	c := &Http2Client{
		incomingFrameFilters: make([]func(frames.Frame) frames.Frame, 0),
		outgoingFrameFilters: make([]func(frames.Frame) frames.Frame, 0),
		tlsConfig:            tlsConfig,
	}

	ro := &http.Client{
		Transport: c,
	}
	return ro

}

func New() *Http2Client {
	return &Http2Client{
		incomingFrameFilters: make([]func(frames.Frame) frames.Frame, 0),
		outgoingFrameFilters: make([]func(frames.Frame) frames.Frame, 0),
	}
}

// The filter is called immediately after a frame is read from the server.
// The filter can be used to inspect and modify the incoming frames.
// WARNING: The filter will called in another go routine.
func (h2c *Http2Client) AddFilterForIncomingFrames(filter func(frames.Frame) frames.Frame) {
	h2c.incomingFrameFilters = append(h2c.incomingFrameFilters, filter)
}

// The filter is called immediately before a frame is sent to the server.
// The filter can be used to inspect and modify the outgoing frames.
// WARNING: The filter will called in another go routine.
func (h2c *Http2Client) AddFilterForOutgoingFrames(filter func(frames.Frame) frames.Frame) {
	h2c.outgoingFrameFilters = append(h2c.outgoingFrameFilters, filter)
}

func (h2c *Http2Client) Connect(url url.URL, config *tls.Config) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if url.Scheme != "https" {
		return "", fmt.Errorf("%v connections not supported.", url.Scheme)
	}
	if h2c.loop != nil && !h2c.loop.IsTerminated() {
		return "", fmt.Errorf("Already connected to %v:%v.", h2c.loop.Host, h2c.loop.Port)
	}
	port, err := strconv.Atoi(url.Port())
	if err != nil {
		return "", fmt.Errorf("Port %v appears not to be a number", url.Port())
	}
	loop, err := eventloop.Start(url.Hostname(), port, h2c.incomingFrameFilters, h2c.outgoingFrameFilters, config)
	if err != nil {
		return "", err
	}
	h2c.loop = loop
	return "", nil
}

func (h2c *Http2Client) isConnected() bool {
	return h2c.loop != nil && !h2c.loop.IsTerminated()
}

func (h2c *Http2Client) Disconnect() (string, error) {
	if h2c.isConnected() {
		// TODO: Send goaway to server.
		h2c.loop.Shutdown <- true
		h2c.loop = nil
	}
	return "", nil
}

func (h2c *Http2Client) Get(req *http.Request, includeHeaders bool, timeoutInSeconds int) (*http.Response, error) {
	return h2c.putOrPostOrGet(req, nil, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) Put(req *http.Request, data []byte, includeHeaders bool, timeoutInSeconds int) (*http.Response, error) {
	return h2c.putOrPostOrGet(req, data, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) Post(req *http.Request, data []byte, includeHeaders bool, timeoutInSeconds int) (*http.Response, error) {
	return h2c.putOrPostOrGet(req, data, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) putOrPostOrGet(req *http.Request, data []byte, includeHeaders bool, timeoutInSeconds int) (*http.Response, error) {
	if h2c.err != nil {
		return nil, h2c.err
	}
	/*url, err := h2c.completeUrlWithCurrentConnectionData(req.URL.Path)
	if err != nil {
		return nil, err
	}
	*/
	if !h2c.isConnected() {
		host, _ := hostAndPort(req.URL)
		if host == "" {
			return nil, fmt.Errorf("Not connected. Run 'h2c connect' first.")
		}
		_, err := h2c.Connect(*req.URL, h2c.tlsConfig)
		if err != nil {
			return nil, err
		}
	}
	if !h2c.urlMatchesCurrentConnection(req.URL) {
		return nil, fmt.Errorf("Cannot query %v while connected to %v", req.URL.Scheme+"://"+req.URL.Host, "https://"+hostAndPortString(h2c.loop.Host, h2c.loop.Port))
	}
	cmd := commands.NewHttpCommand(req.Method, req.URL)
	for _, header := range h2c.customHeaders {
		cmd.Request.AddHeader(header.Name, header.Value)
	}
	if data != nil {
		cmd.Request.SetBody(data, true)
	}
	h2c.loop.HttpCommands <- cmd
	err := cmd.AwaitCompletion(timeoutInSeconds)
	if err != nil {
		return nil, err
	}
	// Create an empty response
	resp := http.Response{Header: map[string][]string{}}
	if includeHeaders {
		for _, header := range cmd.Response.GetHeaders() {
			switch header.Name {
			case ":status":
				code, _ := strconv.Atoi(header.Value)
				resp.StatusCode = code
				resp.Status = http.StatusText(code)

			case "content-length":
				length, _ := strconv.Atoi(header.Value)
				resp.ContentLength = int64(length)
			default:
				resp.Header.Add(header.Name, header.Value)
			}
		}
	}
	// Set the protocol version on the response
	resp.ProtoMajor = 2
	resp.ProtoMinor = 0

	resp.Body = ioutil.NopCloser(bytes.NewBuffer(cmd.Response.GetBody()))
	return &resp, nil

}

func (h2c *Http2Client) completeUrlWithCurrentConnectionData(path string) (*neturl.URL, error) {
	if regexp.MustCompile(":[0-9]+").MatchString(path) && !strings.Contains(path, "://") && !strings.HasPrefix("/", path) {
		path = "/" + path // Treat "localhost:8443" as "/localhost:8443" in GET, PUT, POST, DELETE requests.
	}
	url, err := neturl.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("%v: Invalid path.")
	}
	if !h2c.isConnected() {
		return url, nil
	}
	if url.Scheme == "" {
		url.Scheme = "https"
	}
	if url.Host == "" {
		url.Host = hostAndPortString(h2c.loop.Host, h2c.loop.Port)
	}
	return url, nil
}

func (h2c *Http2Client) urlMatchesCurrentConnection(url *neturl.URL) bool {
	if !h2c.isConnected() {
		return false
	}
	host, port := hostAndPort(url)
	return url.Scheme == "https" && host == h2c.loop.Host && port == h2c.loop.Port
}

func hostAndPort(url *neturl.URL) (string, int) {
	parts := strings.SplitN(url.Host, ":", 2)
	if len(parts) == 2 {
		port, err := strconv.Atoi(parts[1])
		if err == nil {
			return parts[0], port
		}
	}
	return url.Host, 443
}

func hostAndPortString(host string, port int) string {
	result := host
	if port != 443 {
		result = result + ":" + strconv.Itoa(port)
	}
	return result
}

func (h2c *Http2Client) PushList() (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if !h2c.isConnected() {
		return "", fmt.Errorf("Not connected.")
	}
	cmd := commands.NewMonitoringCommand()
	h2c.loop.MonitoringCommands <- cmd
	err := cmd.AwaitCompletion(10)
	if err != nil {
		return "", err
	}
	result := ""
	for _, info := range cmd.Result.StreamInfo {
		if result != "" {
			result = result + "\n"
		}
		if info.IsCachedPushPromise {
			result = result + info.Path
		}
	}
	return result, nil
}

func (h2c *Http2Client) StreamInfo(includeClosedStreams bool) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if !h2c.isConnected() {
		return "", fmt.Errorf("Not connected.")
	}
	cmd := commands.NewMonitoringCommand()
	h2c.loop.MonitoringCommands <- cmd
	err := cmd.AwaitCompletion(10)
	if err != nil {
		return "", err
	}
	result := ""
	for _, info := range cmd.Result.StreamInfo {
		if result != "" {
			result = result + "\n"
		}
		result = result + fmt.Sprintf("%v: %v %v %v", info.StreamId, info.HttpMethod, info.Path, info.State)
		if info.IsCachedPushPromise {
			result = result + " (cached push promise)"
		}
	}
	return result, nil
}

func (h2c *Http2Client) SetHeader(name, value string) (string, error) {
	h2c.customHeaders = append(h2c.customHeaders, hpack.HeaderField{
		Name:  normalizeHeaderName(name),
		Value: value,
	})
	return "", nil
}

func (h2c *Http2Client) PingOnce() (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if !h2c.isConnected() {
		return "", fmt.Errorf("Not connected. Run 'h2c connect' first.")
	}
	pingCmd := commands.NewPingCommand()
	h2c.loop.PingCommands <- pingCmd
	return "", pingCmd.AwaitCompletion(10) // TODO: Hard-coded timeout in seconds.
}

func (h2c *Http2Client) PingRepeatedly(interval time.Duration) (string, error) {
	_, err := h2c.PingOnce()
	if err != nil {
		return "", err
	}
	oldPingTask := h2c.pingTask
	h2c.pingTask = util.StartRepeatedTask(interval, func() {
		_, err := h2c.PingOnce()
		if err != nil {
			h2c.pingTask.Stop()
			h2c.pingTask = nil
		}
	})
	if oldPingTask != nil {
		oldPingTask.Stop()
	}
	return "", nil
}

func (h2c *Http2Client) StopPingRepeatedly() (string, error) {
	if h2c.pingTask != nil {
		h2c.pingTask.Stop()
		h2c.pingTask = nil
	}
	return "", nil
}

// "Content-Type:" -> "content-type"
func normalizeHeaderName(name string) string {
	for name[len(name)-1] == ':' {
		name = name[:len(name)-1]
	}
	// return name // Use this and set header "Content-Type" to provoke RST_STREAM with error CANCEL.
	return strings.ToLower(name)
}

func (h2c *Http2Client) UnsetHeader(nameValue []string) (string, error) {
	if len(nameValue) != 1 && len(nameValue) != 2 {
		return "", errors.New("Syntax error.")
	}
	remainingHeaders := make([]hpack.HeaderField, 0, len(h2c.customHeaders))
	matches := func(field hpack.HeaderField) bool {
		if len(nameValue) == 1 {
			return field.Name == normalizeHeaderName(nameValue[0])
		} else {
			return field.Name == normalizeHeaderName(nameValue[0]) && field.Value == nameValue[1]
		}
	}
	for _, field := range h2c.customHeaders {
		if !matches(field) {
			remainingHeaders = append(remainingHeaders, field)
		}
	}
	h2c.customHeaders = remainingHeaders
	return "", nil
}
