package common

import (
	log "github.com/sirupsen/logrus"
	"os/signal"
	"syscall"
)

func Initialize() {
	signal.Ignore(syscall.SIGPIPE)
	log.Info("GO INIT GO!!!!!")
}
