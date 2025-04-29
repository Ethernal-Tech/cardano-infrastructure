package indexer

type ErrorEmitter interface {
	ErrorCh() <-chan error
}

type Closable interface {
	Close() error
}

type Service interface {
	Closable
	ErrorEmitter
	Start()
}

type BlockTxsRetriever interface {
	GetBlockTransactions(blockHeader BlockHeader) ([]*Tx, error)
}

type BlockSyncer interface {
	Closable
	ErrorEmitter
	Sync() error
}

type BlockSyncerHandler interface {
	RollBackward(point BlockPoint) error
	RollForward(blockHeader BlockHeader, txsRetriver BlockTxsRetriever) error
	Reset() (BlockPoint, error)
}

type NewConfirmedBlockHandler func(*CardanoBlock, []*Tx) error

type TxInfoParserFunc func(rawTx []byte, full bool) (TxInfo, error)
