package commands

import (
	"errors"
	"github.com/fstab/h2c/http2client/internal/util"
	"golang.org/x/net/http2/hpack"
	neturl "net/url"
	"strconv"
)

type HttpCommand struct {
	Request  *httpMsg
	Response *httpMsg
	callback *util.AsyncTask
}

type httpMsg struct {
	headers []hpack.HeaderField
	body    []byte
}

func NewHttpCommand(method string, url *neturl.URL) *HttpCommand {
	result := &HttpCommand{
		Request:  newHttpMsg(),
		Response: newHttpMsg(),
		callback: util.NewAsyncTask(),
	}
	result.Request.AddHeader(":method", method)
	result.Request.AddHeader(":scheme", url.Scheme)
	result.Request.AddHeader(":authority", url.Host)    // TODO: Include user ?
	result.Request.AddHeader(":path", url.RequestURI()) // TODO: Does not include Fragment ?
	return result
}

func newHttpMsg() *httpMsg {
	return &httpMsg{
		headers: make([]hpack.HeaderField, 0),
		body:    make([]byte, 0),
	}
}

func (m *httpMsg) AddHeader(name, value string) {
	m.headers = append(m.headers, hpack.HeaderField{Name: name, Value: value})
}

func (m *httpMsg) GetHeaders() []hpack.HeaderField {
	return m.headers
}

func (m *httpMsg) GetHeader(name string) string {
	for _, header := range m.headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}

func (m *httpMsg) SetBody(data []byte, addContentLengthHeader bool) {
	m.body = data
	if addContentLengthHeader {
		m.AddHeader("content-length", strconv.Itoa(len(data)))
	}
}

func (m *httpMsg) GetBody() []byte {
	return m.body
}

func (c *HttpCommand) CompleteWithError(err error) {
	c.callback.CompleteWithError(err)
}

func (c *HttpCommand) CompleteSuccessfully() {
	c.callback.CompleteSuccessfully()
}

// If AwaitCompletion returns an error, this means that no HTTP response was received.
// If an HTTP response is received, and this response has an error code (like 500),
// AwaitCompletion will not return an error (the HTTP error code is treated like a regular HTTP response).
// There are a number of reasons why no response is received, for example:
//   * Stream error, e.g. RST_STREAM received, illegal stream state, etc.
//   * Connection error, e.g. connection closed, timeout, etc.
//   * HttpRequest is illegal, e.g. it contains an unsupported HTTP method, etc.
func (c *HttpCommand) AwaitCompletion(timeoutInSeconds int) error {
	err := c.callback.WaitForCompletion(timeoutInSeconds)
	if err != nil {
		return err
	}
	if c.Response == nil {
		return errors.New("Request got no error and no response. This is a bug.")
	}
	return nil
}
