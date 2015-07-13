package util

import (
	"fmt"
	"time"
)

type AsyncTask struct {
	success chan bool
	error   chan error
}

func NewAsyncTask() *AsyncTask {
	return &AsyncTask{
		success: make(chan bool, 1),
		error:   make(chan error, 1),
	}
}

func (t *AsyncTask) CompleteSuccessfully() {
	t.success <- true
}

func (t *AsyncTask) CompleteWithError(err error) {
	t.error <- err
}

func (t *AsyncTask) WaitForCompletion(timeoutInSeconds int) error {
	go func() {
		time.Sleep(time.Duration(timeoutInSeconds) * time.Second)
		t.error <- fmt.Errorf("Timeout after %v seconds.", timeoutInSeconds)
	}()
	select {
	case <-t.success:
		return nil
	case err := <-t.error:
		return err
	}
}
