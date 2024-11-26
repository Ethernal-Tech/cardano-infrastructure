package wallet

import (
	"context"
	"fmt"
)

const defaultTimeToLiveInc = 200

func (b *TxBuilder) SetProtocolParametersAndTTL(
	ctx context.Context, retriever ITxDataRetriever, timeToLiveInc uint64,
) error {
	if timeToLiveInc == 0 {
		timeToLiveInc = defaultTimeToLiveInc
	}

	protocolParams, err := retriever.GetProtocolParameters(ctx)
	if err != nil {
		return err
	}

	tip, err := retriever.GetTip(ctx)
	if err != nil {
		return err
	}

	b.SetProtocolParameters(protocolParams).SetTimeToLive(tip.Slot + timeToLiveInc)

	return nil
}

type TxInputs struct {
	Inputs         []TxInput
	Sum            uint64
	NativeTokenSum uint64
}

func GetUTXOsForAmount(
	ctx context.Context, retriever IUTxORetriever, addr string, exactSum uint64, atLeastSum uint64,
) (TxInputs, error) {
	utxos, err := retriever.GetUtxos(ctx, addr)
	if err != nil {
		return TxInputs{}, err
	}

	// Loop through utxos to find first input with enough tokens
	// If we don't have this UTXO we need to use more of them
	//nolint:prealloc
	var (
		amountSum   = uint64(0)
		chosenUTXOs []TxInput
	)

	for _, utxo := range utxos {
		amountSum += utxo.Amount
		chosenUTXOs = append(chosenUTXOs, TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		})

		if amountSum == exactSum || amountSum >= atLeastSum {
			return TxInputs{
				Inputs: chosenUTXOs,
				Sum:    amountSum,
			}, nil
		}
	}

	return TxInputs{}, fmt.Errorf("not enough funds for the transaction: (available, exact, at least) = (%d, %d, %d)",
		amountSum, exactSum, atLeastSum)
}

func GetUTXOsForAmountandNativeTokens(
	ctx context.Context, retriever IUTxORetriever, addr string, exactSum uint64, atLeastSum uint64, nativeTokensSum uint64,
) (TxInputs, error) {
	utxos, err := retriever.GetUtxos(ctx, addr)
	if err != nil {
		return TxInputs{}, err
	}

	// Loop through utxos to find first input with enough tokens
	// If we don't have this UTXO we need to use more of them
	//nolint:prealloc
	var (
		amountSum   = uint64(0)
		ntAmountSum = uint64(0)
		chosenUTXOs []TxInput
	)

	for _, utxo := range utxos {
		amountSum += utxo.Amount
		ntAmountSum += sumOfNativeTokens(utxo)

		chosenUTXOs = append(chosenUTXOs, TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		})

		fmt.Printf("IN amnt: %d, nt amnt: %d \n", utxo.Amount, sumOfNativeTokens(utxo))
		if (amountSum == exactSum || amountSum >= atLeastSum) && (ntAmountSum >= nativeTokensSum) {
			return TxInputs{
				Inputs:         chosenUTXOs,
				Sum:            amountSum,
				NativeTokenSum: ntAmountSum,
			}, nil
		}
	}

	return TxInputs{}, fmt.Errorf("not enough funds for the transaction: (available, exact, at least) = (%d, %d, %d)",
		amountSum, exactSum, atLeastSum)
}

// DN_TODO: FIX get sum per policyID / token name
func sumOfNativeTokens(utxo Utxo) uint64 {

	sum := uint64(0)

	for _, v := range utxo.Tokens {
		for _, value := range v {
			sum += value
		}
	}

	return sum
}
