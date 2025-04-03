package indexerbbolt

import (
	"os"
	"testing"

	indexer "github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/stretchr/testify/require"
)

func TestDatabase(t *testing.T) {
	const filePath = "temp_test.db"

	dbCleanup := func() {
		removeDirOrFilePathIfExists(filePath) //nolint:errcheck
	}

	t.Cleanup(dbCleanup)

	t.Run("InitDatabase", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)
		require.NotNil(t, db)
		require.NotNil(t, db.db)
	})

	t.Run("CloseDatabase", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)
		require.NotNil(t, db)
		require.NotNil(t, db.db)

		err = db.Close()
		require.NoError(t, err)
	})

	t.Run("OpenTx", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		require.NotNil(t, dbTx)
		err = dbTx.Execute()
		require.NoError(t, err)
	})

	t.Run("GetLatestBlockPointNil", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		blockPoint, err := db.GetLatestBlockPoint()
		require.NoError(t, err)
		require.Nil(t, blockPoint)
	})

	t.Run("GetLatestBlockPoint", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		blockPoint1 := &indexer.BlockPoint{
			BlockSlot:   1,
			BlockNumber: 1,
			BlockHash:   indexer.Hash{1},
		}
		blockPoint2 := &indexer.BlockPoint{
			BlockSlot:   2,
			BlockNumber: 2,
			BlockHash:   indexer.Hash{12},
		}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		dbTx.SetLatestBlockPoint(blockPoint1)
		dbTx.SetLatestBlockPoint(blockPoint2)
		err = dbTx.Execute()
		require.NoError(t, err)

		blockPoint, err := db.GetLatestBlockPoint()
		require.NoError(t, err)
		require.NotNil(t, blockPoint)
		require.NotEqualValues(t, blockPoint1, blockPoint)
		require.EqualValues(t, blockPoint2, blockPoint)
	})

	t.Run("GetTxOutputNil", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		txInput := indexer.TxInput{
			Hash:  indexer.Hash{1, 2},
			Index: 1,
		}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		txOutput, err := db.GetTxOutput(txInput)
		require.NoError(t, err)
		require.Empty(t, txOutput)
	})

	t.Run("GetTxOutput", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		txInOut1 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{1, 2},
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
				Hash:  indexer.Hash{14, 43, 78},
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

	t.Run("GetLatestConfirmedBlocksEmpty", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		blocks, err := db.GetLatestConfirmedBlocks(10)
		require.NoError(t, err)
		require.Empty(t, blocks)
	})

	t.Run("GetLatestConfirmedBlocks", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		block1 := &indexer.CardanoBlock{Slot: 1}
		block2 := &indexer.CardanoBlock{Slot: 2}
		block3 := &indexer.CardanoBlock{Slot: 3}
		block4 := &indexer.CardanoBlock{Slot: 4}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		dbTx.AddConfirmedBlock(block1)
		dbTx.AddConfirmedBlock(block2)
		dbTx.AddConfirmedBlock(block3)
		dbTx.AddConfirmedBlock(block4)
		require.NoError(t, dbTx.Execute())

		blocks, err := db.GetLatestConfirmedBlocks(3)
		require.NoError(t, err)
		require.NotEmpty(t, blocks)
		require.Len(t, blocks, 3)

		require.EqualValues(t, block4, blocks[0])
		require.EqualValues(t, block3, blocks[1])
		require.EqualValues(t, block2, blocks[2])
	})

	t.Run("GetConfirmedBlocksFromEmpty", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		blocks, err := db.GetConfirmedBlocksFrom(0, 10)
		require.NoError(t, err)
		require.Empty(t, blocks)
		require.Len(t, blocks, 0)
	})

	t.Run("GetConfirmedBlocksFrom", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		block1 := &indexer.CardanoBlock{Slot: 1}
		block2 := &indexer.CardanoBlock{Slot: 2}
		block3 := &indexer.CardanoBlock{Slot: 3}
		block4 := &indexer.CardanoBlock{Slot: 4}

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		dbTx := db.OpenTx()
		dbTx.AddConfirmedBlock(block1)
		dbTx.AddConfirmedBlock(block2)
		dbTx.AddConfirmedBlock(block3)
		dbTx.AddConfirmedBlock(block4)
		require.NoError(t, dbTx.Execute())

		blocks, err := db.GetConfirmedBlocksFrom(2, 10)
		require.NoError(t, err)
		require.NotEmpty(t, blocks)
		require.Len(t, blocks, 3)

		require.EqualValues(t, block2, blocks[0])
		require.EqualValues(t, block3, blocks[1])
		require.EqualValues(t, block4, blocks[2])
	})

	t.Run("MarkConfirmedTxsProcessed", func(t *testing.T) {
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
		dbTx.AddConfirmedTxs([]*indexer.Tx{tx1, tx2, tx3, tx4, tx5})
		require.NoError(t, dbTx.Execute())

		err = db.MarkConfirmedTxsProcessed([]*indexer.Tx{tx1, tx4, tx5})
		require.NoError(t, err)

		txs, err := db.GetUnprocessedConfirmedTxs(10)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
		require.Len(t, txs, 2)

		require.EqualValues(t, tx2, txs[0])
		require.EqualValues(t, tx3, txs[1])
	})

	t.Run("GetUnprocessedConfirmedTxsEmpty", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		db := &BBoltDatabase{}
		err := db.Init(filePath)
		require.NoError(t, err)

		txs, err := db.GetUnprocessedConfirmedTxs(10)
		require.NoError(t, err)
		require.Empty(t, txs)
	})

	t.Run("GetUnprocessedConfirmedTxs", func(t *testing.T) {
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
		dbTx.AddConfirmedTxs([]*indexer.Tx{tx1, tx2, tx3, tx4, tx5})
		require.NoError(t, dbTx.Execute())

		txs, err := db.GetUnprocessedConfirmedTxs(3)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
		require.Len(t, txs, 3)

		require.EqualValues(t, tx1, txs[0])
		require.EqualValues(t, tx2, txs[1])
		require.EqualValues(t, tx3, txs[2])

		err = db.MarkConfirmedTxsProcessed([]*indexer.Tx{tx1, tx2})
		require.NoError(t, err)

		txs, err = db.GetUnprocessedConfirmedTxs(10)
		require.NoError(t, err)
		require.NotEmpty(t, txs)
		require.Len(t, txs, 3)

		require.EqualValues(t, tx3, txs[0])
		require.EqualValues(t, tx4, txs[1])
		require.EqualValues(t, tx5, txs[2])
	})

	t.Run("GetAllTxOutputs", func(t *testing.T) {
		t.Cleanup(dbCleanup)

		const addr = "0x123445454"

		good1 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{11, 2},
				Index: 2,
			},
			Output: indexer.TxOutput{
				Slot:    100,
				Address: addr,
				Amount:  100,
			},
		}
		good2 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{89, 2},
				Index: 0,
			},
			Output: indexer.TxOutput{
				Slot:    200,
				Address: addr,
				Amount:  150,
			},
		}
		good3 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{111, 2},
				Index: 1,
			},
			Output: indexer.TxOutput{
				Slot:    200,
				Address: addr,
				Amount:  200,
			},
		}
		good4 := &indexer.TxInputOutput{
			Input: indexer.TxInput{
				Hash:  indexer.Hash{111, 2},
				Index: 14,
			},
			Output: indexer.TxOutput{
				Slot:    200,
				Address: addr,
				Amount:  200,
			},
		}

		txOutputs := []*indexer.TxInputOutput{
			good4,
			good3,
			{
				Input: indexer.TxInput{
					Hash:  indexer.Hash{9},
					Index: 2,
				},
				Output: indexer.TxOutput{
					Slot:    50,
					Address: "0x00",
					Amount:  100,
				},
			},
			good2,
			{
				Input: indexer.TxInput{
					Hash:  indexer.Hash{19},
					Index: 4,
				},
				Output: indexer.TxOutput{
					Slot:    100,
					Address: addr,
					Amount:  300,
					IsUsed:  true,
				},
			},
			good1,
		}
		db := &BBoltDatabase{}

		require.NoError(t, db.Init(filePath))
		require.NoError(t, db.OpenTx().AddTxOutputs(txOutputs).Execute())

		result, err := db.GetAllTxOutputs(addr, true)

		require.NoError(t, err)
		require.Equal(t, []*indexer.TxInputOutput{good1, good2, good3, good4}, result)
	})
}

func removeDirOrFilePathIfExists(dirOrFilePath string) (err error) {
	if _, err = os.Stat(dirOrFilePath); err == nil {
		os.RemoveAll(dirOrFilePath)
	}

	return err
}
