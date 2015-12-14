package util

import "time"

type RepeatedTask interface {
	Stop()
}

type repeatedTask struct {
	ticker      *time.Ticker
	doneChannel chan interface{}
	task        func()
}

func StartRepeatedTask(interval time.Duration, task func()) RepeatedTask {
	t := &repeatedTask{
		ticker:      time.NewTicker(interval),
		doneChannel: make(chan interface{}),
		task:        task,
	}
	go func() {
		for {
			select {
			case <-t.ticker.C:
				t.task()
			case <-t.doneChannel:
				// t.Stop() was called.
				return
			}
		}
	}()
	return t
}

func (t *repeatedTask) Stop() {
	t.ticker.Stop()
	close(t.doneChannel)
}
