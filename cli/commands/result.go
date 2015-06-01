package commands

// Result is sent from the h2c process to the command line interface.
type Result struct {
	Message string
	Error   *string // Should be type error, but this doesn't seem to work well with JSON marshalling.
}

func NewResult(msg string, err error) *Result {
	if err == nil {
		return &Result{msg, nil}
	} else {
		errString := err.Error()
		return &Result{msg, &errString}
	}
}

// Marshal returns the base64 encoding of Result.
func (res *Result) Marshal() (string, error) {
	return marshal(res)
}

// Used by the command line interface when receiving a Result from the h2c process.
func UnmarshalResult(encodedResult string) (*Result, error) {
	res := &Result{}
	err := unmarshal(encodedResult, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
