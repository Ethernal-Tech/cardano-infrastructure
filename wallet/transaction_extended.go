package wallet

import (
	"context"
	"fmt"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
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

func GetUTXOsForAmount(
	ctx context.Context,
	retriever IUTxORetriever,
	addr string,
	tokenNames []string,
	exactSum map[string]uint64,
	atLeastSum map[string]uint64,
) (TxInputs, error) {
	utxos, err := common.ExecuteWithRetry(ctx, func(ctx context.Context) ([]Utxo, error) {
		return retriever.GetUtxos(ctx, addr)
	})
	if err != nil {
		return TxInputs{}, err
	}

	// Loop through utxos to find first input with enough tokens
	// If we don't have this UTXO we need to use more of them
	//nolint:prealloc
	var (
		currentSum       = map[string]uint64{}
		chosenUTXOs      []TxInput
		notGoodTokenName string
	)

	for _, utxo := range utxos {
		currentSum[AdaTokenName] += utxo.Amount

		for _, token := range utxo.Tokens {
			currentSum[token.TokenName()] += token.Amount
		}

		chosenUTXOs = append(chosenUTXOs, TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		})

		isOk := true

		for _, tokenName := range tokenNames {
			if currentSum[tokenName] != exactSum[tokenName] && currentSum[tokenName] < atLeastSum[tokenName] {
				isOk = false
				notGoodTokenName = tokenName

				break
			}
		}

		if isOk {
			return TxInputs{
				Inputs: chosenUTXOs,
				Sum:    currentSum,
			}, nil
		}
	}

	return TxInputs{}, fmt.Errorf("not enough funds for the transaction: (available, exact, at least) = (%d, %d, %d)",
		currentSum[notGoodTokenName], exactSum[notGoodTokenName], atLeastSum[notGoodTokenName])
}

func GetTokenCostSum(txBuilder *TxBuilder, userAddress string, utxos []Utxo) (uint64, error) {
	userTokenSum := GetUtxosSum(utxos)

	txOutput := TxOutput{
		Addr:   userAddress,
		Amount: userTokenSum[AdaTokenName],
	}

	for tokenName, amount := range userTokenSum {
		if tokenName != AdaTokenName {
			tokenAmount, err := NewTokenAmountWithFullName(tokenName, amount, false)
			if err != nil {
				tokenAmount, err = NewTokenAmountWithFullName(tokenName, amount, true)
				if err != nil {
					return 0, err
				}
			}

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
			newToken, err := NewTokenAmountWithFullName(tokenName, changeTokenAmount, true)
			if err != nil {
				return TxOutput{}, err
			}

			changeTokens = append(changeTokens, newToken)
		}
	}

	return TxOutput{
		Addr:   baseTxOutput.Addr,
		Amount: changeAmount,
		Tokens: changeTokens,
	}, nil
}
