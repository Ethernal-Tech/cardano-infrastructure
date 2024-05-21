package indexer

type DBTransactionWriter interface {
	SetLatestBlockPoint(point *BlockPoint) DBTransactionWriter
	AddTxOutputs(txOutputs []*TxInputOutput) DBTransactionWriter
	AddConfirmedBlock(block *CardanoBlock) DBTransactionWriter
	AddConfirmedTxs(txs []*Tx) DBTransactionWriter
	RemoveTxOutputs(txInputs []*TxInput, softDelete bool) DBTransactionWriter
	Execute() error
}

type TxOutputRetriever interface {
	GetTxOutput(txInput TxInput) (TxOutput, error)
}

type BlockIndexerDB interface {
	TxOutputRetriever
	GetLatestBlockPoint() (*BlockPoint, error)
	OpenTx() DBTransactionWriter
}

type Database interface {
	BlockIndexerDB
	Init(filepath string) error
	Close() error

	MarkConfirmedTxsProcessed(txs []*Tx) error
	GetUnprocessedConfirmedTxs(maxCnt int) ([]*Tx, error)
	GetLatestConfirmedBlocks(maxCnt int) ([]*CardanoBlock, error)
	GetConfirmedBlocksFrom(slotNumber uint64, maxCnt int) ([]*CardanoBlock, error)
	GetAllTxOutputs(address string, onlyNotUsed bool) ([]*TxInputOutput, error)
}
