package main

import (
	"os"
	"push-sender/cmd"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Errorf("error while starting %s", err)
		os.Exit(2)
	}
}
