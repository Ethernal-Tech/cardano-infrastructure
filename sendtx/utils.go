package sendtx

import (
	"context"
	"errors"
	"math/big"
	"sync"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
)

func WaitForTx(
	ctx context.Context,
	utxoRetriever cardanowallet.IUTxORetriever,
	receivers []BridgingTxReceiver,
	tokenName string,
	options ...infracommon.RetryConfigOption,
) error {
	errs := make([]error, len(receivers))
	wg := sync.WaitGroup{}

	for i, x := range receivers {
		wg.Add(1)

		go func(idx int, recv BridgingTxReceiver) {
			defer wg.Done()

			_, errs[idx] = infracommon.WaitForAmount(
				ctx, new(big.Int).SetUint64(recv.Amount), func(ctx context.Context) (*big.Int, error) {
					utxos, err := utxoRetriever.GetUtxos(ctx, recv.Addr)
					if err != nil {
						return nil, err
					}

					return new(big.Int).SetUint64(cardanowallet.GetUtxosSum(utxos)[tokenName]), nil
				}, options...)
		}(i, x)
	}

	wg.Wait()

	return errors.Join(errs...)
}
