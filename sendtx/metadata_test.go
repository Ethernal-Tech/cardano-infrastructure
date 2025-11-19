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
				Address:      common.SplitString("ffa00021", splitStringLength),
				BridgingType: BridgingTypeWrappedTokenOnSource,
				Amount:       420,
			},
			{
				Address:      common.SplitString("ffa00055a", splitStringLength),
				BridgingType: BridgingTypeCurrencyOnSource,
				Amount:       150,
			},
			{
				Address:      common.SplitString("ffa00022", splitStringLength),
				BridgingType: BridgingTypeWrappedTokenOnSource,
				Amount:       220,
			},
			{
				Address:       common.SplitString("ffa00023", splitStringLength),
				BridgingType:  BridgingTypeColoredCoinOnSource,
				ColoredCoinID: 1,
				Amount:        20,
			},
			{
				Address:       common.SplitString("ffa00024", splitStringLength),
				BridgingType:  BridgingTypeColoredCoinOnSource,
				ColoredCoinID: 1,
				Amount:        320,
			},
			{
				Address:       common.SplitString("ffa00024", splitStringLength),
				BridgingType:  BridgingTypeColoredCoinOnSource,
				ColoredCoinID: 2,
				Amount:        300,
			},
		},
	}

	outputAmounts := metadata.GetOutputAmounts()

	assert.Equal(t, uint64(120+200+200+150), outputAmounts.CurrencyLovelace)
	assert.Equal(t, uint64(420+220), outputAmounts.WrappedTokens)
	assert.Equal(t, uint64(20+320), outputAmounts.ColoredCoins[1])
	assert.Equal(t, uint64(300), outputAmounts.ColoredCoins[2])
}

func TestMetaDataMarshal(t *testing.T) {
	metadata := &BridgingRequestMetadata{}

	bytes, err := metadata.Marshal()

	require.NoError(t, err)
	require.True(t, len(bytes) > 0)
}
