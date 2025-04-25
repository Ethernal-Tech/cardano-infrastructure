package gouroboros

import (
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/hashicorp/go-hclog"
)

const (
	ProtocolTCP  = "tcp"
	ProtocolUnix = "unix"

	syncStartTriesDefault = 4
)

type BlockSyncerConfig struct {
	NetworkMagic   uint32        `json:"networkMagic"`
	NodeAddress    string        `json:"nodeAddress"`
	RestartOnError bool          `json:"restartOnError"`
	RestartDelay   time.Duration `json:"restartDelay"`
	SyncStartTries int           `json:"syncStartTries"`
	KeepAlive      bool          `json:"keepAlive"`
}

func (bsc BlockSyncerConfig) Protocol() string {
	if strings.HasPrefix(bsc.NodeAddress, "/") {
		return ProtocolUnix
	}

	return ProtocolTCP
}

type BlockSyncerImpl struct {
	connection   *ouroboros.Connection
	blockHandler indexer.BlockSyncerHandler
	config       *BlockSyncerConfig
	logger       hclog.Logger

	errorCh  chan error
	closeCh  chan struct{}
	lock     sync.Mutex
	isClosed bool
}

var _ indexer.BlockSyncer = (*BlockSyncerImpl)(nil)

func NewBlockSyncer(
	config *BlockSyncerConfig, blockHandler indexer.BlockSyncerHandler, logger hclog.Logger,
) *BlockSyncerImpl {
	return &BlockSyncerImpl{
		blockHandler: blockHandler,
		config:       config,
		logger:       logger,
		errorCh:      make(chan error, 1),
		closeCh:      make(chan struct{}),
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

	bs.logger.Debug("Start syncing requested", "addr", bs.config.NodeAddress, "magic", bs.config.NetworkMagic)

	// create connection
	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(bs.config.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(bs.config.KeepAlive),
		ouroboros.WithChainSyncConfig(chainsync.NewConfig(
			chainsync.WithRollBackwardFunc(bs.rollBackwardCallback),
			chainsync.WithRollForwardFunc(bs.rollForwardCallback),
		)),
	)
	if err != nil {
		return err
	}

	// dial node -> connect to node
	if err := connection.Dial(bs.config.Protocol(), bs.config.NodeAddress); err != nil {
		return err
	}

	bs.connection = connection

	bs.logger.Debug("Connection established", "addr", bs.config.NodeAddress, "magic", bs.config.NetworkMagic)

	blockPoint, err := bs.blockHandler.Reset()
	if err != nil {
		return err
	}

	// start syncing
	var blockPointLedger common.Point
	if blockPoint.BlockSlot != 0 {
		blockPointLedger = common.NewPoint(blockPoint.BlockSlot, blockPoint.BlockHash[:])
	}

	if err := connection.ChainSync().Client.Sync([]common.Point{blockPointLedger}); err != nil {
		return err
	}

	bs.logger.Debug("Syncing started", "addr", bs.config.NodeAddress,
		"magic", bs.config.NetworkMagic, "point", blockPoint)

	// in separated routine wait for async errors
	go bs.errorHandler(connection.ErrorChan())

	return nil
}

func (bs *BlockSyncerImpl) rollBackwardCallback(
	ctx chainsync.CallbackContext, point common.Point, tip chainsync.Tip,
) error {
	bs.logger.Debug("Roll backward",
		"hash", hex.EncodeToString(point.Hash), "slot", point.Slot,
		"tip_slot", tip.Point.Slot, "tip_hash", hex.EncodeToString(tip.Point.Hash))

	blockPoint := indexer.BlockPoint{BlockSlot: point.Slot}
	if len(point.Hash) == indexer.HashSize {
		blockPoint.BlockHash = indexer.Hash(point.Hash)
	}

	return bs.blockHandler.RollBackward(blockPoint)
}

func (bs *BlockSyncerImpl) rollForwardCallback(
	ctx chainsync.CallbackContext, blockType uint, blockInfo interface{}, tip chainsync.Tip,
) error {
	blockHeader, ok := blockInfo.(ledger.BlockHeader)
	if !ok {
		return errors.New("failed to get block header with gouroboros")
	}

	bs.lock.Lock()
	if bs.connection == nil {
		bs.lock.Unlock()

		return errors.New("failed to get block transactions: no connection")
	}

	txsRetriever := &BlockTxsRetrieverImpl{
		connection: bs.connection,
		logger:     bs.logger,
	}
	bs.lock.Unlock()

	bs.logger.Debug("Roll forward",
		"hash", blockHeader.Hash(), "slot", blockHeader.SlotNumber(), "number", blockHeader.BlockNumber(),
		"tip_slot", tip.Point.Slot, "tip_hash", hex.EncodeToString(tip.Point.Hash))

	return bs.blockHandler.RollForward(indexer.BlockHeader{
		Slot:   blockHeader.SlotNumber(),
		Hash:   indexer.NewHashFromHexString(blockHeader.Hash()),
		Number: blockHeader.BlockNumber(),
		EraID:  blockHeader.Era().Id,
	}, txsRetriever)
}

func (bs *BlockSyncerImpl) errorHandler(errorCh <-chan error) {
	var (
		err error
		ok  bool
	)

	select {
	case <-bs.closeCh:
		return // close routine
	case err, ok = <-errorCh:
		if !ok {
			return
		}
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

func (bs *BlockSyncerImpl) closeConnectionNoLock() {
	if oldConn := bs.connection; oldConn != nil {
		bs.logger.Debug("Closing old connection")

		if err := oldConn.Close(); err != nil { // close previous connection
			bs.logger.Warn("Error while closing previous connection", "err", err)
		} else {
			<-oldConn.ErrorChan() // error channel will be closed after connection closing is done!

			bs.logger.Debug("Old connection has been closed")
		}
	}
}
