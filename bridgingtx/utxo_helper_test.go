package bridgingtx

import (
	"testing"

	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
	"github.com/stretchr/testify/assert"
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
			Tokens: []cardanowallet.TokenAmount{
				{
					PolicyID: "1",
					Name:     "1",
					Amount:   100,
				},
			},
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
			Tokens: []cardanowallet.TokenAmount{
				{
					PolicyID: "1",
					Name:     "1",
					Amount:   50,
				},
			},
		},
		{
			Hash:   "6",
			Amount: 400,
		},
		{
			Hash:   "7",
			Amount: 200,
			Tokens: []cardanowallet.TokenAmount{
				{
					PolicyID: "1",
					Name:     "1",
					Amount:   400,
				},
			},
		},
		{
			Hash:   "8",
			Amount: 50,
			Tokens: []cardanowallet.TokenAmount{
				{
					PolicyID: "1",
					Name:     "1",
					Amount:   200,
				},
			},
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
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 610,
			"1.31":                     50,
		}, txInputs.Sum)
		assert.Equal(t, []cardanowallet.TxInput{
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
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 760,
			"1.31":                     50,
		}, txInputs.Sum)
		assert.Equal(t, []cardanowallet.TxInput{
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

	t.Run("at least tokens", func(t *testing.T) {
		txInputs, err := GetUTXOsForAmounts(append([]cardanowallet.Utxo{}, utxos...), map[string]AmountCondition{
			cardanowallet.AdaTokenName: {
				Exact:   200,
				AtLeast: 200,
			},
			"1.31": {
				Exact:   596,
				AtLeast: 600,
			},
		}, 2)

		require.NoError(t, err)
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 250,
			"1.31":                     600,
		}, txInputs.Sum)
		assert.Equal(t, []cardanowallet.TxInput{
			{
				Hash: "7",
			},
			{
				Hash: "8",
			},
		}, txInputs.Inputs)
	})

	t.Run("exact tokens", func(t *testing.T) {
		txInputs, err := GetUTXOsForAmounts(append([]cardanowallet.Utxo{}, utxos...), map[string]AmountCondition{
			cardanowallet.AdaTokenName: {
				Exact:   200,
				AtLeast: 200,
			},
			"1.31": {
				Exact:   700,
				AtLeast: 800,
			},
		}, 3)

		require.NoError(t, err)
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 300,
			"1.31":                     700,
		}, txInputs.Sum)
		assert.Equal(t, []cardanowallet.TxInput{
			{
				Hash: "7",
			},
			{
				Hash: "2",
			},
			{
				Hash: "8",
			},
		}, txInputs.Inputs)
	})

	t.Run("not enough tokens", func(t *testing.T) {
		_, err := GetUTXOsForAmounts(append([]cardanowallet.Utxo{}, utxos...), map[string]AmountCondition{
			cardanowallet.AdaTokenName: {
				Exact:   200,
				AtLeast: 200,
			},
			"1.31": {
				Exact:   701,
				AtLeast: 1000,
			},
		}, 3)

		require.ErrorContains(t, err, "not enough funds")
	})
}
