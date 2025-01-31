package wallet

import (
	"context"
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
	Inputs []TxInput
	Sum    map[string]uint64
}

func GetTokenCostSum(txBuilder *TxBuilder, userAddress string, utxos []Utxo) (uint64, error) {
	userTokenSum := GetUtxosSum(utxos)

	txOutput := TxOutput{
		Addr:   userAddress,
		Amount: userTokenSum[AdaTokenName],
	}

	for tokenName, amount := range userTokenSum {
		if tokenName != AdaTokenName {
			token, err := NewTokenWithFullName(tokenName, false)
			if err != nil {
				token, err = NewTokenWithFullName(tokenName, true)
				if err != nil {
					return 0, err
				}
			}

			tokenAmount := NewTokenAmount(token, amount)

			txOutput.Tokens = append(txOutput.Tokens, tokenAmount)
		}
	}
	retSum, err := txBuilder.CalculateMinUtxo(txOutput)
	if err != nil {
		return 0, err
	}
	return retSum, nil
}
