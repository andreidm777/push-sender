package ios

import (
	"testing"

	config "github.com/spf13/viper"
)

func init() {
	config.SetDefault("secret", "xxxxxxxxxxxxxxxxxxxx")
}

func TestMain(m *testing.M) {
}
