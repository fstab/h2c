package commands

import (
	"fmt"
	"github.com/fstab/h2c/http2client/internal/util"
)

type PingRequest interface {
	CompleteWithError(err error)
	CompleteSuccessfully(resp PingResponse)
	AwaitCompletion(timeoutInSeconds int) (PingResponse, error)
}

type PingResponse interface{}

type pingRequest struct {
	response PingResponse
	callback *util.AsyncTask
}

type pingResponse struct{}

func NewPingRequest() PingRequest {
	return &pingRequest{
		callback: util.NewAsyncTask(),
	}
}

func NewPingResponse() PingResponse {
	return &pingResponse{}
}

func (req *pingRequest) CompleteWithError(err error) {
	req.callback.CompleteWithError(err)
}

func (req *pingRequest) CompleteSuccessfully(resp PingResponse) {
	req.response = resp
	req.callback.CompleteSuccessfully()
}

func (req *pingRequest) AwaitCompletion(timeoutInSeconds int) (PingResponse, error) {
	err := req.callback.WaitForCompletion(timeoutInSeconds)
	if err != nil {
		return nil, err
	}
	if req.response == nil {
		return nil, fmt.Errorf("Request got no error and no response. This is a bug.")
	}
	return req.response, nil
}
