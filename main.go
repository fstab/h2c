package main

import (
	"fmt"
	"github.com/fstab/h2c/cli"
	"os"
)

func main() {
	msg, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(-1)
	} else if msg != "" {
		fmt.Println(msg)
	}
}
