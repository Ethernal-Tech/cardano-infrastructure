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

// IntersectNotFoundHandler is optionally implemented by a BlockSyncerHandler. The syncer calls it only when
// the node explicitly rejects the point returned by Reset (the node does not know that block). Other failed
// sync starts (dial, handshake, TLS, dropped connection) are not reported
type IntersectNotFoundHandler interface {
	IntersectNotFound(point BlockPoint)
}

type NewConfirmedBlockHandler func(*CardanoBlock, []*Tx) error

type TxInfoParserFunc func(rawTx []byte, full bool) (TxInfo, error)
