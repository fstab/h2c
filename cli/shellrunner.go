package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func runDaemonShellCommand() error {
	h2c := os.Args[0]
	cmd := exec.Command(h2c, "start")
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("Failed to run '%v start': %v", h2c, err)
	}
	for i := 0; i < 3; i++ {
		cmd = exec.Command(h2c, "pid")
		err = cmd.Run()
		if err == nil {
			return nil // If 'h2c pid' returns no error, the process should be running successfully.
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
	return fmt.Errorf("Failed to run '%v start'", h2c)
}
