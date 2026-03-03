package common

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	_, err := ExecuteWithRetry(ctx, func(_ context.Context) (int, error) {
		return 0, errWait
	}, options...)

	require.ErrorIs(t, err, errWait)

	i := 0

	_, err = ExecuteWithRetry(ctx, func(_ context.Context) (int, error) {
		i++
		if i&1 == 1 {
			return 0, &net.DNSError{IsTimeout: true}
		} else if i&3 == 0 {
			return 0, ErrRetryTryAgain
		}

		return 0, errors.New("status code 500")
	}, options...)

	require.ErrorIs(t, err, ErrRetryTimeout)
	require.Equal(t, 10, i)

	ctxWithCancel, cncl := context.WithCancel(ctx)
	go cncl()

	_, err = ExecuteWithRetry(ctxWithCancel, func(_ context.Context) (int, error) {
		return 0, errors.New("status code 500")
	}, options...)

	require.ErrorIs(t, err, ctxWithCancel.Err())

	result, err := ExecuteWithRetry(ctx, func(cnt context.Context) (int, error) {
		return 8930, nil
	}, options...)

	require.NoError(t, err)
	require.Equal(t, 8930, result)
}

func TestIsRetryableError(t *testing.T) {
	assert.False(t, IsRetryableError(nil))
	assert.False(t, IsRetryableError(context.Canceled))
	assert.False(t, IsRetryableError(context.DeadlineExceeded))
	assert.True(t, IsRetryableError(ErrRetryTryAgain))
	assert.False(t, IsRetryableError(net.ErrClosed))
	assert.True(t, IsRetryableError(&net.DNSError{IsTimeout: true}))
	assert.True(t, IsRetryableError(errors.New("replacement tx underpriced")))
}

func TestIsRetryableError_Resolver(t *testing.T) {
	// Reproduce "lookup rpc.nexus.testnet.apexfusion.org: i/o timeout" by using
	// a resolver that dials a non-responsive address so the DNS connection times out.
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Millisecond}
			// RFC 5737 documentation address; no DNS server there, so dial times out
			return d.DialContext(ctx, network, "198.51.100.1:53")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := resolver.LookupHost(ctx, "rpc.nexus.testnet.apexfusion.org")
	require.Error(t, err)
	require.Contains(t, err.Error(), "i/o timeout")
	require.Contains(t, err.Error(), "rpc.nexus.testnet.apexfusion.org")

	var netErr net.Error

	require.True(t, errors.As(err, &netErr))
	assert.True(t, netErr.Timeout(), "DNSError should be a timeout")
	assert.True(t, IsRetryableError(err), "DNS i/o timeout should be retryable")
}
