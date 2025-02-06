package common

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

const (
	defaultRetryCount    = 10
	defaultRetryWaitTime = time.Second * 5
)

var (
	ErrRetryTimeout  = errors.New("timeout")
	ErrRetryTryAgain = errors.New("retry try again")
	defaultLogger    = hclog.NewNullLogger()
)

// RetryConfig defines ExecuteWithRetry configuration
type RetryConfig struct {
	retryCount       int
	retryWaitTime    time.Duration
	isRetryableError func(err error) bool
	logger           hclog.Logger
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

func WithLogger(logger hclog.Logger) RetryConfigOption {
	return func(c *RetryConfig) {
		c.logger = logger
	}
}

// ExecuteWithRetry attempts to execute a provided handler function multiple times
// with retries in case of failure, respecting a specified wait time between attempts.
func ExecuteWithRetry[T any](
	ctx context.Context, handler func(context.Context) (T, error), options ...RetryConfigOption,
) (result T, err error) {
	config := RetryConfig{
		retryCount:       defaultRetryCount,
		retryWaitTime:    defaultRetryWaitTime,
		isRetryableError: IsRetryableError,
		logger:           defaultLogger,
	}

	for _, opt := range options {
		opt(&config)
	}

	for count := 0; count < config.retryCount; count++ {
		result, err = handler(ctx)
		if err != nil {
			if !config.isRetryableError(err) {
				return result, err
			}

			if !errors.Is(err, ErrRetryTryAgain) { // do not log ErrRetryTryAgain errors
				config.logger.Info("ExecuteWithRetry failed. Retrying...", "time", count+1, "err", err)
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

// IsRetryableErrorDefault returns true if error is retryable
// handles context errors, net.Errors and ErrRetryTryAgain
func IsRetryableErrorDefault(err error) bool {
	// Context was explicitly canceled or deadline exceeded; not retryable
	if IsContextDoneErr(err) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return errors.Is(err, ErrRetryTryAgain)
}

// IsRetryableError returns true if error is retryable
// includes ogmios and go-ethereum/blade specific errors
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	retryableMessages := []string{
		"replacement transaction underpriced",
		"replacement tx underpriced",
		"nonce too low",
		"intrinsic gas too low",
		"tx with the same nonce is already present",
		"rejected future tx due to low slots",
		"transaction underpriced",
		"status code 429", // blockfrost rate limiter
		"status code 500", // ogmios "status code 500" error
	}
	errStr := err.Error()

	for _, msg := range retryableMessages {
		if strings.Contains(errStr, msg) {
			return true
		}
	}

	return IsRetryableErrorDefault(err)
}
