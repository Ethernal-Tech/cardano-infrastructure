package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	testDir, err := os.MkdirTemp("", "new-logger-test")
	require.NoError(t, err)

	defer os.RemoveAll(testDir)

	filePath := filepath.Join(testDir, "dummy", "file.log")

	t.Run("rotating empty", func(t *testing.T) {
		_, err := NewLogger(LoggerConfig{
			RotatingLogsEnabled: true,
		})
		require.Error(t, err)
	})

	t.Run("rotating with file path", func(t *testing.T) {
		logger, err := NewLogger(LoggerConfig{
			RotatingLogsEnabled: true,
			LogFilePath:         filePath,
		})

		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("empty", func(t *testing.T) {
		logger, err := NewLogger(LoggerConfig{
			RotatingLogsEnabled: false,
		})

		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("with file path", func(t *testing.T) {
		logger, err := NewLogger(LoggerConfig{
			RotatingLogsEnabled: false,
			LogFilePath:         filePath,
		})

		require.NoError(t, err)
		require.NotNil(t, logger)
	})
}

func TestGetLogFileWriter(t *testing.T) {
	testDir, err := os.MkdirTemp("", "logger-test")
	require.NoError(t, err)

	defer os.RemoveAll(testDir)

	filePathWithExtension := filepath.Join(testDir, "dummy1", "file.log")
	filePathWithoutExtension := filepath.Join(testDir, "dummy2", "file")

	t.Run("empty", func(t *testing.T) {
		f, err := getLogFileWriter(LoggerConfig{LogFilePath: " ", AppendFile: true})
		require.NoError(t, err)
		require.Nil(t, f)
	})

	t.Run("with append", func(t *testing.T) {
		f, err := getLogFileWriter(LoggerConfig{LogFilePath: filePathWithExtension, AppendFile: true})
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Equal(t, filePathWithExtension, f.Name())

		f, err = getLogFileWriter(LoggerConfig{LogFilePath: filePathWithoutExtension, AppendFile: true})
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Equal(t, filePathWithoutExtension, f.Name())
	})

	t.Run("without append", func(t *testing.T) {
		f, err := getLogFileWriter(LoggerConfig{LogFilePath: filePathWithExtension, AppendFile: false})
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("^%s/dummy1/file_.*\\.log$", testDir)), f.Name())

		f, err = getLogFileWriter(LoggerConfig{LogFilePath: filePathWithoutExtension, AppendFile: false})
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("^%s/dummy2/file_.*$", testDir)), f.Name())
	})
}
