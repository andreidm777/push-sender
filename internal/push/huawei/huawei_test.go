package huawei

import (
	"fmt"
	"testing"
)

func TestMain(m *testing.M) {
	opt := HmsMessageOpts{
		ClientId:    "11111",
		ApiKey:      "xxxxxxxxxxxxxxxxxxxxxxxx",
		PackageName: "ru.mail.android_app_test",
	}

	sender := New()

	err := sender.Send("xxxxxxxxx", `{"dry_run":false,"data":{}}`, &opt)

	fmt.Printf("%s", err)
}
