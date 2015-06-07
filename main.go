package main

import (
	"fmt"
	"github.com/fstab/h2c/cli"
	"os"
	"runtime"
)

func main() {
	switch runtime.GOOS {
	case "nacl", "plan9", "windows":
		fmt.Fprintf(os.Stderr, "The current version of h2c uses 'unix' sockets, which are not supported on Windows.\n")
		fmt.Fprintf(os.Stderr, "Sorry. This is one of the top priority bugs to be fixed.\n")
		os.Exit(-1)
	default:
		msg, err := cli.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err.Error())
			os.Exit(-1)
		} else if msg != "" {
			fmt.Println(msg)
		}
	}
}
