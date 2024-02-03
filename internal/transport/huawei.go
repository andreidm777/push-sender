package transport

import (
	"push-sender/internal/push"
	"push-sender/internal/push/huawei"
	"push-sender/internal/task"

	log "github.com/sirupsen/logrus"
)

type huaweiSender struct {
	hmsSender *huawei.HmsSender
	hmsConfig HuaweiConfig
}

func (a *huaweiSender) Send(task *task.Task) error {
	opts := a.hmsConfig.GetConfig(task.Project)

	payload, ok := task.Payload.(string)

	if !ok {
		log.Errorf("android: bad payload %v", task.Payload)
		return push.ErrorRequest
	}

	return a.hmsSender.Send(task.To, payload, opts)
}

func NewHuaweiTransport() Transport {
	return &huaweiSender{
		hmsSender: huawei.New(),
		hmsConfig: newDefaultHuaweiConfig(),
	}
}
