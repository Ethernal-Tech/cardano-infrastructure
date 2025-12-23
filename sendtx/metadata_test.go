package sendtx

import (
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaDataGetOutputAmounts(t *testing.T) {
	currencyID := uint16(1)
	wrappedTokenID := uint16(2)
	coloredCoinID1 := uint16(3)
	coloredCoinID2 := uint16(4)

	metadata := &BridgingRequestMetadata{
		BridgingFee:  120,
		OperationFee: 200,
		Transactions: []BridgingRequestMetadataTransaction{
			{
				Address: common.SplitString("ffa000", splitStringLength),
				Amount:  200,
				TokenID: currencyID,
			},
			{
				Address: common.SplitString("ffa00021", splitStringLength),
				Amount:  420,
				TokenID: wrappedTokenID,
			},
			{
				Address: common.SplitString("ffa00055a", splitStringLength),
				Amount:  150,
				TokenID: currencyID,
			},
			{
				Address: common.SplitString("ffa00022", splitStringLength),
				Amount:  220,
				TokenID: wrappedTokenID,
			},
			{
				Address: common.SplitString("ffa00023", splitStringLength),
				Amount:  20,
				TokenID: coloredCoinID1,
			},
			{
				Address: common.SplitString("ffa00024", splitStringLength),
				Amount:  320,
				TokenID: coloredCoinID1,
			},
			{
				Address: common.SplitString("ffa00024", splitStringLength),
				Amount:  300,
				TokenID: coloredCoinID2,
			},
		},
	}

	outputAmounts := metadata.GetOutputAmounts(currencyID)

	assert.Equal(t, uint64(120+200+200+150), outputAmounts.CurrencyLovelace)
	assert.Equal(t, uint64(420+220), outputAmounts.NativeTokens[wrappedTokenID])
	assert.Equal(t, uint64(20+320), outputAmounts.NativeTokens[coloredCoinID1])
	assert.Equal(t, uint64(300), outputAmounts.NativeTokens[coloredCoinID2])
}

func TestMetaDataMarshal(t *testing.T) {
	metadata := &BridgingRequestMetadata{}

	bytes, err := metadata.Marshal()

	require.NoError(t, err)
	require.True(t, len(bytes) > 0)
}
