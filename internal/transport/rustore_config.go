package transport

import (
	"push-sender/internal/push/rustore"

	config "github.com/spf13/viper"
)

type RustoreConfig interface {
	GetConfig(projectId string) *rustore.RuStoreMessageOpts
}

type defaultRustoreConfig struct {
}

func newDefaultRustoreConfig() RustoreConfig {
	return &defaultRustoreConfig{}
}

func (c *defaultRustoreConfig) GetConfig(projectId string) *rustore.RuStoreMessageOpts {
	return &rustore.RuStoreMessageOpts{
		ApiKey:    config.GetString(projectId + ".apikey"),
		ProjectId: projectId,
	}
}
