package wallet

import "context"

type Utxo struct {
	Hash   string `json:"hsh"`
	Index  uint32 `json:"ind"`
	Amount uint64 `json:"amount"`
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

type IStakeSigner interface {
	ISigner
	GetStakeSigningKey() []byte
	GetStakeVerificationKey() []byte
}

type IWallet interface {
	IStakeSigner
	GetKeyHash() string
}

type IWalletManager interface {
	// Create creates new wallet
	Create(directory string, forceCreate bool) (IWallet, error)
	// Load loads wallet
	Load(directory string) (IWallet, error)
}

type IPolicyScript interface {
	GetPolicyScript() []byte
	GetCount() int
}
