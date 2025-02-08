package conf

import (
	"log/slog"
	"os"
	"strconv"
)

type Log struct {
	*slog.Logger
}

func NewLog() *Log {
	// default log level is info
	return newLog(slog.LevelInfo)
}

func newLog(level slog.Level) *Log {
	logLevel := level
	// override log level if set in env
	if envLevelStr := GetEnv(ENV_LOG_LEVEL, LOG_LEVEL_INFO); envLevelStr != "" {
		envLevel, err := strconv.Atoi(envLevelStr)
		if err == nil {
			logLevel = slog.Level(envLevel)
		}
	}
	opts := slog.HandlerOptions{
		Level: logLevel,
	}
	handler := slog.NewTextHandler(os.Stdout, &opts)
	logger := slog.New(handler)
	// set default logger
	slog.SetDefault(logger)
	return &Log{logger}
}

func (l *Log) WithError(err error, msg string, args ...any) *Log {
	l.With("error", err).With(args...).Error(msg)
	return l
}
