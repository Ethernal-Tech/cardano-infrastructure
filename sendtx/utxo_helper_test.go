package sendtx

import (
	"fmt"
	"testing"

	"slices"

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
			Amount: 20,
			Tokens: []cardanowallet.TokenAmount{
				cardanowallet.NewTokenAmount(cardanowallet.NewToken("1", "1"), 100),
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
				cardanowallet.NewTokenAmount(cardanowallet.NewToken("1", "1"), 50),
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
				cardanowallet.NewTokenAmount(cardanowallet.NewToken("1", "1"), 400),
			},
		},
		{
			Hash:   "8",
			Amount: 50,
			Tokens: []cardanowallet.TokenAmount{
				cardanowallet.NewTokenAmount(cardanowallet.NewToken("1", "1"), 200),
			},
		},
	}

	checkAllAreThere := func(t *testing.T, utxos []cardanowallet.Utxo) {
		t.Helper()

		mp := map[string]bool{}

		for _, utxo := range utxos {
			mp[fmt.Sprintf("%s_%d", utxo.Hash, utxo.Index)] = true
		}

		require.Equal(t, len(utxos), len(mp))
	}

	t.Run("exact amount", func(t *testing.T) {
		utxosNew := slices.Clone(utxos)

		txInputs, err := GetUTXOsForAmounts(utxosNew, map[string]uint64{
			cardanowallet.AdaTokenName: 610,
		}, 50, 4, 1)

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

		checkAllAreThere(t, utxosNew)
	})

	t.Run("greater amount", func(t *testing.T) {
		utxosNew := slices.Clone(utxos)

		txInputs, err := GetUTXOsForAmounts(utxosNew, map[string]uint64{
			cardanowallet.AdaTokenName: 1240,
			"1.31":                     150,
		}, 15, 7, 1)

		require.NoError(t, err)
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 1260,
			"1.31":                     650,
		}, txInputs.Sum)
		assert.Equal(t, []cardanowallet.TxInput{
			{
				Hash: "1",
			},
			{
				Hash: "7",
			},
			{
				Hash: "3",
			},
			{
				Hash: "4",
			},
			{
				Hash: "5",
			},
			{
				Hash: "6",
			},
			{
				Hash: "8",
			},
		}, txInputs.Inputs)

		checkAllAreThere(t, utxosNew)
	})

	t.Run("greater tokens", func(t *testing.T) {
		utxosNew := slices.Clone(utxos)

		txInputs, err := GetUTXOsForAmounts(utxosNew, map[string]uint64{
			cardanowallet.AdaTokenName: 200,
			"1.31":                     410,
		}, 50, 2, 1)

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

		checkAllAreThere(t, utxosNew)
	})

	t.Run("exact tokens", func(t *testing.T) {
		utxosNew := append([]cardanowallet.Utxo{}, utxos...)

		txInputs, err := GetUTXOsForAmounts(utxosNew, map[string]uint64{
			cardanowallet.AdaTokenName: 200,
			"1.31":                     700,
		}, 10, 3, 1)

		require.NoError(t, err)
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 270,
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

		checkAllAreThere(t, utxosNew)
	})

	t.Run("try to consolidate utxos", func(t *testing.T) {
		utxosNew := slices.Clone(utxos)

		_, err := GetUTXOsForAmounts(utxosNew, map[string]uint64{
			cardanowallet.AdaTokenName: 200,
			"1.31":                     500,
		}, 50, 1, 1)

		require.ErrorIs(t, err, cardanowallet.ErrUTXOsLimitReached)

		checkAllAreThere(t, utxosNew)
	})

	t.Run("not enough tokens", func(t *testing.T) {
		utxosNew := slices.Clone(utxos)

		_, err := GetUTXOsForAmounts(utxosNew, map[string]uint64{
			cardanowallet.AdaTokenName: 300,
			"1.31":                     1000,
		}, 50, 3, 1)

		require.ErrorIs(t, err, cardanowallet.ErrUTXOsCouldNotSelect)

		checkAllAreThere(t, utxosNew)
	})

	t.Run("with tryAtLeastInputs", func(t *testing.T) {
		utxos := []cardanowallet.Utxo{
			{
				Hash:   "1",
				Amount: 50,
			},
			{
				Hash:   "2",
				Amount: 1000,
			},
			{
				Hash:   "3",
				Amount: 150,
				Tokens: []cardanowallet.TokenAmount{
					cardanowallet.NewTokenAmount(cardanowallet.NewToken("1", "1"), 100),
				},
			},
			{
				Hash:   "4",
				Amount: 200,
			},
		}

		txInputs, err := GetUTXOsForAmounts(utxos, map[string]uint64{
			cardanowallet.AdaTokenName: 1000,
		}, 50, 5, 4)

		require.NoError(t, err)
		assert.Equal(t, map[string]uint64{
			cardanowallet.AdaTokenName: 1400,
			"1.31":                     100,
		}, txInputs.Sum)
		assert.Equal(t, []cardanowallet.TxInput{
			{
				Hash: "1",
			},
			{
				Hash: "2",
			},
			{
				Hash: "3",
			},
			{
				Hash: "4",
			},
		}, txInputs.Inputs)

		checkAllAreThere(t, utxos)
	})
}

func TestDoesSumSatisfyCondition(t *testing.T) {
	t.Run("equal currency", func(t *testing.T) {
		require.True(t, doesSumSatisfyCondition(map[string]uint64{
			cardanowallet.AdaTokenName: 100,
		}, map[string]uint64{
			cardanowallet.AdaTokenName: 100,
		}, 50))
	})

	t.Run("currency with change", func(t *testing.T) {
		require.True(t, doesSumSatisfyCondition(map[string]uint64{
			cardanowallet.AdaTokenName: 150,
		}, map[string]uint64{
			cardanowallet.AdaTokenName: 100,
		}, 50))
	})

	t.Run("currency not enough change", func(t *testing.T) {
		require.False(t, doesSumSatisfyCondition(map[string]uint64{
			cardanowallet.AdaTokenName: 149,
		}, map[string]uint64{
			cardanowallet.AdaTokenName: 100,
		}, 50))
	})

	t.Run("not enough tokens", func(t *testing.T) {
		require.False(t, doesSumSatisfyCondition(map[string]uint64{
			cardanowallet.AdaTokenName: 150,
			"1.31":                     200,
		}, map[string]uint64{
			cardanowallet.AdaTokenName: 100,
			"1.31":                     250,
		}, 50))
	})

	t.Run("enough tokens and currency", func(t *testing.T) {
		require.True(t, doesSumSatisfyCondition(map[string]uint64{
			cardanowallet.AdaTokenName: 155,
			"1.31":                     250,
		}, map[string]uint64{
			cardanowallet.AdaTokenName: 100,
			"1.31":                     250,
		}, 50))
	})
}
