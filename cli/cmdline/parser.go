// Package cmdline implements the parser for the command line arguments.
package cmdline

import (
	"errors"
	"github.com/fstab/h2c/cli/rpc"
)

var syntaxError = errors.New("Syntax error. Run 'h2c --help' for help.")

func Parse(args []string) (*rpc.Command, error) {
	if len(args) == 1 && (args[0] == HELP_OPTION.short || args[0] == HELP_OPTION.long) {
		// h2c --help
		return nil, errors.New(globalUsage())
	}
	cmd, err := findCommand(args)
	if err != nil {
		return nil, err
	}
	remainingArgs, options, err := parseOptions(args, cmd)
	if err != nil {
		return nil, err
	}
	if len(remainingArgs) < cmd.minArgs+1 || len(remainingArgs) > cmd.maxArgs+1 {
		return nil, errors.New(usage(cmd))
	}
	if HELP_OPTION.IsSet(options) {
		return nil, errors.New(usage(cmd))
	}
	cmdArgs := make([]string, 0)
	if len(remainingArgs) > 1 {
		cmdArgs = remainingArgs[1:]
		if cmd.areArgsValid != nil && !cmd.areArgsValid(cmdArgs) {
			return nil, errors.New(usage(cmd))
		}
	}
	return rpc.NewCommand(cmd.name, cmdArgs, options)
}

func parseOptions(args []string, cmd *command) ([]string, map[string]string, error) {
	foundOptions := make(map[string]string)
	for _, opt := range options {
		if opt.supportsCommand(cmd) {
			i, found := opt.findIndex(args)
			if found {
				if opt.hasParam {
					if len(args) <= i+1 {
						return nil, nil, syntaxError
					}
					if !opt.isParamValid(args[i+1]) {
						return nil, nil, syntaxError
					}
					opt.Set(args[i+1], foundOptions)
					args = append(args[:i], args[i+2:]...)
				} else {
					opt.Set("", foundOptions)
					args = append(args[:i], args[i+1:]...)
				}
			}
		}
	}
	return args, foundOptions, nil
}

func globalUsage() string {
	result := "Usage: h2c ["
	first := true
	for _, cmd := range commands {
		if !first {
			result += "|"
		}
		result += cmd.name
		first = false
	}
	result += "] <args>\nRun 'h2c [cmd] --help' to learn more about a command."
	return result
}

func usage(cmd *command) string {
	result := cmd.description
	result += "\nUsage: " + cmd.usage
	availableOptions := make([]*option, 0)
	for _, opt := range options {
		if opt.supportsCommand(cmd) {
			availableOptions = append(availableOptions, opt)
		}
	}
	if len(availableOptions) > 0 {
		result += "\nAvailable options are:"
		for _, opt := range availableOptions {
			result += "\n    " + opt.short + " " + opt.long + ": " + opt.description
		}
	}
	return result
}

func findCommand(args []string) (*command, error) {
	if len(args) < 1 {
		return nil, syntaxError
	}
	for _, cmd := range commands {
		if args[0] == cmd.name {
			return cmd, nil
		}
	}
	for _, opt := range options {
		if args[0] == opt.short || args[0] == opt.long {
			if opt.hasParam {
				if len(args) < 2 {
					return nil, syntaxError
				} else {
					return findCommand(args[2:])
				}
			}
			return findCommand(args[1:])
		}
	}
	return nil, errors.New(args[0] + ": Unknown command. Run 'h2c --help' for help.")
}

func (opt *option) findIndex(argv []string) (int, bool) {
	for i, arg := range argv {
		if arg == opt.short || arg == opt.long {
			return i, true
		}
	}
	return -1, false
}

func (opt *option) supportsCommand(cmd *command) bool {
	for _, c := range opt.commands {
		if c.name == cmd.name {
			return true
		}
	}
	return false
}
