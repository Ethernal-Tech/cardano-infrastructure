package sendtx

import (
	"context"
	"encoding/hex"
	"fmt"
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
	dummyProtoParams = []byte(`{"costModels":{"PlutusV1":[197209,0,1,1,396231,621,0,1,150000,1000,0,1,150000,32,2477736,29175,4,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,100,100,29773,100,150000,32,150000,32,150000,32,150000,1000,0,1,150000,32,150000,1000,0,8,148000,425507,118,0,1,1,150000,1000,0,8,150000,112536,247,1,150000,10000,1,136542,1326,1,1000,150000,1000,1,150000,32,150000,32,150000,32,1,1,150000,1,150000,4,103599,248,1,103599,248,1,145276,1366,1,179690,497,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,148000,425507,118,0,1,1,61516,11218,0,1,150000,32,148000,425507,118,0,1,1,148000,425507,118,0,1,1,2477736,29175,4,0,82363,4,150000,5000,0,1,150000,32,197209,0,1,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,3345831,1,1],"PlutusV2":[205665,812,1,1,1000,571,0,1,1000,24177,4,1,1000,32,117366,10475,4,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,100,100,23000,100,19537,32,175354,32,46417,4,221973,511,0,1,89141,32,497525,14068,4,2,196500,453240,220,0,1,1,1000,28662,4,2,245000,216773,62,1,1060367,12586,1,208512,421,1,187000,1000,52998,1,80436,32,43249,32,1000,32,80556,1,57667,4,1000,10,197145,156,1,197145,156,1,204924,473,1,208896,511,1,52467,32,64832,32,65493,32,22558,32,16563,32,76511,32,196500,453240,220,0,1,1,69522,11687,0,1,60091,32,196500,453240,220,0,1,1,196500,453240,220,0,1,1,1159724,392670,0,2,806990,30482,4,1927926,82523,4,265318,0,4,0,85931,32,205665,812,1,1,41182,32,212342,32,31220,32,32696,32,43357,32,32247,32,38314,32,35892428,10,9462713,1021,10,38887044,32947,10,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807],"PlutusV3":[100788,420,1,1,1000,173,0,1,1000,59957,4,1,11183,32,201305,8356,4,16000,100,16000,100,16000,100,16000,100,16000,100,16000,100,100,100,16000,100,94375,32,132994,32,61462,4,72010,178,0,1,22151,32,91189,769,4,2,85848,123203,7305,-900,1716,549,57,85848,0,1,1,1000,42921,4,2,24548,29498,38,1,898148,27279,1,51775,558,1,39184,1000,60594,1,141895,32,83150,32,15299,32,76049,1,13169,4,22100,10,28999,74,1,28999,74,1,43285,552,1,44749,541,1,33852,32,68246,32,72362,32,7243,32,7391,32,11546,32,85848,123203,7305,-900,1716,549,57,85848,0,1,90434,519,0,1,74433,32,85848,123203,7305,-900,1716,549,57,85848,0,1,1,85848,123203,7305,-900,1716,549,57,85848,0,1,955506,213312,0,2,270652,22588,4,1457325,64566,4,20467,1,4,0,141992,32,100788,420,1,1,81663,32,59498,32,20142,32,24588,32,20744,32,25933,32,24623,32,43053543,10,53384111,14333,10,43574283,26308,10,16000,100,16000,100,962335,18,2780678,6,442008,1,52538055,3756,18,267929,18,76433006,8868,18,52948122,18,1995836,36,3227919,12,901022,1,166917843,4307,36,284546,36,158221314,26549,36,74698472,36,333849714,1,254006273,72,2174038,72,2261318,64571,4,207616,8310,4,1293828,28716,63,0,1,1006041,43623,251,0,1]},"protocolVersion":{"major":7,"minor":0},"maxBlockHeaderSize":1100,"maxBlockBodySize":65536,"maxTxSize":16384,"txFeeFixed":155381,"txFeePerByte":44,"stakeAddressDeposit":400000,"stakePoolDeposit":0,"minPoolCost":0,"poolRetireMaxEpoch":18,"stakePoolTargetNum":100,"poolPledgeInfluence":0,"monetaryExpansion":0.1,"treasuryCut":0.1,"collateralPercentage":150,"executionUnitPrices":{"priceMemory":0.0577,"priceSteps":0.0000721},"utxoCostPerByte":4310,"maxTxExecutionUnits":{"memory":16000000,"steps":10000000000},"maxBlockExecutionUnits":{"memory":80000000,"steps":40000000000},"maxCollateralInputs":3,"maxValueSize":5000,"extraPraosEntropy":null,"decentralization":null,"minUTxOValue":null,"poolVotingThresholds":{"committeeNoConfidence":0.51,"committeeNormal":0.51,"hardForkInitiation":0.51,"motionNoConfidence":0.51,"ppSecurityGroup":0.51},"dRepVotingThresholds":{"committeeNoConfidence":0.51,"committeeNormal":0.51,"hardForkInitiation":0.51,"motionNoConfidence":0.51,"ppEconomicGroup":0.51,"ppGovGroup":0.51,"ppNetworkGroup":0.51,"ppTechnicalGroup":0.51,"treasuryWithdrawal":0.51,"updateToConstitution":0.51},"dRepActivity":0,"dRepDeposit":0,"govActionDeposit":0,"govActionLifetime":14,"minFeeRefScriptCostPerByte":15,"committeeMaxTermLength":60,"committeeMinSize":0}`)
)

func TestTxSender(t *testing.T) {
	bridgingAddr, err := cardanowallet.NewEnterpriseAddress(0, append(make([]byte, 31), 1))
	require.NoError(t, err)

	txProvider := &txProviderMock{
		protocolParameters: dummyProtoParams,
		utxos: []cardanowallet.Utxo{
			{
				Hash:   "f97a06232cd0998821768cf053964d8c265d28984a1ff29f50de097ed3add8b5",
				Index:  2,
				Amount: uint64(2_000_000_000),
				Tokens: []cardanowallet.TokenAmount{
					{
						Token:  cardanowallet.NewToken(dummyPID, "Route3"),
						Amount: uint64(12_000_000),
					},
				},
			},
		},
	}
	ctx := context.Background()
	privateSigningKeys := []string{
		"a678adbbca14b1fe81e4294f2a5274a25537e164ce839f890ef8f9f29d1e0af2",
		"a7f7a3b37b72924ba926b87e553d587256b92bb14070998491497f9bab22f426",
		"da737464dd5074dfebc34bb90a0cd0e92b06a978a1132d55fda4d2b0df96729c",
		"e16d2bfe1c3aea4c75c314b9067081eaf6c619b5fc5d3b33da155094de05c357",
	}
	configs := map[string]ChainConfig{
		"prime": {
			TestNetMagic:     3113,
			MultiSigAddr:     bridgingAddr.String(),
			CardanoCliBinary: cardanowallet.ResolveCardanoCliBinary(0),
			TxProvider:       txProvider,
			NativeTokens: []TokenExchangeConfig{
				{
					DstChainID: "vector",
					TokenName:  fmt.Sprintf("%s.Route3", dummyPID),
				},
			},
		},
		"vector": {
			TestNetMagic: 1790,
		},
	}

	wallets := make([]*cardanowallet.Wallet, len(privateSigningKeys))
	keyHashes := make([]string, len(privateSigningKeys))

	for i, psk := range privateSigningKeys {
		pskBytes, err := hex.DecodeString(psk)
		require.NoError(t, err)

		wallets[i] = cardanowallet.NewWallet(pskBytes, nil)

		keyHashes[i], err = cardanowallet.GetKeyHash(wallets[i].VerificationKey)
		require.NoError(t, err)
	}

	cliBinary := cardanowallet.ResolveCardanoCliBinary(0)
	cliUtils := cardanowallet.NewCliUtils(cliBinary)

	receiverAddr, err := cardanowallet.NewEnterpriseAddress(0, wallets[0].VerificationKey)
	require.NoError(t, err)

	quorumCount := (len(keyHashes)*2)/3 + 1
	policyScript := cardanowallet.NewPolicyScript(keyHashes, quorumCount)

	multisigAddr, err := cliUtils.GetPolicyScriptEnterpriseAddress(
		configs["prime"].TestNetMagic, policyScript)
	require.NoError(t, err)

	txSender := NewTxSender(configs, WithMaxInputsPerTx(10), WithRetryOptions(nil))

	t.Run("create bridging tx", func(t *testing.T) {
		txInfo, metadata, err := txSender.CreateBridgingTx(ctx, BridgingTxDto{
			SrcChainID: "prime", DstChainID: "vector",
			SenderAddr:             multisigAddr,
			SenderAddrPolicyScript: policyScript,
			Receivers: []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeNativeTokenOnSource,
					Addr:         receiverAddr.String(),
					Amount:       uint64(1_000_000),
				},
			},
			BridgingFee: uint64(1_000_010),
		})

		require.NoError(t, err)
		require.NotNil(t, txInfo)
		require.NotNil(t, metadata)
	})

	t.Run("calculate bridging tx fee", func(t *testing.T) {
		txFeeInfo, metadata, err := txSender.CalculateBridgingTxFee(ctx, BridgingTxDto{
			SrcChainID: "prime", DstChainID: "vector",
			SenderAddr:             multisigAddr,
			SenderAddrPolicyScript: policyScript,
			Receivers: []BridgingTxReceiver{
				{
					BridgingType: BridgingTypeNativeTokenOnSource,
					Addr:         receiverAddr.String(),
					Amount:       uint64(1_000_000),
				},
			},
			BridgingFee: uint64(1_000_010),
		})

		require.NoError(t, err)
		require.NotNil(t, txFeeInfo)
		require.NotNil(t, metadata)
		require.Greater(t, txFeeInfo.Fee, uint64(0))
	})

	t.Run("create generic tx", func(t *testing.T) {
		txInfo, err := txSender.CreateTxGeneric(ctx, GenericTxDto{
			SrcChainID:             "prime",
			SenderAddr:             multisigAddr,
			SenderAddrPolicyScript: policyScript,
			ReceiverAddr:           receiverAddr.String(),
			OutputLovelace:         uint64(1_000_030),
			OutputNativeTokens: []cardanowallet.TokenAmount{
				{
					Token:  cardanowallet.NewToken(dummyPID, "Route3"),
					Amount: uint64(2_000_000),
				},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, txInfo)
	})
}

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

	t.Run("invalid destination", func(t *testing.T) {
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
	const bridgingFee = uint64(1_000_87)

	token := cardanowallet.NewToken(dummyPID, "WADA")
	txProviderMock := &txProviderMock{
		protocolParameters: dummyProtoParams,
	}
	txSnd := NewTxSender(map[string]ChainConfig{
		"prime": {
			MinUtxoValue: 55,
			TestNetMagic: cardanowallet.PreviewProtocolMagic,
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
		bridgingTxInput := BridgingTxDto{
			SrcChainID: "prime",
			DstChainID: "vector",
			SenderAddr: dummyAddr,
			Receivers: []BridgingTxReceiver{
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
			},
			BridgingAddress: dummyAddr,
			BridgingFee:     bridgingFee,
			OperationFee:    0,
		}
		data, err := txSnd.prepareBridgingTx(context.Background(), bridgingTxInput)

		require.NoError(t, err)
		require.NotNil(t, data.TxBuilder)

		defer data.TxBuilder.Dispose()

		expectedTokenAmount := cardanowallet.NewTokenAmount(token, 600_000*2+3)
		calcMinUtxoLovelaceAmount := uint64(1_034_400)
		expectedLovelaceAmount := calcMinUtxoLovelaceAmount + bridgingFee

		assert.Equal(t, expectedLovelaceAmount, data.OutputLovelace)
		assert.Equal(t, calcMinUtxoLovelaceAmount-1_000_001+bridgingFee, data.BridgingFee)
		assert.Len(t, data.OutputNativeTokens, 1)
		assert.Equal(t, expectedTokenAmount, data.OutputNativeTokens[0])
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
		v, err := adjustLovelaceOutput(txBuilder, dummyAddr, []cardanowallet.TokenAmount{{
			Token:  cardanowallet.NewToken(dummyPID, "WADAorWAPEX"),
			Amount: 1_000_000_000_000,
		}}, 1_000_002, 1_000_001)

		require.NoError(t, err)
		require.Equal(t, uint64(1_081_810), v)
	})
}

func Test_populateTxBuilder(t *testing.T) {
	txBuilder, err := cardanowallet.NewTxBuilder(cardanowallet.ResolveCardanoCliBinary(cardanowallet.TestNetNetwork))
	require.NoError(t, err)

	defer txBuilder.Dispose()

	txBuilder.SetProtocolParameters(dummyProtoParams)

	token := cardanowallet.NewToken(dummyPID, "WADA")
	txProviderMock := &txProviderMock{
		protocolParameters: dummyProtoParams,
		utxos: []cardanowallet.Utxo{
			{
				Amount: 10_000_000,
				Tokens: []cardanowallet.TokenAmount{
					cardanowallet.NewTokenAmount(token, 10_000_000),
				},
			},
		},
	}
	txSnd := NewTxSender(map[string]ChainConfig{
		"": {
			MinUtxoValue: 55,
			TestNetMagic: cardanowallet.PreviewProtocolMagic,
			NativeTokens: []TokenExchangeConfig{
				{
					DstChainID: "vector",
					TokenName:  token.String(),
				},
			},
			TxProvider:       txProviderMock,
			CardanoCliBinary: cardanowallet.ResolveCardanoCliBinary(cardanowallet.TestNetNetwork),
		},
	})

	t.Run("valid without token", func(t *testing.T) {
		data, err := txSnd.populateTxBuilder(
			context.Background(), txBuilder,
			GenericTxDto{
				SenderAddr:     dummyAddr,
				ReceiverAddr:   dummyAddr,
				OutputLovelace: 2_000_000,
			})

		require.NoError(t, err)
		assert.Equal(t, uint64(8000000), data.ChangeLovelace)
		assert.Equal(t, uint64(1034400), data.ChangeMinUtxoAmount)
		assert.GreaterOrEqual(t, len(data.ChosenInputs.Inputs), 1)
	})

	t.Run("valid with token", func(t *testing.T) {
		data, err := txSnd.populateTxBuilder(
			context.Background(), txBuilder,
			GenericTxDto{
				SenderAddr:     dummyAddr,
				ReceiverAddr:   dummyAddr,
				OutputLovelace: 1_000_000,
				OutputNativeTokens: []cardanowallet.TokenAmount{{
					Token:  token,
					Amount: 2_000_000,
				}},
			})

		require.NoError(t, err)
		assert.Equal(t, uint64(9000000), data.ChangeLovelace)
		assert.Equal(t, uint64(1034400), data.ChangeMinUtxoAmount)
		assert.GreaterOrEqual(t, len(data.ChosenInputs.Inputs), 1)
	})
}

type txProviderMock struct {
	protocolParameters []byte
	utxos              []cardanowallet.Utxo
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
	return m.utxos, nil
}

func (m *txProviderMock) SubmitTx(ctx context.Context, txSigned []byte) error {
	return nil
}

func (m *txProviderMock) GetStakePools(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *txProviderMock) GetStakeAddressInfo(ctx context.Context, addr string) (cardanowallet.QueryStakeAddressInfo, error) {
	return cardanowallet.QueryStakeAddressInfo{}, nil
}
