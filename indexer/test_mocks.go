package indexer

import (
	"errors"

	"github.com/stretchr/testify/mock"
)

type BlockSyncerMock struct {
	mock.Mock
	CloseFn func() error
	SyncFn  func() error
}

// Close implements BlockSyncer.
func (m *BlockSyncerMock) Close() error {
	args := m.Called()

	if m.CloseFn != nil {
		return m.CloseFn()
	}

	return args.Error(0)
}

// Sync implements BlockSyncer.
func (m *BlockSyncerMock) Sync() error {
	args := m.Called()

	if m.SyncFn != nil {
		return m.SyncFn()
	}

	return args.Error(0)
}

// ErrorCh implements BlockSyncer.
func (m *BlockSyncerMock) ErrorCh() <-chan error {
	return make(<-chan error)
}

var _ BlockSyncer = (*BlockSyncerMock)(nil)

type DatabaseMock struct {
	mock.Mock
	Writter               *DBTransactionWriterMock
	GetLatestBlockPointFn func() (*BlockPoint, error)
	GetTxOutputFn         func(txInput TxInput) (TxOutput, error)
	InitFn                func(filepath string) error
}

// GetLatestBlockPoint implements Database.
func (m *DatabaseMock) GetLatestBlockPoint() (*BlockPoint, error) {
	args := m.Called()

	if m.GetLatestBlockPointFn != nil {
		return m.GetLatestBlockPointFn()
	}

	//nolint:forcetypeassert
	return args.Get(0).(*BlockPoint), args.Error(1)
}

// GetTxOutput implements Database.
func (m *DatabaseMock) GetTxOutput(txInput TxInput) (TxOutput, error) {
	args := m.Called(txInput)

	if m.GetTxOutputFn != nil {
		return m.GetTxOutputFn(txInput)
	}

	//nolint:forcetypeassert
	return args.Get(0).(TxOutput), args.Error(1)
}

// GetUnprocessedConfirmedTxs implements Database.
func (m *DatabaseMock) GetUnprocessedConfirmedTxs(maxCnt int) ([]*Tx, error) {
	args := m.Called(maxCnt)

	//nolint:forcetypeassert
	return args.Get(0).([]*Tx), args.Error(1)
}

// Init implements Database.
func (m *DatabaseMock) Init(filepath string) error {
	args := m.Called(filepath)

	if m.InitFn != nil {
		return m.InitFn(filepath)
	}

	return args.Error(0)
}

// MarkConfirmedTxsProcessed implements Database.
func (m *DatabaseMock) MarkConfirmedTxsProcessed(txs []*Tx) error {
	return m.Called(txs).Error(0)
}

// OpenTx implements Database.
func (m *DatabaseMock) OpenTx() DBTransactionWriter {
	args := m.Called()

	if m.Writter != nil {
		return m.Writter
	}

	//nolint:forcetypeassert
	return args.Get(0).(DBTransactionWriter)
}

func (m *DatabaseMock) Close() error {
	return m.Called().Error(0)
}

func (m *DatabaseMock) GetLatestConfirmedBlocks(maxCnt int) ([]*CardanoBlock, error) {
	args := m.Called(maxCnt)

	//nolint:forcetypeassert
	return args.Get(0).([]*CardanoBlock), args.Error(1)
}

func (m *DatabaseMock) GetConfirmedBlocksFrom(slotNumber uint64, maxCnt int) ([]*CardanoBlock, error) {
	args := m.Called(slotNumber, maxCnt)

	//nolint:forcetypeassert
	return args.Get(0).([]*CardanoBlock), args.Error(1)
}

func (m *DatabaseMock) GetAllTxOutputs(address string, onlyNotUser bool) ([]*TxInputOutput, error) {
	args := m.Called(address, onlyNotUser)

	//nolint:forcetypeassert
	return args.Get(0).([]*TxInputOutput), args.Error(1)
}

var _ Database = (*DatabaseMock)(nil)

type DBTransactionWriterMock struct {
	mock.Mock
	AddConfirmedTxsFn     func(txs []*Tx) DBTransactionWriter
	AddTxOutputsFn        func(txOutputs []*TxInputOutput) DBTransactionWriter
	RemoveTxOutputsFn     func(txInputs []TxInput) DBTransactionWriter
	SetLatestBlockPointFn func(point *BlockPoint) DBTransactionWriter
	ExecuteFn             func() error
}

// AddConfirmedTxs implements DbTransactionWriter.
func (m *DBTransactionWriterMock) AddConfirmedTxs(txs []*Tx) DBTransactionWriter {
	m.Called(txs)

	if m.AddConfirmedTxsFn != nil {
		return m.AddConfirmedTxsFn(txs)
	}

	return m
}

func (m *DBTransactionWriterMock) AddConfirmedBlock(block *CardanoBlock) DBTransactionWriter {
	m.Called(block)

	return m
}

// AddTxOutputs implements DbTransactionWriter.
func (m *DBTransactionWriterMock) AddTxOutputs(txOutputs []*TxInputOutput) DBTransactionWriter {
	m.Called(txOutputs)

	if m.AddTxOutputsFn != nil {
		return m.AddTxOutputsFn(txOutputs)
	}

	return m
}

// Execute implements DbTransactionWriter.
func (m *DBTransactionWriterMock) Execute() error {
	if m.ExecuteFn != nil {
		return m.ExecuteFn()
	}

	return m.Called().Error(0)
}

// RemoveTxOutputs implements DbTransactionWriter.
func (m *DBTransactionWriterMock) RemoveTxOutputs(txInputs []TxInput, softDelete bool) DBTransactionWriter {
	m.Called(txInputs, softDelete)

	if m.RemoveTxOutputsFn != nil {
		return m.RemoveTxOutputsFn(txInputs)
	}

	return m
}

// SetLatestBlockPoint implements DbTransactionWriter.
func (m *DBTransactionWriterMock) SetLatestBlockPoint(point *BlockPoint) DBTransactionWriter {
	m.Called(point)

	if m.SetLatestBlockPointFn != nil {
		return m.SetLatestBlockPointFn(point)
	}

	return m
}

func (m *DBTransactionWriterMock) DeleteAllTxOutputsPhysically() DBTransactionWriter {
	m.Called()

	return m
}

var _ DBTransactionWriter = (*DBTransactionWriterMock)(nil)

type BlockTxsRetrieverMock struct {
	RetrieveFn func(blockHeader BlockHeader) ([]*Tx, error)
}

func (bt *BlockTxsRetrieverMock) GetBlockTransactions(blockHeader BlockHeader) ([]*Tx, error) {
	return bt.RetrieveFn(blockHeader)
}

var _ BlockSyncerHandler = (*BlockSyncerHandlerMock)(nil)

type BlockSyncerHandlerMock struct {
	defBlockPoint      *BlockPoint
	ResetFn            func() (BlockPoint, error)
	RollForwardFn      func(BlockHeader, BlockTxsRetriever) error
	RollBackwardFuncFn func(BlockPoint) error
}

func NewBlockSyncerHandlerMock(slot uint64, hash string) *BlockSyncerHandlerMock {
	return &BlockSyncerHandlerMock{
		defBlockPoint: &BlockPoint{
			BlockSlot: slot,
			BlockHash: NewHashFromHexString(hash),
		},
	}
}

func (hMock *BlockSyncerHandlerMock) RollBackward(point BlockPoint) error {
	if hMock.RollBackwardFuncFn != nil {
		return hMock.RollBackwardFuncFn(point)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) RollForward(
	blockHeader BlockHeader, txsRetriever BlockTxsRetriever,
) error {
	if hMock.RollForwardFn != nil {
		return hMock.RollForwardFn(blockHeader, txsRetriever)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) Reset() (BlockPoint, error) {
	if hMock.ResetFn != nil {
		return hMock.ResetFn()
	}

	if hMock.defBlockPoint == nil {
		return BlockPoint{}, errors.New("error sync block point")
	}

	return *hMock.defBlockPoint, nil
}
