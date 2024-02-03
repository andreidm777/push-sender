package app

import (
	"context"
	"push-sender/internal/fetcher"
	"push-sender/internal/task"
	"sync"

	"push-sender/internal/worker"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

type Application interface {
	Start(ctx context.Context, wg *sync.WaitGroup)
}

type defaultApplication struct {
	channelTask chan *task.Task
	fetch       fetcher.Fetcher
	workers     map[int]worker.Worker
}

func (da *defaultApplication) startFetcher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		mtask, err := da.fetch.Get()
		if err != nil && err != fetcher.ErrContinue {
			log.Errorf("cannot fetch data")
			break
		}
		select {
		case <-ctx.Done():
			close(da.channelTask)
			log.Debug("graceful shutdown fetcher")
			break
		default:
			log.Debugf("get task %v", mtask)
		}
	}
}

func (da *defaultApplication) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	da.fetch = fetcher.NewDefaultFetcher(ctx)
	go da.startFetcher(ctx, wg)
	wg.Add(config.GetInt("worker_count"))
	for _, v := range da.workers {
		v.Start(da.channelTask, wg)
	}
}

func NewDefaultApp() Application {
	app := &defaultApplication{
		channelTask: make(chan *task.Task),
		workers:     make(map[int]worker.Worker, config.GetInt("worker_count")),
	}
	for i := 0; i < config.GetInt("worker_count"); i++ {
		app.workers[i] = worker.NewDefaultWorker()
	}

	return app
}
