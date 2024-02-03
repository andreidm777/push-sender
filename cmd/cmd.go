package cmd

import (
	"flag"
	"push-sender/internal/app"
	"push-sender/internal/liveness"
	"push-sender/internal/runner"
	"sync"

	config "github.com/spf13/viper"
)

var (
	configFileName = flag.String("config", "/usr/local/etc/push-sender.conf", "config file")
)

func init() {
	config.SetDefault("liveness_enabled", false)
}

func Run() error {
	flag.Parse()
	ctx := runner.NewDefaultRunner(
		*configFileName,
		map[string]func(param string){},
	).StartAsync()

	var wg sync.WaitGroup

	liveness.NewDefaultLiveness().Start(ctx, &wg)

	app.NewDefaultApp().Start(ctx, &wg)

	wg.Wait()

	return nil
}
