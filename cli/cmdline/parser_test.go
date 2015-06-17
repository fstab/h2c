package cmdline

import (
	"github.com/fstab/h2c/cli/rpc"
	"testing"
)

func TestGetInclude(t *testing.T) {
	cmd, err := Parse([]string{"get", "--include", "path"})
	expectedCmd := &rpc.Command{
		Name: "get",
		Args: []string{"path"},
		Options: map[string]string{
			"--include": "",
		},
	}
	assertSuccess(cmd, expectedCmd, err, t)
}

func TestGet(t *testing.T) {
	cmd, err := Parse([]string{"get", "path"})
	expectedCmd := &rpc.Command{
		Name:    "get",
		Args:    []string{"path"},
		Options: make(map[string]string),
	}
	assertSuccess(cmd, expectedCmd, err, t)
}

func TestEmpty(t *testing.T) {
	cmd, err := Parse(make([]string, 0))
	assertError(cmd, err, t)
}

func assertSuccess(actual, expected *rpc.Command, err error, t *testing.T) {
	if err != nil {
		t.Error("Unexpected error: ", err.Error())
	}
	if actual.Name != expected.Name {
		t.Error("Expected command ", expected.Name, ", but got ", actual.Name)
	}
	if len(actual.Args) != len(expected.Args) {
		t.Error("Expected ", len(expected.Args), " args, but got ", len(actual.Args), " args.")
	}
	for i, arg := range actual.Args {
		if arg != expected.Args[i] {
			t.Error("Args[", i, "] is ", arg, ", but should be ", expected.Args[i], ".")
		}
	}
	if len(actual.Options) != len(expected.Options) {
		t.Error("Expected ", len(expected.Options), " options, but got ", len(actual.Options), " options.")
	}
	for name, val := range actual.Options {
		expectedVal, exists := expected.Options[name]
		if !exists {
			t.Error("Expected option ", name, " missing.")
		}
		if expectedVal != val {
			t.Error("Expected ", name, " to be ", expectedVal, ", but got ", val, ".")
		}
	}
}

func assertError(cmd *rpc.Command, err error, t *testing.T) {
	if err == nil {
		t.Error("Expected error, but got no error.")
	}
}
