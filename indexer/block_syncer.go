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

type BlockTxsRetriever interface {
	GetBlockTransactions(blockHeader ledger.BlockHeader) ([]ledger.Transaction, error)
}

type BlockSyncer interface {
	Sync() error
	Close() error
	ErrorCh() <-chan error
}

type BlockSyncerHandler interface {
	RollBackwardFunc(point common.Point) error
	RollForwardFunc(blockHeader ledger.BlockHeader, txsRetriver BlockTxsRetriever) error
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

	for i := 1; i <= cntTries; i++ {
		if err = bs.syncExecute(); err == nil {
			break
		} else if i < cntTries {
			bs.logger.Warn("Error while starting syncer", "err", err, "attempt", i, "of", cntTries)
		}

		select {
		case <-bs.closed:
			return
		case <-time.After(bs.config.RestartDelay):
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
	// if the syncer is closed in the meantime -> quit
	select {
	case <-bs.closed:
		return nil
	default:
	}

	// connection should be created inside lock because
	// BlockSyncerImpl->Close can be called from another routine
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if oldConn := bs.connection; oldConn != nil {
		if err := oldConn.Close(); err != nil { // close previous connection
			bs.logger.Warn("Error while closing previous connection", "err", err)
		} else {
			<-oldConn.ErrorChan() // error channel will be closed after connection closing is done!
		}
	}

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
	if err := connection.ChainSync().Client.Sync([]common.Point{blockPoint.ToCommonPoint()}); err != nil {
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

	return bs.blockHandler.RollBackwardFunc(point)
}

func (bs *BlockSyncerImpl) GetBlockTransactions(blockHeader ledger.BlockHeader) ([]ledger.Transaction, error) {
	bs.logger.Debug("Get block transactions", "slot", blockHeader.SlotNumber(), "hash", blockHeader.Hash())

	bs.lock.Lock()
	connection := bs.connection
	bs.lock.Unlock()

	if connection == nil {
		return nil, errors.New("failed to get block transactions: no connection")
	}

	hash := NewHashFromHexString(blockHeader.Hash())

	block, err := connection.BlockFetch().Client.GetBlock(
		common.NewPoint(blockHeader.SlotNumber(), hash[:]),
	)
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
		"hash", blockHeader.Hash(), "slot", blockHeader.SlotNumber(), "number", blockHeader.BlockNumber(),
		"tip_slot", tip.Point.Slot, "tip_hash", hex.EncodeToString(tip.Point.Hash))

	return bs.blockHandler.RollForwardFunc(blockHeader, bs)
}

func (bs *BlockSyncerImpl) errorHandler(errorCh <-chan error) {
	var (
		err error
		ok  bool
	)

	select {
	case <-bs.closed:
		return // close routine
	case err, ok = <-errorCh:
		if !ok {
			return
		}
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
