package indexer

import (
	"errors"
	"fmt"
	"sync"

	infraCommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/common"
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
	ConfirmationBlockCount uint `json:"confirmationBlockCount"`

	AddressesOfInterest []string `json:"addressesOfInterest"`

	KeepAllTxOutputsInDB bool `json:"keepAllTxOutputsInDb"`

	AddressCheck int `json:"addressCheck"`

	SoftDeleteUtxo bool `json:"softDeleteUtxo"`

	KeepAllTxsHashesInBlock bool `json:"keepAllTxsHashesInBlock"`
}

type NewConfirmedBlockHandler func(*CardanoBlock, []*Tx) error

type blockWithLazyTxRetriever struct {
	header ledger.BlockHeader
	getTxs GetTxsFunc
}

type BlockIndexer struct {
	config *BlockIndexerConfig

	// latest confirmed and saved block point
	latestBlockPoint      *BlockPoint
	unconfirmedBlocks     infraCommon.CircularQueue[blockWithLazyTxRetriever]
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
		unconfirmedBlocks:     infraCommon.NewCircularQueue[blockWithLazyTxRetriever](int(config.ConfirmationBlockCount)),
		db:                    db,
		addressesOfInterest:   addressesOfInterest,
		logger:                logger,
	}
}

func (bi *BlockIndexer) RollBackwardFunc(
	ctx chainsync.CallbackContext, point common.Point, tip chainsync.Tip,
) error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	pointHash := bytes2HashString(point.Hash)

	// linear is ok, there will be smaller number of unconfirmed blocks in memory
	indx := bi.unconfirmedBlocks.Find(func(unc blockWithLazyTxRetriever) bool {
		return unc.header.SlotNumber() == point.Slot && unc.header.Hash() == pointHash
	})
	if indx != -1 {
		bi.logger.Info("Roll backward to unconfirmed block",
			"hash", pointHash, "slot", point.Slot, "indx", indx)

		bi.unconfirmedBlocks.SetCount(indx + 1)

		return nil
	}

	if bi.latestBlockPoint.BlockSlot == point.Slot && bi.latestBlockPoint.BlockHash.String() == pointHash {
		bi.unconfirmedBlocks.SetCount(0)

		bi.logger.Info("Roll backward to confirmed block", "hash", pointHash, "slot", point.Slot)

		// everything is ok -> we are reverting to the latest confirmed block
		return nil
	}

	// we have confirmed some block that should not be confirmed!!!! TODO: what to do in this case?
	return errors.Join(errBlockSyncerFatal,
		fmt.Errorf("roll backward block not found. new = (%d, %s) vs latest = (%d, %s)",
			point.Slot, pointHash,
			bi.latestBlockPoint.BlockSlot, bi.latestBlockPoint.BlockHash))
}

func (bi *BlockIndexer) RollForwardFunc(
	blockHeader ledger.BlockHeader, getTxsFunc GetTxsFunc, tip chainsync.Tip,
) error {
	bi.mutex.Lock()
	defer bi.mutex.Unlock()

	if !bi.unconfirmedBlocks.IsFull() {
		// If there are not enough children blocks to promote the first one to the confirmed state,
		// a new block header is added, and the function returns
		_ = bi.unconfirmedBlocks.Push(blockWithLazyTxRetriever{
			header: blockHeader,
			getTxs: getTxsFunc,
		})

		return nil
	}

	firstBlock := bi.unconfirmedBlocks.Peek()

	txs, err := firstBlock.getTxs()
	if err != nil {
		return err
	}

	confirmedBlock, confirmedTxs, latestBlockPoint, err := bi.processConfirmedBlock(firstBlock.header, txs)
	if err != nil {
		return err
	}

	// update latest block point in memory if we have confirmed block
	bi.latestBlockPoint = latestBlockPoint

	bi.unconfirmedBlocks.Pop()
	_ = bi.unconfirmedBlocks.Push(blockWithLazyTxRetriever{
		header: blockHeader,
		getTxs: getTxsFunc,
	})

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
	confirmedBlockHeader ledger.BlockHeader, allBlockTransactions []ledger.Transaction,
) (*CardanoBlock, []*Tx, *BlockPoint, error) {
	var (
		txsHashes         []Hash
		confirmedTxs      []*Tx
		txOutputsToSave   []*TxInputOutput
		txOutputsToRemove []*TxInput

		dbTx = bi.db.OpenTx() // open database tx
	)

	// get all transactions of interest from block
	txsOfInterest, err := bi.filterTxsOfInterest(allBlockTransactions)
	if err != nil {
		return nil, nil, nil, err
	}

	if bi.config.KeepAllTxOutputsInDB {
		txOutputsToSave = bi.getTxOutputs(confirmedBlockHeader.SlotNumber(), allBlockTransactions, nil)
		txOutputsToRemove = bi.getTxInputs(allBlockTransactions)
	} else {
		txOutputsToSave = bi.getTxOutputs(confirmedBlockHeader.SlotNumber(), txsOfInterest, bi.addressesOfInterest)
		txOutputsToRemove = bi.getTxInputs(txsOfInterest)
	}

	// add confirmed block to db and create full block only if there are some transactions of interest
	if len(txsOfInterest) > 0 {
		confirmedTxs = make([]*Tx, len(txsOfInterest))
		for i, ltx := range txsOfInterest {
			confirmedTxs[i], err = bi.createTx(confirmedBlockHeader, ltx, uint32(i))
			if err != nil {
				return nil, nil, nil, err
			}
		}

		dbTx.AddConfirmedTxs(confirmedTxs) // add confirmed txs in db
	}

	if bi.config.KeepAllTxsHashesInBlock {
		txsHashes = getTxHashes(allBlockTransactions)
	} else {
		txsHashes = getTxHashes(txsOfInterest)
	}

	confirmedBlock := NewCardanoBlock(confirmedBlockHeader, txsHashes)
	latestBlockPoint := &BlockPoint{
		BlockSlot:   confirmedBlockHeader.SlotNumber(),
		BlockHash:   NewHashFromHexString(confirmedBlockHeader.Hash()),
		BlockNumber: confirmedBlockHeader.BlockNumber(),
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

	return confirmedBlock, confirmedTxs, latestBlockPoint, nil
}

func (bi *BlockIndexer) filterTxsOfInterest(txs []ledger.Transaction) (result []ledger.Transaction, err error) {
	if len(bi.addressesOfInterest) == 0 {
		return txs, nil
	}

	for _, tx := range txs {
		if bi.config.AddressCheck&AddressCheckOutputs != 0 && bi.isTxOutputOfInterest(tx) {
			result = append(result, tx)
		} else if bi.config.AddressCheck&AddressCheckInputs != 0 {
			txIsGood, err := bi.isTxInputOfInterest(tx)
			if err != nil {
				return nil, err
			} else if txIsGood {
				result = append(result, tx)
			}
		}
	}

	return result, nil
}

func (bi *BlockIndexer) isTxOutputOfInterest(tx ledger.Transaction) bool {
	for _, out := range tx.Outputs() {
		addr := LedgerAddressToString(out.Address())
		if bi.addressesOfInterest[addr] {
			return true
		}
	}

	return false
}

func (bi *BlockIndexer) isTxInputOfInterest(tx ledger.Transaction) (bool, error) {
	for _, inp := range tx.Inputs() {
		txOutput, err := bi.db.GetTxOutput(TxInput{
			Hash:  Hash(inp.Id()),
			Index: inp.Index(),
		})
		if err != nil {
			return false, err
		} else if !txOutput.IsUsed && bi.addressesOfInterest[txOutput.Address] {
			return true, nil
		}
	}

	return false, nil
}

func (bi *BlockIndexer) getTxOutputs(
	slot uint64, txs []ledger.Transaction, addressesOfInterest map[string]bool,
) (res []*TxInputOutput) {
	for _, tx := range txs {
		for ind, txOut := range tx.Outputs() {
			assets := txOut.Assets()

			nativeAsset := NativeAsset{}

			if assets != nil {
				for _, p := range assets.Policies() {
					policyID := p
					for _, as := range assets.Assets(p) {
						assetAmount := assets.Asset(p, as)

						nativeAsset.Name = string(as)
						nativeAsset.Amount = assetAmount
						nativeAsset.PolicyId = policyID.String()
					}
				}
			}

			addr := LedgerAddressToString(txOut.Address())
			if len(addressesOfInterest) > 0 && !bi.addressesOfInterest[addr] {
				continue
			}

			res = append(res, &TxInputOutput{
				Input: TxInput{
					Hash:  NewHashFromHexString(tx.Hash()),
					Index: uint32(ind),
				},
				Output: TxOutput{
					Slot:    slot,
					Address: addr,
					Amount:  txOut.Amount(),
					Assets: NativeAsset{
						Name:     nativeAsset.Name,
						PolicyId: nativeAsset.PolicyId,
						Amount:   nativeAsset.Amount,
					},
				},
			})
		}
	}

	return res
}

func (bi *BlockIndexer) getTxInputs(txs []ledger.Transaction) (res []*TxInput) {
	for _, tx := range txs {
		for _, inp := range tx.Inputs() {
			res = append(res, &TxInput{
				Hash:  Hash(inp.Id()),
				Index: inp.Index(),
			})
		}
	}

	return res
}

func (bi *BlockIndexer) createTx(
	ledgerBlockHeader ledger.BlockHeader, ledgerTx ledger.Transaction, indx uint32,
) (*Tx, error) {
	tx := &Tx{
		Indx:      indx,
		Hash:      NewHashFromHexString(ledgerTx.Hash()),
		Fee:       ledgerTx.Fee(),
		BlockSlot: ledgerBlockHeader.SlotNumber(),
		BlockHash: NewHashFromHexString(ledgerBlockHeader.Hash()),
		Valid:     ledgerTx.IsValid(),
	}

	if inputs := ledgerTx.Inputs(); len(inputs) > 0 {
		tx.Inputs = make([]*TxInputOutput, len(inputs))

		for j, inp := range inputs {
			txInput := TxInput{
				Hash:  Hash(inp.Id()),
				Index: inp.Index(),
			}

			output, err := bi.db.GetTxOutput(txInput)
			if err != nil {
				return nil, err
			}

			tx.Inputs[j] = &TxInputOutput{
				Input:  txInput,
				Output: output,
			}
		}
	}

	if outputs := ledgerTx.Outputs(); len(outputs) > 0 {
		tx.Outputs = make([]*TxOutput, len(outputs))

		for j, out := range outputs {
			tx.Outputs[j] = &TxOutput{
				Slot:    ledgerBlockHeader.SlotNumber(),
				Address: LedgerAddressToString(out.Address()),
				Amount:  out.Amount(),
			}
		}
	}

	if ledgerTx.Metadata() != nil && ledgerTx.Metadata().Cbor() != nil {
		tx.Metadata = ledgerTx.Metadata().Cbor()
	}

	switch realTx := ledgerTx.(type) {
	case *ledger.AllegraTransaction:
		tx.Witnesses = NewWitnesses(realTx.WitnessSet.VkeyWitnesses)
	case *ledger.AlonzoTransaction:
		tx.Witnesses = NewWitnesses(realTx.WitnessSet.VkeyWitnesses)
	case *ledger.BabbageTransaction:
		tx.Witnesses = NewWitnesses(realTx.WitnessSet.VkeyWitnesses)
	case *ledger.ByronTransaction:
		// not supported
	case *ledger.ConwayTransaction:
		tx.Witnesses = NewWitnesses(realTx.WitnessSet.VkeyWitnesses)
	case *ledger.MaryTransaction:
		tx.Witnesses = NewWitnesses(realTx.WitnessSet.VkeyWitnesses)
	case *ledger.ShelleyTransaction:
		tx.Witnesses = NewWitnesses(realTx.WitnessSet.VkeyWitnesses)
	}

	return tx, nil
}

func getTxHashes(txs []ledger.Transaction) []Hash {
	if len(txs) == 0 {
		return nil
	}

	res := make([]Hash, len(txs))
	for i, x := range txs {
		res[i] = NewHashFromHexString(x.Hash())
	}

	return res
}
