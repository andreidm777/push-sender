package transport

import (
	"push-sender/internal/push/ios"

	config "github.com/spf13/viper"
)

type ApnsConfig interface {
	GetConfig(bundleId string) *ios.ApnsOptions
}

type defaultApnsConfig struct {
}

func newDefaultApnsConfig() ApnsConfig {
	return &defaultApnsConfig{}
}

func (c *defaultApnsConfig) GetConfig(bundleId string) *ios.ApnsOptions {
	return &ios.ApnsOptions{
		Cert:     config.GetString(bundleId + ".cert"),
		BundleId: bundleId,
	}

}
