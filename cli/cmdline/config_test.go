package cmdline

import "testing"

func TestConfig(t *testing.T) {
	if len(commands) != len(HELP_OPTION.commands) {
		t.Error("Some command does not have a help option.")
	}
}
