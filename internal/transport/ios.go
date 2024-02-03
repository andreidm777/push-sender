package transport

import (
	"push-sender/internal/push"
	"push-sender/internal/push/ios"
	"push-sender/internal/task"

	log "github.com/sirupsen/logrus"
)

type iosSender struct {
	apnsSender  *ios.IosSender
	apnsConfigs ApnsConfig
}

func NewIosTransport() Transport {
	return &iosSender{
		apnsSender:  ios.New(),
		apnsConfigs: newDefaultApnsConfig(),
	}
}

func (a *iosSender) Send(task *task.Task) error {
	payload, ok := task.Payload.(string)
	if !ok {
		log.Errorf("ios: bad payload %v", task.Payload)
		return push.ErrorRequest
	}
	return a.apnsSender.Send(task.To, payload, a.apnsConfigs.GetConfig(task.Project))
}
