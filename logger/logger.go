package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/hashicorp/go-hclog"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type RotatingLoggerConfig struct {
	MaxSize    int  `json:"maxSize"`
	MaxBackups int  `json:"maxBackups"`
	MaxAge     int  `json:"maxAge"`
	Compress   bool `json:"compress"`
}

type LoggerConfig struct {
	RotatingLogsEnabled bool                 `json:"rotatingLogsEnabled"`
	RotatingLogerConfig RotatingLoggerConfig `json:"rotatingLogerConfig"`
	LogLevel            hclog.Level          `json:"logLevel"`
	JSONLogFormat       bool                 `json:"jsonLogFormat"`
	AppendFile          bool                 `json:"appendFile"`
	LogFilePath         string               `json:"logFilePath"`
	Name                string               `json:"name"`
}

func NewLogger(config LoggerConfig) (hclog.Logger, error) {
	if config.RotatingLogsEnabled {
		return newRotatingLogger(config)
	}

	output, err := getLogFileWriter(config.LogFilePath, config.AppendFile)
	if err != nil {
		return nil, fmt.Errorf("could not create or open log file: %w", err)
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       config.Name,
		Level:      config.LogLevel,
		Output:     output,
		JSONFormat: config.JSONLogFormat,
	}), nil
}

func newRotatingLogger(config LoggerConfig) (hclog.Logger, error) {
	logFilePathTrimmed, _, err := createLogDir(config.LogFilePath)
	if err != nil {
		return nil, err
	}

	lumber := &lumberjack.Logger{
		Filename:   logFilePathTrimmed,
		MaxSize:    config.RotatingLogerConfig.MaxSize,
		MaxBackups: config.RotatingLogerConfig.MaxBackups,
		MaxAge:     config.RotatingLogerConfig.MaxAge,
		Compress:   config.RotatingLogerConfig.Compress,
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       config.Name,
		Level:      config.LogLevel,
		Output:     lumber,
		JSONFormat: config.JSONLogFormat,
	}), nil
}

func getLogFileWriter(logFilePath string, appendFile bool) (*os.File, error) {
	logFilePathTrimmed, logFileDirectory, err := createLogDir(logFilePath)
	if err != nil {
		return nil, err
	}

	if !appendFile {
		suffix := strings.Replace(strings.Replace(time.Now().UTC().Format(time.RFC3339), ":", "_", -1), "-", "_", -1)
		logFileName := filepath.Base(logFilePathTrimmed)

		if parts := strings.SplitN(logFileName, ".", 2); len(parts) == 1 {
			logFileName = fmt.Sprintf("%s_%s", parts[0], suffix)
		} else {
			logFileName = fmt.Sprintf("%s_%s.%s", parts[0], suffix, parts[1])
		}

		logFilePathTrimmed = filepath.Join(logFileDirectory, logFileName)
	}

	return os.OpenFile(logFilePathTrimmed, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
}

func createLogDir(logFilePath string) (string, string, error) {
	logFilePathTrimmed := strings.Trim(logFilePath, " ")
	if logFilePathTrimmed == "" {
		return "", "", fmt.Errorf("log file path is empty")
	}

	logFileDirectory := filepath.Dir(logFilePathTrimmed)

	if err := common.CreateDirSafe(logFileDirectory, 0770); err != nil {
		return "", "", err
	}

	return logFilePathTrimmed, logFileDirectory, nil
}
