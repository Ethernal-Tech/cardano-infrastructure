package indexer

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortTxInputOutputs(t *testing.T) {
	inputs := []*TxInputOutput{
		{
			Input: TxInput{
				Hash:  Hash{1, 1},
				Index: 0,
			},
			Output: TxOutput{
				Slot: 500,
			},
		},
		{
			Input: TxInput{
				Hash:  Hash{1, 2},
				Index: 5,
			},
			Output: TxOutput{
				Slot: 200,
			},
		},
		{
			Input: TxInput{
				Hash:  Hash{89, 2},
				Index: 1,
			},
			Output: TxOutput{
				Slot: 200,
			},
		},
		{
			Input: TxInput{
				Hash:  Hash{1, 2},
				Index: 3,
			},
			Output: TxOutput{
				Slot: 200,
			},
		},
	}
	sorted := SortTxInputOutputs(slices.Clone(inputs))

	require.Equal(t, []*TxInputOutput{
		inputs[3], inputs[1], inputs[2], inputs[0],
	}, sorted)
}

func TestGetTxOutputsAndInputsFuncs(t *testing.T) {
	const address = "address1"

	txs := []*Tx{
		{
			Hash: Hash{1, 1},
			Outputs: []*TxOutput{
				{Address: address, Slot: 100},
				{Address: "a", Slot: 200},
			},
			Inputs: []*TxInputOutput{
				{
					Input: TxInput{Hash: Hash{2}, Index: 1},
				},
			},
		},
		{
			Hash: Hash{1, 1},
			Outputs: []*TxOutput{
				{Address: "b", Slot: 100},
			},
			Inputs: []*TxInputOutput{
				{
					Input:  TxInput{Hash: Hash{8}, Index: 0},
					Output: TxOutput{Address: address, Slot: 100},
				},
				{
					Input: TxInput{Hash: Hash{9}, Index: 1},
				},
			},
		},
		{
			Hash: Hash{1, 1},
			Outputs: []*TxOutput{
				{Address: address, Slot: 100},
			},
			Inputs: []*TxInputOutput{
				{
					Input:  TxInput{Hash: Hash{10}, Index: 0},
					Output: TxOutput{Address: "0", Slot: 100},
				},
				{
					Input:  TxInput{Hash: Hash{12}, Index: 100},
					Output: TxOutput{Address: address, Slot: 100},
				},
			},
		},
	}
	addressesOfInterest := map[string]bool{address: true}

	t.Run("getTxOutputs retrieve all", func(t *testing.T) {
		require.Len(t, getTxOutputs(txs, nil), 4)
	})

	t.Run("getTxOutputs retrieve filtered", func(t *testing.T) {
		require.Len(t, getTxOutputs(txs, addressesOfInterest), 2)
	})

	t.Run("getTxInputs retrieve all", func(t *testing.T) {
		require.Len(t, getTxInputs(txs, nil), 5)
	})

	t.Run("getTxInputs retrieve filtered", func(t *testing.T) {
		require.Len(t, getTxInputs(txs, addressesOfInterest), 2)
	})
}
