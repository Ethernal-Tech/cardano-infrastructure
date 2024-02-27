package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

type LoggerConfig struct {
	LogLevel      hclog.Level
	JSONLogFormat bool
	AppendFile    bool
	LogFilePath   string
	Name          string
}

func NewLogger(config LoggerConfig) (l hclog.Logger, err error) {
	var logFileWriter *os.File

	if config.LogFilePath != "" {
		fullFilePath := filepath.Base(config.LogFilePath)

		if dir := filepath.Dir(config.LogFilePath); dir != "/" && strings.TrimLeft(dir, ".") != "" {
			if dirErr := os.MkdirAll(dir, os.ModePerm); dirErr == nil {
				fullFilePath = filepath.Join(dir, fullFilePath)
			}
		}

		if !config.AppendFile {
			timestamp := strings.Replace(strings.Replace(time.Now().UTC().Format(time.RFC3339), ":", "_", -1), "-", "_", -1)
			fullFilePath = fullFilePath + "_" + timestamp
		}

		logFileWriter, err = os.OpenFile(fullFilePath+".log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, fmt.Errorf("could not create or open log file, %w", err)
		}
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       config.Name,
		Level:      config.LogLevel,
		Output:     logFileWriter,
		JSONFormat: config.JSONLogFormat,
	}), nil
}
