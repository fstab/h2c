package commands

import (
	"github.com/fstab/h2c/http2client/internal/util"
)

type PingCommand struct {
	callback *util.AsyncTask
}

func NewPingCommand() *PingCommand {
	return &PingCommand{
		callback: util.NewAsyncTask(),
	}
}

func (cmd *PingCommand) CompleteWithError(err error) {
	cmd.callback.CompleteWithError(err)
}

func (cmd *PingCommand) CompleteSuccessfully() {
	cmd.callback.CompleteSuccessfully()
}

func (cmd *PingCommand) AwaitCompletion(timeoutInSeconds int) error {
	return cmd.callback.WaitForCompletion(timeoutInSeconds)
}
