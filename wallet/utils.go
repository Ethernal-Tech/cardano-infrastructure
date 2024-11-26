package wallet

import (
	"context"
	"fmt"
)

// GetUtxosSum returns sum of all utxos
func GetUtxosSum(utxos []Utxo) (sum uint64) {
	for _, utxo := range utxos {
		sum += utxo.Amount
	}

	return sum
}

// GetUtxosTokensSum returns sum for tokens in utxos (including lovelace)
func GetUtxosTokensSum(utxos []Utxo) map[string]uint64 {
	result := map[string]uint64{}

	for _, utxo := range utxos {
		result[adaTokenName] += utxo.Amount

		for _, token := range utxo.Tokens {
			result[fmt.Sprintf("%s.%s", token.PolicyID, token.Name)] += token.Amount
		}
	}

	return result
}

// GetOutputsSum returns sum of tx outputs
func GetOutputsSum(outputs []TxOutput) (receiversSum uint64) {
	for _, x := range outputs {
		receiversSum += x.Amount
	}

	return receiversSum
}

// IsTxInUtxos checks whether a specified transaction hash (txHash)
// exists within the UTXOs associated with the given address (addr).
func IsTxInUtxos(ctx context.Context, utxoRetriever IUTxORetriever, addr string, txHash string) (bool, error) {
	utxos, err := utxoRetriever.GetUtxos(ctx, addr)
	if err != nil {
		return false, err
	}

	for _, x := range utxos {
		if x.Hash == txHash {
			return true, nil
		}
	}

	return false, nil
}
