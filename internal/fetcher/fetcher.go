package fetcher

import (
	"context"
	"errors"
	"push-sender/internal/task"
)

var ErrContinue = errors.New("Continue")

type Fetcher interface {
	Get() (*task.Task, error)
}

type defaultFetcher struct {
}

func NewDefaultFetcher(_ context.Context) Fetcher {
	return &defaultFetcher{}
}

func (f *defaultFetcher) Get() (*task.Task, error) {
	return &task.Task{}, nil
}
