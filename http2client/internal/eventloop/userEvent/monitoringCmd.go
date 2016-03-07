package userEvent

import (
	"fmt"
	"github.com/fstab/h2c/http2client/internal/streamstate"
	"github.com/fstab/h2c/http2client/internal/util"
	"sort"
)

type MonitoringRequest interface {
	CompleteWithError(err error)
	CompleteSuccessfully(resp MonitoringResponse)
	AwaitCompletion(timeoutInSeconds int) (MonitoringResponse, error)
}

// As of now, the "monitoring" is just use to retrieve info about stream states.
// However, it could be extended to retrieve other info, like available window sizes, etc.
type MonitoringResponse interface {
	StreamInfo() []StreamInfo
	AddStreamInfo(streamID uint32, httpMethod string, path string, state streamstate.StreamState, isCachedPushPromise bool)
}

type monitoringRequest struct {
	response MonitoringResponse
	callback *util.AsyncTask
}

type monitoringResponse struct {
	streamInfo []StreamInfo
}

type StreamInfo struct {
	StreamId            uint32
	HttpMethod          string
	Path                string
	State               streamstate.StreamState
	IsCachedPushPromise bool
}

func NewMonitoringRequest() MonitoringRequest {
	return &monitoringRequest{
		callback: util.NewAsyncTask(),
	}
}

func NewMonitoringResponse() MonitoringResponse {
	return &monitoringResponse{
		streamInfo: make([]StreamInfo, 0),
	}
}

func (resp *monitoringResponse) StreamInfo() []StreamInfo {
	return resp.streamInfo
}

func (resp *monitoringResponse) AddStreamInfo(streamID uint32, httpMethod string, path string, state streamstate.StreamState, isCachedPushPromise bool) {
	resp.streamInfo = append(resp.streamInfo, StreamInfo{
		StreamId:            streamID,
		HttpMethod:          httpMethod,
		Path:                path,
		State:               state,
		IsCachedPushPromise: isCachedPushPromise,
	})
	sort.Sort(streamInfoSlice(resp.streamInfo))
}

type streamInfoSlice []StreamInfo

func (s streamInfoSlice) Len() int {
	return len(s)
}

func (s streamInfoSlice) Less(i, j int) bool {
	return s[i].StreamId < s[j].StreamId
}

func (s streamInfoSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
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
