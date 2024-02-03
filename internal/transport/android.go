package transport

import (
	"push-sender/internal/push"
	"push-sender/internal/push/android"
	"push-sender/internal/task"

	log "github.com/sirupsen/logrus"
)

type sAndroid struct {
	androidSender *android.AndroidSender
	androidConfig AndroidConfig
	Opts          map[string]*android.FcmMessageOpts
}

func (a *sAndroid) Send(task *task.Task) error {
	opts, ok := a.Opts[task.Project]
	if !ok {
		var err error
		opts, err = a.androidConfig.GetConfig(task.Project)
		if err != nil {
			return err
		}
		a.Opts[task.Project] = opts
	}

	payload, ok := task.Payload.(map[string]string)

	if !ok {
		log.Errorf("android: bad payload %v", task.Payload)
		return push.ErrorRequest
	}

	return a.androidSender.Send(task.To, payload, opts)
}

func NewAndroidTransport() Transport {
	return &sAndroid{
		androidSender: android.New(),
		androidConfig: newDefaultAndroidConfig(),
		Opts:          make(map[string]*android.FcmMessageOpts),
	}
}
