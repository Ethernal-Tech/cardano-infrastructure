package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExecuteWithRetry(t *testing.T) {
	t.Parallel()

	var (
		errWait = errors.New("hello wait")
		ctx     = context.Background()
	)

	options := []RetryConfigOption{
		WithRetryCount(10),
		WithRetryWaitTime(time.Millisecond * 5),
	}

	_, err := ExecuteWithRetry(ctx, func(_ int) (int, error) {
		return 0, errWait
	}, options...)

	require.ErrorIs(t, err, errWait)

	_, err = ExecuteWithRetry(ctx, func(_ int) (int, error) {
		return 0, errors.New("status code 500")
	}, options...)

	require.ErrorIs(t, err, ErrRetryTimeout)

	ctxWithCancel, cncl := context.WithCancel(ctx)
	go cncl()

	_, err = ExecuteWithRetry(ctxWithCancel, func(_ int) (int, error) {
		return 0, errors.New("status code 500")
	}, options...)

	require.ErrorIs(t, err, ctxWithCancel.Err())

	result, err := ExecuteWithRetry(ctx, func(cnt int) (int, error) {
		if cnt == 2 {
			return 8930, nil
		}

		return 0, errors.New("status code 500")
	}, options...)

	require.NoError(t, err)
	require.Equal(t, 8930, result)
}
