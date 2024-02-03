package transport

import (
	"push-sender/internal/push/huawei"

	config "github.com/spf13/viper"
)

type HuaweiConfig interface {
	GetConfig(projectId string) *huawei.HmsMessageOpts
}

type defaultHuaweiConfig struct {
}

func newDefaultHuaweiConfig() HuaweiConfig {
	return &defaultHuaweiConfig{}
}

func (c *defaultHuaweiConfig) GetConfig(projectId string) *huawei.HmsMessageOpts {
	return &huawei.HmsMessageOpts{
		ClientId:    projectId,
		ApiKey:      config.GetString(projectId + ".apikey"),
		PackageName: config.GetString(projectId + ".package_name"),
	}

}
