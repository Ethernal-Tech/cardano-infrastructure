package indexer

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/common"
)

type Witness struct {
	VKey      []byte `json:"vkey"`
	Signature []byte `json:"sign"`
}

type BlockPoint struct {
	BlockSlot   uint64 `json:"slot"`
	BlockHash   []byte `json:"hash"`
	BlockNumber uint64 `json:"num"`
}

type FullBlock struct {
	Slot    uint64 `json:"slot"`
	Hash    string `json:"hash"`
	Number  uint64 `json:"num"`
	EraID   uint8  `json:"era"`
	EraName string `json:"-"`
	Txs     []*Tx  `json:"txs"`
}

type Tx struct {
	Hash      string           `json:"hash"`
	Metadata  []byte           `json:"metadata"`
	Inputs    []*TxInputOutput `json:"inputs"`
	Outputs   []*TxOutput      `json:"outputs"`
	Fee       uint64           `json:"fee"`
	Witnesses []Witness        `json:"witness"`
}

type TxInput struct {
	Hash  string `json:"id"`
	Index uint32 `json:"index"`
}

type TxOutput struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
	IsUsed  bool   `json:"isUsed"`
}

type TxInputOutput struct {
	Input  TxInput  `json:"inp"`
	Output TxOutput `json:"out"`
}

func NewFullBlock(bh ledger.BlockHeader, txs []*Tx) *FullBlock {
	return &FullBlock{
		Slot:    bh.SlotNumber(),
		Hash:    bh.Hash(),
		Number:  bh.BlockNumber(),
		EraID:   bh.Era().Id,
		EraName: bh.Era().Name,
		Txs:     txs,
	}
}

func NewWitnesses(vkeyWitnesses []interface{}) []Witness {
	res := make([]Witness, len(vkeyWitnesses))

	for i, vv := range vkeyWitnesses {
		arr, ok1 := vv.([]interface{})
		if !ok1 || len(arr) != 2 {
			panic("wrong key inside block") //nolint:gocritic
		}

		key, ok2 := arr[0].([]byte)
		sign, ok3 := arr[1].([]byte)
		if !ok2 || !ok3 {
			panic("wrong key inside block") //nolint:gocritic
		}

		res[i] = Witness{
			VKey:      key,
			Signature: sign,
		}
	}

	return res
}

func (fb FullBlock) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("number = %d, hash = %s, tx count = %d\n", fb.Number, fb.Hash, len(fb.Txs)))
	for _, tx := range fb.Txs {
		var (
			sbInp strings.Builder
			sbOut strings.Builder
		)

		for _, x := range tx.Inputs {
			if sbInp.Len() > 0 {
				sbInp.WriteString(", ")
			}

			sbInp.WriteString("[")
			sbInp.WriteString(x.Input.Hash)
			sbInp.WriteString(", ")
			sbInp.WriteString(strconv.FormatUint(uint64(x.Input.Index), 10))
			sbInp.WriteString(", ")
			sbInp.WriteString(x.Output.Address)
			sbInp.WriteString(", ")
			sbInp.WriteString(strconv.FormatUint(x.Output.Amount, 10))
			sbInp.WriteString("]")
		}

		for i, x := range tx.Outputs {
			if sbOut.Len() > 0 {
				sbOut.WriteString(", ")
			}

			sbOut.WriteString("[")
			sbOut.WriteString(strconv.Itoa(i))
			sbOut.WriteString(", ")
			sbOut.WriteString(x.Address)
			sbOut.WriteString(", ")
			sbOut.WriteString(strconv.FormatUint(x.Amount, 10))
			sbOut.WriteString("]")
		}

		sb.WriteString(fmt.Sprintf("  tx hash = %s, fee = %d\n", tx.Hash, tx.Fee))
		if tx.Metadata != nil {
			sb.WriteString(fmt.Sprintf("  meta = %s\n", string(tx.Metadata)))
		}

		sb.WriteString(fmt.Sprintf("   inputs = %s\n", sbInp.String()))
		sb.WriteString(fmt.Sprintf("  outputs = %s\n", sbOut.String()))
	}

	return sb.String()
}

func (to TxOutput) IsNotUsed() bool {
	return to.Address != "" && !to.IsUsed
}

func (ti TxInput) Key() []byte {
	return []byte(fmt.Sprintf("%s_%d", ti.Hash, ti.Index))
}

func (fb FullBlock) Key() []byte {
	return []byte(fmt.Sprintf("%s_%d", fb.Hash, fb.Slot))
}

func (bp BlockPoint) ToCommonPoint() common.Point {
	if len(bp.BlockHash) == 0 {
		return common.NewPointOrigin() // from genesis
	}

	return common.NewPoint(bp.BlockSlot, bp.BlockHash)
}

func (bp BlockPoint) String() string {
	return fmt.Sprintf("slot = %d, hash = %s, num = %d", bp.BlockSlot, hex.EncodeToString(bp.BlockHash), bp.BlockNumber)
}

func hash2Bytes(hash string) []byte {
	v, _ := hex.DecodeString(hash) // nolint

	return v
}

func bytes2Hash(hash []byte) string {
	return hex.EncodeToString(hash)
}
