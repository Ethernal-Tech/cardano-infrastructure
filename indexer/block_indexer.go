package indexer

import (
	"errors"
	"fmt"
	"sync"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/hashicorp/go-hclog"
)

const (
	AddressCheckNone    = 0               // No flags
	AddressCheckInputs  = 1 << (iota - 1) // 1 << 0 = 0x00...0001 = 1
	AddressCheckOutputs                   // 1 << 1 = 0x00...0010 = 2
	AddressCheckAll     = AddressCheckInputs | AddressCheckOutputs

	// how many confirmed blocks before the latest one syncing starts from when the sync could not start
	// from the latest one and deep rollbacks are allowed. Doubled on each further failed start
	deepRollbackResumeBlockCount = 20
	// upper bound of the doubled count: the cardano security parameter k, no rollback is deeper than that
	deepRollbackMaxResumeBlockCount = 2160
)

type BlockIndexerConfig struct {
	StartingBlockPoint *BlockPoint `json:"startingBlockPoint"`
	// how many children blocks is needed for some block to be considered final
	ConfirmationBlockCount  uint     `json:"confirmationBlockCount"`
	AddressesOfInterest     []string `json:"addressesOfInterest"`
	KeepAllTxOutputsInDB    bool     `json:"keepAllTxOutputsInDb"`
	AddressCheck            int      `json:"addressCheck"`
	SoftDeleteUtxo          bool     `json:"softDeleteUtxo"`
	KeepAllTxsHashesInBlock bool     `json:"keepAllTxsHashesInBlock"`
	// A rollback past the latest confirmed block is a fatal error by default. When allowed, the indexer
	// moves back to the rollback point and continues, and if the node does not recognize the latest
	// confirmed block on start (it was rolled back while the indexer was not running), syncing starts
	// from older confirmed blocks. Blocks, txs and outputs after the rollback point are not reverted
	// and blocks may be processed again, so it is meant only for indexers that just observe txs
	AllowDeepRollback bool `json:"allowDeepRollback"`
}

type BlockIndexer struct {
	config *BlockIndexerConfig

	// latest confirmed and saved block point
	latestBlockPoint      *BlockPoint
	unconfirmedBlocks     infracommon.CircularQueue[BlockHeader]
	confirmedBlockHandler NewConfirmedBlockHandler
	addressesOfInterest   map[string]bool
	// number of times in a row the node rejected the point returned by Reset (intersect not found), and the
	// oldest of those points. Cleared by any received block or rollback. Other sync start failures do not
	// change them
	rejectedStarts int
	oldestRejected *BlockPoint
	// Reset returned a point older than the one in the database. It is stored when the node rolls back to it
	resumedFromOlderPoint bool

	db BlockIndexerDB

	mutex  sync.Mutex
	logger hclog.Logger
}

var (
	_ BlockSyncerHandler       = (*BlockIndexer)(nil)
	_ IntersectNotFoundHandler = (*BlockIndexer)(nil)
)

func NewBlockIndexer(
	config *BlockIndexerConfig, confirmedBlockHandler NewConfirmedBlockHandler, db BlockIndexerDB, logger hclog.Logger,
) *BlockIndexer {
	if config.AddressCheck&AddressCheckAll == 0 {
		panic("block indexer must at least check outputs or inputs") //nolint:gocritic
	}

	addressesOfInterest := make(map[string]bool, len(config.AddressesOfInterest))
	for _, x := range config.AddressesOfInterest {
		addressesOfInterest[x] = true
	}

	return &BlockIndexer{
		config:                config,
		latestBlockPoint:      nil,
		confirmedBlockHandler: confirmedBlockHandler,
		unconfirmedBlocks:     infracommon.NewCircularQueue[BlockHeader](int(config.ConfirmationBlockCount)), //nolint
		db:                    db,
		addressesOfInterest:   addressesOfInterest,
		logger:                logger,
	}
}

func (bi *BlockIndexer) AddNewAddressesOfInterest(addresses ...string) {
	bi.mutex.Lock()
	for _, address := range addresses {
		bi.addressesOfInterest[address] = true
	}
	bi.mutex.Unlock()
}

func (bi *BlockIndexer) RollBackward(point BlockPoint) error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	resumedFromOlderPoint := bi.resumedFromOlderPoint
	bi.clearRejectedStarts()

	// linear is ok, there will be smaller number of unconfirmed blocks in memory
	indx := bi.unconfirmedBlocks.Find(func(header BlockHeader) bool {
		return header.Slot == point.BlockSlot && header.Hash == point.BlockHash
	})
	if indx != -1 {
		bi.logger.Info("Roll backward to unconfirmed block", "indx", indx,
			"slot", point.BlockSlot, "hash", point.BlockHash)

		bi.unconfirmedBlocks.SetCount(indx + 1)

		return nil
	}

	if bi.latestBlockPoint.BlockSlot == point.BlockSlot && bi.latestBlockPoint.BlockHash == point.BlockHash {
		bi.unconfirmedBlocks.SetCount(0)

		// the node accepted the older point Reset returned instead of the database one: store it, so a
		// connection that drops before the next block does not start from the rejected point again
		if resumedFromOlderPoint {
			bi.logger.Info("Storing the older sync start point accepted by the node",
				"slot", point.BlockSlot, "hash", point.BlockHash)

			return bi.setLatestBlockPoint(&point)
		}

		bi.logger.Info("Roll backward to confirmed block", "slot", point.BlockSlot, "hash", point.BlockHash)

		// everything is ok -> we are reverting to the latest confirmed block
		return nil
	}

	if bi.config.AllowDeepRollback {
		bi.logger.Warn("Roll backward before the latest confirmed block",
			"slot", point.BlockSlot, "hash", point.BlockHash,
			"latest slot", bi.latestBlockPoint.BlockSlot, "latest hash", bi.latestBlockPoint.BlockHash)

		return bi.setLatestBlockPoint(&point)
	}

	// we have confirmed a block that should NOT have been confirmed!
	// recovering from this error is difficult and requires manual database changes
	return errors.Join(ErrBlockIndexerFatal,
		fmt.Errorf("roll backward block not found. new = (%d, %s) vs latest = (%d, %s)",
			point.BlockSlot, point.BlockHash, bi.latestBlockPoint.BlockSlot, bi.latestBlockPoint.BlockHash))
}

func (bi *BlockIndexer) RollForward(blockHeader BlockHeader, txsRetriever BlockTxsRetriever) error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	bi.clearRejectedStarts()

	// without confirmations the new block is confirmed right away
	firstBlockHeader := blockHeader

	if bi.config.ConfirmationBlockCount > 0 {
		if !bi.unconfirmedBlocks.IsFull() {
			// If there are not enough children blocks to promote the first one to the confirmed state,
			// a new block header is added, and the function returns
			_ = bi.unconfirmedBlocks.Push(blockHeader)

			return nil
		}

		firstBlockHeader = bi.unconfirmedBlocks.Peek()
	}

	txs, err := txsRetriever.GetBlockTransactions(firstBlockHeader)
	if err != nil {
		return &processConfirmedBlockError{err: err}
	}

	confirmedBlock, confirmedTxs, latestBlockPoint, err := bi.processConfirmedBlock(firstBlockHeader, txs)
	if err != nil {
		return &processConfirmedBlockError{err: err}
	}

	// update latest block point in memory if we have confirmed block
	bi.latestBlockPoint = latestBlockPoint

	if bi.config.ConfirmationBlockCount > 0 {
		bi.unconfirmedBlocks.Pop()
		_ = bi.unconfirmedBlocks.Push(blockHeader)
	}

	return bi.confirmedBlockHandler(confirmedBlock, confirmedTxs)
}

func (bi *BlockIndexer) Reset() (BlockPoint, error) {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	// try to read latest point block from the database
	latestPoint, err := bi.db.GetLatestBlockPoint()
	if err != nil {
		return BlockPoint{}, err
	}

	bi.resumedFromOlderPoint = false

	// ...or an older one if the node rejected it (see IntersectNotFound). Connection failures do not move
	// the point back
	if latestPoint != nil && bi.config.AllowDeepRollback && bi.rejectedStarts > 0 {
		resumePoint, err := bi.getDeepRollbackResumePoint(*latestPoint)
		if err != nil {
			return BlockPoint{}, err
		}

		bi.resumedFromOlderPoint = *resumePoint != *latestPoint
		latestPoint = resumePoint
	}

	// ...then if latest point block is not in the database pick it from the configuration
	if latestPoint == nil {
		latestPoint = bi.config.StartingBlockPoint
	}

	// ...then if latest point block is still nil, create default one starting from the genesis block point
	if latestPoint == nil {
		latestPoint = &BlockPoint{}
	}

	bi.latestBlockPoint = latestPoint
	bi.unconfirmedBlocks.SetCount(0) // clear all unconfirmed from the memory

	return *latestPoint, nil
}

// IntersectNotFound is called by the syncer when the node rejected the point returned by Reset, so the next
// Reset starts from an older confirmed block (only when deep rollbacks are allowed)
func (bi *BlockIndexer) IntersectNotFound(point BlockPoint) {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	bi.rejectedStarts++

	if bi.oldestRejected == nil || point.BlockSlot < bi.oldestRejected.BlockSlot {
		bi.oldestRejected = &point
	}

	bi.logger.Warn("The node does not know the sync start point",
		"slot", point.BlockSlot, "hash", point.BlockHash, "rejected starts", bi.rejectedStarts)
}

// getDeepRollbackResumeBlockCount returns how many confirmed blocks before the latest one syncing starts from
// after rejectedStarts sync starts rejected by the node. It doubles on each rejection, so a block the node does
// not know (e.g. from an abandoned fork) is not picked again and again
func getDeepRollbackResumeBlockCount(rejectedStarts int) int {
	cnt := deepRollbackResumeBlockCount

	for i := 1; i < rejectedStarts && cnt < deepRollbackMaxResumeBlockCount; i++ {
		cnt *= 2
	}

	return min(cnt, deepRollbackMaxResumeBlockCount)
}

func (bi *BlockIndexer) clearRejectedStarts() {
	bi.rejectedStarts = 0
	bi.oldestRejected = nil
	bi.resumedFromOlderPoint = false
}

// getDeepRollbackResumePoint returns the point to sync from after the node rejected rejectedStarts points:
//   - the confirmed block getDeepRollbackResumeBlockCount blocks before the latest point. Rollbacks are not
//     deeper than deepRollbackMaxResumeBlockCount, so the block at that limit is retried rather than going
//     further back (the node is behind, not on another fork)
//   - if there are fewer confirmed blocks before the latest point, the oldest of them and then the starting
//     point (older than all confirmed blocks), the first one older than every rejected point
//   - the oldest rejected point if none of them is older
func (bi *BlockIndexer) getDeepRollbackResumePoint(latestPoint BlockPoint) (*BlockPoint, error) {
	cnt := getDeepRollbackResumeBlockCount(bi.rejectedStarts)

	// only blocks before the latest point are read: deep rollbacks do not remove the confirmed blocks after
	// the rollback point, and there may be more of them than cnt
	blocks, err := bi.db.GetConfirmedBlocksBefore(latestPoint.BlockSlot, cnt)
	if err != nil {
		return nil, err
	}

	if len(blocks) == cnt {
		point := &BlockPoint{BlockSlot: blocks[cnt-1].Slot, BlockHash: blocks[cnt-1].Hash}

		bi.logger.Warn("Syncing from an older confirmed block",
			"slot", point.BlockSlot, "hash", point.BlockHash, "blocks back", cnt)

		return point, nil
	}

	candidates := make([]*BlockPoint, 0, 2)

	if len(blocks) > 0 {
		oldest := blocks[len(blocks)-1]
		candidates = append(candidates, &BlockPoint{BlockSlot: oldest.Slot, BlockHash: oldest.Hash})
	}

	candidates = append(candidates, bi.config.StartingBlockPoint)

	for _, point := range candidates {
		if point != nil && point.BlockSlot < bi.oldestRejected.BlockSlot {
			bi.logger.Warn("Syncing from an older point", "slot", point.BlockSlot, "hash", point.BlockHash,
				"starting point", point == bi.config.StartingBlockPoint)

			return point, nil
		}
	}

	bi.logger.Error("No point older than the ones the node rejected, syncing from the oldest rejected one again",
		"slot", bi.oldestRejected.BlockSlot, "hash", bi.oldestRejected.BlockHash)

	point := *bi.oldestRejected

	return &point, nil
}

// setLatestBlockPoint moves the indexer back to the given point
func (bi *BlockIndexer) setLatestBlockPoint(point *BlockPoint) error {
	if err := bi.db.OpenTx().SetLatestBlockPoint(point).Execute(); err != nil {
		return err
	}

	bi.latestBlockPoint = point
	bi.unconfirmedBlocks.SetCount(0)

	return nil
}

func (bi *BlockIndexer) processConfirmedBlock(
	confirmedBlockHeader BlockHeader, allTxs []*Tx,
) (*CardanoBlock, []*Tx, *BlockPoint, error) {
	var (
		txsHashes         []Hash
		txOutputsToSave   []*TxInputOutput
		txOutputsToRemove []TxInput

		dbTx = bi.db.OpenTx() // open database tx
	)
	// populate each input with full output data (address, amount, etc.) based on its (hash, index)
	if err := bi.populateOutputsForEachInput(allTxs); err != nil {
		return nil, nil, nil, err
	}

	// get all transactions of interest from block
	relevantTxs := bi.filterTxsOfInterest(allTxs)

	if bi.config.KeepAllTxOutputsInDB {
		txOutputsToSave = getTxOutputs(allTxs, nil)
		txOutputsToRemove = getTxInputs(allTxs, nil)
	} else {
		txOutputsToSave = getTxOutputs(relevantTxs, bi.addressesOfInterest)
		txOutputsToRemove = getTxInputs(relevantTxs, bi.addressesOfInterest)
	}

	// add all relevant transactions from the confirmed block to the db
	dbTx.AddConfirmedTxs(relevantTxs)

	if bi.config.KeepAllTxsHashesInBlock {
		txsHashes = getTxHashes(allTxs)
	} else {
		txsHashes = getTxHashes(relevantTxs)
	}

	confirmedBlock := confirmedBlockHeader.ToCardanoBlock(txsHashes)
	latestBlockPoint := &BlockPoint{
		BlockSlot: confirmedBlockHeader.Slot,
		BlockHash: confirmedBlockHeader.Hash,
	}
	// save confirmed block (without tx details) in db
	dbTx.AddConfirmedBlock(confirmedBlock)
	// update latest block point in db tx
	dbTx.SetLatestBlockPoint(latestBlockPoint)
	// add all needed outputs, remove used ones in db tx
	dbTx.AddTxOutputs(txOutputsToSave).RemoveTxOutputs(txOutputsToRemove, bi.config.SoftDeleteUtxo)

	// execute all previously queued updates in a single atomic db operation
	if err := dbTx.Execute(); err != nil {
		return nil, nil, nil, err
	}

	return confirmedBlock, relevantTxs, latestBlockPoint, nil
}

func (bi *BlockIndexer) filterTxsOfInterest(txs []*Tx) (result []*Tx) {
	if len(bi.addressesOfInterest) == 0 {
		return txs
	}

	for _, tx := range txs {
		if bi.isTxInputOfInterest(tx) || bi.isTxOutputOfInterest(tx) {
			result = append(result, tx)
		}
	}

	return result
}

func (bi *BlockIndexer) isTxOutputOfInterest(tx *Tx) bool {
	if bi.config.AddressCheck&AddressCheckOutputs == 0 {
		return false
	}

	for _, out := range tx.Outputs {
		if bi.addressesOfInterest[out.Address] {
			return true
		}
	}

	return false
}

func (bi *BlockIndexer) isTxInputOfInterest(tx *Tx) bool {
	if bi.config.AddressCheck&AddressCheckInputs == 0 {
		return false
	}

	for _, inp := range tx.Inputs {
		if bi.addressesOfInterest[inp.Output.Address] {
			return true
		}
	}

	return false
}

func (bi *BlockIndexer) populateOutputsForEachInput(txs []*Tx) (err error) {
	for _, tx := range txs {
		for _, inp := range tx.Inputs {
			if inp.Output.Address != "" {
				continue // output is already set
			}
			// if there is no output for the input, zero address and amount are set
			inp.Output, err = bi.db.GetTxOutput(inp.Input)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
