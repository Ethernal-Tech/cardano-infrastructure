package indexer

import (
	"encoding/binary"
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
	inp2 := TxInput{}

	require.NoError(t, inp2.Set(inp.Key()))

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

func TestTokenAmount_StringFuncs(t *testing.T) {
	token := &TokenAmount{PolicyID: "policyId", Name: "tokenName", Amount: 100}

	t.Run("TokenName", func(t *testing.T) {
		require.Equal(t, "policyId.746f6b656e4e616d65", token.TokenName())
	})

	t.Run("String", func(t *testing.T) {
		require.Equal(t, "100 policyId.746f6b656e4e616d65", token.String())
	})
}
