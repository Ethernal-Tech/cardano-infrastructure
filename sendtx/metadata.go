package sendtx

import (
	"encoding/json"
	"fmt"
	"strings"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
)

type BridgingRequestType string

const (
	metadataMapKey                           = 1
	metadataBoolTrue                         = 1
	bridgingMetaDataType BridgingRequestType = "bridge"

	splitStringLength = 40
)

// BridgingRequestMetadataTransaction represents a single transaction in a bridging request.
// IsNativeTokenOnSrc is true if the user is bridging native tokens (e.g., WSADA, WSAPEX) ...
// ... and false if bridging native currency (e.g., ADA, APEX).
type BridgingRequestMetadataTransaction struct {
	Address            []string `cbor:"a" json:"a"`
	IsNativeTokenOnSrc byte     `cbor:"nt" json:"nt"` // bool is not supported by Cardano!
	Amount             uint64   `cbor:"m" json:"m"`
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

// GetOutputAmounts returns amount needed for outputs in lovelace and native tokens
func (brm *BridgingRequestMetadata) GetOutputAmounts() (outputCurrencyLovelace uint64, outputNativeToken uint64) {
	outputCurrencyLovelace = brm.BridgingFee + brm.OperationFee

	for _, x := range brm.Transactions {
		if x.IsNativeTokenOnSource() {
			outputNativeToken += x.Amount // WSADA/WSAPEX to ADA/APEX
		} else {
			outputCurrencyLovelace += x.Amount // ADA/APEX to WSADA/WSAPEX or Reactor tokens
		}
	}

	return outputCurrencyLovelace, outputNativeToken
}

func (brmt BridgingRequestMetadataTransaction) IsNativeTokenOnSource() bool {
	return brmt.IsNativeTokenOnSrc != 0
}

func addrToMetaDataAddr(addr string) []string {
	addr = strings.TrimPrefix(strings.TrimPrefix(addr, "0x"), "0X")

	return infracommon.SplitString(addr, splitStringLength)
}

// GetOutputAmounts returns amount needed for outputs in lovelace and native tokens
func getOutputAmounts(receivers []BridgingTxReceiver) (outputCurrencyLovelace uint64, outputNativeToken uint64) {
	for _, x := range receivers {
		if x.BridgingType == BridgingTypeNativeTokenOnSource {
			outputNativeToken += x.Amount // WSADA/WSAPEX to ADA/APEX
		} else {
			outputCurrencyLovelace += x.Amount // ADA/APEX to WSADA/WSAPEX or Reactor tokens
		}
	}

	return outputCurrencyLovelace, outputNativeToken
}
