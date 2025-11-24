package sendtx

import cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"

type IUtxosTransformer interface {
	TransformUtxos(utxos []cardanowallet.Utxo) []cardanowallet.Utxo
}

type BridgingType byte

const (
	BridgingTypeNormal BridgingType = iota
	BridgingTypeNativeTokenOnSource
	BridgingTypeCurrencyOnSource

	defaultPotentialFee     = 400_000
	defaultMaxInputsPerTx   = 50
	defaultTTLSlotNumberInc = 500
)

type TokenExchangeConfig struct {
	// Destination chain ID
	DstChainID string `json:"dstChainID"`
	// Token identifier in the format "policyId.name"
	TokenName string `json:"tokenName"`
	// Indicates whether the token is to be minted
	Mint bool `json:"mint"`
}

type ChainConfig struct {
	CardanoCliBinary         string
	TxProvider               cardanowallet.ITxProvider
	MultiSigAddr             string
	TestNetMagic             uint
	TTLSlotNumberInc         uint64
	MinUtxoValue             uint64
	NativeTokens             []TokenExchangeConfig
	DefaultMinFeeForBridging uint64
	MinFeeForBridgingTokens  uint64
	MinOperationFeeAmount    uint64
	PotentialFee             uint64
	ProtocolParameters       []byte
	TreasuryAddress          string
}

type BridgingTxReceiver struct {
	BridgingType BridgingType `json:"type"`
	Addr         string       `json:"addr"`
	Amount       uint64       `json:"amount"`
}

type TxInfo struct {
	TxRaw               []byte
	TxHash              string
	ChangeMinUtxoAmount uint64
	ChosenInputs        cardanowallet.TxInputs
}

type TxFeeInfo struct {
	Fee                 uint64
	ChangeMinUtxoAmount uint64
}

type bridgingTxPreparedData struct {
	TxBuilder          *cardanowallet.TxBuilder
	OutputLovelace     uint64
	OutputNativeTokens []cardanowallet.TokenAmount
	BridgingAddress    string
	BridgingFee        uint64
}

type txBuilderPopulationData struct {
	ChangeLovelace      uint64
	ChangeMinUtxoAmount uint64
	ChosenInputs        cardanowallet.TxInputs
}

type BridgingTxDto struct {
	SrcChainID             string
	DstChainID             string
	SenderAddr             string
	SenderAddrPolicyScript *cardanowallet.PolicyScript
	Receivers              []BridgingTxReceiver
	BridgingAddress        string
	BridgingFee            uint64
	OperationFee           uint64
}

type TxReceiversDto struct {
	Addr         string
	Amount       uint64
	NativeTokens []cardanowallet.TokenAmount
}

type GenericTxDto struct {
	SrcChainID             string
	SenderAddr             string
	SenderAddrPolicyScript *cardanowallet.PolicyScript
	Metadata               []byte
	Receivers              []TxReceiversDto
	OperationFee           uint64
}

func (bt BridgingType) String() string {
	switch bt {
	case BridgingTypeNormal:
		return "Bridging Request Reactor"
	case BridgingTypeNativeTokenOnSource:
		return "Bridging Native Token on Source"
	case BridgingTypeCurrencyOnSource:
		return "Bridging Currency on Source"
	default:
		return "Unknown Bridging Type"
	}
}
