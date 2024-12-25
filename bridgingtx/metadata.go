package bridgingtx

import (
	"encoding/json"
	"fmt"
)

type BridgingRequestType string

const (
	metadataMapKey                           = 1
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
	IsNativeTokenOnSrc bool                                 `cbor:"nt" json:"nt"`
	Amount             uint64                               `cbor:"m" json:"m"`
	Additional         *BridgingRequestMetadataCurrencyInfo `cbor:"ad" json:"ad"`
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
