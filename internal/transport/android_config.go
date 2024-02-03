package transport

import (
	"push-sender/internal/push/android"

	config "github.com/spf13/viper"
)

type AndroidConfig interface {
	GetConfig(bundleId string) (*android.FcmMessageOpts, error)
}

type defaultAndroidConfig struct {
}

func newDefaultAndroidConfig() AndroidConfig {
	return &defaultAndroidConfig{}
}

func (c *defaultAndroidConfig) GetConfig(projectId string) (*android.FcmMessageOpts, error) {
	return android.MakeFcmMessageOpts(
		config.GetString(projectId+".cert"),
		24*60*60,
	)
}
