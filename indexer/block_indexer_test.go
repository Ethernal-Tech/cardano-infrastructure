package indexer

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
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

func TestBlockIndexer_processConfirmedBlockNoTxOfInterest(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	addressesOfInterest := []string{addresses[1]}
	blockHash := []byte{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
	}
	config := &BlockIndexerConfig{
		AddressCheck:        AddressCheckAll,
		AddressesOfInterest: addressesOfInterest,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
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

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	fb, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2Hash(blockHash),
		BlockNumberVal: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	assert.Nil(t, fb)
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_processConfirmedBlockTxOfInterestInOutputs(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	hashTx := []string{"00333", "7873282"}
	addressesOfInterest := []string{addresses[1], addresses[3]}
	blockHash := []byte{100, 200, 100}
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
		AddressCheck:        AddressCheckAll,
		AddressesOfInterest: addressesOfInterest,
		SoftDeleteUtxo:      true,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		return nil
	}

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			HashVal: hashTx[0],
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
			HashVal: hashTx[1],
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
		Hash:  txInputs[1].Id().String(),
		Index: txInputs[1].Index(),
	}).Return(TxOutput{}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  txInputs[0].Id().String(),
		Index: txInputs[0].Index(),
	}).Return(TxOutput{Address: "1", Amount: 2}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  txInputs[2].Id().String(),
		Index: txInputs[2].Index(),
	}).Return(TxOutput{Address: "2", Amount: 4}, error(nil)).Once()

	dbMock.Writter.On("Execute").Return(error(nil)).Once()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		{
			Hash:  txInputs[0].Id().String(),
			Index: txInputs[0].Index(),
		},
		{
			Hash:  txInputs[2].Id().String(),
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
			},
		},
	}).Once()
	dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Run(func(args mock.Arguments) {
		txs := args.Get(0).([]*Tx)
		require.Len(t, txs, 2)
		require.Equal(t, txs[0].BlockHash, bytes2Hash(blockHash))
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2Hash(blockHash),
		BlockNumberVal: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	require.Len(t, txs, 2)
	assert.Equal(t, txs[0].Hash, hashTx[0])
	assert.Equal(t, txs[1].Hash, hashTx[1])
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_processConfirmedBlockTxOfInterestInInputs(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	hashTx := [2]string{"eee", "111"}
	addressesOfInterest := []string{addresses[1], addresses[3]}
	dbInputOutputs := [2]*TxInputOutput{
		{
			Input: TxInput{
				Hash:  string("xyzy"),
				Index: uint32(20),
			},
			Output: TxOutput{
				Address: addressesOfInterest[0],
				Amount:  2000,
			},
		},
		{
			Input: TxInput{
				Hash:  string("abcdef"),
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
		NewLedgerTransactionInputMock(t, []byte(dbInputOutputs[0].Input.Hash), dbInputOutputs[0].Input.Index),
		NewLedgerTransactionInputMock(t, []byte("not_exist_2"), uint32(0)),
		NewLedgerTransactionInputMock(t, []byte(dbInputOutputs[1].Input.Hash), dbInputOutputs[1].Input.Index),
	}
	blockHash := []byte{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
	}
	config := &BlockIndexerConfig{
		AddressCheck:        AddressCheckAll,
		AddressesOfInterest: addressesOfInterest,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		return nil
	}

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			HashVal: hashTx[0],
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
			HashVal: hashTx[1],
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
		Hash:  txInputs[0].Id().String(),
		Index: txInputs[0].Index(),
	}).Return(TxOutput{}, error(nil)).Twice()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  txInputs[2].Id().String(),
		Index: txInputs[2].Index(),
	}).Return(TxOutput{}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  txInputs[1].Id().String(),
		Index: txInputs[1].Index(),
	}).Return(TxOutput{
		Address: addressesOfInterest[0],
	}, error(nil)).Twice()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  txInputs[3].Id().String(),
		Index: txInputs[3].Index(),
	}).Return(TxOutput{
		Address: addressesOfInterest[1],
	}, error(nil)).Twice()
	dbMock.Writter.On("SetLatestBlockPoint", expectedLastBlockPoint).Once()
	dbMock.Writter.On("AddTxOutputs", ([]*TxInputOutput)(nil)).Once()
	dbMock.Writter.On("RemoveTxOutputs", []*TxInput{
		{
			Hash:  txInputs[0].Id().String(),
			Index: txInputs[0].Index(),
		},
		{
			Hash:  txInputs[1].Id().String(),
			Index: txInputs[1].Index(),
		},
		{
			Hash:  txInputs[3].Id().String(),
			Index: txInputs[3].Index(),
		},
	}, false).Once()
	dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Run(func(args mock.Arguments) {
		txs := args.Get(0).([]*Tx)
		require.Len(t, txs, 2)
		require.Equal(t, bytes2Hash(blockHash), txs[0].BlockHash)
		require.Equal(t, bytes2Hash(blockHash), txs[1].BlockHash)
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2Hash(blockHash),
		BlockNumberVal: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	require.Len(t, txs, 2)
	assert.Equal(t, txs[0].Hash, hashTx[0])
	assert.Equal(t, txs[1].Hash, hashTx[1])
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_processConfirmedBlockKeepAllTxOutputsInDb(t *testing.T) {
	t.Parallel()

	const (
		blockNumber = uint64(50)
		blockSlot   = uint64(100)
	)

	hashTx := [2]string{"eee", "111"}
	dbInputOutputs := [2]*TxInputOutput{
		{
			Input: TxInput{
				Hash:  string("xyzy"),
				Index: uint32(20),
			},
			Output: TxOutput{
				Address: addresses[1],
				Amount:  2000,
			},
		},
		{
			Input: TxInput{
				Hash:  string("abcdef"),
				Index: uint32(120),
			},
			Output: TxOutput{
				Address: addresses[1],
				Amount:  2,
			},
		},
	}
	txInputs := [2]*LedgerTransactionInputMock{
		NewLedgerTransactionInputMock(t, []byte(dbInputOutputs[0].Input.Hash), dbInputOutputs[0].Input.Index),
		NewLedgerTransactionInputMock(t, []byte(dbInputOutputs[1].Input.Hash), dbInputOutputs[1].Input.Index),
	}
	blockHash := []byte{100, 200, 100}
	expectedLastBlockPoint := &BlockPoint{
		BlockSlot:   blockSlot,
		BlockHash:   blockHash,
		BlockNumber: blockNumber,
	}
	config := &BlockIndexerConfig{
		AddressCheck:         AddressCheckAll,
		KeepAllTxOutputsInDb: true,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		return nil
	}

	allTransactions := []ledger.Transaction{
		&LedgerTransactionMock{
			HashVal: hashTx[0],
			InputsVal: []ledger.TransactionInput{
				txInputs[0],
			},
			OutputsVal: []ledger.TransactionOutput{
				NewLedgerTransactionOutputMock(t, addresses[1], uint64(200)),
			},
		},
		&LedgerTransactionMock{
			HashVal: hashTx[1],
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
		Hash:  txInputs[0].Id().String(),
		Index: txInputs[0].Index(),
	}).Return(TxOutput{
		Address: "addr1",
	}, error(nil)).Once()
	dbMock.On("GetTxOutput", TxInput{
		Hash:  txInputs[1].Id().String(),
		Index: txInputs[1].Index(),
	}).Return(TxOutput{
		Address: "addr2",
	}, error(nil)).Once()
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
		{
			Hash:  txInputs[0].Id().String(),
			Index: txInputs[0].Index(),
		},
		{
			Hash:  txInputs[1].Id().String(),
			Index: txInputs[1].Index(),
		},
	}, false).Once()
	dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Run(func(args mock.Arguments) {
		txs := args.Get(0).([]*Tx)
		require.Len(t, txs, 2)
		require.Equal(t, bytes2Hash(blockHash), txs[0].BlockHash)
		require.Equal(t, bytes2Hash(blockHash), txs[1].BlockHash)
	}).Once()

	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	assert.NotNil(t, blockIndexer)

	txs, latestBlockPoint, err := blockIndexer.processConfirmedBlock(&LedgerBlockHeaderMock{
		SlotNumberVal:  blockSlot,
		HashVal:        bytes2Hash(blockHash),
		BlockNumberVal: blockNumber,
	}, allTransactions)

	require.Nil(t, err)
	require.Len(t, txs, 2)
	assert.Equal(t, txs[0].Hash, hashTx[0])
	assert.Equal(t, txs[1].Hash, hashTx[1])
	assert.Equal(t, expectedLastBlockPoint, latestBlockPoint)
	dbMock.AssertExpectations(t)
	dbMock.Writter.AssertExpectations(t)
}

func TestBlockIndexer_RollBackwardFuncToUnconfirmed(t *testing.T) {
	t.Parallel()

	uncomfBlocks := []blockWithLazyTxRetriever{
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 6, HashVal: bytes2Hash([]byte{0, 2})},
		},
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 7, HashVal: bytes2Hash([]byte{0, 3})},
		},
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 8, HashVal: bytes2Hash([]byte{0, 4})},
		},
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 9, HashVal: bytes2Hash([]byte{0, 5})},
		},
	}
	bp := &BlockPoint{
		BlockSlot:   5,
		BlockHash:   []byte{0, 1},
		BlockNumber: 1,
	}
	config := &BlockIndexerConfig{
		StartingBlockPoint: bp,
		AddressCheck:       AddressCheckAll,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())

	dbMock.On("GetLatestBlockPoint").Return((*BlockPoint)(nil), error(nil)).Once()

	sp, err := blockIndexer.Reset()
	require.NoError(t, err)
	require.Equal(t, *bp, sp)

	blockIndexer.unconfirmedBlocks = uncomfBlocks

	err = blockIndexer.RollBackwardFunc(common.Point{
		Slot: 7,
		Hash: []byte{0, 3},
	}, chainsync.Tip{})
	require.NoError(t, err)

	require.Equal(t, uncomfBlocks[0:2], blockIndexer.unconfirmedBlocks)
	dbMock.AssertExpectations(t)
}

func TestBlockIndexer_RollBackwardFuncToConfirmed(t *testing.T) {
	t.Parallel()

	uncomfBlocks := []blockWithLazyTxRetriever{
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 6, HashVal: bytes2Hash([]byte{0, 2})},
		},
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 7, HashVal: bytes2Hash([]byte{0, 3})},
		},
	}
	bp := &BlockPoint{
		BlockSlot:   5,
		BlockHash:   []byte{0, 1},
		BlockNumber: 1,
	}
	config := &BlockIndexerConfig{
		StartingBlockPoint: &BlockPoint{},
		AddressCheck:       AddressCheckAll,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	blockIndexer.unconfirmedBlocks = uncomfBlocks
	blockIndexer.latestBlockPoint = bp

	err := blockIndexer.RollBackwardFunc(common.Point{
		Slot: bp.BlockSlot,
		Hash: bp.BlockHash,
	}, chainsync.Tip{})
	require.NoError(t, err)

	require.Len(t, blockIndexer.unconfirmedBlocks, 0)
	dbMock.AssertExpectations(t)
}

func TestBlockIndexer_RollBackwardFuncError(t *testing.T) {
	t.Parallel()

	uncomfBlocks := []blockWithLazyTxRetriever{
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 6, HashVal: bytes2Hash([]byte{0, 2})},
		},
		{
			header: &LedgerBlockHeaderMock{SlotNumberVal: 7, HashVal: bytes2Hash([]byte{0, 3})},
		},
	}
	bp := &BlockPoint{
		BlockSlot:   5,
		BlockHash:   []byte{0, 1},
		BlockNumber: 1,
	}
	config := &BlockIndexerConfig{
		StartingBlockPoint: nil,
		AddressCheck:       AddressCheckAll,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())
	blockIndexer.unconfirmedBlocks = uncomfBlocks

	dbMock.On("GetLatestBlockPoint").Return((*BlockPoint)(nil), error(nil)).Once()

	sp, err := blockIndexer.Reset()
	require.NoError(t, err)
	require.Nil(t, sp.BlockHash)

	err = blockIndexer.RollBackwardFunc(common.Point{
		Slot: bp.BlockSlot + 10003,
		Hash: bp.BlockHash,
	}, chainsync.Tip{})
	require.ErrorIs(t, err, errBlockSyncerFatal)

	dbMock.AssertExpectations(t)
}

func TestBlockIndexer_RollForwardFunc(t *testing.T) {
	t.Parallel()

	confirmedTxs := ([]*Tx)(nil)
	addressesOfInterest := []string{addresses[1]}
	getTxs := []GetTxsFunc{
		func() ([]ledger.Transaction, error) {
			return []ledger.Transaction{
				&LedgerTransactionMock{
					OutputsVal: []ledger.TransactionOutput{
						NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(100)),
					},
				},
			}, nil
		},
		func() ([]ledger.Transaction, error) {
			return []ledger.Transaction{
				&LedgerTransactionMock{
					OutputsVal: []ledger.TransactionOutput{
						NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(100)),
					},
				},
				&LedgerTransactionMock{
					OutputsVal: []ledger.TransactionOutput{
						NewLedgerTransactionOutputMock(t, addressesOfInterest[0], uint64(200)),
					},
				},
			}, nil
		},
		func() ([]ledger.Transaction, error) {
			return []ledger.Transaction{}, nil
		},
		func() ([]ledger.Transaction, error) {
			return []ledger.Transaction{}, nil
		},
	}

	blockHeaders := []*LedgerBlockHeaderMock{
		{SlotNumberVal: 1, HashVal: bytes2Hash([]byte{1})},
		{SlotNumberVal: 2, HashVal: bytes2Hash([]byte{2})},
		{SlotNumberVal: 3, HashVal: bytes2Hash([]byte{3})},
		{SlotNumberVal: 4, HashVal: bytes2Hash([]byte{4})},
	}
	config := &BlockIndexerConfig{
		StartingBlockPoint:     nil,
		AddressCheck:           AddressCheckOutputs,
		ConfirmationBlockCount: 2,
	}
	dbMock := &DatabaseMock{
		Writter: &DbTransactionWriterMock{},
	}
	newConfirmedBlockHandler := func(fb []*Tx) error {
		confirmedTxs = fb

		return nil
	}
	blockIndexer := NewBlockIndexer(config, newConfirmedBlockHandler, dbMock, hclog.NewNullLogger())

	for i, h := range blockHeaders {
		if i >= 2 {
			dbMock.On("OpenTx").Once()
			dbMock.Writter.On("Execute").Return(error(nil)).Once()
			dbMock.Writter.On("AddTxOutputs", ([]*TxInputOutput)(nil)).Once()
			dbMock.Writter.On("RemoveTxOutputs", []*TxInput(nil), false).Once()
			dbMock.Writter.On("AddConfirmedTxs", mock.Anything).Once()
			dbMock.Writter.On("SetLatestBlockPoint", &BlockPoint{
				BlockSlot: blockHeaders[i-2].SlotNumberVal,
				BlockHash: hash2Bytes(blockHeaders[i-2].HashVal),
			}).Once()
		}

		err := blockIndexer.RollForwardFunc(h, getTxs[i], chainsync.Tip{})

		require.NoError(t, err)

		if i < 2 {
			require.Len(t, blockIndexer.unconfirmedBlocks, i+1)
		} else {
			require.Len(t, blockIndexer.unconfirmedBlocks, 2)
			require.Len(t, confirmedTxs, i-1)
			require.Equal(t, blockHeaders[i-2].Hash(), confirmedTxs[0].BlockHash)
		}

		dbMock.AssertExpectations(t)
	}
}
