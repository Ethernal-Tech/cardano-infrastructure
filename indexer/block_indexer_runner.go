package indexer

import (
	"errors"
	"fmt"
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
	AutoStart        bool          `json:"autoStart"`
	QueueChannelSize int           `json:"queueChannelSize"`
	RetryDelay       time.Duration `json:"retryDelay"`
}

type BlockIndexerRunner struct {
	blockSyncerHandler BlockSyncerHandler
	config             *BlockIndexerRunnerConfig
	isClosed           uint32
	isLoopStarted      uint32
	errorCh            chan error
	stopLoopCh         chan struct{}
	loopFinishedCh     chan struct{}
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
		config:             config,
		errorCh:            make(chan error, 1),
		loopFinishedCh:     make(chan struct{}, 1),
		closeCh:            make(chan struct{}),
		stopLoopCh:         make(chan struct{}, 1),
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
		// check if loop is already started
		if !atomic.CompareAndSwapUint32(&br.isLoopStarted, 0, 1) {
			return
		}

		br.logger.Info("Block indexer runner has been started")

		defer func() {
			br.logger.Info("Block indexer runner has been stopped")
			// Signal that the loop has finished (Reset method requires this signal)
			// The signal will only be sent if there is already a listener waiting for the loop to end
			select {
			case br.loopFinishedCh <- struct{}{}:
			default:
			}
		}()

		for {
			select {
			case <-br.closeCh:
				return
			case <-br.stopLoopCh:
				return
			case item := <-br.queueCh:
				if err := br.execute(item); err != nil {
					br.errorCh <- err // quit if error is uncoverable error

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
	// stop main runner loop only if it is started
	if atomic.LoadUint32(&br.isLoopStarted) == 1 {
		br.stopLoopCh <- struct{}{} // notify that the main loop needs to terminate
		// wait for runner main loop to finish
		select {
		case <-br.loopFinishedCh:
		case <-br.closeCh:
		}
	}
	// reset indexer before recreating queue channel and restart main runner loop
	bp, err := br.blockSyncerHandler.Reset()
	if err != nil {
		return bp, err
	}
	// create queue channel again
	br.queueCh = make(chan blockIndexerRunnerQueueItem, br.config.QueueChannelSize)
	// start runner main loop again
	br.Start()

	return bp, nil
}

func (br *BlockIndexerRunner) ErrorCh() <-chan error {
	return br.errorCh
}

func (br *BlockIndexerRunner) execute(item blockIndexerRunnerQueueItem) (err error) {
	// each item from the queue must be processed before moving to the next
	// the loop is infinite if the item cannot be processed and the error is non-fatal
	for {
		if item.Point != nil {
			err = br.blockSyncerHandler.RollBackward(*item.Point)
		} else {
			err = br.blockSyncerHandler.RollForward(*item.BlockHeader, item.TxsRetriever)
		}

		if err == nil {
			return nil // item processed successfully
		}

		br.logger.Error("Runner failed", "item", item, "error", err)

		if errors.Is(err, ErrBlockIndexerFatal) {
			return err
		}

		select {
		case <-br.closeCh:
			return nil
		case <-br.stopLoopCh:
			return nil
		case <-time.After(br.config.RetryDelay):
		}
	}
}
