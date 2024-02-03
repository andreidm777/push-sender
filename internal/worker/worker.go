package worker

import (
	"push-sender/internal/task"
	"push-sender/internal/transport"
	"sync"
)

type Worker interface {
	Start(channel <-chan *task.Task, wg *sync.WaitGroup)
}

type defaultWorker struct {
}

func NewDefaultWorker() Worker {
	return &defaultWorker{}
}

func (dw *defaultWorker) Start(channel <-chan *task.Task, wg *sync.WaitGroup) {
	go func(dw *defaultWorker) {
		defer wg.Done()
		for qtask := range channel {
			dw.push(qtask)
		}
	}(dw)
}

func (dw *defaultWorker) push(qtask *task.Task) {
	sender := transport.GetTransport(qtask.Type)
	sender.Send(qtask)
}
