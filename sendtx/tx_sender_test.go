package sendtx

import (
	"encoding/hex"
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMetaData(t *testing.T) {
	const (
		bridgingFeeAmount  = uint64(110)
		operationFeeAmount = uint64(50)
		senderAddr         = "addr1_xghghg3sdss"
	)

	primeCfg := &ChainConfig{
		MinUtxoValue: 55,
	}
	vectorCfg := &ChainConfig{
		MinUtxoValue: 20,
	}
	configs := map[string]ChainConfig{
		"prime":  *primeCfg,
		"vector": *vectorCfg,
	}

	t.Run("valid", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		metadata, err := txSnd.CreateMetadata(
			senderAddr, "prime", "vector", []BridgingTxReceiver{
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
			}, bridgingFeeAmount, operationFeeAmount)

		require.NoError(t, err)
		assert.Equal(t, common.SplitString(senderAddr, splitStringLength), metadata.SenderAddr)
		assert.Equal(t, bridgingMetaDataType, metadata.BridgingTxType)
		assert.Equal(t, "vector", metadata.DestinationChainID)
		assert.Equal(t, bridgingFeeAmount, metadata.BridgingFee)
		assert.Equal(t, operationFeeAmount, metadata.OperationFee)
		assert.Equal(t, []BridgingRequestMetadataTransaction{
			{
				Address: common.SplitString("addr1_aa", splitStringLength),
				Amount:  uint64(100),
			},
			{
				Address: common.SplitString("addr1_ab", splitStringLength),
				Amount:  uint64(61),
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

		metadata, err := txSnd.CreateMetadata(
			senderAddr, "prime", "vector", []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeNativeTokenOnSource,
					Addr:         "addr1_ab",
					Amount:       uint64(200),
				},
			}, bridgingFeeAmount, operationFeeAmount)

		require.NoError(t, err)
		assert.Equal(t, common.SplitString(senderAddr, splitStringLength), metadata.SenderAddr)
		assert.Equal(t, bridgingMetaDataType, metadata.BridgingTxType)
		assert.Equal(t, "vector", metadata.DestinationChainID)
		assert.Equal(t, bridgingFeeAmount, metadata.BridgingFee)
		assert.Equal(t, operationFeeAmount, metadata.OperationFee)
		assert.Equal(t, []BridgingRequestMetadataTransaction{
			{
				Address:            common.SplitString("addr1_ab", splitStringLength),
				IsNativeTokenOnSrc: metadataBoolTrue,
				Amount:             200,
			},
		}, metadata.Transactions)
	})

	t.Run("invalid amount native token on source", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(
			senderAddr, "prime", "vector", []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeNativeTokenOnSource,
					Amount:       19,
				},
			}, bridgingFeeAmount, operationFeeAmount)
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid amount currency on source", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		_, err := txSnd.CreateMetadata(
			senderAddr, "prime", "vector", []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeCurrencyOnSource,
					Amount:       9,
				},
			}, bridgingFeeAmount, operationFeeAmount)
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid source", func(t *testing.T) {
		_, err := NewTxSender(configs).CreateMetadata(
			senderAddr, "prime1", "vector", []BridgingTxReceiver{}, bridgingFeeAmount, operationFeeAmount)

		require.ErrorContains(t, err, "source")
	})

	t.Run("invalid source", func(t *testing.T) {
		_, err := NewTxSender(configs).CreateMetadata(
			senderAddr, "prime", "vector2", []BridgingTxReceiver{}, bridgingFeeAmount, operationFeeAmount)

		require.ErrorContains(t, err, "destination")
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

		_, err := txSnd.CreateMetadata(
			senderAddr, "prime", "vector", []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeNormal,
					Amount:       189,
				},
			}, bridgingFeeAmount, operationFeeAmount)
		require.ErrorContains(t, err, "amount for receiver ")
	})
}

func Test_checkFees(t *testing.T) {
	const (
		bridgingFeeAmount  = 1_000_005
		operationFeeAmount = 34
	)

	cfg := &ChainConfig{
		MinBridgingFeeAmount:  bridgingFeeAmount,
		MinOperationFeeAmount: operationFeeAmount,
	}

	t.Run("invalid bridging fee", func(t *testing.T) {
		err := checkFees(cfg, bridgingFeeAmount-1, operationFeeAmount)
		require.ErrorContains(t, err, "bridging fee")
	})

	t.Run("invalid operation fee", func(t *testing.T) {
		err := checkFees(cfg, bridgingFeeAmount, operationFeeAmount-1)
		require.ErrorContains(t, err, "operation fee")
	})

	t.Run("good", func(t *testing.T) {
		err := checkFees(cfg, bridgingFeeAmount, operationFeeAmount)
		require.NoError(t, err)
	})
}

func Test_getOutputAmounts(t *testing.T) {
	lovelace, nativeTokens := getOutputAmounts([]BridgingTxReceiver{
		{
			BridgingType: BridgingTypeCurrencyOnSource,
			Amount:       1,
		},
		{
			BridgingType: BridgingTypeNativeTokenOnSource,
			Amount:       2,
		},
		{
			BridgingType: BridgingTypeCurrencyOnSource,
			Amount:       3,
		},
		{
			BridgingType: BridgingTypeNormal,
			Amount:       4,
		},
		{
			BridgingType: BridgingTypeNativeTokenOnSource,
			Amount:       5,
		},
	})

	assert.Equal(t, uint64(8), lovelace)
	assert.Equal(t, uint64(7), nativeTokens)
}

func TestGetTokenFromTokenExchangeConfig(t *testing.T) {
	cfg := []TokenExchangeConfig{
		{
			DstChainID: "prime",
			TokenName:  "pid.ffaabb",
		},
		{
			DstChainID: "nexus",
			TokenName:  "pid.roko",
		},
		{
			DstChainID: "vector",
			TokenName:  "pidffaabb",
		},
	}

	token, err := GetTokenFromTokenExchangeConfig(cfg, "prime")

	require.NoError(t, err)
	assert.Equal(t, cfg[0].TokenName, token.String())

	token, err = GetTokenFromTokenExchangeConfig(cfg, "nexus")

	require.NoError(t, err)
	assert.Equal(t, "pid."+hex.EncodeToString([]byte("roko")), token.String())

	_, err = GetTokenFromTokenExchangeConfig(cfg, "vector")

	assert.Error(t, err)
}
