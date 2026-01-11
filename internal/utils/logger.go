package utils

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// InitLogger initializes the global logger
func InitLogger(level string) {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Log.SetLevel(logLevel)
}

// GetLogger returns the global logger
func GetLogger() *logrus.Logger {
	if Log == nil {
		InitLogger("info")
	}
	return Log
}
