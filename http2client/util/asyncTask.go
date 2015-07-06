package util

import (
	"fmt"
	"time"
)

type AsyncTask struct {
	completed chan bool
	timeout   chan bool
}

func NewAsyncTask() *AsyncTask {
	return &AsyncTask{
		completed: make(chan bool, 1),
		timeout:   make(chan bool, 1),
	}
}

func (t *AsyncTask) Complete() {
	t.completed <- true
}

func (t *AsyncTask) WaitForCompletion(timeoutInSeconds int) error {
	go func() {
		time.Sleep(time.Duration(timeoutInSeconds) * time.Second)
		t.timeout <- true
	}()
	select {
	case <-t.completed:
		return nil
	case <-t.timeout:
		return fmt.Errorf("Timeout after %v seconds.", timeoutInSeconds)
	}
}
