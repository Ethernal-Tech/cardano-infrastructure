package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getLogFileWriter(t *testing.T) {
	testDir, err := os.MkdirTemp("", "logger-test")
	require.NoError(t, err)

	defer func() {
		os.RemoveAll(testDir)
		os.Remove(testDir)
	}()

	filePathWithExtension := filepath.Join(testDir, "dummy1", "file.log")
	filePathWithoutExtension := filepath.Join(testDir, "dummy2", "file")

	t.Run("empty", func(t *testing.T) {
		f, err := getLogFileWriter(" ", true)
		require.NoError(t, err)
		require.Nil(t, f)
	})

	t.Run("with append", func(t *testing.T) {
		f, err := getLogFileWriter(filePathWithExtension, true)
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Equal(t, filePathWithExtension, f.Name())

		f, err = getLogFileWriter(filePathWithoutExtension, true)
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Equal(t, filePathWithoutExtension, f.Name())
	})

	t.Run("without append", func(t *testing.T) {
		f, err := getLogFileWriter(filePathWithExtension, false)
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("^%s/dummy1/file_.*\\.log$", testDir)), f.Name())

		f, err = getLogFileWriter(filePathWithoutExtension, false)
		require.NoError(t, err)
		require.NotNil(t, f)
		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("^%s/dummy2/file_.*$", testDir)), f.Name())
	})
}
