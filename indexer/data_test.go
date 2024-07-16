package indexer

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxInputKey(t *testing.T) {
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

func TestTxKey(t *testing.T) {
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

func TestCardanoBlockKey(t *testing.T) {
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
