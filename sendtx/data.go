package sendtx

import (
	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
)

type IUtxosTransformer interface {
	TransformUtxos(utxos []cardanowallet.Utxo) []cardanowallet.Utxo
}

const (
	defaultPotentialFee     = 400_000
	defaultMaxInputsPerTx   = 50
	defaultTTLSlotNumberInc = 500
)

type ApexToken struct {
	FullName          string
	IsWrappedCurrency bool
}

type ChainConfig struct {
	CardanoCliBinary           string
	TxProvider                 cardanowallet.ITxProvider
	MultiSigAddr               string
	TestNetMagic               uint
	TTLSlotNumberInc           uint64
	MinUtxoValue               uint64
	MinColCoinsAllowedToBridge uint64
	Tokens                     map[uint16]ApexToken
	DefaultMinFeeForBridging   uint64
	MinFeeForBridgingTokens    uint64
	MinOperationFeeAmount      uint64
	PotentialFee               uint64
	ProtocolParameters         []byte
}

type BridgingTxReceiver struct {
	Addr    string `json:"addr"`
	Amount  uint64 `json:"amount"`
	TokenID uint16 `json:"token"`
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
}

func (config ChainConfig) GetCurrencyID() (uint16, bool) {
	for id, token := range config.Tokens {
		if token.FullName == cardanowallet.AdaTokenName {
			return id, true
		}
	}

	return 0, false
}
