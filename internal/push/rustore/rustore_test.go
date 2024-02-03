package rustore

import (
	"fmt"
	"testing"
)

func TestRuStore(m *testing.T) {
	opt := RuStoreMessageOpts{
		ApiKey:    "xxxxxxxxxxxxxxx",
		ProjectId: "xxxxxxxx",
	}

	data := `{"dry_run":false}`

	err := Send("xxxxxxxxxxxxxx", data, &opt)

	fmt.Printf("%s", err)
}
