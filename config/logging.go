package config

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = logrus.New()

// InitLogger initializes the logging setup using Logrus
func InitLogger() {
	// Create log directory if not exists
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0755)
	}

	// Set output to a log file with rotation (using lumberjack)
	Log.Out = &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10,   // Megabytes before log is rotated
		MaxBackups: 3,    // Number of old logs to keep
		MaxAge:     28,   // Maximum number of days to retain old log files
		Compress:   true, // Compress backups
	}

	// Get log level from config (e.g., from .env or config.yaml)
	logLevel := viper.GetString("LOG_LEVEL")
	switch logLevel {
	case "debug":
		Log.SetLevel(logrus.DebugLevel)
	case "warn":
		Log.SetLevel(logrus.WarnLevel)
	case "error":
		Log.SetLevel(logrus.ErrorLevel)
	default:
		Log.SetLevel(logrus.InfoLevel) // Default log level is info
	}

	// Set log format to JSON
	Log.SetFormatter(&logrus.JSONFormatter{})

	Log.Info("Logger telah diinisialisasi dengan sukses!")
}
