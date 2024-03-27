package indexer

import (
	ouroboros "github.com/blinklabs-io/gouroboros/ledger"
)

type TxInfo struct {
	Hash     string `json:"hash"`
	MetaData []byte `json:"md"`
	TTL      uint64 `json:"ttl"`
	Fee      uint64 `json:"fee"`
}

func ParseTxInfo(rawTx []byte) (TxInfo, error) {
	typ, err := ouroboros.DetermineTransactionType(rawTx)
	if err != nil {
		return TxInfo{}, err
	}

	gtx, err := ouroboros.NewTransactionFromCbor(typ, rawTx)
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
	}, nil
}
