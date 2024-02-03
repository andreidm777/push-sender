package transport

import "push-sender/internal/task"

type Transport interface {
	Send(qtask *task.Task) error
}

func GetTransport(t task.Platform) Transport {
	switch t {
	case task.Huawei:
		return NewHuaweiTransport()
	case task.Android:
		return NewAndroidTransport()
	case task.Rustore:
		return NewRustoreTransport()
	case task.Ios:
		return NewIosTransport()
	}
	return nil
}
