package wallet

import "context"

const (
	demeterAuthHeaderKey = "dmtr-api-key"
)

type TxProviderDemeter struct {
	TxProviderBlockFrost
	submitAPIURL string
	submitAPIKey string
}

var _ ITxProvider = (*TxProviderDemeter)(nil)

func NewTxProviderDemeter(
	blockfrostURL, blockfrostAPIKey, submitAPIURL, submitAPIKey string,
) *TxProviderDemeter {
	blockfrost := NewTxProviderBlockFrost(blockfrostURL, blockfrostAPIKey)
	blockfrost.authHeaderKey = demeterAuthHeaderKey

	return &TxProviderDemeter{
		TxProviderBlockFrost: *blockfrost,
		submitAPIURL:         submitAPIURL,
		submitAPIKey:         submitAPIKey,
	}
}

func (b *TxProviderDemeter) SubmitTx(ctx context.Context, txSigned []byte) error {
	return blockfrostSubmitTx(ctx, b.submitAPIURL+"/api/submit/tx", b.authHeaderKey, b.submitAPIKey, txSigned)
}
