package cmdline

import (
	"regexp"
)

type command struct {
	name         string
	description  string
	nArgs        int
	areArgsValid func([]string) bool
	usage        string
}

var (
	START_COMMAND = &command{
		name:        "start",
		description: "Start the h2c process.\nThe h2c process must be started before any other command can be run.",
		nArgs:       0,
		usage:       StartCmd,
	}
	CONNECT_COMMAND = &command{
		name:        "connect",
		description: "Connect to a server using https.",
		nArgs:       1,
		areArgsValid: func(args []string) bool {
			return regexp.MustCompile("^[^:]+(:[0-9]+)?$").MatchString(args[0])
		},
		usage: "h2c connect [options] <host>:<port>",
	}
	GET_COMMAND = &command{
		name:        "get",
		description: "Perform a GET request.",
		nArgs:       1,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c get [options] <path>",
	}
	SET_COMMAND = &command{
		name:        "set",
		description: "Set a header.\nThe header will be included in any subsequent request, unless 'h2c unset' is called.",
		nArgs:       2,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c set <header-name> <header-value>",
	}
	PID_COMMAND = &command{
		name:        "pid",
		description: "Show the process id of the h2c process.",
		nArgs:       0,
		usage:       "h2c pid",
	}
	STOP_COMMAND = &command{
		name:        "stop",
		description: "Stop the h2c process.",
		nArgs:       0,
		usage:       "h2c stop",
	}
)

func (c *command) Name() string {
	return c.name
}

var commands = []*command{
	START_COMMAND,
	CONNECT_COMMAND,
	GET_COMMAND,
	SET_COMMAND,
	PID_COMMAND,
	STOP_COMMAND,
}

type option struct {
	short        string
	long         string
	description  string
	commands     []*command
	hasParam     bool
	isParamValid func(string) bool
}

func (o *option) IsSet(m map[string]string) bool {
	_, ok := m[o.long]
	return ok
}

func (o *option) Get(m map[string]string) string {
	val, _ := m[o.long]
	return val
}

func (o *option) Set(val string, m map[string]string) {
	m[o.long] = val
}

var (
	INCLUDE_OPTION = &option{
		short:       "-i",
		long:        "--include",
		description: "Show response headers in the output.",
		commands:    []*command{GET_COMMAND},
		hasParam:    false,
	}
	TIMEOUT_OPTION = &option{
		short:       "-t",
		long:        "--timeout",
		description: "Timeout in seconds while waiting for response.",
		commands:    []*command{GET_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return regexp.MustCompile("^[0-9]+$").MatchString(param)
		},
	}
	HELP_OPTION = &option{
		short:       "-h",
		long:        "--help",
		description: "Show this help message.",
		commands:    []*command{START_COMMAND, CONNECT_COMMAND, GET_COMMAND, SET_COMMAND, PID_COMMAND, STOP_COMMAND},
		hasParam:    false,
	}
	DUMP_OPTION = &option{
		short:       "-d",
		long:        "--dump",
		description: "Dump traffic to console.",
		commands:    []*command{START_COMMAND},
		hasParam:    false,
	}
)

var options = []*option{
	INCLUDE_OPTION,
	TIMEOUT_OPTION,
	HELP_OPTION,
	DUMP_OPTION,
}
