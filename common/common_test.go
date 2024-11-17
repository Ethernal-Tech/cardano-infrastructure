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

	require.ErrorIs(t, ExecuteWithRetry(ctx, func() (bool, error) {
		return true, errWait
	}, 10, time.Millisecond*5), errWait)

	require.ErrorIs(t, ExecuteWithRetry(ctx, func() (bool, error) {
		return false, errWait
	}, 10, time.Millisecond*5), ErrTimeout)

	ctx, cncl := context.WithCancel(ctx)
	go cncl()

	require.ErrorIs(t, ExecuteWithRetry(ctx, func() (bool, error) {
		return false, nil
	}, 10, time.Millisecond*5), ctx.Err())

	require.NoError(t, ExecuteWithRetry(ctx, func() (bool, error) {
		return true, nil
	}, 10, time.Millisecond*5))
}
