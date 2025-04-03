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
	MaxSizeInMB  int  `json:"maxSizeInMB"`
	MaxBackups   int  `json:"maxBackups"`
	MaxAgeInDays int  `json:"maxAgeInDays"`
	Compress     bool `json:"compress"`
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
	logFilePath, _, err := createLogDir(config.LogFilePath)
	if err != nil {
		return nil, err
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       config.Name,
		Level:      config.LogLevel,
		JSONFormat: config.JSONLogFormat,
		Output: &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    config.RotatingLogerConfig.MaxSizeInMB,
			MaxBackups: config.RotatingLogerConfig.MaxBackups,
			MaxAge:     config.RotatingLogerConfig.MaxAgeInDays,
			Compress:   config.RotatingLogerConfig.Compress,
		},
	}), nil
}

func getLogFileWriter(logFilePath string, appendFile bool) (*os.File, error) {
	logFilePath, logFileDirectory, err := createLogDir(logFilePath)
	if err != nil {
		return nil, err
	}

	if !appendFile {
		suffix := strings.Replace(strings.Replace(time.Now().UTC().Format(time.RFC3339), ":", "_", -1), "-", "_", -1)
		logFileName := filepath.Base(logFilePath)

		if parts := strings.SplitN(logFileName, ".", 2); len(parts) == 1 {
			logFileName = fmt.Sprintf("%s_%s", parts[0], suffix)
		} else {
			logFileName = fmt.Sprintf("%s_%s.%s", parts[0], suffix, parts[1])
		}

		logFilePath = filepath.Join(logFileDirectory, logFileName)
	}

	return os.OpenFile(logFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
}

func createLogDir(logFilePath string) (string, string, error) {
	logFilePathTrimmed := strings.Trim(logFilePath, " ")
	logFileDirectory := filepath.Dir(logFilePathTrimmed)

	if err := common.CreateDirSafe(logFileDirectory, 0770); err != nil {
		return "", "", err
	}

	return logFilePathTrimmed, logFileDirectory, nil
}
