package bridgingtx

import (
	"fmt"
	"strconv"
	"strings"

	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
)

type AmountCondition struct {
	Exact   uint64
	AtLeast uint64
}

// GetUTXOsForAmounts selects UTXOs that fulfill specified token amount conditions while adhering
// to a maximum input limit per transaction.
//
// Parameters:
// - utxos: A list of available UTXOs for selection.
// - conditions: A map defining required token conditions (e.g., exact or minimum amounts).
// - maxInputsPerTx: The maximum number of UTXOs allowed in the transaction.
//
// Returns:
// - cardanowallet.TxInputs: Selected UTXOs and their total sum if conditions are met.
// - error: An error if no valid selection can satisfy the conditions.
//
// The function iteratively selects UTXOs, replacing the smallest ones when the limit is reached,
// until the specified conditions are satisfied or no valid selection is possible.
func GetUTXOsForAmounts(
	utxos []cardanowallet.Utxo,
	conditions map[string]AmountCondition,
	maxInputsPerTx int,
) (cardanowallet.TxInputs, error) {
	currentSum := map[string]uint64{}
	choosenCount := 0

	for _, utxo := range utxos {
		utxos[choosenCount] = utxo
		choosenCount++
		currentSum[cardanowallet.AdaTokenName] += utxo.Amount

		for _, token := range utxo.Tokens {
			currentSum[token.TokenName()] += token.Amount
		}

		if isSumSatisfiesCondition(currentSum, conditions) {
			return cardanowallet.TxInputs{
				Inputs: utxosToTxInputs(utxos[:choosenCount]),
				Sum:    currentSum,
			}, nil
		}

		// replace the smallest utxo with the current one
		if choosenCount == maxInputsPerTx {
			minChosenUTXO, minChosenUTXOIdx := findMinUtxo(utxos[:choosenCount], currentSum, conditions)

			choosenCount--
			utxos[minChosenUTXOIdx] = utxo
			currentSum[cardanowallet.AdaTokenName] -= minChosenUTXO.Amount

			for _, token := range minChosenUTXO.Tokens {
				currentSum[token.TokenName()] -= token.Amount
			}
		}
	}

	return cardanowallet.TxInputs{}, fmt.Errorf(
		"not enough funds for the transaction: (available, conditions) = (%s, %s)",
		mapStrUInt64ToStr(currentSum), condMapToStr(conditions))
}

func utxosToTxInputs(utxos []cardanowallet.Utxo) []cardanowallet.TxInput {
	txInputs := make([]cardanowallet.TxInput, len(utxos))
	for i, utxo := range utxos {
		txInputs[i] = cardanowallet.TxInput{
			Hash:  utxo.Hash,
			Index: utxo.Index,
		}
	}

	return txInputs
}

func findMinUtxo(
	utxos []cardanowallet.Utxo, currentSum map[string]uint64, conditions map[string]AmountCondition,
) (cardanowallet.Utxo, int) {
	replaceTokenName := ""
	biggestDiff := uint64(0)
	// take the token with the biggest difference as the one to replace
	for tokenName, amount := range conditions {
		if diff := amount.AtLeast - currentSum[tokenName]; diff > biggestDiff {
			diff = biggestDiff
			replaceTokenName = tokenName
		}
	}

	min := utxos[0]
	idx := 0

	// two lops, one for ada and one for tokens
	if replaceTokenName == cardanowallet.AdaTokenName {
		for i, utxo := range utxos[1:] {
			if utxo.Amount < min.Amount {
				min = utxo
				idx = i + 1
			}
		}
	} else {
		for i, utxo := range utxos[1:] {
			for _, token := range utxo.Tokens {
				if token.TokenName() == replaceTokenName && token.Amount < min.Amount {
					min = utxo
					idx = i + 1
				}
			}
		}
	}

	return min, idx
}

func isSumSatisfiesCondition(
	currentSum map[string]uint64, conditions map[string]AmountCondition,
) bool {
	for tokenName, amount := range conditions {
		if currentSum[tokenName] != amount.Exact && currentSum[tokenName] < amount.AtLeast {
			return false
		}
	}

	return true
}

func condMapToStr(m map[string]AmountCondition) string {
	var sb strings.Builder
	for k, v := range m {
		if sb.Len() > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(strconv.FormatUint(v.Exact, 10))
		sb.WriteString(":")
		sb.WriteString(strconv.FormatUint(v.AtLeast, 10))
	}

	return sb.String()
}

func mapStrUInt64ToStr(m map[string]uint64) string {
	var sb strings.Builder
	for k, v := range m {
		if sb.Len() > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(strconv.FormatUint(v, 10))
	}

	return sb.String()
}
