package indexer

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxInput_Key(t *testing.T) {
	t.Parallel()

	const (
		index = uint32(17878329)
		hash  = "FF00FFCCFF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00AAFF00AA"
	)

	inp := TxInput{
		Hash:  NewHashFromHexString(hash),
		Index: index,
	}

	inp2, err := NewTxInputFromBytes(inp.Key())
	require.NoError(t, err)

	require.Equal(t, inp, inp2)
	require.Equal(t, strings.ToLower(hash)[:64], inp2.Hash.String())
}

func TestTx_Key(t *testing.T) {
	t.Parallel()

	const (
		blockSlot = uint64(78_023_893_190_777_456)
		index     = uint32(2_889_111_003)
	)

	inp := Tx{
		BlockSlot: blockSlot,
		Indx:      index,
	}

	bytes := inp.Key()

	require.Len(t, bytes, 12)

	bs := binary.BigEndian.Uint64(bytes[:8])
	idx := binary.BigEndian.Uint32(bytes[8:])

	require.Equal(t, blockSlot, bs)
	require.Equal(t, idx, idx)
}

func TestCardanoBlock_Key(t *testing.T) {
	t.Parallel()

	const (
		blockSlot = uint64(942_623_893_190_777_456)
	)

	inp := CardanoBlock{
		Slot: blockSlot,
	}

	bytes := inp.Key()

	require.Len(t, bytes, 8)

	bs := binary.BigEndian.Uint64(bytes[:8])
	require.Equal(t, blockSlot, bs)
}

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

func TestTokenAmount_StringFuncs(t *testing.T) {
	token := &TokenAmount{PolicyID: "policyId", Name: "tokenName", Amount: 100}

	t.Run("TokenName", func(t *testing.T) {
		require.Equal(t, "policyId.746f6b656e4e616d65", token.TokenName())
	})

	t.Run("String", func(t *testing.T) {
		require.Equal(t, "100 policyId.746f6b656e4e616d65", token.String())
	})
}
