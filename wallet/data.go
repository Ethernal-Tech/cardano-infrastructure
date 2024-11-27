package wallet

import (
	"context"
	"fmt"
	"strings"
)

const (
	adaTokenPolicyID = "ada"
	adaTokenName     = "lovelace"
)

type TokenAmount struct {
	PolicyID string `json:"pid"`
	Name     string `json:"nam"`
	Amount   uint64 `json:"val"`
}

func NewTokenAmount(policyID string, name string, amount uint64) TokenAmount {
	return TokenAmount{
		PolicyID: policyID,
		Name:     name,
		Amount:   amount,
	}
}

func NewTokenAmountWithFullName(name string, amount uint64) (TokenAmount, error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return TokenAmount{}, fmt.Errorf("name should have two parts but instead has: %d", len(parts))
	}

	return TokenAmount{
		PolicyID: parts[0],
		Name:     parts[1],
		Amount:   amount,
	}, nil
}

func (tt TokenAmount) TokenName() string {
	return fmt.Sprintf("%s.%s", tt.PolicyID, tt.Name)
}

func (tt TokenAmount) String() string {
	return fmt.Sprintf("%d %s.%s", tt.Amount, tt.PolicyID, tt.Name)
}

type Utxo struct {
	Hash   string        `json:"hsh"`
	Index  uint32        `json:"ind"`
	Amount uint64        `json:"amount"`
	Tokens []TokenAmount `json:"tokens,omitempty"`
}

type QueryTipData struct {
	Block           uint64 `json:"block"`
	Epoch           uint64 `json:"epoch"`
	Era             string `json:"era"`
	Hash            string `json:"hash"`
	Slot            uint64 `json:"slot"`
	SlotInEpoch     uint64 `json:"slotInEpoch"`
	SlotsToEpochEnd uint64 `json:"slotsToEpochEnd"`
	SyncProgress    string `json:"syncProgress"`
}

type ITxSubmitter interface {
	// SubmitTx submits transaction - txSigned should be cbor serialized signed transaction
	SubmitTx(ctx context.Context, txSigned []byte) error
}

type ITxRetriever interface {
	GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error)
}

type ITxDataRetriever interface {
	GetTip(ctx context.Context) (QueryTipData, error)
	GetProtocolParameters(ctx context.Context) ([]byte, error)
}

type IUTxORetriever interface {
	GetUtxos(ctx context.Context, addr string) ([]Utxo, error)
}

type ITxProvider interface {
	ITxSubmitter
	ITxDataRetriever
	IUTxORetriever
	Dispose()
}

type ISigner interface {
	GetSigningKey() []byte
	GetVerificationKey() []byte
}

type IWallet interface {
	ISigner
	GetStakeSigningKey() []byte
	GetStakeVerificationKey() []byte
}

type IPolicyScript interface {
	GetPolicyScriptJSON() ([]byte, error)
	GetCount() int
}
