package indexer

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

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

	return fmt.Sprintf("forward (%d, %s)", qi.BlockHeader.Slot, qi.BlockHeader.Hash)
}

type BlockIndexerRunnerConfig struct {
	QueueChannelSize int           `json:"queueChannelSize"`
	RetryDelay       time.Duration `json:"retryDelay"`
}

type BlockIndexerRunner struct {
	blockSyncerHandler BlockSyncerHandler
	config             *BlockIndexerRunnerConfig
	isClosed           uint32
	lock               sync.RWMutex
	errorCh            chan error
	closeCh            chan struct{}
	stopLoopCh         chan struct{}
	loopFinishedCh     chan struct{}
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
		config:             config,
		errorCh:            make(chan error, 1),
		closeCh:            make(chan struct{}),
		loopFinishedCh:     make(chan struct{}),
		stopLoopCh:         make(chan struct{}),
		queueCh:            make(chan blockIndexerRunnerQueueItem, config.QueueChannelSize),
		logger:             logger,
	}

	close(runner.loopFinishedCh) // signal once for Reset

	return runner
}

func (br *BlockIndexerRunner) Close() error {
	if atomic.CompareAndSwapUint32(&br.isClosed, 0, 1) {
		br.logger.Info("Closing block indexer runner")

		close(br.closeCh)
	}

	return nil
}

func (br *BlockIndexerRunner) RollBackward(point BlockPoint) error {
	br.lock.RLock()
	queueCh := br.queueCh
	stopLoopCh := br.stopLoopCh
	br.lock.RUnlock()

	select {
	case queueCh <- blockIndexerRunnerQueueItem{Point: &point}:
	case <-stopLoopCh:
	case <-br.closeCh:
	}

	return nil
}

func (br *BlockIndexerRunner) RollForward(blockHeader BlockHeader, txsRetriever BlockTxsRetriever) error {
	br.lock.RLock()
	queueCh := br.queueCh
	stopLoopCh := br.stopLoopCh
	br.lock.RUnlock()

	select {
	case queueCh <- blockIndexerRunnerQueueItem{BlockHeader: &blockHeader, TxsRetriever: txsRetriever}:
	case <-stopLoopCh:
	case <-br.closeCh:
	}

	return nil
}

func (br *BlockIndexerRunner) Reset() (BlockPoint, error) {
	// stop main runner loop if started
	close(br.stopLoopCh)
	// wait for runner main loop to finish
	select {
	case <-br.loopFinishedCh:
	case <-br.closeCh:
		return BlockPoint{}, nil
	}
	// reset indexer before recreating channels and restart main runner loop
	bp, err := br.blockSyncerHandler.Reset()
	if err != nil {
		return bp, err
	}
	// create channels again
	br.lock.Lock()
	br.queueCh = make(chan blockIndexerRunnerQueueItem, br.config.QueueChannelSize)
	br.loopFinishedCh = make(chan struct{})
	br.stopLoopCh = make(chan struct{})
	br.lock.Unlock()
	// start runner main loop again
	br.runMainLoop()

	return bp, nil
}

func (br *BlockIndexerRunner) ErrorCh() <-chan error {
	return br.errorCh
}

func (br *BlockIndexerRunner) runMainLoop() {
	go func() {
		br.logger.Info("Block indexer runner has been started")

		br.lock.RLock()
		queueCh := br.queueCh
		stopLoopCh := br.stopLoopCh
		loopFinishedCh := br.loopFinishedCh
		br.lock.RUnlock()

		defer func() {
			br.logger.Info("Block indexer runner has been stopped")
			// Signal that the loop has finished (Reset method requires this signal)
			close(loopFinishedCh)
		}()

		for {
			select {
			case <-br.closeCh:
				return
			case <-stopLoopCh:
				return
			case item := <-queueCh:
				if br.execute(item, stopLoopCh) {
					return
				}
			}
		}
	}()
}

func (br *BlockIndexerRunner) execute(
	item blockIndexerRunnerQueueItem, stopLoopCh <-chan struct{},
) (breakLoop bool) {
	var err error
	// each item from the queue must be processed before moving to the next
	// the loop is infinite if the item cannot be processed and the error is non-fatal
	for {
		if item.Point != nil {
			err = br.blockSyncerHandler.RollBackward(*item.Point)
		} else {
			err = br.blockSyncerHandler.RollForward(*item.BlockHeader, item.TxsRetriever)
		}

		if err == nil {
			return false // item processed successfully
		}

		br.logger.Error("Runner failed", "item", item, "error", err)

		if errors.Is(err, ErrBlockIndexerFatal) {
			br.errorCh <- err // send fatal error to error channel

			return true
		}

		select {
		case <-br.closeCh:
			return true
		case <-stopLoopCh:
			return true
		case <-time.After(br.config.RetryDelay):
		}
	}
}
