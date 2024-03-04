package indexer

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	Writter               *DbTransactionWriterMock
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

	return args.Get(0).(*BlockPoint), args.Error(1)
}

// GetTxOutput implements Database.
func (m *DatabaseMock) GetTxOutput(txInput TxInput) (TxOutput, error) {
	args := m.Called(txInput)

	if m.GetTxOutputFn != nil {
		return m.GetTxOutputFn(txInput)
	}

	return args.Get(0).(TxOutput), args.Error(1)
}

// GetUnprocessedConfirmedTxs implements Database.
func (m *DatabaseMock) GetUnprocessedConfirmedTxs(maxCnt int) ([]*Tx, error) {
	args := m.Called(maxCnt)

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
func (m *DatabaseMock) OpenTx() DbTransactionWriter {
	args := m.Called()

	if m.Writter != nil {
		return m.Writter
	}

	return args.Get(0).(DbTransactionWriter)
}

func (m *DatabaseMock) Close() error {
	return m.Called().Error(0)
}

var _ Database = (*DatabaseMock)(nil)

type DbTransactionWriterMock struct {
	mock.Mock
	AddConfirmedTxsFn     func(txs []*Tx) DbTransactionWriter
	AddTxOutputsFn        func(txOutputs []*TxInputOutput) DbTransactionWriter
	RemoveTxOutputsFn     func(txInputs []*TxInput) DbTransactionWriter
	SetLatestBlockPointFn func(point *BlockPoint) DbTransactionWriter
	ExecuteFn             func() error
}

// AddConfirmedTxs implements DbTransactionWriter.
func (m *DbTransactionWriterMock) AddConfirmedTxs(txs []*Tx) DbTransactionWriter {
	m.Called(txs)

	if m.AddConfirmedTxsFn != nil {
		return m.AddConfirmedTxsFn(txs)
	}

	return m
}

// AddTxOutputs implements DbTransactionWriter.
func (m *DbTransactionWriterMock) AddTxOutputs(txOutputs []*TxInputOutput) DbTransactionWriter {
	m.Called(txOutputs)

	if m.AddTxOutputsFn != nil {
		return m.AddTxOutputsFn(txOutputs)
	}

	return m
}

// Execute implements DbTransactionWriter.
func (m *DbTransactionWriterMock) Execute() error {
	if m.ExecuteFn != nil {
		return m.ExecuteFn()
	}

	return m.Called().Error(0)
}

// RemoveTxOutputs implements DbTransactionWriter.
func (m *DbTransactionWriterMock) RemoveTxOutputs(txInputs []*TxInput, softDelete bool) DbTransactionWriter {
	m.Called(txInputs, softDelete)

	if m.RemoveTxOutputsFn != nil {
		return m.RemoveTxOutputsFn(txInputs)
	}

	return m
}

// SetLatestBlockPoint implements DbTransactionWriter.
func (m *DbTransactionWriterMock) SetLatestBlockPoint(point *BlockPoint) DbTransactionWriter {
	m.Called(point)

	if m.SetLatestBlockPointFn != nil {
		return m.SetLatestBlockPointFn(point)
	}

	return m
}

var _ DbTransactionWriter = (*DbTransactionWriterMock)(nil)

type LedgerBlockHeaderMock struct {
	BlockNumberVal uint64
	SlotNumberVal  uint64
	EraVal         ledger.Era
	HashVal        string
}

// BlockBodySize implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) BlockBodySize() uint64 {
	panic("unimplemented")
}

// BlockNumber implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) BlockNumber() uint64 {
	return m.BlockNumberVal
}

// Cbor implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) Cbor() []byte {
	panic("unimplemented")
}

// Era implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) Era() ledger.Era {
	return m.EraVal
}

// Hash implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) Hash() string {
	return m.HashVal
}

// IssuerVkey implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) IssuerVkey() ledger.IssuerVkey {
	panic("unimplemented")
}

// SlotNumber implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) SlotNumber() uint64 {
	return m.SlotNumberVal
}

var _ ledger.BlockHeader = (*LedgerBlockHeaderMock)(nil)

type LedgerBlockMock struct {
	TransactionsVal []ledger.Transaction
}

// BlockBodySize implements ledger.Block.
func (m *LedgerBlockMock) BlockBodySize() uint64 {
	panic("unimplemented")
}

// BlockNumber implements ledger.Block.
func (*LedgerBlockMock) BlockNumber() uint64 {
	panic("unimplemented")
}

// Cbor implements ledger.Block.
func (m *LedgerBlockMock) Cbor() []byte {
	panic("unimplemented")
}

// Era implements ledger.Block.
func (m *LedgerBlockMock) Era() ledger.Era {
	panic("unimplemented")
}

// Hash implements ledger.Block.
func (m *LedgerBlockMock) Hash() string {
	panic("unimplemented")
}

// IssuerVkey implements ledger.Block.
func (m *LedgerBlockMock) IssuerVkey() ledger.IssuerVkey {
	panic("unimplemented")
}

// SlotNumber implements ledger.Block.
func (m *LedgerBlockMock) SlotNumber() uint64 {
	panic("unimplemented")
}

// Transactions implements ledger.Block.
func (m *LedgerBlockMock) Transactions() []ledger.Transaction {
	return m.TransactionsVal
}

var _ ledger.Block = (*LedgerBlockMock)(nil)

type LedgerTransactionMock struct {
	FeeVal      uint64
	HashVal     string
	InputsVal   []ledger.TransactionInput
	OutputsVal  []ledger.TransactionOutput
	MetadataVal *cbor.Value
	TTLVal      uint64
}

// Cbor implements ledger.Transaction.
func (m *LedgerTransactionMock) Cbor() []byte {
	panic("unimplemented")
}

// Fee implements ledger.Transaction.
func (m *LedgerTransactionMock) Fee() uint64 {
	return m.FeeVal
}

// Hash implements ledger.Transaction.
func (m *LedgerTransactionMock) Hash() string {
	return m.HashVal
}

// Inputs implements ledger.Transaction.
func (m *LedgerTransactionMock) Inputs() []ledger.TransactionInput {
	return m.InputsVal
}

// Metadata implements ledger.Transaction.
func (m *LedgerTransactionMock) Metadata() *cbor.Value {
	return m.MetadataVal
}

// Outputs implements ledger.Transaction.
func (m *LedgerTransactionMock) Outputs() []ledger.TransactionOutput {
	return m.OutputsVal
}

// TTL implements ledger.Transaction.
func (m *LedgerTransactionMock) TTL() uint64 {
	return m.TTLVal
}

var _ ledger.Transaction = (*LedgerTransactionMock)(nil)

type LedgerTransactionInputMock struct {
	HashVal  ledger.Blake2b256
	IndexVal uint32
}

func NewLedgerTransactionInputMock(t *testing.T, hash []byte, index uint32) *LedgerTransactionInputMock {
	t.Helper()

	return &LedgerTransactionInputMock{
		HashVal:  ledger.NewBlake2b256(hash),
		IndexVal: index,
	}
}

// Id implements ledger.TransactionInput.
func (m *LedgerTransactionInputMock) Id() ledger.Blake2b256 {
	return m.HashVal
}

// Index implements ledger.TransactionInput.
func (m *LedgerTransactionInputMock) Index() uint32 {
	return m.IndexVal
}

var _ ledger.TransactionInput = (*LedgerTransactionInputMock)(nil)

type LedgerTransactionOutputMock struct {
	AddressVal ledger.Address
	AmountVal  uint64
}

func NewLedgerTransactionOutputMock(t *testing.T, addr string, amount uint64) *LedgerTransactionOutputMock {
	t.Helper()

	a, err := ledger.NewAddress(addr)
	require.NoError(t, err)

	return &LedgerTransactionOutputMock{
		AddressVal: a,
		AmountVal:  amount,
	}
}

// Address implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Address() ledger.Address {
	return m.AddressVal
}

// Amount implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Amount() uint64 {
	return m.AmountVal
}

// Assets implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Assets() *ledger.MultiAsset[uint64] {
	panic("unimplemented")
}

// Cbor implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Cbor() []byte {
	panic("unimplemented")
}

// Datum implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Datum() *cbor.LazyValue {
	panic("unimplemented")
}

// DatumHash implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) DatumHash() *ledger.Blake2b256 {
	panic("unimplemented")
}

var _ ledger.TransactionOutput = (*LedgerTransactionOutputMock)(nil)
