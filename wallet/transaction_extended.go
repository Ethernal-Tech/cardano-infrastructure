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

// CreateTxOutputChange generates a TxOutput representing the change
// by subtracting the total sum of outputs from the total available amount of each currency/token.
func CreateTxOutputChange(
	baseTxOutput TxOutput, totalSum map[string]uint64, outputsSum map[string]uint64,
) (TxOutput, error) {
	totalAmount := baseTxOutput.Amount + totalSum[AdaTokenName]
	outputAmount := outputsSum[AdaTokenName]

	if totalAmount < outputAmount {
		return TxOutput{}, fmt.Errorf("invalid amount: has = %d, required = %d", totalAmount, outputAmount)
	}

	changeAmount := totalAmount - outputAmount
	changeTokens := []TokenAmount(nil)

	for tokenName, tokenAmount := range totalSum {
		if tokenName == AdaTokenName {
			continue
		}

		outputTokenAmount := outputsSum[tokenName]
		totalTokenAmount := tokenAmount

		for _, token := range baseTxOutput.Tokens {
			if token.TokenName() == tokenName {
				totalTokenAmount += token.Amount

				break
			}
		}

		if totalTokenAmount < outputTokenAmount {
			return TxOutput{}, fmt.Errorf("invalid token amount: has = %d, required = %d",
				totalTokenAmount, outputTokenAmount)
		}

		changeTokenAmount := totalTokenAmount - outputTokenAmount
		if changeTokenAmount > 0 {
			newToken, err := NewTokenWithFullName(tokenName, true)
			if err != nil {
				return TxOutput{}, err
			}

			changeTokens = append(changeTokens, NewTokenAmount(newToken, changeTokenAmount))
		}
	}

	return TxOutput{
		Addr:   baseTxOutput.Addr,
		Amount: changeAmount,
		Tokens: changeTokens,
	}, nil
}
