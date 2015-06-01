package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func marshal(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("JSON marshalling error: %v", err.Error())
	}
	result := base64.StdEncoding.EncodeToString(data)
	if strings.Contains(result, "\n") {
		return "", fmt.Errorf("Base64 encoding error: Received newline in base64 string.")
	}
	return result, nil
}

func unmarshal(data string, v interface{}) error {
	jsonData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(data))
	if err != nil {
		return fmt.Errorf("Failed to decode base64 data: %v", err.Error())
	}
	err = json.Unmarshal(jsonData, v)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal json data: %v %v", err.Error())
	}
	return nil
}
