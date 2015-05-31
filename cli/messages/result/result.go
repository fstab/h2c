package result

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type Result struct {
	Message string
	Error   error
}

// TODO: This is copy-and-paste from command.go

func New(msg string, err error) *Result {
	return &Result{msg, err}
}

// The resulting string is a single line, it does not contain newlines.
func (res *Result) Encode() (string, error) {
	data, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("Marshalling error: %v", err.Error())
	}
	result := base64.StdEncoding.EncodeToString(data)
	if strings.Contains(result, "\n") {
		return "", fmt.Errorf("Base64 encoding error: Received newline in base64 string.")
	}
	return result, nil
}

func Decode(encodedResult string) (*Result, error) {
	jsonData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedResult))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode base64 data: %v", err.Error())
	}
	res := &Result{}
	err = json.Unmarshal(jsonData, res)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode json data: %v %v", err.Error())
	}
	return res, nil
}
