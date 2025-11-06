package sendtx

import (
	"encoding/json"
	"fmt"
)

type BridgingRequestType string

const (
	metadataMapKey                           = 1
	bridgingMetaDataType BridgingRequestType = "bridge"

	splitStringLength = 40
)

// BridgingRequestMetadataTransaction represents a single transaction in a bridging request.
type BridgingRequestMetadataTransaction struct {
	Address []string `cbor:"a" json:"a"`
	Amount  uint64   `cbor:"m" json:"m"`
}

// BridgingRequestMetadata represents metadata for a bridging request
// BridgingFee fee for fee multisig address
// OperationFee fee for bridging currency to native tokens and other operational costs
type BridgingRequestMetadata struct {
	BridgingTxType     BridgingRequestType                  `cbor:"t" json:"t"`
	DestinationChainID string                               `cbor:"d" json:"d"`
	SenderAddr         []string                             `cbor:"s" json:"s"`
	Transactions       []BridgingRequestMetadataTransaction `cbor:"tx" json:"tx"`
	BridgingFee        uint64                               `cbor:"fa" json:"fa"`
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
func (brm *BridgingRequestMetadata) GetOutputAmounts() (outputCurrencyLovelace uint64) {
	outputCurrencyLovelace = brm.BridgingFee

	for _, x := range brm.Transactions {
		outputCurrencyLovelace += x.Amount
	}

	return outputCurrencyLovelace
}
