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

type OutputAmounts struct {
	CurrencyLovelace uint64
	NativeTokens     map[uint16]uint64
}

// BridgingRequestMetadataTransaction represents a single transaction in a bridging request.
// IsNativeTokenOnSrc is true if the user is bridging native tokens (e.g., WSADA, WSAPEX) ...
// ... and false if bridging native currency (e.g., ADA, APEX).
type BridgingRequestMetadataTransaction struct {
	Address []string `cbor:"a" json:"a"`
	Amount  uint64   `cbor:"m" json:"m"`
	Token   uint16   `cbor:"t" json:"t"`
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
	OperationFee       uint64                               `cbor:"of" json:"of"`
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

// GetOutputAmounts returns the required output amounts in lovelace, wrapped tokens, and colored coins.
func (brm *BridgingRequestMetadata) GetOutputAmounts(currencyID uint16) OutputAmounts {
	amounts := OutputAmounts{
		CurrencyLovelace: brm.BridgingFee + brm.OperationFee,
		NativeTokens:     make(map[uint16]uint64),
	}

	for _, tx := range brm.Transactions {
		if tx.Token == currencyID {
			amounts.CurrencyLovelace += tx.Amount
		} else {
			amounts.NativeTokens[tx.Token] += tx.Amount
		}
	}

	return amounts
}
