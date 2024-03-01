package indexer

type DbTransactionWriter interface {
	SetLatestBlockPoint(point *BlockPoint) DbTransactionWriter
	AddTxOutputs(txOutputs []*TxInputOutput) DbTransactionWriter
	AddConfirmedBlock(block *FullBlock) DbTransactionWriter
	RemoveTxOutputs(txInputs []*TxInput, softDelete bool) DbTransactionWriter
	Execute() error
}

type TxOutputRetriever interface {
	GetTxOutput(txInput TxInput) (TxOutput, error)
}

type BlockIndexerDb interface {
	TxOutputRetriever
	GetLatestBlockPoint() (*BlockPoint, error)
	OpenTx() DbTransactionWriter
}

type Database interface {
	BlockIndexerDb
	Init(filepath string) error
	Close() error

	MarkConfirmedBlocksProcessed(blocks []*FullBlock) error
	GetUnprocessedConfirmedBlocks(maxCnt int) ([]*FullBlock, error)
}
