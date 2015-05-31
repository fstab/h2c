// Package marshaller implements helper functions for marshalling command and result.
package marshaller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Marshal is a helper function for Command.Marshal() and Result.Marshal().
//
// The resulting string does not contain newlines,
// so newlines can be used as separators between multiple encoded values.
func Marshal(v interface{}) (string, error) {
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

// Unmarshal is a helper function for command.Unmarshal() and result.Unmarshal().
//
// The result is stored in the value pointed to by v.
func Unmarshal(data string, v interface{}) error {
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
