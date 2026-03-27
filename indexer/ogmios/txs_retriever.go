package ogmios

import (
	"encoding/hex"
	"sync"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/Ethernal-Tech/cardano-infrastructure/wallet"
	"github.com/hashicorp/go-hclog"
)

type blockTxsRetrieverImpl struct {
	lock            sync.Mutex
	txsMap          map[uint64][]*ogmiosTransaction
	lastUsedSlotNum uint64
	logger          hclog.Logger
}

var (
	_ indexer.BlockTxsRetriever = (*blockTxsRetrieverImpl)(nil)
	_ blockTxsRetrieverExtended = (*blockTxsRetrieverImpl)(nil)
)

func newBlockTxsRetrieverImpl(logger hclog.Logger) *blockTxsRetrieverImpl {
	return &blockTxsRetrieverImpl{
		txsMap: map[uint64][]*ogmiosTransaction{},
		logger: logger,
	}
}

// Add implements blockTxsRetrieverExtended.
func (br *blockTxsRetrieverImpl) Add(slot uint64, txs []*ogmiosTransaction) {
	br.lock.Lock()
	defer br.lock.Unlock()

	br.txsMap[slot] = txs
}

func (br *blockTxsRetrieverImpl) GetBlockTransactions(blockHeader indexer.BlockHeader) ([]*indexer.Tx, error) {
	const ada, lovelace = wallet.AdaTokenPolicyID, wallet.AdaTokenName

	br.logger.Debug("Get block transactions", "slot", blockHeader.Slot, "hash", blockHeader.Hash)

	ogmiosTxs := br.getTxsForSlot(blockHeader.Slot)

	if len(ogmiosTxs) == 0 {
		return nil, nil
	}

	txs := make([]*indexer.Tx, len(ogmiosTxs))
	for i, otx := range ogmiosTxs {
		var (
			metadata []byte
			inputs   []*indexer.TxInputOutput
			outputs  []*indexer.TxOutput
			tokens   []indexer.TokenAmount
		)

		if otx.Metadata != nil {
			metadata = otx.Metadata.Labels
		}

		if len(otx.Inputs) > 0 {
			inputs = make([]*indexer.TxInputOutput, len(otx.Inputs))

			for j, inp := range otx.Inputs {
				inputs[j] = &indexer.TxInputOutput{
					Input: indexer.TxInput{
						Hash:  indexer.NewHashFromHexString(inp.Transaction.Hash),
						Index: inp.Index,
					},
				}
			}
		}

		if len(otx.Outputs) > 0 {
			outputs = make([]*indexer.TxOutput, len(otx.Outputs))

			for j, out := range otx.Outputs {
				dt, _ := hex.DecodeString(out.Datum)

				if len(out.Value) > 1 {
					tokens = make([]indexer.TokenAmount, 0, len(out.Value)-1)

					for policyID, vmap := range out.Value {
						if policyID == ada {
							continue
						}

						for tokenName, amount := range vmap {
							tokens = append(tokens, indexer.TokenAmount{
								PolicyID: policyID,
								Name:     tokenName,
								Amount:   amount,
							})
						}
					}
				}

				outputs[j] = &indexer.TxOutput{
					Slot:      blockHeader.Slot,
					Address:   out.Address,
					Amount:    out.Value[ada][lovelace],
					Datum:     dt,
					DatumHash: indexer.NewHashFromHexString(out.DatumHash),
					Tokens:    tokens,
				}
			}
		}

		txs[i] = &indexer.Tx{
			BlockSlot: blockHeader.Slot,
			BlockHash: blockHeader.Hash,
			Indx:      uint32(i), //nolint:gosec
			Hash:      indexer.NewHashFromHexString(otx.Hash),
			Metadata:  metadata,
			Fee:       otx.Fee[ada][lovelace],
			Inputs:    inputs,
			Outputs:   outputs,
			Valid:     true,
		}
	}

	return txs, nil
}

func (br *blockTxsRetrieverImpl) getTxsForSlot(slot uint64) []*ogmiosTransaction {
	br.lock.Lock()
	defer br.lock.Unlock()

	result := br.txsMap[slot]

	if br.lastUsedSlotNum != 0 && br.lastUsedSlotNum != slot {
		br.txsMap[br.lastUsedSlotNum] = nil // clear previosly read slot
	}

	br.lastUsedSlotNum = slot

	return result
}
