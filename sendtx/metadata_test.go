package sendtx

import (
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaDataGetOutputAmounts(t *testing.T) {
	metadata := &BridgingRequestMetadata{
		BridgingFee: 120,
		Transactions: []BridgingRequestMetadataTransaction{
			{
				Address: common.SplitString("ffa000", splitStringLength),
				Amount:  200,
			},
			{
				Address: common.SplitString("ffa00021", splitStringLength),
				Amount:  420,
			},
			{
				Address: common.SplitString("ffa00055a", splitStringLength),
				Amount:  150,
			},
		},
	}

	amount := metadata.GetOutputAmounts()

	assert.Equal(t, uint64(120+200+420+150), amount)
}

func TestMetaDataMarshal(t *testing.T) {
	metadata := &BridgingRequestMetadata{}

	bytes, err := metadata.Marshal()

	require.NoError(t, err)
	require.True(t, len(bytes) > 0)
}
