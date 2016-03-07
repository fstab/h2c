package cmdline

import (
	"regexp"
)

type command struct {
	name         string
	description  string
	minArgs      int
	maxArgs      int
	areArgsValid func([]string) bool
	usage        string
}

var (
	START_COMMAND = &command{
		name: "start",
		description: "Start the h2c process. The h2c process must be started before running any other\n" +
			"command. To run h2c as a background process, run '" + StartCmd + "'.",
		minArgs: 0,
		maxArgs: 0,
		usage:   "h2c start [options]",
	}
	CONNECT_COMMAND = &command{
		name:        "connect",
		description: "Connect to a server using https.",
		minArgs:     1,
		maxArgs:     1,
		areArgsValid: func(args []string) bool {
			return regexp.MustCompile("^(https?://)?[^:]+(:[0-9]+)?$").MatchString(args[0])
		},
		usage: "h2c connect [options] <host>:<port>",
	}
	DISCONNECT_COMMAND = &command{
		name:        "disconnect",
		description: "Disconnect from server.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c disconnect",
	}
	GET_COMMAND = &command{
		name:        "get",
		description: "Perform a GET request.",
		minArgs:     1,
		maxArgs:     1,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c get [options] <path>",
	}
	PUT_COMMAND = &command{
		name:        "put",
		description: "Perform a PUT request.",
		minArgs:     1,
		maxArgs:     1,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c put [options] <path>",
	}
	POST_COMMAND = &command{
		name:        "post",
		description: "Perform a POST request.",
		minArgs:     1,
		maxArgs:     1,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c post [options] <path>",
	}
	SET_COMMAND = &command{
		name:        "set",
		description: "Set a header. The header will be included in any subsequent request.",
		minArgs:     2,
		maxArgs:     2,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c set <header-name> <header-value>",
	}
	UNSET_COMMAND = &command{
		name: "unset",
		description: "Undo 'h2c set'. The header will no longer be included in subsequent requests.\n" +
			"If <header-value> is omitted, all headers with <header-name> are removed.\n" +
			"Otherwise, only the specific value is removed but other headers with the same\n" +
			"name remain.",
		minArgs: 1,
		maxArgs: 2,
		areArgsValid: func(args []string) bool {
			return true
		},
		usage: "h2c unset <header-name> [<header-value>]",
	}
	PING_COMMAND = &command{
		name:        "ping",
		description: "Send ping frames.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c ping [options]",
	}
	PID_COMMAND = &command{
		name:        "pid",
		description: "Show the process id of the h2c process.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c pid",
	}
	STREAM_INFO_COMMAND = &command{
		name:        "stream-info",
		description: "List streams and their status.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c stream-info",
	}
	PUSH_LIST_COMMAND = &command{
		name:        "push-list",
		description: "List responses that are available as push promises.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c push-list",
	}
	STOP_COMMAND = &command{
		name:        "stop",
		description: "Stop the h2c process.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c stop",
	}
	WIRETAP_COMMAND = &command{
		name: "wiretap",
		description: "Forward HTTP/2 traffic and print captured frames to the console.\n" +
			"The wiretap command listens on localhost:port and fowards all traffic to remotehost:port.",
		minArgs: 2,
		maxArgs: 2,
		usage:   "h2c wiretap <localhost:port> <remotehost:port>\n",
	}
	VERSION_COMMAND = &command{
		name:        "version",
		description: "Print the version of h2c.",
		minArgs:     0,
		maxArgs:     0,
		usage:       "h2c version",
	}
)

func (c *command) Name() string {
	return c.name
}

var commands = []*command{
	START_COMMAND,
	CONNECT_COMMAND,
	DISCONNECT_COMMAND,
	GET_COMMAND,
	PUT_COMMAND,
	POST_COMMAND,
	SET_COMMAND,
	UNSET_COMMAND,
	PING_COMMAND,
	PID_COMMAND,
	PUSH_LIST_COMMAND,
	STREAM_INFO_COMMAND,
	STOP_COMMAND,
	WIRETAP_COMMAND,
	VERSION_COMMAND,
}

type option struct {
	short        string
	long         string
	description  string
	commands     []*command
	hasParam     bool
	isParamValid func(string) bool
}

func (o *option) Name() string {
	return o.long
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

func (o *option) Delete(m map[string]string) {
	delete(m, o.long)
}

var (
	INCLUDE_FRAMES_OPTION = &option{
		short:       "-i",
		long:        "--include",
		description: "Use with --dump to show only the specified frame times. Example: --include HEADERS,CONTINUATION",
		commands:    []*command{START_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return regexp.MustCompile("^[A-Za-z_]+(,\\s*[A-Za-z_]+)?$").MatchString(param)
		},
	}
	EXCLUDE_FRAMES_OPTION = &option{
		short:       "-e",
		long:        "--exclude",
		description: "Use with --dump to exclude the specified frame times. Example: --exclude PING,PRIORITY",
		commands:    []*command{START_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return INCLUDE_FRAMES_OPTION.isParamValid(param)
		},
	}
	INCLUDE_HEADERS_OPTION = &option{
		short:       "-i",
		long:        "--include",
		description: "Show response headers in the output.",
		commands:    []*command{GET_COMMAND, PUT_COMMAND, POST_COMMAND},
		hasParam:    false,
	}
	INCLUDE_CLOSED_STREAMS_OPTION = &option{
		short:       "-c",
		long:        "--closed",
		description: "Include closed streams.",
		commands:    []*command{STREAM_INFO_COMMAND},
		hasParam:    false,
	}
	TIMEOUT_OPTION = &option{
		short:       "-t",
		long:        "--timeout",
		description: "Timeout in seconds while waiting for response.",
		commands:    []*command{GET_COMMAND, PUT_COMMAND, POST_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return regexp.MustCompile("^[0-9]+$").MatchString(param)
		},
	}
	CONTENT_TYPE_OPTION = &option{
		short:       "-c",
		long:        "--content-type",
		description: "Value of the Content-Type header.",
		commands:    []*command{PUT_COMMAND, POST_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return true
		},
	}
	DATA_OPTION = &option{
		short:       "-d",
		long:        "--data",
		description: "The data to be sent. May not be used when --file is present.",
		commands:    []*command{PUT_COMMAND, POST_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return true
		},
	}
	FILE_OPTION = &option{
		short:       "-f",
		long:        "--file",
		description: "Post the content of file. Use '--file -' to read from stdin.",
		commands:    []*command{PUT_COMMAND, POST_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return true
		},
	}
	HELP_OPTION = &option{
		short:       "-h",
		long:        "--help",
		description: "Show this help message.",
		commands:    commands, // help option is available for all commands.
		hasParam:    false,
	}
	DUMP_OPTION = &option{
		short:       "-d",
		long:        "--dump",
		description: "Dump traffic to console.",
		commands:    []*command{START_COMMAND},
		hasParam:    false,
	}
	INTERVAL_OPTION = &option{
		short:       "-i",
		long:        "--interval",
		description: "Ping repeatedly. The time interval can be milliseconds (example: 500ms), seconds (example: 1s), or minutes (example: 2m). If ping is already running, this will update the time interval.",
		commands:    []*command{PING_COMMAND},
		hasParam:    true,
		isParamValid: func(param string) bool {
			return regexp.MustCompile("^[1-9][0-9]*(ms|s|m)$").MatchString(param)
		},
	}
	STOP_OPTION = &option{
		short:       "-s",
		long:        "--stop",
		description: "Stop pinging repeatedly.",
		commands:    []*command{PING_COMMAND},
		hasParam:    false,
	}
)

var options = []*option{
	INCLUDE_FRAMES_OPTION,
	EXCLUDE_FRAMES_OPTION,
	INCLUDE_HEADERS_OPTION,
	INCLUDE_CLOSED_STREAMS_OPTION,
	TIMEOUT_OPTION,
	CONTENT_TYPE_OPTION,
	HELP_OPTION,
	DUMP_OPTION,
	DATA_OPTION,
	FILE_OPTION,
	INTERVAL_OPTION,
	STOP_OPTION,
}
