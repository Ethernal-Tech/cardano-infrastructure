package logger

import (
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
)

type ILoggerContainer interface {
	GetLogger(s string) (hclog.Logger, error)
}

type LoggerContainerImpl struct {
	lock sync.Mutex

	loggers map[string]hclog.Logger
	config  LoggerConfig
}

func NewLoggerContainer(config LoggerConfig) *LoggerContainerImpl {
	return &LoggerContainerImpl{
		loggers: map[string]hclog.Logger{},
		config:  config,
	}
}

func (l *LoggerContainerImpl) GetLogger(s string) (hclog.Logger, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	logger, exists := l.loggers[s]
	if exists {
		return logger, nil
	}

	nc := l.config
	if nc.LogFilePath != "" {
		nc.LogFilePath = filepath.Join(nc.LogFilePath, s+".log")
	}

	newLogger, err := NewLogger(nc)
	if err != nil {
		return nil, err
	}

	l.loggers[s] = newLogger

	return newLogger, nil
}

type NullLoggerContainer struct{}

func NewNullLoggerContainer() *NullLoggerContainer {
	return &NullLoggerContainer{}
}

func (l *NullLoggerContainer) GetLogger(s string) (hclog.Logger, error) {
	return hclog.NewNullLogger(), nil
}
