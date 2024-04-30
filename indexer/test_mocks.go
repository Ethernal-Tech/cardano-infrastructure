package indexer

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	utxorpc "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
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

	return args.Get(0).(*BlockPoint), args.Error(1) //nolint:forcetypeassert
}

// GetTxOutput implements Database.
func (m *DatabaseMock) GetTxOutput(txInput TxInput) (TxOutput, error) {
	args := m.Called(txInput)

	if m.GetTxOutputFn != nil {
		return m.GetTxOutputFn(txInput)
	}

	return args.Get(0).(TxOutput), args.Error(1) //nolint:forcetypeassert
}

// GetUnprocessedConfirmedTxs implements Database.
func (m *DatabaseMock) GetUnprocessedConfirmedTxs(maxCnt int) ([]*Tx, error) {
	args := m.Called(maxCnt)

	return args.Get(0).([]*Tx), args.Error(1) //nolint:forcetypeassert
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

	return args.Get(0).(DBTransactionWriter) //nolint:forcetypeassert
}

func (m *DatabaseMock) Close() error {
	return m.Called().Error(0)
}

func (m *DatabaseMock) GetLatestConfirmedBlocks(maxCnt int) ([]*CardanoBlock, error) {
	args := m.Called(maxCnt)

	return args.Get(0).([]*CardanoBlock), args.Error(1) //nolint:forcetypeassert
}

func (m *DatabaseMock) GetConfirmedBlocksFrom(slotNumber uint64, maxCnt int) ([]*CardanoBlock, error) {
	args := m.Called(slotNumber, maxCnt)

	return args.Get(0).([]*CardanoBlock), args.Error(1) //nolint:forcetypeassert
}

func (m *DatabaseMock) GetProcessedTx(txHash string) (*Tx, error) {
	args := m.Called(txHash)

	return args.Get(0).(*Tx), args.Error(1) //nolint:forcetypeassert
}

func (m *DatabaseMock) GetUnprocessedTx(txHash string) (*Tx, error) {
	args := m.Called(txHash)

	return args.Get(0).(*Tx), args.Error(1) //nolint:forcetypeassert
}

var _ Database = (*DatabaseMock)(nil)

type DBTransactionWriterMock struct {
	mock.Mock
	AddConfirmedTxsFn     func(txs []*Tx) DBTransactionWriter
	AddTxOutputsFn        func(txOutputs []*TxInputOutput) DBTransactionWriter
	RemoveTxOutputsFn     func(txInputs []*TxInput) DBTransactionWriter
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
func (m *DBTransactionWriterMock) RemoveTxOutputs(txInputs []*TxInput, softDelete bool) DBTransactionWriter {
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

var _ DBTransactionWriter = (*DBTransactionWriterMock)(nil)

type LedgerBlockHeaderMock struct {
	BlockNumberVal uint64
	SlotNumberVal  uint64
	EraVal         ledger.Era
	HashVal        string
}

// BlockBodySize implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) BlockBodySize() uint64 {
	panic("unimplemented") //nolint
}

// BlockNumber implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) BlockNumber() uint64 {
	return m.BlockNumberVal
}

// Cbor implements ledger.BlockHeader.
func (m *LedgerBlockHeaderMock) Cbor() []byte {
	panic("unimplemented") //nolint
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
	panic("unimplemented") //nolint
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
	panic("unimplemented") //nolint
}

// BlockNumber implements ledger.Block.
func (*LedgerBlockMock) BlockNumber() uint64 {
	panic("unimplemented") //nolint
}

// Cbor implements ledger.Block.
func (m *LedgerBlockMock) Cbor() []byte {
	panic("unimplemented") //nolint
}

// Era implements ledger.Block.
func (m *LedgerBlockMock) Era() ledger.Era {
	panic("unimplemented") //nolint
}

// Hash implements ledger.Block.
func (m *LedgerBlockMock) Hash() string {
	panic("unimplemented") //nolint
}

// IssuerVkey implements ledger.Block.
func (m *LedgerBlockMock) IssuerVkey() ledger.IssuerVkey {
	panic("unimplemented") //nolint
}

// SlotNumber implements ledger.Block.
func (m *LedgerBlockMock) SlotNumber() uint64 {
	panic("unimplemented") //nolint
}

// Transactions implements ledger.Block.
func (m *LedgerBlockMock) Transactions() []ledger.Transaction {
	return m.TransactionsVal
}

func (m *LedgerBlockMock) Utxorpc() *utxorpc.Block {
	return nil
}

var _ ledger.Block = (*LedgerBlockMock)(nil)

type LedgerTransactionMock struct {
	FeeVal             uint64
	HashVal            string
	InputsVal          []ledger.TransactionInput
	OutputsVal         []ledger.TransactionOutput
	MetadataVal        *cbor.Value
	TTLVal             uint64
	IsInvalid          bool
	ReferenceInputsVal []ledger.TransactionInput
}

// Cbor implements ledger.Transaction.
func (m *LedgerTransactionMock) Cbor() []byte {
	panic("unimplemented") //nolint
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

func (m *LedgerTransactionMock) Utxorpc() *utxorpc.Tx {
	return nil
}

func (m *LedgerTransactionMock) IsValid() bool {
	return !m.IsInvalid
}

func (m *LedgerTransactionMock) ReferenceInputs() []ledger.TransactionInput {
	return m.ReferenceInputsVal
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
func (m *LedgerTransactionInputMock) Id() ledger.Blake2b256 { //nolint
	return m.HashVal
}

// Index implements ledger.TransactionInput.
func (m *LedgerTransactionInputMock) Index() uint32 {
	return m.IndexVal
}

func (m *LedgerTransactionInputMock) Utxorpc() *utxorpc.TxInput {
	return nil
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
	panic("unimplemented") //nolint
}

// Cbor implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Cbor() []byte {
	panic("unimplemented") //nolint
}

// Datum implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) Datum() *cbor.LazyValue {
	panic("unimplemented") //nolint
}

// DatumHash implements ledger.TransactionOutput.
func (m *LedgerTransactionOutputMock) DatumHash() *ledger.Blake2b256 {
	panic("unimplemented") //nolint
}

func (m *LedgerTransactionOutputMock) Utxorpc() *utxorpc.TxOutput {
	return nil
}

var _ ledger.TransactionOutput = (*LedgerTransactionOutputMock)(nil)
