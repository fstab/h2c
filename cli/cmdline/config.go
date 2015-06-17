package cmdline

import (
	"regexp"
)

type command struct {
	name         string
	nArgs        int
	areArgsValid func([]string) bool
	usage        string
}

var commands = []*command{
	&command{
		name:  "start",
		nArgs: 0,
		usage: StartCmd,
	},
	&command{
		name:  "connect",
		nArgs: 1,
		areArgsValid: func(args []string) bool {
			return regexp.MustCompile("^[^:]+(:[0-9]+)?$").MatchString(args[0])
		},
		usage: "h2c connect [options] <host>:<port>",
	},
	&command{
		name:  "get",
		nArgs: 1,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c get [options] <path>",
	},
	&command{
		name:  "pid",
		nArgs: 0,
		usage: "h2c pid",
	},
	&command{
		name:  "stop",
		nArgs: 0,
		usage: "h2c stop",
	},
}

type option struct {
	short        string
	long         string
	description  string
	commands     []string
	hasParam     bool
	isParamValid func(string) bool
}

var options = []*option{
	&option{
		short:       "-i",
		long:        "--include",
		description: "Show response headers in the output.",
		commands:    []string{"get"},
		hasParam:    false,
	},
	&option{
		short:       "-t",
		long:        "--timeout",
		description: "Timeout in seconds while waiting for response.",
		commands:    []string{"get"},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return regexp.MustCompile("^[0-9]+$").MatchString(param)
		},
	},
	&option{
		short:       "-h",
		long:        "--help",
		description: "Show this help message.",
		commands:    []string{"start", "get", "pid", "stop"},
		hasParam:    false,
	},
}
