package indexer

import (
	"sync/atomic"

	"github.com/hashicorp/go-hclog"
)

type blockIndexerRunnerQueueItem struct {
	BlockHeader  *BlockHeader
	TxsRetriever BlockTxsRetriever
	Point        *BlockPoint
}

type BlockIndexerRunnerConfig struct {
	AutoStart        bool `json:"autoStart"`
	QueueChannelSize int  `json:"queueChannelSize"`
}

type BlockIndexerRunner struct {
	blockSyncerHandler BlockSyncerHandler
	isClosed           uint32
	closeCh            chan struct{}
	queueCh            chan blockIndexerRunnerQueueItem
	logger             hclog.Logger
}

var _ BlockSyncerHandler = (*BlockIndexerRunner)(nil)

func NewBlockIndexerRunner(
	blockSyncerHandler BlockSyncerHandler, config *BlockIndexerRunnerConfig, logger hclog.Logger,
) *BlockIndexerRunner {
	runner := &BlockIndexerRunner{
		blockSyncerHandler: blockSyncerHandler,
		closeCh:            make(chan struct{}),
		queueCh:            make(chan blockIndexerRunnerQueueItem, config.QueueChannelSize),
		logger:             logger,
	}

	if config.AutoStart {
		runner.Start()
	}

	return runner
}

func (br *BlockIndexerRunner) Start() {
	br.logger.Info("Starting block indexer runner")

	go func() {
		for {
			select {
			case <-br.closeCh:
				return
			case item := <-br.queueCh:
				if item.Point != nil {
					if err := br.blockSyncerHandler.RollBackward(*item.Point); err != nil {
						br.logger.Error("Failed to roll backward", "error", err)
					}
				} else {
					if err := br.blockSyncerHandler.RollForward(*item.BlockHeader, item.TxsRetriever); err != nil {
						br.logger.Error("Failed to roll forward", "error", err)
					}
				}
			}
		}
	}()
}

func (br *BlockIndexerRunner) Close() {
	if atomic.CompareAndSwapUint32(&br.isClosed, 0, 1) {
		br.logger.Info("Closing block indexer runner")

		close(br.closeCh)
	}
}

func (br *BlockIndexerRunner) RollBackward(point BlockPoint) error {
	br.queueCh <- blockIndexerRunnerQueueItem{Point: &point}

	return nil
}

func (br *BlockIndexerRunner) RollForward(blockHeader BlockHeader, txsRetriever BlockTxsRetriever) error {
	br.queueCh <- blockIndexerRunnerQueueItem{BlockHeader: &blockHeader, TxsRetriever: txsRetriever}

	return nil
}

func (br *BlockIndexerRunner) Reset() (BlockPoint, error) {
	return br.blockSyncerHandler.Reset()
}
