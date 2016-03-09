package commands

import (
	"github.com/fstab/h2c/http2client/internal/streamstate"
	"github.com/fstab/h2c/http2client/internal/util"
	"sort"
)

// As of now, the "monitoring" is just use to retrieve info about stream states.
// However, it could be extended to retrieve other info, like available window sizes, response times, etc.
type MonitoringCommand struct {
	Result   *monitoringCommandResult
	callback *util.AsyncTask
}

type monitoringCommandResult struct {
	StreamInfo sortableStreamInfoSlice
}

type sortableStreamInfoSlice []StreamInfo

type StreamInfo struct {
	StreamId            uint32
	HttpMethod          string
	Path                string
	State               streamstate.StreamState
	IsCachedPushPromise bool
}

func NewMonitoringCommand() *MonitoringCommand {
	return &MonitoringCommand{
		Result:   newMonitoringCommandResult(),
		callback: util.NewAsyncTask(),
	}
}

func newMonitoringCommandResult() *monitoringCommandResult {
	return &monitoringCommandResult{
		StreamInfo: make([]StreamInfo, 0),
	}
}

func (res *monitoringCommandResult) AddStreamInfo(streamID uint32, httpMethod string, path string, state streamstate.StreamState, isCachedPushPromise bool) {
	res.StreamInfo = append(res.StreamInfo, StreamInfo{
		StreamId:            streamID,
		HttpMethod:          httpMethod,
		Path:                path,
		State:               state,
		IsCachedPushPromise: isCachedPushPromise,
	})
	sort.Sort(res.StreamInfo)
}

func (s sortableStreamInfoSlice) Len() int {
	return len(s)
}

func (s sortableStreamInfoSlice) Less(i, j int) bool {
	return s[i].StreamId < s[j].StreamId
}

func (s sortableStreamInfoSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (cmd *MonitoringCommand) CompleteWithError(err error) {
	cmd.callback.CompleteWithError(err)
}

func (cmd *MonitoringCommand) CompleteSuccessfully() {
	cmd.callback.CompleteSuccessfully()
}

func (cmd *MonitoringCommand) AwaitCompletion(timeoutInSeconds int) error {
	return cmd.callback.WaitForCompletion(timeoutInSeconds)
}
