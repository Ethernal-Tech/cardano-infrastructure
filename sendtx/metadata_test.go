package sendtx

import (
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaDataGetOutputAmounts(t *testing.T) {
	metadata := &BridgingRequestMetadata{
		BridgingFee:  120,
		OperationFee: 200,
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
			},
		},
	}

	v1, v2 := metadata.GetOutputAmounts()

	assert.Equal(t, uint64(120+200+200+150), v1)
	assert.Equal(t, uint64(420), v2)
}

func TestMetaDataMarshal(t *testing.T) {
	metadata := &BridgingRequestMetadata{}

	bytes, err := metadata.Marshal()

	require.NoError(t, err)
	require.True(t, len(bytes) > 0)
}
