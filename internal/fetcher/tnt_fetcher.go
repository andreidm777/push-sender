package fetcher

import (
	"context"
	"push-sender/internal/task"
	"time"

	"push-sender/internal/tnt"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
	"github.com/tarantool/go-tarantool/v2/pool"
)

type tntFetcher struct {
	queue tnt.Queue
}

func NewTntFetcher(ctx context.Context) Fetcher {
	qu, err := tnt.NewTntQueue(ctx,
		config.GetString("tarantool.queue_name"),
		&tnt.TntCfg{
			Addrs:    config.GetStringSlice("tarantool.queue"),
			User:     config.GetString("tarantool.user"),
			Password: config.GetString("tarantool.password"),
			Timeout:  config.GetDuration("tarantool.timeout"),
		})

	if err != nil {
		log.Fatal("cannot connection to tarantool queue")
		return nil
	}

	return &tntFetcher{
		queue: qu,
	}
}

func (f *tntFetcher) Get() (*task.Task, error) {
	qtask, err := f.queue.TakeTimeout(1 * time.Second)
	if err == pool.ErrNoRwInstance || qtask == nil {
		return nil, ErrContinue
	}

	ret_task := &task.Task{
		ID: qtask.Id(),
	}

	if err := tnt.ScanFieldsAnyToStruct(qtask.Data(), ret_task); err != nil {
		return nil, err
	}

	return ret_task, nil
}
