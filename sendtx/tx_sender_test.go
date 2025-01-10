package sendtx

import (
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMetaData(t *testing.T) {
	const (
		bridgingFeeAmount = uint64(110)
		senderAddr        = "addr1_xghghg3sdss"
	)

	configs := map[string]ChainConfig{
		"prime": {
			MinUtxoValue: 55,
			ExchangeRate: map[string]float64{
				"vector": 2.0,
			},
		},
		"vector": {
			MinUtxoValue: 20,
			ExchangeRate: map[string]float64{
				"prime": 0.5,
			},
		},
	}

	t.Run("valid", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(50), uint64(0), 0, configs)

		metadata, err := txSnd.CreateMetadata(senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNormal,
				Addr:         "addr1_aa",
				Amount:       uint64(100),
			},
			{
				BridgingType: BridgingTypeCurrencyOnSource,
				Addr:         "addr1_ab",
				Amount:       uint64(61),
			},
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Addr:         "addr1_ac",
				Amount:       uint64(33),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, common.SplitString(senderAddr, splitStringLength), metadata.SenderAddr)
		assert.Equal(t, bridgingMetaDataType, metadata.BridgingTxType)
		assert.Equal(t, "vector", metadata.DestinationChainID)
		assert.Equal(t, BridgingRequestMetadataCurrencyInfo{
			SrcAmount:  55,
			DestAmount: 110,
		}, metadata.FeeAmount)
		assert.Equal(t, []BridgingRequestMetadataTransaction{
			{
				Address: common.SplitString("addr1_aa", splitStringLength),
				Amount:  uint64(100),
			},
			{
				Address: common.SplitString("addr1_ab", splitStringLength),
				Amount:  uint64(61),
				Additional: &BridgingRequestMetadataCurrencyInfo{
					SrcAmount:  10,
					DestAmount: 20,
				},
			},
			{
				Address:            common.SplitString("addr1_ac", splitStringLength),
				IsNativeTokenOnSrc: metadataBoolTrue,
				Amount:             33,
			},
		}, metadata.Transactions)
	})

	t.Run("valid 2", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(50), uint64(0), 0, map[string]ChainConfig{
			"prime": {
				MinUtxoValue: 550,
				ExchangeRate: map[string]float64{
					"vector": 2.0,
				},
			},
			"vector": {
				MinUtxoValue: 200,
				ExchangeRate: map[string]float64{
					"prime": 0.5,
				},
			},
		})

		metadata, err := txSnd.CreateMetadata(senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Addr:         "addr1_ab",
				Amount:       uint64(200),
			},
		})

		require.NoError(t, err)
		assert.Equal(t, common.SplitString(senderAddr, splitStringLength), metadata.SenderAddr)
		assert.Equal(t, bridgingMetaDataType, metadata.BridgingTxType)
		assert.Equal(t, "vector", metadata.DestinationChainID)
		assert.Equal(t, BridgingRequestMetadataCurrencyInfo{
			SrcAmount:  550,
			DestAmount: 1100,
		}, metadata.FeeAmount)
		assert.Equal(t, []BridgingRequestMetadataTransaction{
			{
				Address:            common.SplitString("addr1_ab", splitStringLength),
				IsNativeTokenOnSrc: metadataBoolTrue,
				Amount:             200,
			},
		}, metadata.Transactions)
	})

	t.Run("invalid destination", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(50), uint64(0), 0, configs)

		_, err := txSnd.CreateMetadata(senderAddr, "prime", "vector1", []BridgingTxReceiver{})
		require.ErrorContains(t, err, "destination chain ")
	})

	t.Run("invalid source", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(50), uint64(0), 0, configs)

		_, err := txSnd.CreateMetadata(senderAddr, "prime1", "vector", []BridgingTxReceiver{})
		require.ErrorContains(t, err, "source chain ")
	})

	t.Run("invalid amount native token on source", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(50), uint64(0), 0, configs)

		_, err := txSnd.CreateMetadata(senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Amount:       19,
			},
		})
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid amount currency on source", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(50), uint64(0), 0, configs)

		_, err := txSnd.CreateMetadata(senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeCurrencyOnSource,
				Amount:       9,
			},
		})
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid amount reactor", func(t *testing.T) {
		txSnd := NewTxSender(bridgingFeeAmount, uint64(190), uint64(0), 0, configs)

		_, err := txSnd.CreateMetadata(senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNormal,
				Amount:       189,
			},
		})
		require.ErrorContains(t, err, "amount for receiver ")
	})
}
