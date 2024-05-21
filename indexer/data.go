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
	VKey      []byte `json:"key"`
	Signature []byte `json:"sgn"`
}

type BlockPoint struct {
	BlockSlot   uint64 `json:"slot"`
	BlockHash   []byte `json:"hash"`
	BlockNumber uint64 `json:"num"`
}

type Tx struct {
	BlockSlot uint64           `json:"slot"`
	BlockHash string           `json:"bhash"`
	Indx      uint32           `json:"ind"`
	Hash      string           `json:"hash"`
	Metadata  []byte           `json:"metadata"`
	Inputs    []*TxInputOutput `json:"inp"`
	Outputs   []*TxOutput      `json:"out"`
	Fee       uint64           `json:"fee"`
	Witnesses []Witness        `json:"ws"`
	Valid     bool             `json:"valid"`
}

type TxInput struct {
	Hash  string `json:"id"`
	Index uint32 `json:"ind"`
}

type TxOutput struct {
	Address string `json:"addr"`
	Block   uint64 `json:"block"`
	Amount  uint64 `json:"amnt"`
	IsUsed  bool   `json:"used"`
}

type TxInputOutput struct {
	Input  TxInput  `json:"inp"`
	Output TxOutput `json:"out"`
}

type CardanoBlock struct {
	Slot    uint64   `json:"slot"`
	Hash    string   `json:"hash"`
	Number  uint64   `json:"num"`
	EraID   uint8    `json:"era"`
	EraName string   `json:"-"`
	Txs     []string `json:"txs"`
}

func NewCardanoBlock(header ledger.BlockHeader, txs []string) *CardanoBlock {
	return &CardanoBlock{
		Slot:    header.SlotNumber(),
		Hash:    header.Hash(),
		Number:  header.BlockNumber(),
		EraID:   header.Era().Id,
		EraName: header.Era().Name,
		Txs:     txs,
	}
}

func (cb CardanoBlock) Key() []byte {
	return SlotNumberToKey(cb.Slot)
}

func SlotNumberToKey(slotNumber uint64) []byte {
	return []byte(fmt.Sprintf("%20d", slotNumber))
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

func (tx Tx) Key() []byte {
	return []byte(fmt.Sprintf("%20d_%6d", tx.BlockSlot, tx.Indx))
}

func (tx Tx) String() string {
	var (
		sb    strings.Builder
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

	sb.WriteString("hash = ")
	sb.WriteString(tx.Hash)
	sb.WriteString("\nblock hash = ")
	sb.WriteString(tx.BlockHash)
	sb.WriteString("\nblock slot = ")
	sb.WriteString(strconv.FormatUint(tx.BlockSlot, 10))
	sb.WriteString("\nfee = ")
	sb.WriteString(strconv.FormatUint(tx.Fee, 10))

	if tx.Metadata != nil {
		sb.WriteString("\nmeta = ")
		sb.WriteString(string(tx.Metadata))
	}

	sb.WriteString("\ninputs = ")
	sb.WriteString(sbInp.String())
	sb.WriteString("\noutputs = ")
	sb.WriteString(sbOut.String())

	return sb.String()
}

func (to TxOutput) IsNotUsed() bool {
	return to.Address != "" && !to.IsUsed
}

func (ti TxInput) Key() []byte {
	return []byte(fmt.Sprintf("%s_%d", ti.Hash, ti.Index))
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
	v, _ := hex.DecodeString(hash)

	return v
}

func bytes2Hash(hash []byte) string {
	return hex.EncodeToString(hash)
}
