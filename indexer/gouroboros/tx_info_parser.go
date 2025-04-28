package gouroboros

import (
	"fmt"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/blinklabs-io/gouroboros/ledger"
)

var _ indexer.TxInfoParserFunc = ParseTxInfo

func ParseTxInfo(rawTx []byte, full bool) (indexer.TxInfo, error) {
	gtx, err := tryParseTxRaw(rawTx)
	if err != nil {
		return indexer.TxInfo{}, err
	}

	var (
		metadata  []byte
		txOutputs []*indexer.TxOutput
		txInputs  []indexer.TxInput
	)

	if gtx.Metadata() != nil && gtx.Metadata().Cbor() != nil {
		metadata = gtx.Metadata().Cbor()
	}

	if full {
		if libOutputs := gtx.Outputs(); len(libOutputs) > 0 {
			txOutputs = make([]*indexer.TxOutput, len(libOutputs))
			for j, out := range libOutputs {
				txOutputs[j] = createTxOutput(0, out)
			}
		}

		if libInputs := gtx.Inputs(); len(libInputs) > 0 {
			txInputs = make([]indexer.TxInput, len(libInputs))
			for i, inp := range libInputs {
				txInputs[i] = indexer.TxInput{
					Hash:  indexer.Hash(inp.Id()),
					Index: inp.Index(),
				}
			}
		}
	}

	return indexer.TxInfo{
		Hash:     gtx.Hash(),
		TTL:      gtx.TTL(),
		MetaData: metadata,
		Fee:      gtx.Fee(),
		IsValid:  gtx.IsValid(),
		Inputs:   txInputs,
		Outputs:  txOutputs,
	}, nil
}

func tryParseTxRaw(data []byte) (ledger.Transaction, error) {
	if tx, err := ledger.NewAlonzoTransactionFromCbor(data); err == nil {
		return tx, nil
	}

	if tx, err := ledger.NewConwayTransactionFromCbor(data); err == nil {
		return tx, nil
	}

	if tx, err := ledger.NewBabbageTransactionFromCbor(data); err == nil {
		return tx, nil
	}

	if tx, err := ledger.NewMaryTransactionFromCbor(data); err == nil {
		return tx, nil
	}

	if tx, err := ledger.NewAllegraTransactionFromCbor(data); err == nil {
		return tx, nil
	}

	if tx, err := ledger.NewShelleyTransactionFromCbor(data); err == nil {
		return tx, nil
	}

	return nil, fmt.Errorf("unknown transaction type")
}
