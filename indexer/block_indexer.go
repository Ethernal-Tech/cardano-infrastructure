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
}

type BlockIndexer struct {
	config *BlockIndexerConfig

	// latest confirmed and saved block point
	latestBlockPoint      *BlockPoint
	unconfirmedBlocks     infracommon.CircularQueue[BlockHeader]
	confirmedBlockHandler NewConfirmedBlockHandler
	addressesOfInterest   map[string]bool

	db BlockIndexerDB

	mutex  sync.Mutex
	logger hclog.Logger
}

var _ BlockSyncerHandler = (*BlockIndexer)(nil)

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

func (bi *BlockIndexer) RollBackward(point BlockPoint) error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

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

		bi.logger.Info("Roll backward to confirmed block", "slot", point.BlockSlot, "hash", point.BlockHash)

		// everything is ok -> we are reverting to the latest confirmed block
		return nil
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

	if !bi.unconfirmedBlocks.IsFull() {
		// If there are not enough children blocks to promote the first one to the confirmed state,
		// a new block header is added, and the function returns
		_ = bi.unconfirmedBlocks.Push(blockHeader)

		return nil
	}

	firstBlockHeader := bi.unconfirmedBlocks.Peek()

	txs, err := txsRetriever.GetBlockTransactions(firstBlockHeader)
	if err != nil {
		return err
	}

	confirmedBlock, confirmedTxs, latestBlockPoint, err := bi.processConfirmedBlock(firstBlockHeader, txs)
	if err != nil {
		return err
	}

	// update latest block point in memory if we have confirmed block
	bi.latestBlockPoint = latestBlockPoint

	bi.unconfirmedBlocks.Pop()
	_ = bi.unconfirmedBlocks.Push(blockHeader)

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

func (bi *BlockIndexer) processConfirmedBlock(
	confirmedBlockHeader BlockHeader, allTxs []*Tx,
) (*CardanoBlock, []*Tx, *BlockPoint, error) {
	var (
		txsHashes         []Hash
		txOutputsToSave   []*TxInputOutput
		txOutputsToRemove []*TxInput

		dbTx = bi.db.OpenTx() // open database tx
	)

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

	// add confirmed block to db and create full block only if there are some transactions of interest
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

	// update database -> execute db transaction
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
