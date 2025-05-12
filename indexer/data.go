package indexer

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrBlockIndexerFatal = errors.New("block indexer fatal error")
)

const HashSize = 32

type Hash [HashSize]byte

type BlockPoint struct {
	BlockSlot uint64 `json:"slot"`
	BlockHash Hash   `json:"hash"`
}

type BlockHeader struct {
	Slot   uint64 `json:"slot"`
	Hash   Hash   `json:"hash"`
	Number uint64 `json:"num"`
	EraID  uint8  `json:"era"`
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
	DatumHash Hash          `json:"datumHash,omitempty"`
	IsUsed    bool          `json:"used"`
	Tokens    []TokenAmount `json:"assets,omitempty"`
}

type TxInputOutput struct {
	Input  TxInput  `json:"inp"`
	Output TxOutput `json:"out"`
}

type CardanoBlock struct {
	Slot   uint64 `json:"slot"`
	Hash   Hash   `json:"hash"`
	Number uint64 `json:"num"`
	EraID  uint8  `json:"era"`
	Txs    []Hash `json:"txs"`
}

type TxInfo struct {
	Hash     string      `json:"hash"`
	MetaData []byte      `json:"md"`
	TTL      uint64      `json:"ttl"`
	Fee      uint64      `json:"fee"`
	IsValid  bool        `json:"isValid"`
	Inputs   []TxInput   `json:"inputs,omitempty"`
	Outputs  []*TxOutput `json:"outputs,omitempty"`
}

type processConfirmedBlockError struct {
	err error
}

func (e *processConfirmedBlockError) Error() string {
	return "process confirmed block error: " + e.err.Error()
}

func (e *processConfirmedBlockError) Unwrap() error {
	return e.err
}

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

func (cb *CardanoBlock) Key() []byte {
	return SlotNumberToKey(cb.Slot)
}

func (tx *Tx) Key() []byte {
	key := make([]byte, 8+4)

	binary.BigEndian.PutUint64(key[:8], tx.BlockSlot)
	binary.BigEndian.PutUint32(key[8:], tx.Indx)

	return key
}

func (tx *Tx) String() string {
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

func (t *TxOutput) IsNotUsed() bool {
	return t.Address != "" && !t.IsUsed
}

func (t *TxOutput) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s+%d", t.Address, t.Amount))

	for _, token := range t.Tokens {
		sb.WriteRune('+')
		sb.WriteString(token.String())
	}

	return sb.String()
}

func (ti *TxInput) Key() []byte {
	key := make([]byte, HashSize+4)

	copy(key, ti.Hash[:])
	binary.BigEndian.PutUint32(key[HashSize:], ti.Index)

	return key
}

func (ti *TxInput) Set(bytes []byte) error {
	if len(bytes) != HashSize+4 {
		return fmt.Errorf("invalid bytes size: %d", len(bytes))
	}

	ti.Hash = Hash(bytes[:HashSize])
	ti.Index = binary.BigEndian.Uint32(bytes[HashSize:])

	return nil
}

func (ti *TxInput) String() string {
	return fmt.Sprintf("%s#%d", ti.Hash, ti.Index)
}

func (t *TxInputOutput) String() string {
	if t.Output.Address == "" {
		return t.Input.String()
	}

	return fmt.Sprintf("%s::%s", t.Input.String(), t.Output.String())
}

func (bp *BlockPoint) String() string {
	return fmt.Sprintf("slot = %d, hash = %s", bp.BlockSlot, bp.BlockHash)
}

func (tt *TokenAmount) TokenName() string {
	return fmt.Sprintf("%s.%s", tt.PolicyID, hex.EncodeToString([]byte(tt.Name)))
}

func (tt *TokenAmount) String() string {
	return fmt.Sprintf("%d %s.%s", tt.Amount, tt.PolicyID, hex.EncodeToString([]byte(tt.Name)))
}

func (header BlockHeader) ToCardanoBlock(txs []Hash) *CardanoBlock {
	return &CardanoBlock{
		Slot:   header.Slot,
		Hash:   header.Hash,
		Number: header.Number,
		EraID:  header.EraID,
		Txs:    txs,
	}
}
