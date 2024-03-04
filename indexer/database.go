package indexer

type DbTransactionWriter interface {
	SetLatestBlockPoint(point *BlockPoint) DbTransactionWriter
	AddTxOutputs(txOutputs []*TxInputOutput) DbTransactionWriter
	AddConfirmedTxs(txs []*Tx) DbTransactionWriter
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

	MarkConfirmedTxsProcessed(txs []*Tx) error
	GetUnprocessedConfirmedTxs(maxCnt int) ([]*Tx, error)
}
