package transport

import (
	"push-sender/internal/push"
	"push-sender/internal/push/rustore"
	"push-sender/internal/task"

	log "github.com/sirupsen/logrus"
)

type rustoreSender struct {
	rustoreConfig RustoreConfig
}

func (a *rustoreSender) Send(task *task.Task) error {
	payload, ok := task.Payload.(string)

	if !ok {
		log.Errorf("rustore: bad payload %v", task.Payload)
		return push.ErrorRequest
	}

	return rustore.Send(task.To, payload, a.rustoreConfig.GetConfig(task.Project))
}

func NewRustoreTransport() Transport {
	return &rustoreSender{
		rustoreConfig: newDefaultRustoreConfig(),
	}
}
