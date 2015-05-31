package result

import (
	"github.com/fstab/h2c/cli/messages/marshaller"
)

type Result struct {
	Message string
	Error   error
}

func New(msg string, err error) *Result {
	return &Result{msg, err}
}

// Marshal returns the base64 encoding of Result.
func (res *Result) Marshal() (string, error) {
	return marshaller.Marshal(res)
}

// Unmarshal is the inverse function of Marshal().
func Unmarshal(encodedResult string) (*Result, error) {
	res := &Result{}
	err := marshaller.Unmarshal(encodedResult, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
