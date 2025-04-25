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
