package sendtx

import cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"

type ChainConfig struct {
	CardanoCliBinary   string
	TxProvider         cardanowallet.ITxProvider
	MultiSigAddr       string
	TestNetMagic       uint
	TTLSlotNumberInc   uint64
	MinUtxoValue       uint64
	NativeToken        cardanowallet.Token
	BridgingFeeAmount  uint64
	ProtocolParameters []byte
}
