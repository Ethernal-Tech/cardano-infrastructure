package wallet

import (
	"context"
	"fmt"
)

// GetUtxosSum returns sum for tokens in utxos (including lovelace)
func GetUtxosSum(utxos []Utxo) map[string]uint64 {
	result := map[string]uint64{}

	for _, utxo := range utxos {
		result[adaTokenName] += utxo.Amount

		for _, token := range utxo.Tokens {
			result[fmt.Sprintf("%s.%s", token.PolicyID, token.Name)] += token.Amount
		}
	}

	return result
}

// GetOutputsSum returns sum or tokens in outputs (including lovelace)
func GetOutputsSum(outputs []TxOutput) map[string]uint64 {
	result := map[string]uint64{}

	for _, output := range outputs {
		result[adaTokenName] += output.Amount

		for _, token := range output.Tokens {
			result[token.TokenName()] += token.Amount
		}
	}

	return result
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
