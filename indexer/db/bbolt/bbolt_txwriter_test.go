package indexerbbolt

import (
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/stretchr/testify/require"
)

func TestTxWriter(t *testing.T) {
	const filePath = "temp_test.db"

	dbCleanup := func() {
		removeDirOrFilePathIfExists(filePath) //nolint:errcheck
	}

	t.Cleanup(dbCleanup)

	t.Run("ExecuteEmpty", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		err = dbTx.Execute()
		require.NoError(t, err)
	})

	t.Run("SetLatestBlockPoint", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		blockPoint := &indexer.BlockPoint{BlockSlot: 1}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		dbTx.SetLatestBlockPoint(blockPoint)
		require.NoError(t, dbTx.Execute())

		bp, err := db.GetLatestBlockPoint()
		require.NoError(t, err)
		require.EqualValues(t, blockPoint, bp)

		dbTx = db.OpenTx()
		dbTx.SetLatestBlockPoint(nil)
		require.NoError(t, dbTx.Execute())

		bp, err = db.GetLatestBlockPoint()
		require.NoError(t, err)
		require.Nil(t, bp)
	})

	t.Run("AddTxOutputs", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		txInOut1 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{1, 2, 3},
				Index: 1,
			},
			Output: indexer.TxOutput{
				Address: "addr_out_1",
				Amount:  1000000,
				IsUsed:  false,
			},
		}
		txInOut2 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{1, 2, 78},
				Index: 2,
			},
			Output: indexer.TxOutput{
				Address: "addr_out_2",
				Amount:  1000000,
				IsUsed:  false,
			},
		}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		dbTx.AddTxOutputs([]*indexer.TxInputOutput{txInOut1, txInOut2})
		err = dbTx.Execute()
		require.NoError(t, err)

		txOutput1, err := db.GetTxOutput(txInOut1.Input)
		require.NoError(t, err)
		require.NotNil(t, txOutput1)
		require.EqualValues(t, txInOut1.Output, txOutput1)

		txOutput2, err := db.GetTxOutput(txInOut2.Input)
		require.NoError(t, err)
		require.NotNil(t, txOutput2)
		require.EqualValues(t, txInOut2.Output, txOutput2)
	})

	t.Run("RemoveTxOutputsSoftDelete", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		softDelete := true

		txInOut1 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{48},
				Index: 1,
			},
			Output: indexer.TxOutput{
				Address: "addr_out_1",
				Amount:  1000000,
				IsUsed:  false,
			},
		}
		txInOut2 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{11},
				Index: 2,
			},
			Output: indexer.TxOutput{
				Address: "addr_out_2",
				Amount:  1000000,
				IsUsed:  false,
			},
		}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		dbTx.AddTxOutputs([]*indexer.TxInputOutput{txInOut1, txInOut2})
		err = dbTx.Execute()
		require.NoError(t, err)

		dbTx = db.OpenTx()
		dbTx.RemoveTxOutputs([]*indexer.TxInput{&txInOut1.Input, &txInOut2.Input}, softDelete)
		err = dbTx.Execute()
		require.NoError(t, err)

		txOutput1, err := db.GetTxOutput(txInOut1.Input)
		require.NoError(t, err)
		require.NotNil(t, txOutput1)

		expectedOutput1 := txInOut1.Output
		expectedOutput1.IsUsed = true

		require.EqualValues(t, expectedOutput1, txOutput1)

		txOutput2, err := db.GetTxOutput(txInOut2.Input)
		require.NoError(t, err)
		require.NotNil(t, txOutput2)

		expectedOutput2 := txInOut2.Output
		expectedOutput2.IsUsed = true

		require.EqualValues(t, expectedOutput2, txOutput2)
	})

	t.Run("RemoveTxOutputsHardDelete", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		softDelete := false

		txInOut1 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{24},
				Index: 1,
			},
			Output: indexer.TxOutput{
				Address: "addr_out_1",
				Amount:  1000000,
				IsUsed:  false,
			},
		}
		txInOut2 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{44},
				Index: 2,
			},
			Output: indexer.TxOutput{
				Address: "addr_out_2",
				Amount:  1000000,
				IsUsed:  false,
			},
		}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		dbTx.AddTxOutputs([]*indexer.TxInputOutput{txInOut1, txInOut2})
		err = dbTx.Execute()
		require.NoError(t, err)

		dbTx = db.OpenTx()
		dbTx.RemoveTxOutputs([]*indexer.TxInput{&txInOut1.Input, &txInOut2.Input}, softDelete)
		err = dbTx.Execute()
		require.NoError(t, err)

		txOutput1, err := db.GetTxOutput(txInOut1.Input)
		require.NoError(t, err)
		require.Empty(t, txOutput1)

		txOutput2, err := db.GetTxOutput(txInOut2.Input)
		require.NoError(t, err)
		require.Empty(t, txOutput2)
	})

	t.Run("AddConfirmedBlock", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		block1 := &indexer.CardanoBlock{Slot: 1}
		block2 := &indexer.CardanoBlock{Slot: 2}
		block3 := &indexer.CardanoBlock{Slot: 3}
		block4 := &indexer.CardanoBlock{Slot: 4}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		dbTx.AddConfirmedBlock(block1)
		dbTx.AddConfirmedBlock(block2)
		err = dbTx.Execute()
		require.NoError(t, err)

		blocks, err := db.GetConfirmedBlocksFrom(0, 10)
		require.NoError(t, err)
		require.NotEmpty(t, blocks)
		require.Len(t, blocks, 2)

		dbTx = db.OpenTx()
		dbTx.AddConfirmedBlock(block3)
		dbTx.AddConfirmedBlock(block4)
		err = dbTx.Execute()
		require.NoError(t, err)

		blocks, err = db.GetConfirmedBlocksFrom(0, 10)
		require.NoError(t, err)
		require.NotEmpty(t, blocks)
		require.Len(t, blocks, 4)

		require.EqualValues(t, []*indexer.CardanoBlock{block1, block2, block3, block4}, blocks)
	})

	t.Run("AddConfirmedTxs", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		tx1 := &indexer.Tx{BlockSlot: 1, Indx: 1}
		tx2 := &indexer.Tx{BlockSlot: 1, Indx: 2}
		tx3 := &indexer.Tx{BlockSlot: 2, Indx: 1}
		tx4 := &indexer.Tx{BlockSlot: 2, Indx: 2}
		tx5 := &indexer.Tx{BlockSlot: 2, Indx: 3}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)

		dbTx.AddConfirmedTxs([]*indexer.Tx{tx1, tx2})
		err = dbTx.Execute()
		require.NoError(t, err)

		txs, err := db.GetUnprocessedConfirmedTxs(10)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
		require.Len(t, txs, 2)

		dbTx = db.OpenTx()
		dbTx.AddConfirmedTxs([]*indexer.Tx{tx3, tx4, tx5})
		err = dbTx.Execute()
		require.NoError(t, err)

		txs, err = db.GetUnprocessedConfirmedTxs(10)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
		require.Len(t, txs, 5)

		require.EqualValues(t, []*indexer.Tx{tx1, tx2, tx3, tx4, tx5}, txs)
	})

	t.Run("DeleteAllTxOutputsPhysically", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		const addr = "addr_1_test"

		txInOuts := []*indexer.TxInputOutput{
			{
				Input: indexer.TxInput{
					Hash: indexer.Hash{1, 2, 3},
				},
				Output: indexer.TxOutput{
					Address: addr,
					Amount:  100,
				},
			},
			{
				Input: indexer.TxInput{
					Hash: indexer.Hash{1, 2, 78},
				},
				Output: indexer.TxOutput{
					Address: addr,
					Amount:  200,
				},
			},
		}

		db := &BBoltDatabase{}

		require.NoError(t, db.Init(filePath))
		require.NoError(t, db.OpenTx().AddTxOutputs(txInOuts).Execute())

		result, err := db.GetAllTxOutputs(addr, true)

		require.NoError(t, err)
		require.Equal(t, txInOuts, result)

		require.NoError(t, db.OpenTx().DeleteAllTxOutputsPhysically().Execute())

		result, err = db.GetAllTxOutputs(addr, true)

		require.NoError(t, err)
		require.Len(t, result, 0)

		require.NoError(t, db.OpenTx().AddTxOutputs(txInOuts).Execute())

		require.NoError(t, db.OpenTx().DeleteAllTxOutputsPhysically().AddTxOutputs(txInOuts[1:]).Execute())

		result, err = db.GetAllTxOutputs(addr, true)

		require.NoError(t, err)
		require.Equal(t, txInOuts[1:], result)
	})
}
