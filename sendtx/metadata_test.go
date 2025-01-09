package sendtx

import (
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/stretchr/testify/assert"
)

func TestGetOutputAmounts(t *testing.T) {
	metadata := &BridgingRequestMetadata{
		FeeAmount: BridgingRequestMetadataCurrencyInfo{
			SrcAmount:  uint64(110),
			DestAmount: uint64(127),
		},
		Transactions: []BridgingRequestMetadataTransaction{
			{
				Address: common.SplitString("ffa000", splitStringLength),
				Amount:  200,
			},
			{
				Address:            common.SplitString("ffa00021", splitStringLength),
				IsNativeTokenOnSrc: metadataBoolTrue,
				Amount:             420,
			},
			{
				Address: common.SplitString("ffa00055a", splitStringLength),
				Amount:  150,
				Additional: &BridgingRequestMetadataCurrencyInfo{
					DestAmount: uint64(301),
					SrcAmount:  uint64(20),
				},
			},
		},
	}

	v1, v2 := GetOutputAmounts(metadata)

	assert.Equal(t, uint64(110+200+20+150), v1)
	assert.Equal(t, uint64(420), v2)
}
