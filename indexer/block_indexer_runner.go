package indexer

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/go-hclog"
)

type blockIndexerRunnerQueueItem struct {
	BlockHeader  *BlockHeader
	TxsRetriever BlockTxsRetriever
	Point        *BlockPoint
}

func (qi blockIndexerRunnerQueueItem) String() string {
	if qi.Point != nil {
		return fmt.Sprintf("backward (%d, %s)", qi.Point.BlockSlot, qi.Point.BlockHash)
	}

	return fmt.Sprintf("foward (%d, %s)", qi.BlockHeader.Slot, qi.BlockHeader.Hash)
}

type BlockIndexerRunnerConfig struct {
	AutoStart        bool `json:"autoStart"`
	QueueChannelSize int  `json:"queueChannelSize"`
}

type BlockIndexerRunner struct {
	blockSyncerHandler BlockSyncerHandler
	isClosed           uint32
	errorCh            chan error
	closeCh            chan struct{}
	queueCh            chan blockIndexerRunnerQueueItem
	logger             hclog.Logger
}

var (
	_ BlockSyncerHandler = (*BlockIndexerRunner)(nil)
	_ Service            = (*BlockIndexerRunner)(nil)
)

func NewBlockIndexerRunner(
	blockSyncerHandler BlockSyncerHandler, config *BlockIndexerRunnerConfig, logger hclog.Logger,
) *BlockIndexerRunner {
	runner := &BlockIndexerRunner{
		blockSyncerHandler: blockSyncerHandler,
		closeCh:            make(chan struct{}),
		errorCh:            make(chan error, 1),
		queueCh:            make(chan blockIndexerRunnerQueueItem, config.QueueChannelSize),
		logger:             logger,
	}

	if config.AutoStart {
		runner.Start()
	}

	return runner
}

func (br *BlockIndexerRunner) Start() {
	go func() {
		br.logger.Info("Block indexer runner has been started")

		var err error

		for {
			select {
			case <-br.closeCh:
				return
			case item := <-br.queueCh:
				if item.Point != nil {
					err = br.blockSyncerHandler.RollBackward(*item.Point)
				} else {
					err = br.blockSyncerHandler.RollForward(*item.BlockHeader, item.TxsRetriever)
				}

				if err != nil {
					br.logger.Error("Runner failed", "item", item, "error", err)
				}

				if errors.Is(err, ErrBlockIndexerFatal) {
					br.errorCh <- err

					return
				}
			}
		}
	}()
}

func (br *BlockIndexerRunner) Close() error {
	if atomic.CompareAndSwapUint32(&br.isClosed, 0, 1) {
		br.logger.Info("Closing block indexer runner")

		close(br.closeCh)
	}

	return nil
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

func (br *BlockIndexerRunner) ErrorCh() <-chan error {
	return br.errorCh
}
