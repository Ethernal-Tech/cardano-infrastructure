package sendtx

import (
	"encoding/json"
	"fmt"
)

type BridgingRequestType string

const (
	metadataMapKey                           = 1
	metadataBoolTrue                         = 1
	bridgingMetaDataType BridgingRequestType = "bridge"
)

type BridgingRequestMetadataCurrencyInfo struct {
	SrcAmount  uint64 `cbor:"sa" json:"sa"`
	DestAmount uint64 `cbor:"da" json:"da"`
}

// IsNativeTokenOnSrc - is the user trying to bridge native tokens (WAda, WApex), or native currency (Ada, APEX)
// Additional will be counted towards the bridging fee shown to the user, but it will actually be assigned to
// the user on destination chain, because of the technical limitation for creating utxos on source and destination
// Additional will be nil for reactor
// If source is native currency then:
// Additional.DestAmount is minUtxoAmount on destination chain
// Additional.SrcAmount is Additional.DestAmount * exchangeRate
// If source is native token then:
// Additional.SrcAmount is up to minUtxoAmount on source chain
// Additional.DestAmount is Additional.SrcAmount * exchangeRate
type BridgingRequestMetadataTransaction struct {
	Address            []string                             `cbor:"a" json:"a"`
	IsNativeTokenOnSrc byte                                 `cbor:"nt" json:"nt"` // bool is not supported by cardano!
	Amount             uint64                               `cbor:"m" json:"m"`
	Additional         *BridgingRequestMetadataCurrencyInfo `cbor:"ad,omitempty" json:"ad,omitempty"`
}

// FeeAmount.DestAmount is minBridgingFee on destination chain
// FeeAmount.SrcAmount is FeeAmount.DestAmount * exchangeRate
type BridgingRequestMetadata struct {
	BridgingTxType     BridgingRequestType                  `cbor:"t" json:"t"`
	DestinationChainID string                               `cbor:"d" json:"d"`
	SenderAddr         []string                             `cbor:"s" json:"s"`
	Transactions       []BridgingRequestMetadataTransaction `cbor:"tx" json:"tx"`
	FeeAmount          BridgingRequestMetadataCurrencyInfo  `cbor:"fa" json:"fa"`
}

func (brm BridgingRequestMetadata) Marshal() ([]byte, error) {
	result, err := json.Marshal(map[int]BridgingRequestMetadata{
		metadataMapKey: brm,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %v, err: %w", brm, err)
	}

	return result, nil
}

// GetOutputAmounts returns amount needed for outputs in lovelace and native tokens
func GetOutputAmounts(metadata *BridgingRequestMetadata) (outputCurrencyLovelace uint64, outputNativeToken uint64) {
	outputCurrencyLovelace = metadata.FeeAmount.SrcAmount

	for _, x := range metadata.Transactions {
		if x.IsNativeTokenOnSource() {
			// WADA/WAPEX to ADA/APEX
			outputNativeToken += x.Amount
		} else {
			// ADA/APEX to WADA/WAPEX or reactor
			outputCurrencyLovelace += x.Amount
		}

		if x.Additional != nil {
			outputCurrencyLovelace += x.Additional.SrcAmount
		}
	}

	return outputCurrencyLovelace, outputNativeToken
}

func (brmt BridgingRequestMetadataTransaction) IsNativeTokenOnSource() bool {
	return brmt.IsNativeTokenOnSrc != 0
}
