package message

import (
	"fmt"
	"github.com/fstab/h2c/http2client/internal/util"
)

type MonitoringRequest interface {
	CompleteWithError(err error)
	CompleteSuccessfully(resp MonitoringResponse)
	AwaitCompletion(timeoutInSeconds int) (MonitoringResponse, error)
}

// As of now, the "monitoring" is just use to retrieve the available push promise paths.
// However, it could be extended to retrieve other info, like available window sizes, open streams, etc.
// The idea is that this data should not be accessed directly, because the connection state is maintained
// in another thread (go routine) than the command line. Multi-threaded applications in Go should communicate
// via messages and not via shared memory.
type MonitoringResponse interface {
	AvailablePushResponses() []string
	AddPromisedPath(path string)
}

type monitoringRequest struct {
	response MonitoringResponse
	callback *util.AsyncTask
}

type monitoringResponse struct {
	promisedPaths []string
}

func NewMonitoringRequest() MonitoringRequest {
	return &monitoringRequest{
		callback: util.NewAsyncTask(),
	}
}

func NewMonitoringResponse() MonitoringResponse {
	return &monitoringResponse{
		promisedPaths: make([]string, 0),
	}
}

func (resp *monitoringResponse) AddPromisedPath(path string) {
	resp.promisedPaths = append(resp.promisedPaths, path)
}

func (resp *monitoringResponse) AvailablePushResponses() []string {
	return resp.promisedPaths
}

func (req *monitoringRequest) CompleteWithError(err error) {
	req.callback.CompleteWithError(err)
}

func (req *monitoringRequest) CompleteSuccessfully(resp MonitoringResponse) {
	req.response = resp
	req.callback.CompleteSuccessfully()
}

func (req *monitoringRequest) AwaitCompletion(timeoutInSeconds int) (MonitoringResponse, error) {
	err := req.callback.WaitForCompletion(timeoutInSeconds)
	if err != nil {
		return nil, err
	}
	if req.response == nil {
		return nil, fmt.Errorf("Request got no error and no response. This is a bug.")
	}
	return req.response, nil
}
