package indexer

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var addresses = []string{
	"addr1vxrmu3m2cc5k6xltupj86a2uzcuq8r4nhznrhfq0pkwl4hgqj2v8w",
	"addr1v9kganeshgdqyhwnyn9stxxgl7r4y2ejfyqjn88n7ncapvs4sugsd",
	"addr1v8hrxaz0yqkfdsszfvjmdnqh0tv4xl2xgd7dfrxzj86cqzghu5c6p",
	"addr1qxh7y2ezyt7hcraew7q0s8fg36usm049ktf4m9rly220snm0tf3rte5f4wequeg86kww58hp34qpwxdpl76tfuwmk77qjstmmj",
}

func TestBlockIndexer_ProcessConfirmedBlock_NoTxOfInterest(t *testing.T) {
	t.Parallel()

	blockHeader := BlockHeader{
		Slot:   200,
		Hash:   Hash{100, 200, 100},
		Number: 11,
		EraID:  5,
	}
	addressesOfInterest := []string{addresses[1]}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot: blockHeader.Slot,
		BlockHash: blockHeader.Hash,
	}
	config := &BlockIndexerConfig{
		AddressCheck:            AddressCheckAll,
		AddressesOfInterest:     addressesOfInterest,
		KeepAllTxsHashesInBlock: true,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}

	allTransactions := []*Tx{
		{
			Inputs: []*TxInputOutput{
				{Input: TxInput{Hash: Hash{1, 2}, Index: 0}},
			},
			Outputs: []*TxOutput{
				{Address: addresses[2], Amount: 100},
			},
		},
		{
			Inputs: []*TxInputOutput{
				{Input: TxInput{Hash: Hash{1, 2}, Index: 1}},
				{Input: TxInput{Hash: Hash{1, 2, 3}, Index: 1}},
			},
			Outputs: []*TxOutput{
				{Address: addresses[0], Amount: 100},
			},
		},
	}

	dbMock.On("OpenTx").Once()
	dbMock.On("GetTxOutput", mock.Anything).Return(TxOutput{}, error(nil)).Times(3)
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("AddConfirmedTxs", []*Tx(nil)).Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("RemoveTxOutputs", ([]*TxInput)(nil), false).Once()
	dbMock.Writter.On("AddTxOutputs", ([]*TxInputOutput)(nil)).Once()
	dbMock.Writter.On("AddConfirmedBlock", blockHeader.ToCardanoBlock(getTxHashes(allTransactions))).Once()

	blockIndexer := NewBlockIndexer(config, nil, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cardanoBlock, relevantTxs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(blockHeader, allTransactions)

	require.Nil(t, err)
	assert.Len(t, relevantTxs, 0)
	assert.Len(t, cardanoBlock.Txs, 2)
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_ProcessConfirmedBlock_TxOfInterestInOutputs(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	hashTx := []Hash{
		{1, 2, 3, 4, 5, 6, 7, 17},
		{5, 2, 9, 4, 8, 6, 7, 27},
	}
	addressesOfInterest := []string{addresses[1], addresses[3]}
	blockHash := Hash{100, 200, 100}
	txInputs := []*TxInputOutput{
		{Input: TxInput{Hash: Hash{1}, Index: 0}},
		{Input: TxInput{Hash: Hash{1, 2}, Index: 1}},
		{Input: TxInput{Hash: Hash{1, 2, 3}, Index: 2}},
	}
	txOutputs := []*TxOutput{
		{Address: addressesOfInterest[0], Amount: 100},
		{Address: addressesOfInterest[1], Amount: 200},
		{Address: addresses[0], Amount: 100}, // not address of interest
		{Address: addresses[0], Amount: 100}, // not address of interest
	}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot: blockSlot,
		BlockHash: blockHash,
	}
	config := &BlockIndexerConfig{
		AddressCheck:            AddressCheckAll,
		AddressesOfInterest:     addressesOfInterest,
		SoftDeleteUtxo:          true,
		KeepAllTxsHashesInBlock: true,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}

	allTransactions := []*Tx{
		{
			Hash:    hashTx[0],
			Inputs:  []*TxInputOutput{txInputs[0]},
			Outputs: []*TxOutput{txOutputs[0]},
		},
		{
			Inputs:  []*TxInputOutput{txInputs[1]},
			Outputs: []*TxOutput{txOutputs[3]},
		},
		{
			Hash:    hashTx[1],
			Inputs:  []*TxInputOutput{txInputs[2]},
			Outputs: []*TxOutput{txOutputs[2], txOutputs[1]},
		},
	}

	dbMock.On("OpenTx").Once()

	dbMock.On("GetTxOutput", txInputs[1].Input).Return(TxOutput{}, error(nil)).Once()
	dbMock.On("GetTxOutput", txInputs[0].Input).Return(TxOutput{Address: "1", Amount: 2}, error(nil)).Once()
	dbMock.On("GetTxOutput", txInputs[2].Input).Return(TxOutput{Address: "2", Amount: 4}, error(nil)).Once()

	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs:    getTxHashes(allTransactions),
	}).Once()
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput(nil), true).Once()
	dbMock.Writter.On("AddTxOutputs", []*TxInputOutput{
		{
			Input: TxInput{
				Hash:  hashTx[0],
				Index: 0,
			},
			Output: *txOutputs[0],
		},
		{
			Input: TxInput{
				Hash:  hashTx[1],
				Index: 1,
			},
			Output: *txOutputs[1],
		},
	}).Once()
	dbMock.Writter.On("AddConfirmedTxs", []*Tx{allTransactions[0], allTransactions[2]}).Once()

	blockIndexer := NewBlockIndexer(config, nil, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(BlockHeader{
		Slot:   blockSlot,
		Hash:   blockHash,
		Number: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	require.Len(t, txs, 2)
	assert.Len(t, cb.Txs, 3)
	assert.Equal(t, txs[0].Hash, hashTx[0])
	assert.Equal(t, txs[1].Hash, hashTx[1])
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_ProcessConfirmedBlock_TxOfInterestInInputs(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	hashTx := [2]Hash{
		{1, 19},
		{29, 4},
	}
	addressesOfInterest := []string{addresses[1], addresses[3]}
	dbInputOutputs := [2]*TxInputOutput{
		{
			Input: TxInput{
				Hash:  Hash{20, 21},
				Index: uint32(20),
			},
			Output: TxOutput{
				Address: addressesOfInterest[0],
				Amount:  2000,
			},
		},
		{
			Input: TxInput{
				Hash:  Hash{30, 31},
				Index: uint32(120),
			},
			Output: TxOutput{
				Address: addressesOfInterest[1],
				Amount:  2,
			},
		},
	}
	txInputs := []TxInput{
		{Hash: NewHashFromHexString("FFAAFFAAFFAAFFAAFFAA"), Index: dbInputOutputs[0].Input.Index},
		{Hash: dbInputOutputs[0].Input.Hash, Index: dbInputOutputs[0].Input.Index},
		{Hash: dbInputOutputs[1].Input.Hash, Index: dbInputOutputs[1].Input.Index + 1203489},
		{Hash: dbInputOutputs[1].Input.Hash, Index: dbInputOutputs[1].Input.Index},
	}
	blockHash := Hash{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot: blockSlot,
		BlockHash: blockHash,
	}
	config := &BlockIndexerConfig{
		AddressCheck:            AddressCheckAll,
		AddressesOfInterest:     addressesOfInterest,
		KeepAllTxsHashesInBlock: false,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}

	allTransactions := []*Tx{
		{
			Hash: hashTx[0],
			Inputs: []*TxInputOutput{
				{Input: txInputs[0]},
				{Input: txInputs[1]},
			},
			Outputs: []*TxOutput{
				{Address: addresses[0], Amount: 200},
			},
		},
		{
			Inputs: []*TxInputOutput{
				{Input: txInputs[2]},
			},
			Outputs: []*TxOutput{
				{Address: addresses[0], Amount: 100},
			},
		},
		{
			Hash: hashTx[1],
			Inputs: []*TxInputOutput{
				{Input: txInputs[3]},
			},
			Outputs: []*TxOutput{
				{Address: addresses[0], Amount: 200},
			},
		},
	}

	dbMock.On("OpenTx").Once()
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.On("GetTxOutput", txInputs[0]).Return(TxOutput{
		Address: "dummy",
	}, error(nil)).Once()
	dbMock.On("GetTxOutput", txInputs[2]).Return(TxOutput{}, error(nil)).Once()
	dbMock.On("GetTxOutput", txInputs[1]).Return(TxOutput{
		Address: addressesOfInterest[0],
	}, error(nil)).Once()
	dbMock.On("GetTxOutput", txInputs[3]).Return(TxOutput{
		Address: addressesOfInterest[1],
	}, error(nil)).Once()
	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs:    getTxHashes([]*Tx{allTransactions[0], allTransactions[2]}),
	}).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("AddTxOutputs", ([]*TxInputOutput)(nil)).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		&txInputs[1], &txInputs[3],
	}, false).Once()
	dbMock.Writter.On("AddConfirmedTxs", []*Tx{
		allTransactions[0], allTransactions[2],
	}).Once()

	blockIndexer := NewBlockIndexer(config, nil, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(BlockHeader{
		Slot:   blockSlot,
		Hash:   blockHash,
		Number: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	require.Len(t, txs, 2)
	assert.Len(t, cb.Txs, 2)
	assert.Equal(t, txs[0].Hash, hashTx[0])
	assert.Equal(t, txs[1].Hash, hashTx[1])
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_ProcessConfirmedBlock_KeepAllTxOutputsInDb(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	hashTx := [2]Hash{
		{89, 19},
		{66, 4},
	}
	dbInputOutputs := []*TxInputOutput{
		{
			Input: TxInput{
				Hash:  Hash{1, 1},
				Index: uint32(20),
			},
			Output: TxOutput{
				Address: addresses[1],
				Amount:  2000,
			},
		},
		{
			Input: TxInput{
				Hash:  Hash{2, 2},
				Index: uint32(120),
			},
			Output: TxOutput{
				Address: addresses[1],
				Amount:  2,
			},
		},
	}
	txInputs := []*TxInputOutput{
		{Input: dbInputOutputs[0].Input}, {Input: dbInputOutputs[1].Input},
	}
	blockHash := Hash{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot: blockSlot,
		BlockHash: blockHash,
	}
	config := &BlockIndexerConfig{
		AddressCheck:         AddressCheckAll,
		KeepAllTxOutputsInDB: true,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}

	allTransactions := []*Tx{
		{
			Hash:   hashTx[0],
			Inputs: []*TxInputOutput{txInputs[0]},
			Outputs: []*TxOutput{
				{Address: addresses[1], Amount: 200},
			},
		},
		{
			Hash:   hashTx[1],
			Inputs: []*TxInputOutput{txInputs[1]},
			Outputs: []*TxOutput{
				{Address: addresses[1], Amount: 100},
			},
		},
	}

	for i, txInput := range txInputs {
		dbMock.On("GetTxOutput", txInput.Input).Return(TxOutput{Address: fmt.Sprintf("addr%d", i)}, error(nil)).Once()
	}

	dbMock.On("OpenTx").Once()
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("AddTxOutputs", []*TxInputOutput{
		{
			Input:  TxInput{Hash: hashTx[0], Index: 0},
			Output: TxOutput{Address: addresses[1], Amount: uint64(200)},
		},
		{
			Input:  TxInput{Hash: hashTx[1], Index: 0},
			Output: TxOutput{Address: addresses[1], Amount: uint64(100)},
		},
	}).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		&txInputs[0].Input, &txInputs[1].Input,
	}, false).Once()
	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs:    getTxHashes(allTransactions),
	}).Once()
	dbMock.Writter.On("AddConfirmedTxs", allTransactions).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(BlockHeader{
		Slot:   blockSlot,
		Hash:   blockHash,
		Number: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	assert.Len(t, cb.Txs, 2)
	require.Len(t, txs, 2)
	assert.Equal(t, txs[0].Hash, hashTx[0])
	assert.Equal(t, txs[1].Hash, hashTx[1])
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

/*
func TestBlockIndexer_RollBackwardFunc_RollbackToUnconfirmed(t *testing.T) {
	t.Parallel()

	uncomfBlocks := []ledger.BlockHeader{
		&LedgerBlockHeaderMock{SlotNumberVal: 6, HashVal: bytes2HashString([]byte{0, 2})},
		&LedgerBlockHeaderMock{SlotNumberVal: 7, HashVal: bytes2HashString([]byte{0, 3})},
		&LedgerBlockHeaderMock{SlotNumberVal: 8, HashVal: bytes2HashString([]byte{0, 4})},
		&LedgerBlockHeaderMock{SlotNumberVal: 9, HashVal: bytes2HashString([]byte{0, 5})},
	}
	bp := &BlockPoint{
		BlockSlot:   5,
		BlockHash:   Hash{0, 1},
		BlockNumber: 1,
	}
	config := &BlockIndexerConfig{
		ConfirmationBlockCount: 5,
		StartingBlockPoint:     bp,
		AddressCheck:           AddressCheckAll,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())

	dbMock.On("GetLatestBlockPoint").Return((*BlockPoint)(nil), error(nil)).Once()

	sp, err := blockIndexer.Reset()
	require.NoError(t, err)
	require.Equal(t, *bp, sp)

	for _, x := range uncomfBlocks {
		require.NoError(t, blockIndexer.unconfirmedBlocks.Push(x))
	}

	err = blockIndexer.RollBackwardFunc(common.Point{
		Slot: 7,
		Hash: []byte{0, 3},
	})
	require.NoError(t, err)

	require.Equal(t, uncomfBlocks[0:2], blockIndexer.unconfirmedBlocks.ToList())
	dbMock.AssertExpectations(t)
}

func TestBlockIndexer_RollBackwardFunc_RollbackToConfirmed(t *testing.T) {
	t.Parallel()

	uncomfBlocks := []ledger.BlockHeader{
		&LedgerBlockHeaderMock{SlotNumberVal: 6, HashVal: bytes2HashString([]byte{0, 2})},
		&LedgerBlockHeaderMock{SlotNumberVal: 7, HashVal: bytes2HashString([]byte{0, 3})},
	}
	bp := &BlockPoint{
		BlockSlot:   5,
		BlockHash:   Hash{0, 1},
		BlockNumber: 1,
	}
	config := &BlockIndexerConfig{
		ConfirmationBlockCount: 5,
		StartingBlockPoint:     &BlockPoint{},
		AddressCheck:           AddressCheckAll,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	blockIndexer.latestBlockPoint = bp

	for _, x := range uncomfBlocks {
		require.NoError(t, blockIndexer.unconfirmedBlocks.Push(x))
	}

	err := blockIndexer.RollBackwardFunc(common.Point{
		Slot: bp.BlockSlot,
		Hash: bp.BlockHash[:],
	})
	require.NoError(t, err)

	require.Equal(t, 0, blockIndexer.unconfirmedBlocks.Len())
	dbMock.AssertExpectations(t)
}

func TestBlockIndexer_RollBackwardFunc_Error(t *testing.T) {
	t.Parallel()

	uncomfBlocks := []ledger.BlockHeader{
		&LedgerBlockHeaderMock{SlotNumberVal: 6, HashVal: bytes2HashString([]byte{0, 2})},
		&LedgerBlockHeaderMock{SlotNumberVal: 7, HashVal: bytes2HashString([]byte{0, 3})},
	}
	bp := &BlockPoint{
		BlockSlot: 5,
		BlockHash: Hash{0, 1},
	}
	config := &BlockIndexerConfig{
		ConfirmationBlockCount: 5,
		StartingBlockPoint:     nil,
		AddressCheck:           AddressCheckAll,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())

	for _, x := range uncomfBlocks {
		require.NoError(t, blockIndexer.unconfirmedBlocks.Push(x))
	}

	dbMock.On("GetLatestBlockPoint").Return((*BlockPoint)(nil), error(nil)).Once()

	sp, err := blockIndexer.Reset()
	require.NoError(t, err)
	require.Equal(t, Hash{}, sp.BlockHash) // all zeroes

	err = blockIndexer.RollBackwardFunc(common.Point{
		Slot: bp.BlockSlot + 10003,
		Hash: bp.BlockHash[:],
	})
	require.ErrorIs(t, err, ErrBlockSyncerFatal)

	dbMock.AssertExpectations(t)
}

func TestBlockIndexer_RollForwardFunc(t *testing.T) {
	t.Parallel()

	inputTxHash := NewHashFromHexString("01FFaa")
	inputTxIndex := uint32(43)
	confirmedTxs := ([]*Tx)(nil)
	addressesOfInterest := []string{addresses[1]}
	getTxsMock := &BlockTxsRetrieverMock{
		RetrieveFn: func(blockHeader ledger.BlockHeader) ([]ledger.Transaction, error) {
			switch blockHeader.SlotNumber() {
			case 1:
				return []ledger.Transaction{
					&LedgerTransactionMock{
						HashVal: "01",
						OutputsVal: []ledger.TransactionOutput{
							NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(50)),
						},
					},
				}, nil
			case 2:
				return []ledger.Transaction{
					&LedgerTransactionMock{
						HashVal: "02",
						InputsVal: []ledger.TransactionInput{
							NewLedgerTransactionInputMock(t, inputTxHash[:], inputTxIndex),
						},
						OutputsVal: []ledger.TransactionOutput{
							NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(100)),
						},
					},
					&LedgerTransactionMock{
						HashVal: "03",
						OutputsVal: []ledger.TransactionOutput{
							NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(200)),
						},
					},
				}, nil
			default:
				return []ledger.Transaction{}, nil
			}
		},
	}
	blockHeaders := []*LedgerBlockHeaderMock{
		{SlotNumberVal: 1, HashVal: bytes2HashString([]byte{1})},
		{SlotNumberVal: 2, HashVal: bytes2HashString([]byte{2})},
		{SlotNumberVal: 3, HashVal: bytes2HashString([]byte{3})},
		{SlotNumberVal: 4, HashVal: bytes2HashString([]byte{4})},
	}
	config := &BlockIndexerConfig{
		StartingBlockPoint:     nil,
		AddressCheck:           AddressCheckOutputs,
		ConfirmationBlockCount: 2,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		confirmedTxs = fb

		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())

	for i, h := range blockHeaders {
		if i >= 2 {
			txsRetrievedWithGetTxs, err := getTxsMock.GetBlockTransactions(blockHeaders[i-2])
			require.NoError(t, err)

			dbMock.On("OpenTx").Once()
			dbMock.Writter.On("Execute").Return(error(nil)).Once()

			// first block... then second block
			if i == 2 {
				dbMock.Writter.On("AddTxOutputs", []*TxInputOutput{
					{
						Input: TxInput{
							Hash: NewHashFromHexString("0x01"),
						},
						Output: TxOutput{
							Slot:    1,
							Address: addressesOfInterest[0],
							Amount:  50,
						},
					},
				}).Once()
				dbMock.Writter.On("RemoveTxOutputs", []*TxInput(nil), false).Once()
			} else {
				dbMock.Writter.On("AddTxOutputs", []*TxInputOutput{
					{
						Input: TxInput{
							Hash: NewHashFromHexString("0x02"),
						},
						Output: TxOutput{
							Slot:    2,
							Address: addressesOfInterest[0],
							Amount:  100,
						},
					},
					{
						Input: TxInput{
							Hash: NewHashFromHexString("0x03"),
						},
						Output: TxOutput{
							Slot:    2,
							Address: addressesOfInterest[0],
							Amount:  200,
						},
					},
				}).Once()
				dbMock.On("GetTxOutput", TxInput{
					Hash: inputTxHash, Index: inputTxIndex,
				}).Return(TxOutput{}, error(nil)).Once()
				dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
					{
						Hash:  inputTxHash,
						Index: inputTxIndex,
					},
				}, false).Once()
			}

			dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Once()
			dbMock.Writter.On("AddConfirmedBlock", blockHeaders[i-2].ToCardanoBlock(getTxHashes(txsRetrievedWithGetTxs))).Once()
			dbMock.Writter.On("SetLatestBlockPoint", &BlockPoint{
				BlockSlot: blockHeaders[i-2].SlotNumberVal,
				BlockHash: NewHashFromHexString(blockHeaders[i-2].HashVal),
			}).Once()
		}

		require.NoError(t, blockIndexer.RollForwardFunc(h, getTxsMock))

		if i < 2 {
			require.Equal(t, i+1, blockIndexer.unconfirmedBlocks.Len())
		} else {
			require.Equal(t, 2, blockIndexer.unconfirmedBlocks.Len())
			require.Len(t, confirmedTxs, i-1)
			require.Equal(t, blockHeaders[i-2].Hash(), bytes2HashString(confirmedTxs[0].BlockHash[:]))
		}

		dbMock.AssertExpectations(t)
	}
}
*/
