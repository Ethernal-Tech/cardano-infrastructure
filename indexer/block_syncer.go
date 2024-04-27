package indexer

import (
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/hashicorp/go-hclog"
)

var (
	errBlockSyncerFatal = errors.New("block syncer fatal error")
)

const (
	ProtocolTCP  = "tcp"
	ProtocolUnix = "unix"

	syncStartTriesDefault = 4
)

type GetTxsFunc func() ([]ledger.Transaction, error)

type BlockSyncer interface {
	Sync() error
	Close() error
	ErrorCh() <-chan error
}

type BlockSyncerHandler interface {
	RollBackwardFunc(ctx chainsync.CallbackContext, point common.Point, tip chainsync.Tip) error
	RollForwardFunc(blockHeader ledger.BlockHeader, getTxsFunc GetTxsFunc, tip chainsync.Tip) error
	Reset() (BlockPoint, error)
}

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
	blockHandler BlockSyncerHandler
	config       *BlockSyncerConfig
	logger       hclog.Logger

	errorCh chan error
	lock    sync.Mutex
	closed  chan struct{}
}

var _ BlockSyncer = (*BlockSyncerImpl)(nil)

func NewBlockSyncer(config *BlockSyncerConfig, blockHandler BlockSyncerHandler, logger hclog.Logger) *BlockSyncerImpl {
	return &BlockSyncerImpl{
		blockHandler: blockHandler,
		config:       config,
		logger:       logger,
		errorCh:      make(chan error, 1),
		closed:       make(chan struct{}),
	}
}

func (bs *BlockSyncerImpl) Sync() (err error) {
	cntTries := bs.config.SyncStartTries
	if cntTries <= 0 {
		cntTries = syncStartTriesDefault
	}

	ticker := time.NewTicker(bs.config.RestartDelay)
	defer ticker.Stop()

	for i := 1; i <= cntTries; i++ {
		if err = bs.syncExecute(); err == nil {
			break
		} else if i < cntTries {
			bs.logger.Warn("Error while starting syncer", "err", err, "attempt", i, "of", cntTries)
		}

		select {
		case <-bs.closed:
			return
		case <-ticker.C:
		}
	}

	return err
}

func (bs *BlockSyncerImpl) Close() error {
	close(bs.closed)

	// connection should be closed inside lock
	// because BlockSyncerImpl->syncExecute can create new one from another routine
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if bs.connection == nil {
		return nil
	}

	return bs.connection.Close()
}

func (bs *BlockSyncerImpl) ErrorCh() <-chan error {
	return bs.errorCh
}

func (bs *BlockSyncerImpl) syncExecute() error {
	// connection should be created inside lock because
	// BlockSyncerImpl->Close can be called from another routine
	bs.lock.Lock()
	defer bs.lock.Unlock()

	// if the syncer is closed in the meantime -> quit
	select {
	case <-bs.closed:
		return nil
	default:
	}

	if oldConn := bs.connection; oldConn != nil {
		if err := oldConn.Close(); err != nil { // close previous connection
			bs.logger.Warn("Error while closing previous connection", "err", err)
		} else {
			<-oldConn.ErrorChan() // error channel will be closed after connection closing is done!
		}
	}

	blockPoint, err := bs.blockHandler.Reset()
	if err != nil {
		return err
	}

	bs.logger.Debug("Start syncing requested",
		"networkMagic", bs.config.NetworkMagic, "addr", bs.config.NodeAddress, "point", blockPoint)

	// create connection
	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(bs.config.NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(bs.config.KeepAlive),
		ouroboros.WithChainSyncConfig(chainsync.NewConfig(
			chainsync.WithRollBackwardFunc(bs.blockHandler.RollBackwardFunc),
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

	bs.logger.Debug("Syncing started",
		"networkMagic", bs.config.NetworkMagic, "addr", bs.config.NodeAddress, "point", blockPoint)

	// start syncing
	if err := connection.ChainSync().Client.Sync([]common.Point{blockPoint.ToCommonPoint()}); err != nil {
		return err
	}

	// in separated routine wait for async errors
	go bs.errorHandler()

	return nil
}

func (bs *BlockSyncerImpl) getBlock(slot uint64, hash []byte) (ledger.Block, error) {
	bs.logger.Debug("Get full block", "slot", slot, "hash", hex.EncodeToString(hash), "connected", bs.connection != nil)

	if bs.connection == nil {
		return nil, errors.New("no connection")
	}

	return bs.connection.BlockFetch().Client.GetBlock(common.NewPoint(slot, hash))
}

func (bs *BlockSyncerImpl) getBlockTransactions(blockHeader ledger.BlockHeader) ([]ledger.Transaction, error) {
	block, err := bs.getBlock(blockHeader.SlotNumber(), hash2Bytes(blockHeader.Hash()))
	if err != nil {
		return nil, err
	}

	return block.Transactions(), nil
}

func (bs *BlockSyncerImpl) rollForwardCallback(
	ctx chainsync.CallbackContext, blockType uint, blockInfo interface{}, tip chainsync.Tip,
) error {
	blockHeader, ok := blockInfo.(ledger.BlockHeader)
	if !ok {
		return errors.Join(errBlockSyncerFatal, errors.New("invalid header"))
	}

	bs.logger.Debug("Roll forward",
		"number", blockHeader.BlockNumber(), "hash", blockHeader.Hash(), "slot", blockHeader.SlotNumber(),
		"tip_slot", tip.Point.Slot, "tip_hash", hex.EncodeToString(tip.Point.Hash))

	getTxsFunc := func() ([]ledger.Transaction, error) {
		return bs.getBlockTransactions(blockHeader)
	}

	return bs.blockHandler.RollForwardFunc(blockHeader, getTxsFunc, tip)
}

func (bs *BlockSyncerImpl) errorHandler() {
	if bs.connection == nil {
		return
	}

	err, ok := <-bs.connection.ErrorChan()
	if !ok {
		close(bs.errorCh)

		return
	}

	// retry syncing again if not fatal error and if RestartOnError is true (errors.Is does not work in this case)
	if !strings.Contains(err.Error(), errBlockSyncerFatal.Error()) && bs.config.RestartOnError {
		bs.logger.Warn("Error happened during synchronization", "err", err)

		select {
		case <-bs.closed:
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
