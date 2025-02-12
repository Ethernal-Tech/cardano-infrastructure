package sendtx

import (
	"context"
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
			MinBridgingFeeAmount: bridgingFeeAmount,
			MinUtxoValue:         55,
		},
		"vector": {
			MinBridgingFeeAmount: bridgingFeeAmount,
			MinUtxoValue:         20,
		},
	}

	exchangeRate := NewExchangeRate(NewExchangeRateEntry("prime", "vector", 2.0))

	t.Run("valid", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		metadata, err := txSnd.CreateMetadata(context.Background(), senderAddr, "prime", "vector", []BridgingTxReceiver{
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
		}, bridgingFeeAmount, exchangeRate)

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
		txSnd := NewTxSender(map[string]ChainConfig{
			"prime": {
				MinBridgingFeeAmount: bridgingFeeAmount,
				MinUtxoValue:         550,
			},
			"vector": {
				MinBridgingFeeAmount: bridgingFeeAmount,
				MinUtxoValue:         200,
			},
		})

		metadata, err := txSnd.CreateMetadata(context.Background(), senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Addr:         "addr1_ab",
				Amount:       uint64(200),
			},
		}, bridgingFeeAmount, exchangeRate)

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
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(
			context.Background(), senderAddr, "prime", "vector1", []BridgingTxReceiver{}, bridgingFeeAmount, exchangeRate)
		require.ErrorContains(t, err, "destination chain ")
	})

	t.Run("invalid source", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(
			context.Background(), senderAddr, "prime1", "vector", []BridgingTxReceiver{}, bridgingFeeAmount, exchangeRate)
		require.ErrorContains(t, err, "source chain ")
	})

	t.Run("invalid amount native token on source", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(context.Background(), senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Amount:       19,
			},
		}, bridgingFeeAmount, exchangeRate)
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid amount currency on source", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(context.Background(), senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeCurrencyOnSource,
				Amount:       9,
			},
		}, bridgingFeeAmount, exchangeRate)
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid bridging fee", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(context.Background(), senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeCurrencyOnSource,
				Amount:       9,
			},
		}, bridgingFeeAmount-1, exchangeRate)
		require.ErrorContains(t, err, "bridging fee")
	})

	t.Run("invalid amount reactor", func(t *testing.T) {
		txSnd := NewTxSender(map[string]ChainConfig{
			"prime": {
				MinBridgingFeeAmount: bridgingFeeAmount,
				MinUtxoValue:         190,
			},
			"vector": {
				MinBridgingFeeAmount: bridgingFeeAmount,
				MinUtxoValue:         20,
			},
		})

		_, err := txSnd.CreateMetadata(context.Background(), senderAddr, "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeNormal,
				Amount:       189,
			},
		}, bridgingFeeAmount, exchangeRate)
		require.ErrorContains(t, err, "amount for receiver ")
	})
}
