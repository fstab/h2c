// Package http2client is a HTTP/2 client library.
package http2client

import (
	"errors"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/eventloop"
	"github.com/fstab/h2c/http2client/internal/message"
	"github.com/fstab/http2/hpack"
	neturl "net/url"
	"regexp"
	"strconv"
	"strings"
)

type Http2Client struct {
	loop                 *eventloop.Loop
	customHeaders        []hpack.HeaderField // filled with 'h2c set'
	err                  error               // if != nil, the Http2Client becomes unusable
	incomingFrameFilters []func(frames.Frame) frames.Frame
	outgoingFrameFilters []func(frames.Frame) frames.Frame
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

func (h2c *Http2Client) Connect(scheme string, host string, port int) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	if scheme != "https" {
		return "", fmt.Errorf("%v connections not supported.", scheme)
	}
	if h2c.loop != nil {
		return "", fmt.Errorf("Already connected to %v:%v.", h2c.loop.Host, h2c.loop.Port)
	}
	loop, err := eventloop.Start(host, port, h2c.incomingFrameFilters, h2c.outgoingFrameFilters)
	if err != nil {
		return "", err
	}
	h2c.loop = loop
	return "", nil
}

func (h2c *Http2Client) isConnected() bool {
	return h2c.loop != nil
}

func (h2c *Http2Client) Disconnect() (string, error) {
	if h2c.isConnected() {
		// TODO: Send goaway to server.
		h2c.loop.Shutdown <- true
		h2c.loop = nil
	}
	return "", nil
}

func (h2c *Http2Client) Get(path string, includeHeaders bool, timeoutInSeconds int) (string, error) {
	return h2c.putOrPostOrGet("GET", path, nil, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) Put(path string, data []byte, includeHeaders bool, timeoutInSeconds int) (string, error) {
	return h2c.putOrPostOrGet("PUT", path, data, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) Post(path string, data []byte, includeHeaders bool, timeoutInSeconds int) (string, error) {
	return h2c.putOrPostOrGet("POST", path, data, includeHeaders, timeoutInSeconds)
}

func (h2c *Http2Client) putOrPostOrGet(method string, path string, data []byte, includeHeaders bool, timeoutInSeconds int) (string, error) {
	if h2c.err != nil {
		return "", h2c.err
	}
	url, err := h2c.completeUrlWithCurrentConnectionData(path)
	if err != nil {
		return "", err
	}
	if !h2c.isConnected() {
		scheme := "https"
		if url.Scheme != "" {
			scheme = url.Scheme
		}
		host, port := hostAndPort(url)
		if host == "" {
			return "", fmt.Errorf("Not connected. Run 'h2c connect' first.")
		}
		_, err := h2c.Connect(scheme, host, port)
		if err != nil {
			return "", err
		}
	}
	if !h2c.urlMatchesCurrentConnection(url) {
		return "", fmt.Errorf("Cannot query %v while connected to %v", url.Scheme+"://"+url.Host, "https://"+hostAndPortString(h2c.loop.Host, h2c.loop.Port))
	}
	request := message.NewRequest(method, url)
	for _, header := range h2c.customHeaders {
		request.AddHeader(header.Name, header.Value)
	}
	if data != nil {
		request.AddData(data, true)
	}
	h2c.loop.HttpRequests <- request
	response, err := request.AwaitCompletion(timeoutInSeconds)
	if err != nil {
		return "", err
	}
	result := ""
	if includeHeaders {
		for _, header := range response.GetHeaders() {
			result = result + header.Name + ": " + header.Value + "\n"
		}
	}
	if response.GetData() != nil {
		result = result + string(response.GetData())
	}
	return result, nil
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
	request := message.NewMonitoringRequest()
	h2c.loop.MonitoringRequests <- request
	response, err := request.AwaitCompletion(10)
	if err != nil {
		return "", err
	}
	return strings.Join(response.AvailablePushResponses(), "\n"), nil
}

func (h2c *Http2Client) SetHeader(name, value string) (string, error) {
	h2c.customHeaders = append(h2c.customHeaders, hpack.HeaderField{
		Name:  normalizeHeaderName(name),
		Value: value,
	})
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
