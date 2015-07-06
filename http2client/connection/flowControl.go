package connection

//
//import (
//	"github.com/fstab/h2c/http2client/frames"
//	"github.com/fstab/h2c/http2client/util"
//	"sync"
//)
//
//type flowControl struct {
//	quit           chan bool
//	windowUpdate   chan int64
//	availableBytes int64
//	queue          *synchronizedQueue
//}
//
//func Run(initialWindowSize int64) *flowControl {
//	result := &flowControl{
//		quit:           make(chan bool),
//		windowUpdate:   make(chan int64),
//		availableBytes: initialWindowSize,
//		queue: &synchronizedQueue{
//			requests: make([]*request, 0),
//		},
//	}
//	go func() {
//		for {
//			select {
//			case <-result.quit:
//				return
//			case diff := <-result.windowUpdate:
//				result.availableBytes += diff
//				result.processPendingTask()
//			}
//		}
//	}()
//}
//
//func (f *flowControl) processPendingTasks() {
//	req := f.queue.popIfLessOrEqual(f.availableBytes)
//	for req != nil {
//		if req.preventTimeout() != nil {
//
//		}
//	}
//}
//
//func (f *flowControl) take(nBytes int64, timeoutInSeconds int) error {
//	task := util.NewAsyncTask()
//	req := &request{
//		task:   task,
//		nBytes: nBytes,
//		done:   false,
//	}
//	f.queue.put(req)
//	err := task.WaitForCompletion(timeoutInSeconds)
//	req.done = true
//	return err
//}
//
//type request struct {
//	task   *util.AsyncTask
//	nBytes int64
//	done   bool
//}
//
//type synchronizedQueue struct {
//	mutex    sync.Mutex
//	requests []*request
//}
//
//func (q *synchronizedQueue) put(r *request) {
//	q.mutex.Lock()
//	defer q.mutex.Unlock()
//	q.requests = append(q.requests, r)
//}
//
//func (q *synchronizedQueue) popIfLessOrEqual(nBytes int64) *request {
//	q.mutex.Lock()
//	defer q.mutex.Unlock()
//	for len(q.requests) > 0 && q.requests[0].done {
//		q.requests = q.requests[1:]
//	}
//	if len(q.requests) == 0 {
//		return nil
//	}
//	if nBytes <= q.requests[0].nBytes {
//		result := q.requests[0]
//		q.requests = q.requests[1:]
//		return result
//	}
//	return nil
//}
