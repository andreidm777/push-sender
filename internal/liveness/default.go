package liveness

import (
	"context"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

type Liveness interface {
	Start(ctx context.Context, wg *sync.WaitGroup)
}

type defaultLiveness struct {
}

func (dl defaultLiveness) Start(ctx context.Context, wg *sync.WaitGroup) {
	// for init liveness and readness k8s docker probe
	http.HandleFunc("/ping/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	srv := http.Server{
		Addr: config.GetString("liveness_port"),
	}

	wg.Add(2)

	go func(ctx context.Context) {
		defer wg.Done()
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Errorf("cannot shutdown server %s", err)
			return
		}
	}(ctx)

	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// Error starting or closing listener:
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()
}

func NewDefaultLiveness() Liveness {
	return defaultLiveness{}
}
