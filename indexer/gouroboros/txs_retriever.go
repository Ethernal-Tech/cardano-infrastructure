package gouroboros

import (
	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/hashicorp/go-hclog"
)

type BlockTxsRetrieverImpl struct {
	connection *ouroboros.Connection
	logger     hclog.Logger
}

var _ indexer.BlockTxsRetriever = (*BlockTxsRetrieverImpl)(nil)

func (br *BlockTxsRetrieverImpl) GetBlockTransactions(blockHeader indexer.BlockHeader) ([]*indexer.Tx, error) {
	br.logger.Debug("Get block transactions", "slot", blockHeader.Slot, "hash", blockHeader.Hash)

	block, err := br.connection.BlockFetch().Client.GetBlock(
		common.NewPoint(blockHeader.Slot, blockHeader.Hash[:]),
	)
	if err != nil {
		return nil, err
	}

	legderTxs := block.Transactions()
	txs := make([]*indexer.Tx, len(legderTxs))

	for i, ledgerTx := range legderTxs {
		tx, err := createTx(&blockHeader, ledgerTx, uint32(i))
		if err != nil {
			return nil, err
		}

		txs[i] = tx
	}

	return txs, nil
}
