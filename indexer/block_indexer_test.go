package indexer

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/common"
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

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	addressesOfInterest := []string{addresses[1]}
	blockHash := Hash{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
	}
	config := &BlockIndexerConfig{
		AddressCheck:            AddressCheckAll,
		AddressesOfInterest:     addressesOfInterest,
		KeepAllTxsHashesInBlock: true,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			InputsVal: []ledger.TransactionInput{
				NewLedgerTransactionInputMock(t, []byte{1, 2}, uint32(0)),
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[2], uint64(100)),
			},
		},
		&LedgerTransactionMock{
			InputsVal: []ledger.TransactionInput{
				NewLedgerTransactionInputMock(t, []byte{1, 2}, uint32(1)),
				NewLedgerTransactionInputMock(t, []byte{1, 2, 3}, uint32(1)),
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[0], uint64(100)),
			},
		},
	}

	dbMock.On("OpenTx").Once()
	dbMock.On("GetTxOutput", mock.Anything).Return(TxOutput{}, error(nil)).Times(3)
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("RemoveTxOutputs", ([]*TxInput)(nil), false).Once()
	dbMock.Writter.On("AddTxOutputs", ([]*TxInputOutput)(nil)).Once()
	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs:    getTxHashes(allTransactions),
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, fb, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2HashString(blockHash[:]),
		BlockNumberVal: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	assert.Nil(t, fb)
	assert.Len(t, cb.Txs, 2)
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
	txInputs := [3]ledger.TransactionInput{
		NewLedgerTransactionInputMock(t, []byte{1}, uint32(0)),
		NewLedgerTransactionInputMock(t, []byte{1, 2}, uint32(1)),
		NewLedgerTransactionInputMock(t, []byte{1, 2, 3}, uint32(2)),
	}
	txOutputs := [4]ledger.TransactionOutput{
		NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(100)),
		NewLedgerTransactionOutputMock(t, addressesOfInterest[1], uint64(200)),
		NewLedgerTransactionOutputMock(t, addresses[0], uint64(100)), // not address of interest
		NewLedgerTransactionOutputMock(t, addresses[0], uint64(100)), // not address of interest
	}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
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
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			HashVal: bytes2HashString(hashTx[0][:]),
			InputsVal: []ledger.TransactionInput{
				txInputs[0],
			},
			OutputsVal: []ledger.TransactionOutput{
				txOutputs[0],
			},
		},
		&LedgerTransactionMock{
			InputsVal: []ledger.TransactionInput{
				txInputs[1],
			},
			OutputsVal: []ledger.TransactionOutput{
				txOutputs[3],
			},
		},
		&LedgerTransactionMock{
			HashVal: bytes2HashString(hashTx[1][:]),
			InputsVal: []ledger.TransactionInput{
				txInputs[2],
			},
			OutputsVal: []ledger.TransactionOutput{
				txOutputs[2],
				txOutputs[1],
			},
		},
	}

	dbMock.On("OpenTx").Once()

	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[1].Id()),
		Index: txInputs[1].Index(),
	}).Return(TxOutput{}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[0].Id()),
		Index: txInputs[0].Index(),
	}).Return(TxOutput{Address: "1", Amount: 2}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[2].Id()),
		Index: txInputs[2].Index(),
	}).Return(TxOutput{Address: "2", Amount: 4}, error(nil)).Once()

	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs:    getTxHashes(allTransactions),
	}).Once()
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		{
			Hash:  Hash(txInputs[0].Id()),
			Index: txInputs[0].Index(),
		},
		{
			Hash:  Hash(txInputs[2].Id()),
			Index: txInputs[2].Index(),
		},
	}, true).Once()
	dbMock.Writter.On("AddTxOutputs", []*TxInputOutput{
		{
			Input: TxInput{
				Hash:  hashTx[0],
				Index: 0,
			},
			Output: TxOutput{
				Address: txOutputs[0].Address().String(),
				Amount:  txOutputs[0].Amount(),
				Slot:    blockSlot,
			},
		},
		{
			Input: TxInput{
				Hash:  hashTx[1],
				Index: 1,
			},
			Output: TxOutput{
				Address: txOutputs[1].Address().String(),
				Amount:  txOutputs[1].Amount(),
				Slot:    blockSlot,
			},
		},
	}).Once()
	dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Run(func(args mock.Arguments) {
		txs := args.Get(0).([]*Tx)
		require.Len(t, txs, 2)
		require.Equal(t, txs[0].BlockHash, blockHash)
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2HashString(blockHash[:]),
		BlockNumberVal: blockNumber,
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
	txInputs := [4]*LedgerTransactionInputMock{
		NewLedgerTransactionInputMock(t, []byte("not_exist_1"), uint32(0)),
		NewLedgerTransactionInputMock(t, dbInputOutputs[0].Input.Hash[:], dbInputOutputs[0].Input.Index),
		NewLedgerTransactionInputMock(t, []byte("not_exist_2"), uint32(0)),
		NewLedgerTransactionInputMock(t, dbInputOutputs[1].Input.Hash[:], dbInputOutputs[1].Input.Index),
	}
	blockHash := Hash{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
	}
	config := &BlockIndexerConfig{
		AddressCheck:            AddressCheckAll,
		AddressesOfInterest:     addressesOfInterest,
		KeepAllTxsHashesInBlock: false,
	}
	dbMock := &DatabaseMock{
		Writter: &DBTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(cb *CardanoBlock, fb []*Tx) error {
		return nil
	}

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			HashVal: bytes2HashString(hashTx[0][:]),
			InputsVal: []ledger.TransactionInput{
				txInputs[0], txInputs[1],
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[0], uint64(200)),
			},
		},
		&LedgerTransactionMock{
			InputsVal: []ledger.TransactionInput{
				txInputs[2],
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[0], uint64(100)),
			},
		},
		&LedgerTransactionMock{
			HashVal: bytes2HashString(hashTx[1][:]),
			InputsVal: []ledger.TransactionInput{
				txInputs[3],
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[0], uint64(200)),
			},
		},
	}

	dbMock.On("OpenTx").Once()
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[0].Id()),
		Index: txInputs[0].Index(),
	}).Return(TxOutput{}, error(nil)).Twice()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[2].Id()),
		Index: txInputs[2].Index(),
	}).Return(TxOutput{}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[1].Id()),
		Index: txInputs[1].Index(),
	}).Return(TxOutput{
		Address: addressesOfInterest[0],
	}, error(nil)).Twice()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[3].Id()),
		Index: txInputs[3].Index(),
	}).Return(TxOutput{
		Address: addressesOfInterest[1],
	}, error(nil)).Twice()
	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs: getTxHashes([]ledger.Transaction{
			allTransactions[0], allTransactions[2],
		}),
	}).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("AddTxOutputs", ([]*TxInputOutput)(nil)).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		{
			Hash:  Hash(txInputs[0].Id()),
			Index: txInputs[0].Index(),
		},
		{
			Hash:  Hash(txInputs[1].Id()),
			Index: txInputs[1].Index(),
		},
		{
			Hash:  Hash(txInputs[3].Id()),
			Index: txInputs[3].Index(),
		},
	}, false).Once()
	dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Run(func(args mock.Arguments) {
		txs := args.Get(0).([]*Tx)
		require.Len(t, txs, 2)
		require.Equal(t, blockHash, txs[0].BlockHash)
		require.Equal(t, blockHash, txs[1].BlockHash)
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2HashString(blockHash[:]),
		BlockNumberVal: blockNumber,
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
	dbInputOutputs := [2]*TxInputOutput{
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
	txInputs := [2]*LedgerTransactionInputMock{
		NewLedgerTransactionInputMock(t, dbInputOutputs[0].Input.Hash[:], dbInputOutputs[0].Input.Index),
		NewLedgerTransactionInputMock(t, dbInputOutputs[1].Input.Hash[:], dbInputOutputs[1].Input.Index),
	}
	blockHash := Hash{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
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

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			HashVal: bytes2HashString(hashTx[0][:]),
			InputsVal: []ledger.TransactionInput{
				txInputs[0],
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[1], uint64(200)),
			},
		},
		&LedgerTransactionMock{
			HashVal: bytes2HashString(hashTx[1][:]),
			InputsVal: []ledger.TransactionInput{
				txInputs[1],
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[1], uint64(100)),
			},
		},
	}

	dbMock.On("OpenTx").Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[0].Id()),
		Index: txInputs[0].Index(),
	}).Return(TxOutput{
		Address: "addr1",
	}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  Hash(txInputs[1].Id()),
		Index: txInputs[1].Index(),
	}).Return(TxOutput{
		Address: "addr2",
	}, error(nil)).Once()
	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("AddTxOutputs", []*TxInputOutput{
		{
			Input:  TxInput{Hash: hashTx[0], Index: 0},
			Output: TxOutput{Address: addresses[1], Amount: uint64(200), Slot: blockSlot},
		},
		{
			Input:  TxInput{Hash: hashTx[1], Index: 0},
			Output: TxOutput{Address: addresses[1], Amount: uint64(100), Slot: blockSlot},
		},
	}).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		{
			Hash:  Hash(txInputs[0].Id()),
			Index: txInputs[0].Index(),
		},
		{
			Hash:  Hash(txInputs[1].Id()),
			Index: txInputs[1].Index(),
		},
	}, false).Once()
	dbMock.Writter.On("AddConfirmedBlock", &CardanoBlock{
		Slot:   blockSlot,
		Number: blockNumber,
		Hash:   blockHash,
		Txs:    getTxHashes(allTransactions),
	}).Once()
	dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Run(func(args mock.Arguments) {
		txs := args.Get(0).([]*Tx)
		require.Len(t, txs, 2)
		require.Equal(t, blockHash, txs[0].BlockHash)
		require.Equal(t, blockHash, txs[1].BlockHash)
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	cb, txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2HashString(blockHash[:]),
		BlockNumberVal: blockNumber,
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
			dbMock.Writter.On("AddConfirmedBlock", NewCardanoBlock(blockHeaders[i-2], getTxHashes(txsRetrievedWithGetTxs))).Once()
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
