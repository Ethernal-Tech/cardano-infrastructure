package common

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var ErrTimeout = errors.New("timeout")

// ExecuteWithRetry attempts to execute a provided handler function multiple times
// with retries in case of failure, respecting a specified wait time between attempts.
func ExecuteWithRetry(
	ctx context.Context, handler func() (bool, error), numRetries int, waitTime time.Duration,
) error {
	for count := 0; count < numRetries; count++ {
		stop, err := handler()
		if stop {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}

	return ErrTimeout
}

// SetupDataDir sets up the data directory and the corresponding sub-directories
func SetupDataDir(dataDir string, paths []string, perms fs.FileMode) error {
	if err := CreateDirSafe(dataDir, perms); err != nil {
		return fmt.Errorf("failed to create data dir: (%s): %w", dataDir, err)
	}

	for _, path := range paths {
		path := filepath.Join(dataDir, path)
		if err := CreateDirSafe(path, perms); err != nil {
			return fmt.Errorf("failed to create path: (%s): %w", path, err)
		}
	}

	return nil
}

// Creates a directory at path and with perms level permissions.
// If directory already exists, owner and permissions are verified.
func CreateDirSafe(path string, perms fs.FileMode) error {
	info, err := os.Stat(path)
	// check if an error occurred other than path not exists
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// create directory if it does not exist
	if !DirectoryExists(path) {
		if err := os.MkdirAll(path, perms); err != nil {
			return err
		}

		return nil
	}

	// verify that existing directory's owner and permissions are safe
	return verifyFileOwnerAndPermissions(path, info, perms)
}

// Creates a file at path and with perms level permissions.
// If file already exists, owner and permissions are
// verified, and the file is overwritten.
func SaveFileSafe(path string, data []byte, perms fs.FileMode) error {
	info, err := os.Stat(path)
	// check if an error occurred other than path not exists
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if FileExists(path) {
		// verify that existing file's owner and permissions are safe
		if err := verifyFileOwnerAndPermissions(path, info, perms); err != nil {
			return err
		}
	}

	// create or overwrite the file
	return os.WriteFile(path, data, perms)
}

// Verifies that the file owner is the current user,
// or the file owner is in the same group as current user
// and permissions are set correctly by the owner.
func verifyFileOwnerAndPermissions(path string, info fs.FileInfo, expectedPerms fs.FileMode) error {
	// get stats
	stat, ok := info.Sys().(*syscall.Stat_t)
	if stat == nil || !ok {
		return fmt.Errorf("failed to get stats of %s", path)
	}

	// get current user
	currUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user")
	}

	// get user id of the owner
	ownerUID := strconv.FormatUint(uint64(stat.Uid), 10)
	if currUser.Uid == ownerUID {
		return nil
	}

	// get group id of the owner
	ownerGID := strconv.FormatUint(uint64(stat.Gid), 10)
	if currUser.Gid != ownerGID {
		return fmt.Errorf("file/directory created by a user from a different group: %s", path)
	}

	// check if permissions are set correctly by the owner
	if info.Mode() != expectedPerms {
		return fmt.Errorf("permissions of the file/directory '%s' are set incorrectly by another user", path)
	}

	return nil
}

// DirectoryExists checks if the directory at the specified path exists
func DirectoryExists(directoryPath string) bool {
	// Check if path is empty
	if directoryPath == "" {
		return false
	}

	// Grab the absolute filepath
	pathAbs, err := filepath.Abs(directoryPath)
	if err != nil {
		return false
	}

	// Check if the directory exists, and that it's actually a directory if there is a hit
	if fileInfo, statErr := os.Stat(pathAbs); os.IsNotExist(statErr) || (fileInfo != nil && !fileInfo.IsDir()) {
		return false
	}

	return true
}

// Checks if the file at the specified path exists
func FileExists(filePath string) bool {
	// Check if path is empty
	if filePath == "" {
		return false
	}

	// Grab the absolute filepath
	pathAbs, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	// Check if the file exists, and that it's actually a file if there is a hit
	if fileInfo, statErr := os.Stat(pathAbs); os.IsNotExist(statErr) || (fileInfo != nil && fileInfo.IsDir()) {
		return false
	}

	return true
}
