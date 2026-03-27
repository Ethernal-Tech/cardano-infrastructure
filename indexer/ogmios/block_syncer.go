package ogmios

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
)

const (
	syncStartTriesDefault = 4

	findIntersectionMethod = "findIntersection"
	nextBlockMethod        = "nextBlock"

	findIntersectionID = "int"
	nextBlockID        = "nb"
)

type BlockSyncerConfig struct {
	URL            string        `json:"url"`
	RestartOnError bool          `json:"restartOnError"`
	RestartDelay   time.Duration `json:"restartDelay"`
	SyncStartTries int           `json:"syncStartTries"`
}

type BlockSyncerImpl struct {
	connection   *websocket.Conn
	blockHandler indexer.BlockSyncerHandler
	config       *BlockSyncerConfig
	logger       hclog.Logger

	errorCh  chan error
	closeCh  chan struct{}
	lock     sync.Mutex
	isClosed bool

	blockTxsRetriever blockTxsRetrieverExtended
}

var _ indexer.BlockSyncer = (*BlockSyncerImpl)(nil)

func NewBlockSyncer(
	config *BlockSyncerConfig, blockHandler indexer.BlockSyncerHandler, logger hclog.Logger,
) *BlockSyncerImpl {
	return &BlockSyncerImpl{
		blockHandler:      blockHandler,
		config:            config,
		errorCh:           make(chan error, 1),
		closeCh:           make(chan struct{}),
		blockTxsRetriever: newBlockTxsRetrieverImpl(logger),
		logger:            logger,
	}
}

func (bs *BlockSyncerImpl) Sync() (err error) {
	cntTries := bs.config.SyncStartTries
	if cntTries <= 0 {
		cntTries = syncStartTriesDefault
	}

	for i := 1; i <= cntTries; i++ {
		if err = bs.syncExecute(); err == nil {
			break
		} else if i < cntTries {
			bs.logger.Warn("Error while starting syncer", "err", err, "attempt", i, "of", cntTries)
		}

		select {
		case <-bs.closeCh:
			return
		case <-time.After(bs.config.RestartDelay):
		}
	}

	return err
}

func (bs *BlockSyncerImpl) Close() error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if !bs.isClosed {
		bs.isClosed = true

		close(bs.closeCh)
		bs.closeConnectionNoLock()
	}

	return nil
}

func (bs *BlockSyncerImpl) ErrorCh() <-chan error {
	return bs.errorCh
}

func (bs *BlockSyncerImpl) syncExecute() error {
	// if the syncer is closed in the meantime -> quit
	select {
	case <-bs.closeCh:
		return nil
	default:
	}

	// close the old connection and create a new one within the lock.
	// two syncExecute calls should not run in parallel
	bs.lock.Lock()
	defer bs.lock.Unlock()

	bs.closeConnectionNoLock()

	bs.logger.Debug("Start syncing requested", "addr", bs.config.URL)

	connection, _, err := websocket.DefaultDialer.Dial(bs.config.URL, nil)
	if err != nil {
		return err
	}

	bs.connection = connection

	bs.logger.Debug("Connection established", "addr", bs.config.URL)

	blockPoint, err := bs.blockHandler.Reset()
	if err != nil {
		return err
	}

	// in separated routine wait for messages
	go func() {
		err := bs.mainLoop(connection)
		bs.handleError(err)
	}()

	// start syncing
	if err := sendFindIntersection(connection, blockPoint); err != nil {
		return err
	}

	bs.logger.Debug("Syncing started", "url", bs.config.URL, "point", blockPoint)

	return nil
}

func (bs *BlockSyncerImpl) handleError(err error) {
	// if the syncer is closed in the meantime -> quit
	select {
	case <-bs.closeCh:
		return
	default:
	}

	// retry syncing again if not fatal error and if RestartOnError is true (errors.Is does not work in this case)
	if !strings.Contains(err.Error(), indexer.ErrBlockIndexerFatal.Error()) && bs.config.RestartOnError {
		bs.logger.Warn("Error happened during synchronization", "err", err)

		select {
		case <-bs.closeCh:
			return
		case <-time.After(bs.config.RestartDelay):
		}

		if err := bs.Sync(); err != nil {
			bs.logger.Error("Error happened while trying to restart the synchronization", "err", err)
			bs.errorCh <- err // propagate error
		}
	} else {
		bs.logger.Error("Error happened during synchronization. Restart the syncer manually.", "err", err)
		bs.errorCh <- err // propagate error
	}
}

func (bs *BlockSyncerImpl) mainLoop(conn *websocket.Conn) error {
	// BlockSyncerImpl->Close will close connection which will break the loop
	// later on code bellow is mandatory to skip closing syncer error
	// select {
	// case <-bs.closeCh:
	// 	return
	// default:
	// }
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var response ogmiosResponse

		if err = json.Unmarshal(message, &response); err != nil {
			return err
		}

		if response.Error != nil {
			return fmt.Errorf("reader error %d: %s", response.Error.Code, response.Error.Message)
		}

		// find interesetion just sets cursor on cardano node side. nothing to do on client side
		if response.ID == nextBlockID {
			if err := bs.handleNextBlock(response.Result); err != nil {
				return err
			}
		}

		// send message in order to receive next block from the cursor
		if err = sendNextBlock(conn); err != nil {
			return err
		}
	}
}

func (bs *BlockSyncerImpl) closeConnectionNoLock() {
	if oldConn := bs.connection; oldConn != nil {
		bs.logger.Debug("Closing old connection")

		if err := oldConn.Close(); err != nil { // close previous connection
			bs.logger.Warn("Error while closing previous connection", "err", err)
		} else {
			bs.logger.Debug("Old connection has been closed")
		}
	}
}

func (bs *BlockSyncerImpl) handleNextBlock(result json.RawMessage) error {
	var nextBlockResult ogmiosResponseNextBlock

	if err := json.Unmarshal(result, &nextBlockResult); err != nil {
		return err
	}

	if nextBlockResult.Direction == "forward" {
		var block ogmiosBlock

		if err := json.Unmarshal(nextBlockResult.Block, &block); err != nil {
			return err
		}

		bs.blockTxsRetriever.Add(block.Slot, block.Transactions) // set transactions for block set in tx retriever

		return bs.blockHandler.RollForward(block.ToBlockHeader(), bs.blockTxsRetriever)
	}

	var point ogmiosPoint

	if errp := json.Unmarshal(nextBlockResult.Point, &point); errp != nil {
		return bs.blockHandler.RollBackward(indexer.BlockPoint{}) // roll back to origin
	}

	return bs.blockHandler.RollBackward(point.ToBlockPoint()) // roll back to point
}
