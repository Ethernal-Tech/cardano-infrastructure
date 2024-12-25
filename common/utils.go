package common

import (
	"context"
	"math/big"
)

// SplitString splits large string into slice of substrings
func SplitString(s string, mxlen int) (res []string) {
	for i := 0; i < len(s); i += mxlen {
		res = append(res, s[i:min(i+mxlen, len(s))])
	}

	return res
}

func WaitForAmount(
	ctx context.Context, receivedAmount *big.Int,
	getBalanceFn func(ctx context.Context) (*big.Int, error), options ...RetryConfigOption,
) (*big.Int, error) {
	originalAmount, err := ExecuteWithRetry(ctx, func(ctx context.Context) (*big.Int, error) {
		return getBalanceFn(ctx)
	})
	if err != nil {
		return nil, err
	}

	expectedBalance := originalAmount.Add(originalAmount, receivedAmount)

	return ExecuteWithRetry(ctx, func(ctx context.Context) (*big.Int, error) {
		balance, err := getBalanceFn(ctx)
		if err != nil {
			return nil, err
		}

		if balance.Cmp(expectedBalance) < 0 {
			return balance, ErrRetryTryAgain
		}

		return balance, nil
	}, options...)
}
