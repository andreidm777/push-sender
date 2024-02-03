package runner

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	config "github.com/spf13/viper"
)

type Runner interface {
	StartAsync() (ctx context.Context)
}

type defaultRunner struct {
	fileName       string
	onChangeConfig map[string]func(param string)
}

func (r defaultRunner) StartAsync() (ctx context.Context) {
	r.initConfig()
	initLog()
	ctx, cancel := context.WithCancel(context.Background())
	go initSignals(cancel)
	return
}

func NewDefaultRunner(configFile string, onChangeConfig map[string]func(param string)) Runner {
	return &defaultRunner{
		fileName:       configFile,
		onChangeConfig: onChangeConfig,
	}
}

func (r *defaultRunner) initConfig() {
	config.AllowEmptyEnv(true)
	config.SetConfigType("yaml")
	config.SetConfigFile(r.fileName)
	config.SetDefault("max_procs", 1)

	rand.Seed(time.Now().UnixNano())

	if err := config.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	runtime.GOMAXPROCS(config.GetInt("max_procs"))

	config.WatchConfig()
	config.OnConfigChange(func(e fsnotify.Event) {
		log.Trace("Config file changed: ", e.Name)
		for k, v := range r.onChangeConfig {
			v(k)
		}
	})
}

func initLog() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	log.SetOutput(os.Stdout)
	switch config.GetString("log_level") {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func initSignals(cancel context.CancelFunc) {
	osSignal := make(chan os.Signal, 1)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Debug("Listening to signals...")

	for {
		sig := <-osSignal

		log.Debugf("Caught signal: %v", sig)

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			cancel()
			return
		}
	}
}
