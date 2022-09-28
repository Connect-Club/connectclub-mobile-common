package common

import (
	"github.com/Connect-Club/connectclub-mobile-common/logs"
)

type PublicLogger interface {
	logs.Logger
}

func NewLogger(caller string) PublicLogger {
	return logs.New(caller)
}

func InitLoggerFile(filesDir string, filename string) {
	logs.InitLoggerFile(filesDir, filename)
}

func PreserveLogFile(filesDir string, filename string) {
	logs.PreserveLogFile(filesDir, filename)
}
