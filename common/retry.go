package common

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

var ErrRetryTimeout = errors.New("timeout")

// RetryConfig defines ExecuteWithRetry configuration
type RetryConfig struct {
	retryCount       int
	retryWaitTime    time.Duration
	isRetryableError func(err error) bool
}

// RetryConfigOption defines ExecuteWithRetry configuration option
type RetryConfigOption func(c *RetryConfig)

func WithRetryCount(retryCount int) RetryConfigOption {
	return func(c *RetryConfig) {
		c.retryCount = retryCount
	}
}

func WithRetryWaitTime(retryWaitTime time.Duration) RetryConfigOption {
	return func(c *RetryConfig) {
		c.retryWaitTime = retryWaitTime
	}
}

func WithIsRetryableError(fn func(err error) bool) RetryConfigOption {
	return func(c *RetryConfig) {
		c.isRetryableError = fn
	}
}

// ExecuteWithRetry attempts to execute a provided handler function multiple times
// with retries in case of failure, respecting a specified wait time between attempts.
func ExecuteWithRetry[T any](
	ctx context.Context, handler func(int) (T, error), options ...RetryConfigOption,
) (result T, err error) {
	config := RetryConfig{
		retryCount:       10,
		retryWaitTime:    time.Second * 2,
		isRetryableError: isRetryableErrorDefault,
	}

	for _, opt := range options {
		opt(&config)
	}

	for count := 0; count < config.retryCount; count++ {
		result, err = handler(count)
		if err != nil {
			if !config.isRetryableError(err) {
				return result, err
			}
		} else {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(config.retryWaitTime):
		}
	}

	return result, ErrRetryTimeout
}

// IsContextDoneErr returns true if the error is due to the context being cancelled
// or expired. This is useful for determining if a function should retry.
func IsContextDoneErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func isRetryableErrorDefault(err error) bool {
	// Context was explicitly canceled or deadline exceeded; not retryable
	if IsContextDoneErr(err) {
		return false
	}

	if _, isNetError := err.(net.Error); isNetError {
		return true
	}

	return strings.Contains(err.Error(), "status code 500") // retry if error is ogmios "status code 500"
}