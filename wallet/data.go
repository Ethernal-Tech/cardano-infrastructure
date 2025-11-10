package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	AdaTokenName     = "lovelace"
	AdaTokenPolicyID = "ada"
	DefaultEra       = "latest"
)

type Token struct {
	// Hexadecimal hash of the monetary policy script
	PolicyID string `json:"pid"`
	// Human-readable name of the token
	Name string `json:"nam"`
}

func NewToken(policyID string, name string) Token {
	return Token{
		PolicyID: policyID,
		Name:     name,
	}
}

func NewTokenWithFullName(name string, isNameEncoded bool) (Token, error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return Token{}, fmt.Errorf("invalid full token name: %s", name)
	}

	if !isNameEncoded {
		return Token{
			PolicyID: parts[0],
			Name:     parts[1],
		}, nil
	}

	decodedName, err := hex.DecodeString(parts[1])
	if err != nil {
		return Token{}, fmt.Errorf("invalid full token name: %s", name)
	}

	return Token{
		PolicyID: parts[0],
		Name:     string(decodedName),
	}, nil
}

func NewTokenWithFullNameTry(name string) (Token, error) {
	token, err := NewTokenWithFullName(name, true)
	if err == nil {
		return token, nil
	}

	token, err = NewTokenWithFullName(name, false)
	if err == nil {
		return token, nil
	}

	return token, err
}

func (tt Token) String() string {
	return fmt.Sprintf("%s.%s", tt.PolicyID, hex.EncodeToString([]byte(tt.Name)))
}

type TokenAmount struct {
	Token
	// Quantity of the token
	Amount uint64 `json:"val"`
}

func NewTokenAmount(token Token, amount uint64) TokenAmount {
	return TokenAmount{
		Token:  token,
		Amount: amount,
	}
}

func (tt TokenAmount) TokenName() string {
	return tt.Token.String()
}

func (tt TokenAmount) String() string {
	return fmt.Sprintf("%d %s.%s", tt.Amount, tt.PolicyID, hex.EncodeToString([]byte(tt.Name)))
}

type Utxo struct {
	Hash   string        `json:"hsh"`
	Index  uint32        `json:"ind"`
	Amount uint64        `json:"amount"`
	Tokens []TokenAmount `json:"tokens,omitempty"`
}

func (utxo Utxo) GetTokenAmount(tokenName string) uint64 {
	if tokenName == AdaTokenName {
		return utxo.Amount
	}

	for _, token := range utxo.Tokens {
		if token.TokenName() == tokenName {
			return token.Amount
		}
	}

	return 0
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

type QueryStakeAddressInfo struct {
	Address              string `json:"address"`
	DelegationDeposit    uint64 `json:"delegationDeposit"`
	RewardAccountBalance uint64 `json:"rewardAccountBalance"`
	StakeDelegation      string `json:"delegation"`
	VoteDelegation       string `json:"voteDelegation"`
}

type QueryEvaluateTxData struct {
	Memory uint64 `json:"memory"`
	CPU    uint64 `json:"cpu"`
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
	EvaluateTx(ctx context.Context, rawTx []byte) (QueryEvaluateTxData, error)
	IUTxORetriever
}

type IUTxORetriever interface {
	GetUtxos(ctx context.Context, addr string) ([]Utxo, error)
}

type IStakeDataRetriever interface {
	GetStakePools(ctx context.Context) ([]string, error)
	GetStakeAddressInfo(ctx context.Context, stakeAddress string) (QueryStakeAddressInfo, error)
}

type ITxProvider interface {
	ITxSubmitter
	ITxDataRetriever
	IUTxORetriever
	IStakeDataRetriever
	Dispose()
}

type ITxSigner interface {
	CreateTxWitness(txHash []byte) ([]byte, error)
	GetSigningKeys() ([]byte, []byte)
}

type ISerializable interface {
	GetBytesJSON() ([]byte, error)
}

type IPolicyScript interface {
	ICardanoArtifact
	GetCount() int
}

type ICardanoArtifact interface {
	ISerializable
}

type CardanoArtifact struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	CborHex     string `json:"cborHex"`
}

// GetBytesJSON returns Cardano artifact as JSON byte array.
func (c CardanoArtifact) GetBytesJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

type Certificate = CardanoArtifact
type PlutusScript = CardanoArtifact
