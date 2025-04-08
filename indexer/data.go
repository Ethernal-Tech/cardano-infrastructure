package indexer

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Ethernal-Tech/cardano-infrastructure/wallet"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/common"
)

const HashSize = 32

type Hash [HashSize]byte

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

func NewHashFromHexString(hash string) Hash {
	v, _ := hex.DecodeString(strings.TrimPrefix(hash, "0x"))

	return NewHashFromBytes(v)
}

func NewHashFromBytes(bytes []byte) Hash {
	if len(bytes) != HashSize {
		result := Hash{}
		size := min(HashSize, len(bytes))

		copy(result[HashSize-size:], bytes[:size])

		return result
	}

	return Hash(bytes)
}

type Witness struct {
	VKey      []byte `json:"key"`
	Signature []byte `json:"sgn"`
}

type BlockPoint struct {
	BlockSlot   uint64 `json:"slot"`
	BlockHash   Hash   `json:"hash"`
	BlockNumber uint64 `json:"num"`
}

type Tx struct {
	BlockSlot uint64           `json:"slot"`
	BlockHash Hash             `json:"bhash"`
	Indx      uint32           `json:"ind"`
	Hash      Hash             `json:"hash"`
	Metadata  []byte           `json:"metadata,omitempty"`
	Inputs    []*TxInputOutput `json:"inp"`
	Outputs   []*TxOutput      `json:"out"`
	Fee       uint64           `json:"fee"`
	Valid     bool             `json:"valid"`
}

type TxInput struct {
	Hash  Hash   `json:"id"`
	Index uint32 `json:"ind"`
}

type TokenAmount struct {
	PolicyID string `json:"polid"`
	Name     string `json:"name"`
	Amount   uint64 `json:"amnt"`
}

type TxOutput struct {
	Address   string        `json:"addr"`
	Slot      uint64        `json:"slot"`
	Amount    uint64        `json:"amnt"`
	Datum     []byte        `json:"datum,omitempty"`
	DatumHash Hash          `json:"datumHsh,omitempty"`
	IsUsed    bool          `json:"used"`
	Tokens    []TokenAmount `json:"assets,omitempty"`
}

type TxInputOutput struct {
	Input  TxInput  `json:"inp"`
	Output TxOutput `json:"out"`
}

type CardanoBlock struct {
	Slot    uint64 `json:"slot"`
	Hash    Hash   `json:"hash"`
	Number  uint64 `json:"num"`
	EraID   uint8  `json:"era"`
	EraName string `json:"-"`
	Txs     []Hash `json:"txs"`
}

func NewCardanoBlock(header ledger.BlockHeader, txs []Hash) *CardanoBlock {
	return &CardanoBlock{
		Slot:    header.SlotNumber(),
		Hash:    NewHashFromHexString(header.Hash()),
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
	bytes := make([]byte, 8)

	binary.BigEndian.PutUint64(bytes, slotNumber)

	return bytes
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
	key := make([]byte, 8+4)

	binary.BigEndian.PutUint64(key[:8], tx.BlockSlot)
	binary.BigEndian.PutUint32(key[8:], tx.Indx)

	return key
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
		sbInp.WriteString(x.String())
		sbInp.WriteString("]")
	}

	for _, x := range tx.Outputs {
		if sbOut.Len() > 0 {
			sbOut.WriteString(", ")
		}

		sbOut.WriteString("[")
		sbOut.WriteString(x.String())
		sbOut.WriteString("]")
	}

	sb.WriteString("hash = ")
	sb.WriteString(tx.Hash.String())
	sb.WriteString("\nblock hash = ")
	sb.WriteString(tx.BlockHash.String())
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
	key := make([]byte, HashSize+4)

	copy(key[:], ti.Hash[:])
	binary.BigEndian.PutUint32(key[HashSize:], ti.Index)

	return key
}

func NewTxInputFromBytes(bytes []byte) (TxInput, error) {
	if len(bytes) != HashSize+4 {
		return TxInput{}, fmt.Errorf("invalid bytes size: %d", len(bytes))
	}

	return TxInput{
		Hash:  Hash(bytes[:HashSize]),
		Index: binary.BigEndian.Uint32(bytes[HashSize:]),
	}, nil
}

func (bp BlockPoint) ToCommonPoint() common.Point {
	if bp.BlockSlot == 0 {
		return common.NewPointOrigin() // from genesis
	}

	return common.NewPoint(bp.BlockSlot, bp.BlockHash[:])
}

func (bp BlockPoint) String() string {
	return fmt.Sprintf("slot = %d, hash = %s, num = %d",
		bp.BlockSlot, hex.EncodeToString(bp.BlockHash[:]), bp.BlockNumber)
}

func bytes2HashString(bytes []byte) string {
	if len(bytes) == HashSize {
		return hex.EncodeToString(bytes)
	}

	h := NewHashFromBytes(bytes)

	return hex.EncodeToString(h[:])
}

func (t TxInput) String() string {
	return fmt.Sprintf("%s#%d", t.Hash, t.Index)
}

func (t TxOutput) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s+%d", t.Address, t.Amount))

	for _, token := range t.Tokens {
		sb.WriteRune('+')
		sb.WriteString(token.String())
	}

	return sb.String()
}

func (t TxInputOutput) String() string {
	if t.Output.Address == "" {
		return t.Input.String()
	}

	return fmt.Sprintf("%s::%s", t.Input, t.Output)
}

func (tt TokenAmount) TokenName() string {
	return fmt.Sprintf("%s.%s", tt.PolicyID, hex.EncodeToString([]byte(tt.Name)))
}

func (tt TokenAmount) String() string {
	return fmt.Sprintf("%d %s.%s", tt.Amount, tt.PolicyID, hex.EncodeToString([]byte(tt.Name)))
}

// LedgerAddressToString translates string representation of address to our wallet representation
// this will handle vector and other specific cases
func LedgerAddressToString(addr ledger.Address) string {
	ourAddr, err := wallet.NewCardanoAddress(addr.Bytes())
	if err != nil {
		return addr.String()
	}

	return ourAddr.String()
}

// SortTxInputOutputs sorts a slice of TxInputOutput pointers based on the following priority:
//  1. Inputs with lower Output.Slot values come first — this is critical because inputs added earlier must be processed first
//  2. If Slot values are equal, inputs are sorted lexicographically by their Input.Hash
//  3. If both Slot and Hash are equal, inputs are sorted by Input.Index
//
// The returned slice reflects this ordering and ensures deterministic processing of inputs in the correct chronological order.
func SortTxInputOutputs(txInputsOutputs []*TxInputOutput) []*TxInputOutput {
	sort.Slice(txInputsOutputs, func(i, j int) bool {
		first, second := txInputsOutputs[i], txInputsOutputs[j]

		if first.Output.Slot != second.Output.Slot {
			return first.Output.Slot < second.Output.Slot
		}

		if cmp := bytes.Compare(first.Input.Hash[:], second.Input.Hash[:]); cmp != 0 {
			return cmp < 0
		}

		return first.Input.Index < second.Input.Index
	})

	return txInputsOutputs
}
