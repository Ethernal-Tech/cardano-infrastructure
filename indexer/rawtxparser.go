package indexer

import (
	"fmt"

	"github.com/blinklabs-io/gouroboros/ledger"
)

type TxInfo struct {
	Hash     string `json:"hash"`
	MetaData []byte `json:"md"`
	TTL      uint64 `json:"ttl"`
	Fee      uint64 `json:"fee"`
	IsValid  bool   `json:"isValid"`
}

type TxInfoFull struct {
	TxInfo
	Outputs []TxOutput
	Inputs  []TxInput
}

func ParseTxInfo(rawTx []byte) (TxInfo, error) {
	gtx, err := tryParseTxRaw(rawTx)
	if err != nil {
		return TxInfo{}, err
	}

	var metadata []byte
	if gtx.Metadata() != nil && gtx.Metadata().Cbor() != nil {
		metadata = gtx.Metadata().Cbor()
	}

	return TxInfo{
		Hash:     gtx.Hash(),
		TTL:      gtx.TTL(),
		MetaData: metadata,
		Fee:      gtx.Fee(),
		IsValid:  gtx.IsValid(),
	}, nil
}

func ParseTxFull(rawTx []byte) (TxInfoFull, error) {
	gtx, err := tryParseTxRaw(rawTx)
	if err != nil {
		return TxInfoFull{}, err
	}

	var (
		metadata  []byte
		txOutputs []TxOutput
		txInputs  []TxInput
	)

	if gtx.Metadata() != nil && gtx.Metadata().Cbor() != nil {
		metadata = gtx.Metadata().Cbor()
	}

	if outputs := gtx.Outputs(); len(outputs) > 0 {
		txOutputs = make([]TxOutput, len(outputs))
		for j, out := range outputs {
			txOutputs[j] = createTxOutput(0, LedgerAddressToString(out.Address()), out)
		}
	}

	if inputs := gtx.Inputs(); len(inputs) > 0 {
		txInputs = make([]TxInput, len(inputs))
		for i, inp := range inputs {
			txInputs[i] = TxInput{
				Hash:  Hash(inp.Id()),
				Index: inp.Index(),
			}
		}
	}

	return TxInfoFull{
		TxInfo: TxInfo{
			Hash:     gtx.Hash(),
			TTL:      gtx.TTL(),
			MetaData: metadata,
			Fee:      gtx.Fee(),
			IsValid:  gtx.IsValid(),
		},
		Outputs: txOutputs,
		Inputs:  txInputs,
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
