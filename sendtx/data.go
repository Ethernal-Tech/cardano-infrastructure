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
	DstChainID string `json:"dstChainID"`
	TokenName  string `json:"tokenName"`
}

type ChainConfig struct {
	CardanoCliBinary      string
	TxProvider            cardanowallet.ITxProvider
	MultiSigAddr          string
	TestNetMagic          uint
	TTLSlotNumberInc      uint64
	MinUtxoValue          uint64
	NativeTokens          []TokenExchangeConfig
	MinBridgingFeeAmount  uint64
	MinOperationFeeAmount uint64
	PotentialFee          uint64
	ProtocolParameters    []byte
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
	BridgingFee        uint64
	SrcConfig          *ChainConfig
}

type txBuilderPopulationData struct {
	ChangeLovelace      uint64
	ChangeMinUtxoAmount uint64
	ChosenInputs        cardanowallet.TxInputs
}
