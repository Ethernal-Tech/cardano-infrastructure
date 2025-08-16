package ogmios

import (
	"encoding/json"
	"strings"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
)

type blockTxsRetrieverExtended interface {
	indexer.BlockTxsRetriever
	Add(slot uint64, txs []*ogmiosTransaction)
}

type ogmiosPoint struct {
	Slot uint64 `json:"slot"`
	Hash string `json:"id"`
}

// type ogmiosTip struct {
// 	Slot   uint64 `json:"slot"`
// 	Hash   string `json:"id"`
// 	Height uint64 `json:"height"`
// }

type ogmiosTxInput struct {
	Transaction struct {
		Hash string `json:"id"`
	} `json:"transaction"`
	Index uint32 `json:"index"`
}

type ogmiosTxOutput struct {
	Address   string                       `json:"address"`
	Value     map[string]map[string]uint64 `json:"value"`
	DatumHash string                       `json:"datumHash"`
	Datum     string                       `json:"datum"`
}

type ogmiosMetadata struct {
	Hash   []byte          `json:"hash"`
	Labels json.RawMessage `json:"labels"`
}

type ogmiosTransaction struct {
	Hash     string                       `json:"id"`
	Metadata *ogmiosMetadata              `json:"metadata"`
	Fee      map[string]map[string]uint64 `json:"fee"`
	Spends   string                       `json:"spends"`
	Inputs   []*ogmiosTxInput             `json:"inputs"`
	Outputs  []*ogmiosTxOutput            `json:"outputs"`
}

type ogmiosBlock struct {
	Type         string               `json:"type"`
	Era          string               `json:"era"`
	Slot         uint64               `json:"slot"`
	Hash         string               `json:"id"`
	Height       uint64               `json:"height"`
	Ancestor     string               `json:"ancestor"`
	Transactions []*ogmiosTransaction `json:"transactions"`
}

type ogmiosError struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ogmiosIntersection[T ogmiosPoint | string] struct {
	Points []T `json:"points"`
}

type ogmiosRequest struct {
	Version string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      string `json:"id"`
}

type ogmiosResponse struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ogmiosError    `json:"error,omitempty"`
	ID      string          `json:"id"`
}

type ogmiosResponseNextBlock struct {
	Direction string          `json:"direction"`
	Point     json.RawMessage `json:"point,omitempty"`
	Block     json.RawMessage `json:"block,omitempty"`
	// Tip       ogmiosTip       `json:"tip"`
}

func (a *ogmiosPoint) ToBlockPoint() indexer.BlockPoint {
	return indexer.BlockPoint{
		BlockSlot: a.Slot,
		BlockHash: indexer.NewHashFromHexString(a.Hash),
	}
}

func newOgmiosPoint(bp indexer.BlockPoint) ogmiosPoint {
	return ogmiosPoint{
		Slot: bp.BlockSlot,
		Hash: bp.BlockHash.String(),
	}
}

func (b *ogmiosBlock) ToBlockHeader() indexer.BlockHeader {
	return indexer.BlockHeader{
		Slot:   b.Slot,
		Hash:   indexer.NewHashFromHexString(b.Hash),
		Number: b.Height,
		EraID:  eraToID(b.Era),
	}
}

func eraToID(name string) uint8 {
	switch strings.ToLower(name) {
	case "byron":
		return 1
	case "shelley":
		return 2
	case "allegra":
		return 3
	case "mary":
		return 4
	case "alonzo":
		return 5
	case "babbage":
		return 6
	case "conway":
		return 7
	default:
		return 0
	}
}
