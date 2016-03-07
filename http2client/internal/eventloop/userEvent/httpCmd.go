package userEvent

import (
	"errors"
	"github.com/fstab/h2c/http2client/internal/util"
	"golang.org/x/net/http2/hpack"
	neturl "net/url"
	"strconv"
)

type HttpMessage interface {
	AddHeader(name, value string)
	GetHeader(name string) string
	GetHeaders() []hpack.HeaderField
	AddData(data []byte, addContentLengthHeader bool)
	GetData() []byte
}

type HttpRequest interface {
	HttpMessage
	CompleteWithError(err error)
	CompleteSuccessfully(resp HttpResponse)

	// The error is only returned if no HTTP response was received.
	// This can have a number of reasons, for example:
	//   * Stream error, e.g. RST_STREAM received, illegal stream state, etc.
	//   * Connection error, e.g. connection closed, timeout, etc.
	//   * HttpRequest is illegal, e.g. it contains an unsupported HTTP method, etc.
	// HTTP error codes (like 500, 404, etc.) are returned as regular HttpResponse and will not trigger the error.
	AwaitCompletion(timeoutInSeconds int) (HttpResponse, error)
}

type HttpResponse interface {
	HttpMessage
}

func NewRequest(method string, url *neturl.URL) HttpRequest {
	headers := make([]hpack.HeaderField, 4)
	headers[0] = hpack.HeaderField{Name: ":method", Value: method}
	headers[1] = hpack.HeaderField{Name: ":scheme", Value: url.Scheme}
	headers[2] = hpack.HeaderField{Name: ":authority", Value: url.Host}    // TODO: Include user ?
	headers[3] = hpack.HeaderField{Name: ":path", Value: url.RequestURI()} // TODO: Does not include Fragment ?
	return &request{
		headers:  headers,
		callback: util.NewAsyncTask(),
	}
}

type request struct {
	headers  []hpack.HeaderField
	data     []byte
	response HttpResponse
	callback *util.AsyncTask
}

func (req *request) AddHeader(name, value string) {
	req.headers = append(req.headers, hpack.HeaderField{Name: name, Value: value})
}

func (req *request) GetHeader(name string) string {
	for _, header := range req.headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}

func (req *request) AddData(data []byte, addContentLengthHeader bool) {
	req.data = data
	if addContentLengthHeader {
		req.AddHeader("content-length", strconv.Itoa(len(data)))
	}
}

func (req *request) CompleteWithError(err error) {
	req.callback.CompleteWithError(err)
}

func (req *request) CompleteSuccessfully(resp HttpResponse) {
	req.response = resp
	req.callback.CompleteSuccessfully()
}

func (req *request) AwaitCompletion(timeoutInSeconds int) (HttpResponse, error) {
	err := req.callback.WaitForCompletion(timeoutInSeconds)
	if err != nil {
		return nil, err
	}
	if req.response == nil {
		return nil, errors.New("Request got no error and no response. This is a bug.")
	}
	return req.response, nil
}

func (req *request) GetHeaders() []hpack.HeaderField {
	return req.headers
}

func (req *request) GetData() []byte {
	return req.data
}

type response struct {
	headers []hpack.HeaderField
	data    []byte
}

func NewResponse() HttpResponse {
	return &response{
		headers: make([]hpack.HeaderField, 0),
	}
}

func (resp *response) AddHeader(name, value string) {
	resp.headers = append(resp.headers, hpack.HeaderField{Name: name, Value: value})
}

func (resp *response) AddData(data []byte, addContentLengthHeader bool) {
	resp.data = data
}

func (resp *response) GetHeader(name string) string {
	for _, header := range resp.headers {
		if header.Name == name {
			return header.Value
		}
	}
	return ""
}

func (res *response) GetHeaders() []hpack.HeaderField {
	return res.headers
}

func (res *response) GetData() []byte {
	return res.data
}
