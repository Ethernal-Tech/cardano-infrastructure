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

	var metadata []byte
	if gtx.Metadata() != nil && gtx.Metadata().Cbor() != nil {
		metadata = gtx.Metadata().Cbor()
	}

	outputs := gtx.Outputs()
	outputsFull := make([]TxOutput, len(outputs))
	inputs := gtx.Inputs()
	inputsFull := make([]TxInput, len(inputs))

	for i, out := range outputs {
		outputsFull[i] = TxOutput{
			Address: out.Address().String(),
			Amount:  out.Amount(),
		}
	}

	for i, inp := range inputs {
		inputsFull[i] = TxInput{
			Hash:  inp.Id().String(),
			Index: inp.Index(),
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
		Outputs: outputsFull,
		Inputs:  inputsFull,
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
