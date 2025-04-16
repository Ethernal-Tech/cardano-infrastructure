package sendtx

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/common"
	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dummyAddr = "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u"
	dummyPID  = "29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8"
)

var (
	dummyProtoParams = []byte(`{"costModels":{"PlutusV1":[197209,0,1,1,396231,621,0,1,150000,1000,0,1,150000,32,2477736,29175,4,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,100,100,29773,100,150000,32,150000,32,150000,32,150000,1000,0,1,150000,32,150000,1000,0,8,148000,425507,118,0,1,1,150000,1000,0,8,150000,112536,247,1,150000,10000,1,136542,1326,1,1000,150000,1000,1,150000,32,150000,32,150000,32,1,1,150000,1,150000,4,103599,248,1,103599,248,1,145276,1366,1,179690,497,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,148000,425507,118,0,1,1,61516,11218,0,1,150000,32,148000,425507,118,0,1,1,148000,425507,118,0,1,1,2477736,29175,4,0,82363,4,150000,5000,0,1,150000,32,197209,0,1,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,3345831,1,1],"PlutusV2":[205665,812,1,1,1000,571,0,1,1000,24177,4,1,1000,32,117366,10475,4,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,100,100,23000,100,19537,32,175354,32,46417,4,221973,511,0,1,89141,32,497525,14068,4,2,196500,453240,220,0,1,1,1000,28662,4,2,245000,216773,62,1,1060367,12586,1,208512,421,1,187000,1000,52998,1,80436,32,43249,32,1000,32,80556,1,57667,4,1000,10,197145,156,1,197145,156,1,204924,473,1,208896,511,1,52467,32,64832,32,65493,32,22558,32,16563,32,76511,32,196500,453240,220,0,1,1,69522,11687,0,1,60091,32,196500,453240,220,0,1,1,196500,453240,220,0,1,1,1159724,392670,0,2,806990,30482,4,1927926,82523,4,265318,0,4,0,85931,32,205665,812,1,1,41182,32,212342,32,31220,32,32696,32,43357,32,32247,32,38314,32,35892428,10,9462713,1021,10,38887044,32947,10]},"protocolVersion":{"major":7,"minor":0},"maxBlockHeaderSize":1100,"maxBlockBodySize":65536,"maxTxSize":16384,"txFeeFixed":155381,"txFeePerByte":44,"stakeAddressDeposit":0,"stakePoolDeposit":0,"minPoolCost":0,"poolRetireMaxEpoch":18,"stakePoolTargetNum":100,"poolPledgeInfluence":0,"monetaryExpansion":0.1,"treasuryCut":0.1,"collateralPercentage":150,"executionUnitPrices":{"priceMemory":0.0577,"priceSteps":0.0000721},"utxoCostPerByte":4310,"maxTxExecutionUnits":{"memory":16000000,"steps":10000000000},"maxBlockExecutionUnits":{"memory":80000000,"steps":40000000000},"maxCollateralInputs":3,"maxValueSize":5000,"extraPraosEntropy":null,"decentralization":null,"minUTxOValue":null}`)
)

func TestCreateMetaData(t *testing.T) {
	const (
		bridgingFeeAmount  = uint64(110)
		operationFeeAmount = uint64(50)
	)

	configs := map[string]ChainConfig{
		"prime": {
			MinUtxoValue: 55,
		},
		"vector": {
			MinUtxoValue: 20,
		},
	}

	t.Run("valid", func(t *testing.T) {
		txSnd := NewTxSender(configs)

		metadata, err := txSnd.CreateMetadata(
			dummyAddr, "prime", "vector", []BridgingTxReceiver{
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
		assert.Equal(t, common.SplitString(dummyAddr, splitStringLength), metadata.SenderAddr)
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
			dummyAddr, "prime", "vector", []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeNativeTokenOnSource,
					Addr:         "addr1_ab",
					Amount:       uint64(200),
				},
			}, bridgingFeeAmount, operationFeeAmount)

		require.NoError(t, err)
		assert.Equal(t, common.SplitString(dummyAddr, splitStringLength), metadata.SenderAddr)
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
			dummyAddr, "prime", "vector", []BridgingTxReceiver{
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
			dummyAddr, "prime", "vector", []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeCurrencyOnSource,
					Amount:       9,
				},
			}, bridgingFeeAmount, operationFeeAmount)
		require.ErrorContains(t, err, "amount for receiver ")
	})

	t.Run("invalid source", func(t *testing.T) {
		_, err := NewTxSender(configs).CreateMetadata(
			dummyAddr, "prime1", "vector", []BridgingTxReceiver{}, bridgingFeeAmount, operationFeeAmount)

		require.ErrorContains(t, err, "source")
	})

	t.Run("invalid source", func(t *testing.T) {
		_, err := NewTxSender(configs).CreateMetadata(
			dummyAddr, "prime", "vector2", []BridgingTxReceiver{}, bridgingFeeAmount, operationFeeAmount)

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
			dummyAddr, "prime", "vector", []BridgingTxReceiver{
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
		require.ErrorContains(t, checkFees(cfg, bridgingFeeAmount-1, operationFeeAmount), "bridging fee")
	})

	t.Run("invalid operation fee", func(t *testing.T) {
		require.ErrorContains(t, checkFees(cfg, bridgingFeeAmount, operationFeeAmount-1), "operation fee")
	})

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, checkFees(cfg, bridgingFeeAmount, operationFeeAmount))
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

	token, err := getTokenFromTokenExchangeConfig(cfg, "prime")

	require.NoError(t, err)
	assert.Equal(t, cfg[0].TokenName, token.String())

	token, err = getTokenFromTokenExchangeConfig(cfg, "nexus")

	require.NoError(t, err)
	assert.Equal(t, "pid."+hex.EncodeToString([]byte("roko")), token.String())

	_, err = getTokenFromTokenExchangeConfig(cfg, "vector")

	assert.Error(t, err)
}

func Test_prepareBridgingTx(t *testing.T) {
	const bridgingFee = uint64(100)

	token := cardanowallet.NewToken(dummyPID, "WADA")
	txProviderMock := &txProviderMock{
		protocolParameters: dummyProtoParams,
	}
	txSnd := NewTxSender(map[string]ChainConfig{
		"prime": {
			MinUtxoValue: 55,
			TestNetMagic: cardanowallet.PreviewProtocolMagic,
			MultiSigAddr: "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			NativeTokens: []TokenExchangeConfig{
				{
					DstChainID: "vector",
					TokenName:  token.String(),
				},
			},
			TxProvider:       txProviderMock,
			CardanoCliBinary: cardanowallet.ResolveCardanoCliBinary(cardanowallet.TestNetNetwork),
		},
		"vector": {
			MinUtxoValue: 20,
		},
	})

	t.Run("valid", func(t *testing.T) {
		data, err := txSnd.prepareBridgingTx(context.Background(), "prime", "vector", []BridgingTxReceiver{
			{
				BridgingType: BridgingTypeCurrencyOnSource,
				Amount:       500_000,
			},
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Amount:       600_000,
			},
			{
				BridgingType: BridgingTypeCurrencyOnSource,
				Amount:       500_001,
			},
			{
				BridgingType: BridgingTypeNativeTokenOnSource,
				Amount:       600_003,
			},
		}, bridgingFee, 0)

		require.NoError(t, err)
		require.NotNil(t, data.TxBuilder)
		assert.NotNil(t, data.SrcConfig)

		defer data.TxBuilder.Dispose()

		expectedTokenAmount := cardanowallet.NewTokenAmount(token, 600_000*2+3)

		assert.Equal(t, uint64(1034400), data.OutputLovelace)
		assert.Equal(t, uint64(1034400-1_000_001), data.BridgingFee)
		assert.Equal(t, &expectedTokenAmount, data.OutputNativeToken)
	})
}

func Test_adjustLovelaceOutput(t *testing.T) {
	txBuilder, err := cardanowallet.NewTxBuilder(cardanowallet.ResolveCardanoCliBinary(cardanowallet.TestNetNetwork))
	require.NoError(t, err)

	defer txBuilder.Dispose()

	txBuilder.SetProtocolParameters(dummyProtoParams)

	t.Run("without native token case 1", func(t *testing.T) {
		v, err := adjustLovelaceOutput(txBuilder, dummyAddr, nil, 1_000_000, 1_000_001)

		require.NoError(t, err)
		require.Equal(t, uint64(1_000_001), v)
	})

	t.Run("without native token case 2", func(t *testing.T) {
		v, err := adjustLovelaceOutput(txBuilder, dummyAddr, nil, 1_000_002, 1_000_001)

		require.NoError(t, err)
		require.Equal(t, uint64(1_000_002), v)
	})

	t.Run("with native token", func(t *testing.T) {
		v, err := adjustLovelaceOutput(txBuilder, dummyAddr, &cardanowallet.TokenAmount{
			Token:  cardanowallet.NewToken(dummyPID, "WADAorWAPEX"),
			Amount: 1_000_000_000_000,
		}, 1_000_002, 1_000_001)

		require.NoError(t, err)
		require.Equal(t, uint64(1_081_810), v)
	})
}

type txProviderMock struct {
	protocolParameters []byte
}

func (m *txProviderMock) Dispose() {
}

func (m *txProviderMock) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	return m.protocolParameters, nil
}

func (m *txProviderMock) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *txProviderMock) GetTip(ctx context.Context) (cardanowallet.QueryTipData, error) {
	return cardanowallet.QueryTipData{}, nil
}

func (m *txProviderMock) GetUtxos(ctx context.Context, addr string) ([]cardanowallet.Utxo, error) {
	return nil, nil
}

func (m *txProviderMock) SubmitTx(ctx context.Context, txSigned []byte) error {
	return nil
}
