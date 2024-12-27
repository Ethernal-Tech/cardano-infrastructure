package bridgingtx

import (
	"testing"

	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
	"github.com/stretchr/testify/require"
)

func TestGetUTXOsForAmounts(t *testing.T) {
	utxos := []cardanowallet.Utxo{
		{
			Hash:   "1",
			Amount: 100,
		},
		{
			Hash:   "2",
			Amount: 50,
		},
		{
			Hash:   "3",
			Amount: 150,
		},
		{
			Hash:   "4",
			Amount: 200,
		},
		{
			Hash:   "5",
			Amount: 160,
		},
		{
			Hash:   "6",
			Amount: 400,
		},
		{
			Hash:   "7",
			Amount: 200,
		},
	}

	t.Run("exact amount", func(t *testing.T) {
		txInputs, err := GetUTXOsForAmounts(append([]cardanowallet.Utxo{}, utxos...), map[string]AmountCondition{
			cardanowallet.AdaTokenName: {
				Exact:   610,
				AtLeast: 710,
			},
		}, 4)

		require.NoError(t, err)
		require.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 610,
		}, txInputs.Sum)
		require.Equal(t, []cardanowallet.TxInput{
			{
				Hash: "1",
			},
			{
				Hash: "4",
			},
			{
				Hash: "3",
			},
			{
				Hash: "5",
			},
		}, txInputs.Inputs)
	})

	t.Run("at least", func(t *testing.T) {
		txInputs, err := GetUTXOsForAmounts(append([]cardanowallet.Utxo{}, utxos...), map[string]AmountCondition{
			cardanowallet.AdaTokenName: {
				Exact:   660,
				AtLeast: 710,
			},
		}, 3)

		require.NoError(t, err)
		require.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 760,
		}, txInputs.Sum)
		require.Equal(t, []cardanowallet.TxInput{
			{
				Hash: "4",
			},
			{
				Hash: "5",
			},
			{
				Hash: "6",
			},
		}, txInputs.Inputs)
	})
}
