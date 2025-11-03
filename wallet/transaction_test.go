package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	protocolParameters         = []byte(`{"costModels":{"PlutusV1":[197209,0,1,1,396231,621,0,1,150000,1000,0,1,150000,32,2477736,29175,4,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,100,100,29773,100,150000,32,150000,32,150000,32,150000,1000,0,1,150000,32,150000,1000,0,8,148000,425507,118,0,1,1,150000,1000,0,8,150000,112536,247,1,150000,10000,1,136542,1326,1,1000,150000,1000,1,150000,32,150000,32,150000,32,1,1,150000,1,150000,4,103599,248,1,103599,248,1,145276,1366,1,179690,497,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,148000,425507,118,0,1,1,61516,11218,0,1,150000,32,148000,425507,118,0,1,1,148000,425507,118,0,1,1,2477736,29175,4,0,82363,4,150000,5000,0,1,150000,32,197209,0,1,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,3345831,1,1],"PlutusV2":[205665,812,1,1,1000,571,0,1,1000,24177,4,1,1000,32,117366,10475,4,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,100,100,23000,100,19537,32,175354,32,46417,4,221973,511,0,1,89141,32,497525,14068,4,2,196500,453240,220,0,1,1,1000,28662,4,2,245000,216773,62,1,1060367,12586,1,208512,421,1,187000,1000,52998,1,80436,32,43249,32,1000,32,80556,1,57667,4,1000,10,197145,156,1,197145,156,1,204924,473,1,208896,511,1,52467,32,64832,32,65493,32,22558,32,16563,32,76511,32,196500,453240,220,0,1,1,69522,11687,0,1,60091,32,196500,453240,220,0,1,1,196500,453240,220,0,1,1,1159724,392670,0,2,806990,30482,4,1927926,82523,4,265318,0,4,0,85931,32,205665,812,1,1,41182,32,212342,32,31220,32,32696,32,43357,32,32247,32,38314,32,35892428,10,9462713,1021,10,38887044,32947,10,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807,9223372036854775807],"PlutusV3":[100788,420,1,1,1000,173,0,1,1000,59957,4,1,11183,32,201305,8356,4,16000,100,16000,100,16000,100,16000,100,16000,100,16000,100,100,100,16000,100,94375,32,132994,32,61462,4,72010,178,0,1,22151,32,91189,769,4,2,85848,123203,7305,-900,1716,549,57,85848,0,1,1,1000,42921,4,2,24548,29498,38,1,898148,27279,1,51775,558,1,39184,1000,60594,1,141895,32,83150,32,15299,32,76049,1,13169,4,22100,10,28999,74,1,28999,74,1,43285,552,1,44749,541,1,33852,32,68246,32,72362,32,7243,32,7391,32,11546,32,85848,123203,7305,-900,1716,549,57,85848,0,1,90434,519,0,1,74433,32,85848,123203,7305,-900,1716,549,57,85848,0,1,1,85848,123203,7305,-900,1716,549,57,85848,0,1,955506,213312,0,2,270652,22588,4,1457325,64566,4,20467,1,4,0,141992,32,100788,420,1,1,81663,32,59498,32,20142,32,24588,32,20744,32,25933,32,24623,32,43053543,10,53384111,14333,10,43574283,26308,10,16000,100,16000,100,962335,18,2780678,6,442008,1,52538055,3756,18,267929,18,76433006,8868,18,52948122,18,1995836,36,3227919,12,901022,1,166917843,4307,36,284546,36,158221314,26549,36,74698472,36,333849714,1,254006273,72,2174038,72,2261318,64571,4,207616,8310,4,1293828,28716,63,0,1,1006041,43623,251,0,1]},"protocolVersion":{"major":7,"minor":0},"maxBlockHeaderSize":1100,"maxBlockBodySize":65536,"maxTxSize":16384,"txFeeFixed":155381,"txFeePerByte":44,"stakeAddressDeposit":400000,"stakePoolDeposit":0,"minPoolCost":0,"poolRetireMaxEpoch":18,"stakePoolTargetNum":100,"poolPledgeInfluence":0,"monetaryExpansion":0.1,"treasuryCut":0.1,"collateralPercentage":150,"executionUnitPrices":{"priceMemory":0.0577,"priceSteps":0.0000721},"utxoCostPerByte":4310,"maxTxExecutionUnits":{"memory":16000000,"steps":10000000000},"maxBlockExecutionUnits":{"memory":80000000,"steps":40000000000},"maxCollateralInputs":3,"maxValueSize":5000,"extraPraosEntropy":null,"decentralization":null,"minUTxOValue":null,"poolVotingThresholds":{"committeeNoConfidence":0.51,"committeeNormal":0.51,"hardForkInitiation":0.51,"motionNoConfidence":0.51,"ppSecurityGroup":0.51},"dRepVotingThresholds":{"committeeNoConfidence":0.51,"committeeNormal":0.51,"hardForkInitiation":0.51,"motionNoConfidence":0.51,"ppEconomicGroup":0.51,"ppGovGroup":0.51,"ppNetworkGroup":0.51,"ppTechnicalGroup":0.51,"treasuryWithdrawal":0.51,"updateToConstitution":0.51},"dRepActivity":0,"dRepDeposit":0,"govActionDeposit":0,"govActionLifetime":14,"minFeeRefScriptCostPerByte":15,"committeeMaxTermLength":60,"committeeMinSize":0}`)
	primeTestnetProtocolParams = []byte(`{"collateralPercentage":150,"costModels":{"PlutusV1":[197209,0,1,1,396231,621,0,1,150000,1000,0,1,150000,32,2477736,29175,4,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,100,100,29773,100,150000,32,150000,32,150000,32,150000,1000,0,1,150000,32,150000,1000,0,8,148000,425507,118,0,1,1,150000,1000,0,8,150000,112536,247,1,150000,10000,1,136542,1326,1,1000,150000,1000,1,150000,32,150000,32,150000,32,1,1,150000,1,150000,4,103599,248,1,103599,248,1,145276,1366,1,179690,497,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,148000,425507,118,0,1,1,61516,11218,0,1,150000,32,148000,425507,118,0,1,1,148000,425507,118,0,1,1,2477736,29175,4,0,82363,4,150000,5000,0,1,150000,32,197209,0,1,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,3345831,1,1],"PlutusV2":[205665,812,1,1,1000,571,0,1,1000,24177,4,1,1000,32,117366,10475,4,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,100,100,23000,100,19537,32,175354,32,46417,4,221973,511,0,1,89141,32,497525,14068,4,2,196500,453240,220,0,1,1,1000,28662,4,2,245000,216773,62,1,1060367,12586,1,208512,421,1,187000,1000,52998,1,80436,32,43249,32,1000,32,80556,1,57667,4,1000,10,197145,156,1,197145,156,1,204924,473,1,208896,511,1,52467,32,64832,32,65493,32,22558,32,16563,32,76511,32,196500,453240,220,0,1,1,69522,11687,0,1,60091,32,196500,453240,220,0,1,1,196500,453240,220,0,1,1,1159724,392670,0,2,806990,30482,4,1927926,82523,4,265318,0,4,0,85931,32,205665,812,1,1,41182,32,212342,32,31220,32,32696,32,43357,32,32247,32,38314,32,35892428,10,9462713,1021,10,38887044,32947,10]},"decentralization":null,"executionUnitPrices":{"priceMemory":5.77,"priceSteps":0.00721},"extraPraosEntropy":null,"maxBlockBodySize":65536,"maxBlockExecutionUnits":{"memory":80000000,"steps":40000000000},"maxBlockHeaderSize":1100,"maxCollateralInputs":3,"maxTxExecutionUnits":{"memory":16000000,"steps":10000000000},"maxTxSize":16384,"maxValueSize":5000,"minPoolCost":0,"minUTxOValue":null,"monetaryExpansion":0.0038,"poolPledgeInfluence":0,"poolRetireMaxEpoch":18,"protocolVersion":{"major":7,"minor":0},"stakeAddressDeposit":0,"stakePoolDeposit":0,"stakePoolTargetNum":100,"treasuryCut":1.0e-8,"txFeeFixed":158298,"txFeePerByte":47,"utxoCostPerByte":4310}`)
	eraName                    = DefaultEra
)

func Test_TransactionBuilder(t *testing.T) {
	t.Parallel()

	const (
		testNetMagic = 203
		ttl          = uint64(28096)
	)

	walletsKeyHashes := []string{
		"d6b67f93ffa4e2651271cc9bcdbdedb2539911266b534d9c163cba21",
		"cba89c7084bf0ce4bf404346b668a7e83c8c9c250d1cafd8d8996e41",
		"79df3577e4c7d7da04872c2182b8d8829d7b477912dbf35d89287c39",
		"2368e8113bd5f32d713751791d29acee9e1b5a425b0454b963b2558b",
		"06b4c7f5254d6395b527ac3de60c1d77194df7431d85fe55ca8f107d",
	}
	walletsFeeKeyHashes := []string{
		"f0f4837b3a306752a2b3e52394168bc7391de3dce11364b723cc55cf",
		"47344d5bd7b2fea56336ba789579705a944760032585ef64084c92db",
		"f01018c1d8da54c2f557679243b09af1c4dd4d9c671512b01fa5f92b",
		"6837232854849427dae7c45892032d7ded136c5beb13c68fda635d87",
		"d215701e2eb17c741b9d306cba553f9fbaaca1e12a5925a065b90fa8",
	}

	policyScriptMultiSig := NewPolicyScript(walletsKeyHashes, len(walletsKeyHashes)*2/3+1)
	policyScriptFeeMultiSig := NewPolicyScript(walletsFeeKeyHashes, len(walletsFeeKeyHashes)*2/3+1)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	multisigPolicyID, err := cliUtils.GetPolicyID(policyScriptMultiSig)
	require.NoError(t, err)

	feeMultisigPolicyID, err := cliUtils.GetPolicyID(policyScriptFeeMultiSig)
	require.NoError(t, err)

	multiSigAddr, err := NewPolicyScriptEnterpriseAddress(TestNetNetwork, multisigPolicyID)
	require.NoError(t, err)

	multiSigFeeAddr, err := NewPolicyScriptEnterpriseAddress(TestNetNetwork, feeMultisigPolicyID)
	require.NoError(t, err)

	type metaDataKey0 struct {
		Type       string `json:"type" cbor:"type"`
		Signers    int    `json:"signers" cbor:"signers"`
		FeeSigners int    `json:"feeSigners" cbor:"feeSigners"`
	}

	type metaDataKey1 struct {
		Company string `json:"comp" cbor:"comp"`
		City    string `json:"city" cbor:"city"`
	}

	metadata := map[uint64]interface{}{
		0: metaDataKey0{
			Type:       "multi",
			Signers:    len(walletsKeyHashes),
			FeeSigners: len(walletsFeeKeyHashes),
		},
		4: metaDataKey1{
			Company: "Ethernal",
			City:    "Novi Sad",
		},
	}
	outputs := []TxOutput{
		{
			Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
			Amount: uint64(1_000_000),
		},
	}
	outputsSum := GetOutputsSum(outputs)

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err)

	multiSigInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "e99a5bde15aa05f24fcc04b7eabc1520d3397283b1ee720de9fe2653abbb0c9f",
				Index: 0,
			},
			{
				Hash:  "d1fd0d772be7741d9bfaf0b037d02d2867a987ccba3e6ba2ee9aa2a861b73145",
				Index: 2,
			},
		},
		Sum: map[string]uint64{
			AdaTokenName: uint64(1_000_000)*3 - 10,
		},
	}

	multiSigFeeInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e",
				Index: 0,
			},
		},
		Sum: map[string]uint64{
			AdaTokenName: uint64(1_000_000) * 2,
		},
	}

	builder.SetTimeToLive(ttl).SetProtocolParameters(protocolParameters)
	builder.SetMetaData(metadataBytes).SetTestNetMagic(testNetMagic)
	builder.AddOutputs(outputs...).AddOutputs(TxOutput{
		Addr: multiSigAddr.String(),
	}).AddOutputs(TxOutput{
		Addr: multiSigFeeAddr.String(),
	})
	builder.AddInputsWithScript(policyScriptMultiSig, multiSigInputs.Inputs...)
	builder.AddInputsWithScript(policyScriptFeeMultiSig, multiSigFeeInputs.Inputs...)

	fee, err := builder.CalculateFee(0)
	require.NoError(t, err)

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-2, multiSigInputs.Sum[AdaTokenName]-outputsSum[AdaTokenName])
	builder.UpdateOutputAmount(-1, multiSigFeeInputs.Sum[AdaTokenName]-fee)

	txRaw, txHash, err := builder.Build()
	require.NoError(t, err)

	assert.Equal(t, "84a50083825820098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e00825820d1fd0d772be7741d9bfaf0b037d02d2867a987ccba3e6ba2ee9aa2a861b7314502825820e99a5bde15aa05f24fcc04b7eabc1520d3397283b1ee720de9fe2653abbb0c9f00018382581d60244877c1aeefc7fd5405a6e14d927d91758d45e37c20fa2ac89cb1671a000f424082581d704aaad0f0626a8ce7b097497e542055b6520842ade881f980e002ae661a001e847682581d703ea4c4aef89a27f111e78464d7d6717b099f85ce27109ee9e5fbddec1a001a6863021a00041c1d03196dc0075820802e4d6f15ce98826886a5451e94855e77aae779cb341d3aab1e3bae4fb2f78da10182830304858200581c47344d5bd7b2fea56336ba789579705a944760032585ef64084c92db8200581c6837232854849427dae7c45892032d7ded136c5beb13c68fda635d878200581cd215701e2eb17c741b9d306cba553f9fbaaca1e12a5925a065b90fa88200581cf01018c1d8da54c2f557679243b09af1c4dd4d9c671512b01fa5f92b8200581cf0f4837b3a306752a2b3e52394168bc7391de3dce11364b723cc55cf830304858200581c06b4c7f5254d6395b527ac3de60c1d77194df7431d85fe55ca8f107d8200581c2368e8113bd5f32d713751791d29acee9e1b5a425b0454b963b2558b8200581c79df3577e4c7d7da04872c2182b8d8829d7b477912dbf35d89287c398200581ccba89c7084bf0ce4bf404346b668a7e83c8c9c250d1cafd8d8996e418200581cd6b67f93ffa4e2651271cc9bcdbdedb2539911266b534d9c163cba21f5d90103a100a200a36a6665655369676e65727305677369676e657273056474797065656d756c746904a26463697479684e6f76692053616464636f6d706845746865726e616c", hex.EncodeToString(txRaw))

	txHashUtil, err := cliUtils.GetTxHash(txRaw)
	require.NoError(t, err)

	require.Equal(t, "55f7fbf8772bfb35640b62694f0e5c6e2baddee02ae2dd1881943d5bf3d4030a", txHashUtil)
	require.Equal(t, txHash, txHashUtil)

	signer, err := GenerateWallet(false)
	require.NoError(t, err)

	txSigned, err := builder.SignTx(txRaw, []ITxSigner{signer})

	require.NoError(t, err)
	require.NotEmpty(t, txSigned)
}

func Test_TransactionBuilderWithRegistrationCertificate(t *testing.T) {
	policyPaymentKeyHashes := []string{
		"0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c6",
		"41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a0411",
		"5282885af1f234cb9407f05b120f2eb06872f297864ca9066a657011",
		"6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42",
	}

	policyStakeKeyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}

	policyScriptPaymentMultiSig := NewPolicyScript(policyPaymentKeyHashes, len(policyPaymentKeyHashes)*2/3+1)
	policyScriptStakeMultiSig := NewPolicyScript(policyStakeKeyHashes, len(policyStakeKeyHashes)*2/3+1)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	multisigPaymentPolicyID, err := cliUtils.GetPolicyID(policyScriptPaymentMultiSig)
	require.NoError(t, err)

	multisigStakePolicyID, err := cliUtils.GetPolicyID(policyScriptStakeMultiSig)
	require.NoError(t, err)

	multiSigAddr, err := NewPolicyScriptBaseAddress(TestNetNetwork, multisigPaymentPolicyID, multisigStakePolicyID)
	require.NoError(t, err)
	require.Equal(t, "addr_test1xqdt3kene0l87agrdcsn7jzspfrj83h5svgmaw8rnzzva644n47f76yle0p2r8dzdz0elefvtaju8v79ddahutcg790s37mp24", multiSigAddr.String())

	multiSigStakeAddr, err := NewPolicyScriptRewardAddress(TestNetNetwork, multisigStakePolicyID)
	require.NoError(t, err)
	require.Equal(t, "stake_test17z6e6lyldz0uhs4pnk3x38ulu5k97ewrk0zkk7m79uy0zhcp9x067", multiSigStakeAddr.String())

	// Create registration certificate
	registrationCertificate, err := cliUtils.CreateRegistrationCertificate(multiSigStakeAddr.String(), 0)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(registrationCertificate.Type, "Certificate"))
	require.Equal(t, "Stake Address Registration Certificate", registrationCertificate.Description)
	require.Equal(t, "82008201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f", registrationCertificate.CborHex)

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	multiSigInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "bb88a2541d545044e400d37c3db3eeb7a452fd9f2c461c89451f7191cc4f4079",
				Index: 0,
			},
		},
		Sum: map[string]uint64{
			AdaTokenName: uint64(10_000_000),
		},
	}

	builder.SetTimeToLive(uint64(9211)).SetProtocolParameters(protocolParameters)
	builder.SetTestNetMagic(2)
	builder.AddInputsWithScript(policyScriptPaymentMultiSig, multiSigInputs.Inputs...)
	builder.AddOutputs(TxOutput{
		Addr: multiSigAddr.String(),
	})

	builder.AddCertificates(policyScriptStakeMultiSig, registrationCertificate)

	fee, err := builder.CalculateFee(0)
	require.NoError(t, err)
	require.Equal(t, uint64(212361), fee)

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-1, multiSigInputs.Sum[AdaTokenName]-fee)

	txRaw, txHash, err := builder.Build()
	require.NoError(t, err)

	require.Equal(t, "953405612f0be25d65c11407494a72ccaac2c4b40c8f577fbf609856f81fb429", txHash)
	require.Equal(t, "84a50081825820bb88a2541d545044e400d37c3db3eeb7a452fd9f2c461c89451f7191cc4f4079000181825839301ab8db33cbfe7f75036e213f48500a4723c6f48311beb8e39884ceeab59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f1a009558f7021a00033d89031923fb048182008201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15fa10181830303848200581c0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c68200581c41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a04118200581c5282885af1f234cb9407f05b120f2eb06872f297864ca9066a6570118200581c6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42f5f6", hex.EncodeToString(txRaw))
}

func Test_TransactionBuilderWithDelegationCertificate(t *testing.T) {
	policyPaymentKeyHashes := []string{
		"0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c6",
		"41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a0411",
		"5282885af1f234cb9407f05b120f2eb06872f297864ca9066a657011",
		"6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42",
	}

	policyStakeKeyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}

	policyScriptPaymentMultiSig := NewPolicyScript(policyPaymentKeyHashes, len(policyPaymentKeyHashes)*2/3+1)
	policyScriptStakeMultiSig := NewPolicyScript(policyStakeKeyHashes, len(policyStakeKeyHashes)*2/3+1)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	multisigPaymentPolicyID, err := cliUtils.GetPolicyID(policyScriptPaymentMultiSig)
	require.NoError(t, err)

	multisigStakePolicyID, err := cliUtils.GetPolicyID(policyScriptStakeMultiSig)
	require.NoError(t, err)

	multiSigAddr, err := NewPolicyScriptBaseAddress(TestNetNetwork, multisigPaymentPolicyID, multisigStakePolicyID)
	require.NoError(t, err)
	require.Equal(t, "addr_test1xqdt3kene0l87agrdcsn7jzspfrj83h5svgmaw8rnzzva644n47f76yle0p2r8dzdz0elefvtaju8v79ddahutcg790s37mp24", multiSigAddr.String())

	multiSigStakeAddr, err := NewPolicyScriptRewardAddress(TestNetNetwork, multisigStakePolicyID)
	require.NoError(t, err)
	require.Equal(t, "stake_test17z6e6lyldz0uhs4pnk3x38ulu5k97ewrk0zkk7m79uy0zhcp9x067", multiSigStakeAddr.String())

	// Create delegation certificate
	poolID := "pool1ttxrlraudm8msm88x4pjz75xqwrug2qmkw2tfgfr7ddjgqfa43q"
	delegationCertificate, err := cliUtils.CreateDelegationCertificate(multiSigStakeAddr.String(), poolID)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(delegationCertificate.Type, "Certificate"))
	require.Equal(t, "Stake Delegation Certificate", delegationCertificate.Description)
	require.Equal(t, "83028201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f581c5acc3f8fbc6ecfb86ce73543217a860387c4281bb394b4a123f35b24", delegationCertificate.CborHex)

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	multiSigInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "c6edbde4bf6421ddf7f51643da7ce602cd63ef396053c7a39bc081d332ca8009",
				Index: 0,
			},
		},
		Sum: map[string]uint64{
			AdaTokenName: uint64(9792083),
		},
	}

	builder.SetTimeToLive(uint64(11704)).SetProtocolParameters(protocolParameters)
	builder.SetTestNetMagic(2)
	builder.AddInputsWithScript(policyScriptPaymentMultiSig, multiSigInputs.Inputs...)
	builder.AddOutputs(TxOutput{
		Addr: multiSigAddr.String(),
	})
	builder.AddCertificates(policyScriptStakeMultiSig, delegationCertificate)

	fee, err := builder.CalculateFee(0)
	require.NoError(t, err)
	require.Equal(t, uint64(219489), fee)

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-1, multiSigInputs.Sum[AdaTokenName]-fee)

	txRaw, txHash, err := builder.Build()
	require.NoError(t, err)

	require.Equal(t, "7055918a634a92221d0112ace86561b8fff39d51d9abf29729a6c659a8c37e7a", txHash)
	require.Equal(t, "84a50081825820c6edbde4bf6421ddf7f51643da7ce602cd63ef396053c7a39bc081d332ca8009000181825839301ab8db33cbfe7f75036e213f48500a4723c6f48311beb8e39884ceeab59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f1a009210f2021a0003596103192db8048183028201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f581c5acc3f8fbc6ecfb86ce73543217a860387c4281bb394b4a123f35b24a10182830303848200581c0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c68200581c41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a04118200581c5282885af1f234cb9407f05b120f2eb06872f297864ca9066a6570118200581c6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42830303848200581c30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d8200581c794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c98200581c8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a98200581c9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7dbf5f6", hex.EncodeToString(txRaw))
}

func Test_TransactionBuilderWithRegAndDelegCertificates(t *testing.T) {
	policyPaymentKeyHashes := []string{
		"0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c6",
		"41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a0411",
		"5282885af1f234cb9407f05b120f2eb06872f297864ca9066a657011",
		"6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42",
	}

	policyStakeKeyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}

	policyScriptPaymentMultiSig := NewPolicyScript(policyPaymentKeyHashes, len(policyPaymentKeyHashes)*2/3+1)
	policyScriptStakeMultiSig := NewPolicyScript(policyStakeKeyHashes, len(policyStakeKeyHashes)*2/3+1)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	multisigPaymentPolicyID, err := cliUtils.GetPolicyID(policyScriptPaymentMultiSig)
	require.NoError(t, err)

	multisigStakePolicyID, err := cliUtils.GetPolicyID(policyScriptStakeMultiSig)
	require.NoError(t, err)

	multiSigAddr, err := NewPolicyScriptBaseAddress(TestNetNetwork, multisigPaymentPolicyID, multisigStakePolicyID)
	require.NoError(t, err)
	require.Equal(t, "addr_test1xqdt3kene0l87agrdcsn7jzspfrj83h5svgmaw8rnzzva644n47f76yle0p2r8dzdz0elefvtaju8v79ddahutcg790s37mp24", multiSigAddr.String())

	multiSigStakeAddr, err := NewPolicyScriptRewardAddress(TestNetNetwork, multisigStakePolicyID)
	require.NoError(t, err)
	require.Equal(t, "stake_test17z6e6lyldz0uhs4pnk3x38ulu5k97ewrk0zkk7m79uy0zhcp9x067", multiSigStakeAddr.String())

	// Create registration certificate
	registrationCertificate, err := cliUtils.CreateRegistrationCertificate(multiSigStakeAddr.String(), 0)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(registrationCertificate.Type, "Certificate"))
	require.Equal(t, "Stake Address Registration Certificate", registrationCertificate.Description)
	require.Equal(t, "82008201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f", registrationCertificate.CborHex)
	// Create delegation certificate
	poolID := "pool1p8kqagxz54eqtuc7tl8d99jvyevt43drejxlcr39n32vk078j5v"
	delegationCertificate, err := cliUtils.CreateDelegationCertificate(multiSigStakeAddr.String(), poolID)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(delegationCertificate.Type, "Certificate"))
	require.Equal(t, "Stake Delegation Certificate", delegationCertificate.Description)
	require.Equal(t, "83028201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f581c09ec0ea0c2a57205f31e5fced2964c2658bac5a3cc8dfc0e259c54cb", delegationCertificate.CborHex)
	certs := []ICardanoArtifact{registrationCertificate, delegationCertificate}

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	multiSigInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "a266468e13942a5a016c12f941864d13a6e82dce3073a7ec7e1a680c2011f1d4",
				Index: 0,
			},
		},
		Sum: map[string]uint64{
			AdaTokenName: uint64(10000000),
		},
	}

	builder.SetTimeToLive(uint64(2193)).SetProtocolParameters(protocolParameters)
	builder.SetTestNetMagic(2)
	builder.AddInputsWithScript(policyScriptPaymentMultiSig, multiSigInputs.Inputs...)
	builder.AddOutputs(TxOutput{
		Addr: multiSigAddr.String(),
	})
	builder.AddCertificates(policyScriptStakeMultiSig, certs...)

	fee, err := builder.CalculateFee(0)
	require.NoError(t, err)
	require.Equal(t, uint64(220985), fee)

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-1, multiSigInputs.Sum[AdaTokenName]-fee)

	txRaw, txHash, err := builder.Build()
	require.NoError(t, err)

	require.Equal(t, "8ade5179d0cbea6af3ee424276f0206d55594157302501d765d29791e46a31cf", txHash)
	require.Equal(t, "84a50081825820a266468e13942a5a016c12f941864d13a6e82dce3073a7ec7e1a680c2011f1d4000181825839301ab8db33cbfe7f75036e213f48500a4723c6f48311beb8e39884ceeab59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f1a00953747021a00035f3903190891048282008201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f83028201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f581c09ec0ea0c2a57205f31e5fced2964c2658bac5a3cc8dfc0e259c54cba10182830303848200581c0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c68200581c41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a04118200581c5282885af1f234cb9407f05b120f2eb06872f297864ca9066a6570118200581c6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42830303848200581c30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d8200581c794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c98200581c8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a98200581c9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7dbf5f6", hex.EncodeToString(txRaw))
}

func Test_TransactionBuilderWithWithdraw(t *testing.T) {
	policyPaymentKeyHashes := []string{
		"0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c6",
		"41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a0411",
		"5282885af1f234cb9407f05b120f2eb06872f297864ca9066a657011",
		"6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42",
	}

	policyStakeKeyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}

	policyScriptPaymentMultiSig := NewPolicyScript(policyPaymentKeyHashes, len(policyPaymentKeyHashes)*2/3+1)
	policyScriptStakeMultiSig := NewPolicyScript(policyStakeKeyHashes, len(policyStakeKeyHashes)*2/3+1)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	multisigPaymentPolicyID, err := cliUtils.GetPolicyID(policyScriptPaymentMultiSig)
	require.NoError(t, err)

	multisigStakePolicyID, err := cliUtils.GetPolicyID(policyScriptStakeMultiSig)
	require.NoError(t, err)

	multiSigAddr, err := NewPolicyScriptBaseAddress(TestNetNetwork, multisigPaymentPolicyID, multisigStakePolicyID)
	require.NoError(t, err)
	multiSigRewardAddr, err := NewPolicyScriptRewardAddress(TestNetNetwork, multisigStakePolicyID)
	require.NoError(t, err)

	require.Equal(t, "addr_test1xqdt3kene0l87agrdcsn7jzspfrj83h5svgmaw8rnzzva644n47f76yle0p2r8dzdz0elefvtaju8v79ddahutcg790s37mp24", multiSigAddr.String())
	require.Equal(t, "stake_test17z6e6lyldz0uhs4pnk3x38ulu5k97ewrk0zkk7m79uy0zhcp9x067", multiSigRewardAddr.String())

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	multiSigInputs := TxInputs{
		Inputs: []TxInput{
			{
				Hash:  "19fc8df9a93cd82d0c3a36d2bf7b8b8d9bc00f1918b0e0ac1ec11ee49345d6ff",
				Index: 0,
			},
		},
		Sum: map[string]uint64{
			AdaTokenName: uint64(9577038),
		},
	}

	builder.SetTimeToLive(uint64(19910)).SetProtocolParameters(protocolParameters)
	builder.SetTestNetMagic(2)
	builder.AddInputsWithScript(policyScriptPaymentMultiSig, multiSigInputs.Inputs...)
	builder.AddOutputs(TxOutput{
		Addr: multiSigAddr.String(),
	})

	rewardAmount := uint64(1539043)
	builder.SetWithdrawalData(multiSigRewardAddr.String(), rewardAmount, policyScriptStakeMultiSig)

	fee, err := builder.CalculateFee(0)
	require.NoError(t, err)
	require.Equal(t, uint64(218257), fee)

	builder.SetFee(fee)

	builder.UpdateOutputAmount(-1, multiSigInputs.Sum[AdaTokenName]+rewardAmount-fee)

	txRaw, txHash, err := builder.Build()
	require.NoError(t, err)

	require.Equal(t, "b0ea2c0c50f60d17a55177edcb2e52238cee859a8dba9217dd9137cf2b0b90c9", txHash)
	require.Equal(t, "84a5008182582019fc8df9a93cd82d0c3a36d2bf7b8b8d9bc00f1918b0e0ac1ec11ee49345d6ff000181825839301ab8db33cbfe7f75036e213f48500a4723c6f48311beb8e39884ceeab59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f1a00a649a0021a0003549103194dc605a1581df0b59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f1a00177be3a10182830303848200581c0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c68200581c41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a04118200581c5282885af1f234cb9407f05b120f2eb06872f297864ca9066a6570118200581c6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42830303848200581c30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d8200581c794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c98200581c8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a98200581c9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7dbf5f6", hex.EncodeToString(txRaw))
}

func Test_TransactionBuilderWithPlutusMint(t *testing.T) {
	inputs := []TxInput{
		{
			Hash:  "2bbfe495f75b5bcb6953b437533beb5aed4ee5d07ce886dac100ae6977349b53",
			Index: 0,
		},
		{
			Hash:  "2bbfe495f75b5bcb6953b437533beb5aed4ee5d07ce886dac100ae6977349b53",
			Index: 3,
		},
	}

	collateralInputs := []TxInput{
		{
			Hash:  "2bbfe495f75b5bcb6953b437533beb5aed4ee5d07ce886dac100ae6977349b53",
			Index: 3,
		},
	}

	collateralOutput := TxOutput{
		Addr:   "addr_test1vq7qupkksergwqyqa0l33f0ksad7w6zk72n7af43veyv7dsyux62h",
		Amount: 0,
	}

	tokensPolicyID := "626cad0064f02def9d61824cac7b9e9fef4292bcab4e439b78bc69bd"
	token1 := NewToken(tokensPolicyID, "sara")
	token2 := NewToken(tokensPolicyID, "sara1")
	nft := NewToken("14b249936a64cbc96bde5a46e04174e7fb58b565103d0c3a32f8d61f", "TestToken")

	mintToknes := []MintTokenAmount{
		NewMintTokenAmount(token1, 15000000),
		NewMintTokenAmount(token2, 15000000),
	}

	txInReference := TxInput{
		Hash:  "20bbbeffee0f48bc03e4226e91cab16bd5778474c121365e13295da459a4251c",
		Index: 0,
	}

	outputs := []TxOutput{
		{
			Addr:   "addr_test1vq7qupkksergwqyqa0l33f0ksad7w6zk72n7af43veyv7dsyux62h",
			Amount: 1500000,
			Tokens: []TokenAmount{
				{
					Token:  nft,
					Amount: 1,
				},
			},
		},
		{
			Addr:   "addr_test1vq7qupkksergwqyqa0l33f0ksad7w6zk72n7af43veyv7dsyux62h",
			Amount: 1500000,
			Tokens: []TokenAmount{
				{
					Token:  token1,
					Amount: 15000000,
				},
			},
		},
		{
			Addr:   "addr_test1vq7qupkksergwqyqa0l33f0ksad7w6zk72n7af43veyv7dsyux62h",
			Amount: 1500000,
			Tokens: []TokenAmount{
				{
					Token:  token2,
					Amount: 15000000,
				},
			},
		},
		{
			Addr: "addr_test1vq7qupkksergwqyqa0l33f0ksad7w6zk72n7af43veyv7dsyux62h",
		},
	}

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	primeTestnetProtocolParams := []byte(`{"collateralPercentage":150,"costModels":{"PlutusV1":[197209,0,1,1,396231,621,0,1,150000,1000,0,1,150000,32,2477736,29175,4,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,100,100,29773,100,150000,32,150000,32,150000,32,150000,1000,0,1,150000,32,150000,1000,0,8,148000,425507,118,0,1,1,150000,1000,0,8,150000,112536,247,1,150000,10000,1,136542,1326,1,1000,150000,1000,1,150000,32,150000,32,150000,32,1,1,150000,1,150000,4,103599,248,1,103599,248,1,145276,1366,1,179690,497,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,148000,425507,118,0,1,1,61516,11218,0,1,150000,32,148000,425507,118,0,1,1,148000,425507,118,0,1,1,2477736,29175,4,0,82363,4,150000,5000,0,1,150000,32,197209,0,1,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,3345831,1,1],"PlutusV2":[205665,812,1,1,1000,571,0,1,1000,24177,4,1,1000,32,117366,10475,4,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,100,100,23000,100,19537,32,175354,32,46417,4,221973,511,0,1,89141,32,497525,14068,4,2,196500,453240,220,0,1,1,1000,28662,4,2,245000,216773,62,1,1060367,12586,1,208512,421,1,187000,1000,52998,1,80436,32,43249,32,1000,32,80556,1,57667,4,1000,10,197145,156,1,197145,156,1,204924,473,1,208896,511,1,52467,32,64832,32,65493,32,22558,32,16563,32,76511,32,196500,453240,220,0,1,1,69522,11687,0,1,60091,32,196500,453240,220,0,1,1,196500,453240,220,0,1,1,1159724,392670,0,2,806990,30482,4,1927926,82523,4,265318,0,4,0,85931,32,205665,812,1,1,41182,32,212342,32,31220,32,32696,32,43357,32,32247,32,38314,32,35892428,10,9462713,1021,10,38887044,32947,10]},"decentralization":null,"executionUnitPrices":{"priceMemory":5.77,"priceSteps":7.21e-3},"extraPraosEntropy":null,"maxBlockBodySize":65536,"maxBlockExecutionUnits":{"memory":80000000,"steps":40000000000},"maxBlockHeaderSize":1100,"maxCollateralInputs":3,"maxTxExecutionUnits":{"memory":16000000,"steps":10000000000},"maxTxSize":16384,"maxValueSize":5000,"minPoolCost":0,"minUTxOValue":null,"monetaryExpansion":3.8e-3,"poolPledgeInfluence":0,"poolRetireMaxEpoch":18,"protocolVersion":{"major":7,"minor":0},"stakeAddressDeposit":0,"stakePoolDeposit":0,"stakePoolTargetNum":100,"treasuryCut":1e-8,"txFeeFixed":158298,"txFeePerByte":47,"utxoCostPerByte":4310}`)

	builder.SetProtocolParameters(primeTestnetProtocolParams)

	builder.AddInputs(inputs...)
	builder.AddCollateralInputs(collateralInputs)
	builder.AddPlutusTokenMints(mintToknes, txInReference, tokensPolicyID)
	builder.AddCollateralOutput(collateralOutput)
	builder.AddOutputs(outputs...)
	builder.SetTimeToLive(44552853)

	_, _, err = builder.UncheckedBuild()
	require.NoError(t, err)
}

func Test_TransactionBuilderWithPlutusDeployment(t *testing.T) {
	const txRaw = "84a40081825820ed985f36a35bde10d4476720b5153c57ed098d962f5f1425c25a97993df80630010182a300581d70626cad0064f02def9d61824cac7b9e9fef4292bcab4e439b78bc69bd011a003567e003d818590167820259016259015f010000223232323232323232533357340022930b19baf32357426aae78dd50009aba1357446aae78dd50011aba135573c6ea8004c8c8cc004004008894ccd55cf8008b0992999ab9a3370e600e6eacd5d09aba235573c6ea800520021001133003003357440046ae84004dd61aba1357446ae88010c8c8cc004004008894ccd55cf8008b0992999ab9a3370e600c646eacd5d09aba235573c6ea8004d5d09aba235573c6ea800520021001133003003357440046ae84004dd61aba1003375a00c46466600200244a666aae7c0045200015333573466ebcd55ce9aba10014c10a4954657374546f6b656e001375a6aae78d5d08008998010011aba200100222253335573e002290000a999ab9a3375e6aae74d5d0800a611e581c14b249936a64cbc96bde5a46e04174e7fb58b565103d0c3a32f8d61f0013300200237566aae78d5d080089998018018011aba200135573c0026ea8004d5d09aab9e375400382581d603c0e06d68646870080ebff18a5f6875be76856f2a7eea6b16648cf36000200031a02b3de4aa0f5f6"

	inputs := []TxInput{
		{
			Hash:  "ed985f36a35bde10d4476720b5153c57ed098d962f5f1425c25a97993df80630",
			Index: 1,
		},
	}

	outputs := []TxOutput{
		{
			Addr: "addr_test1vq7qupkksergwqyqa0l33f0ksad7w6zk72n7af43veyv7dsyux62h",
		},
	}

	plutusScript := PlutusScript{
		Type:        "PlutusScriptV2",
		Description: "",
		CborHex:     "59016259015f010000223232323232323232533357340022930b19baf32357426aae78dd50009aba1357446aae78dd50011aba135573c6ea8004c8c8cc004004008894ccd55cf8008b0992999ab9a3370e600e6eacd5d09aba235573c6ea800520021001133003003357440046ae84004dd61aba1357446ae88010c8c8cc004004008894ccd55cf8008b0992999ab9a3370e600c646eacd5d09aba235573c6ea8004d5d09aba235573c6ea800520021001133003003357440046ae84004dd61aba1003375a00c46466600200244a666aae7c0045200015333573466ebcd55ce9aba10014c10a4954657374546f6b656e001375a6aae78d5d08008998010011aba200100222253335573e002290000a999ab9a3375e6aae74d5d0800a611e581c14b249936a64cbc96bde5a46e04174e7fb58b565103d0c3a32f8d61f0013300200237566aae78d5d080089998018018011aba200135573c0026ea8004d5d09aab9e3754003",
	}

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	builder.SetTimeToLive(45342282).SetProtocolParameters(primeTestnetProtocolParams)
	builder.SetTestNetMagic(3311)
	builder.AddInputs(inputs...)

	_, plutusScriptAddr, err := builder.AddOutputWithPlutusScript(plutusScript, 3500000)
	require.NoError(t, err)
	require.NotEqual(t, "", plutusScriptAddr)

	builder.AddOutputs(outputs...)
	builder.SetFee(0)

	txRawRes, _, err := builder.UncheckedBuild()
	require.NoError(t, err)

	require.Equal(t, hex.EncodeToString(txRawRes), txRaw)
}

func Test_TxBuilder_UpdateOutputAmountAndRemoveOutput(t *testing.T) {
	t.Parallel()

	builder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer builder.Dispose()

	builder.AddOutputs(
		TxOutput{Addr: "0x1"},
		TxOutput{Addr: "0x2"},
		TxOutput{Addr: "0x3"},
		TxOutput{Addr: "0x4"},
	)

	require.Len(t, builder.outputs, 4)
	assert.Equal(t, uint64(0), builder.outputs[2].TxOutput.Amount)
	assert.Equal(t, uint64(0), builder.outputs[3].TxOutput.Amount)

	builder.UpdateOutputAmount(2, 200)
	builder.UpdateOutputAmount(-1, 500)

	assert.Equal(t, uint64(200), builder.outputs[2].TxOutput.Amount)
	assert.Equal(t, "0x3", builder.outputs[2].TxOutput.Addr)
	assert.Equal(t, uint64(500), builder.outputs[3].TxOutput.Amount)
	assert.Equal(t, "0x4", builder.outputs[3].TxOutput.Addr)

	builder.RemoveOutput(1)

	require.Len(t, builder.outputs, 3)
	assert.Equal(t, "0x1", builder.outputs[0].TxOutput.Addr)
	assert.Equal(t, uint64(0), builder.outputs[0].TxOutput.Amount)
	assert.Equal(t, "0x3", builder.outputs[1].TxOutput.Addr)
	assert.Equal(t, uint64(200), builder.outputs[1].TxOutput.Amount)
	assert.Equal(t, "0x4", builder.outputs[2].TxOutput.Addr)
	assert.Equal(t, uint64(500), builder.outputs[2].TxOutput.Amount)

	builder.RemoveOutput(0)

	require.Len(t, builder.outputs, 2)
	assert.Equal(t, "0x3", builder.outputs[0].TxOutput.Addr)
	assert.Equal(t, uint64(200), builder.outputs[0].TxOutput.Amount)
	assert.Equal(t, "0x4", builder.outputs[1].TxOutput.Addr)
	assert.Equal(t, uint64(500), builder.outputs[1].TxOutput.Amount)

	builder.RemoveOutput(1)

	require.Len(t, builder.outputs, 1)
	assert.Equal(t, "0x3", builder.outputs[0].TxOutput.Addr)
	assert.Equal(t, uint64(200), builder.outputs[0].TxOutput.Amount)

	builder.RemoveOutput(0)

	require.Len(t, builder.outputs, 0)
}

func Test_TxBuilder_CheckOutputs(t *testing.T) {
	t.Parallel()

	b, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer b.Dispose()

	b.AddOutputs(TxOutput{
		Addr:   "x1",
		Amount: 2,
	}, TxOutput{
		Addr:   "x2",
		Amount: 1,
	})

	require.NoError(t, b.CheckOutputs())

	b.AddOutputs(TxOutput{
		Addr:   "x3",
		Amount: 2,
	}, TxOutput{
		Addr:   "x4",
		Amount: 0,
	})

	require.Error(t, b.CheckOutputs(), errors.New("output (x4, 3) amount not specified"))
}

func TestCreateTxWitnessAndAssembleTxWitnesses(t *testing.T) {
	t.Parallel()

	const (
		skey        = "58800800c832ac40041bcbd83fc7b6be8f9a93c508d06f767518bad3266d62c3ad497d022a84b1b6663e0c3c62955c43bdfc333b3434ea232ab4e8c41d6b99c7ee12c73cd59dbfba2e07577ad69621e964d404c7bef56f69e1691438abd373561999899ccba5b358e8e3af736263283a472bb941c185ff4b523f532800766f1427c2"
		witnessData = "825820c73cd59dbfba2e07577ad69621e964d404c7bef56f69e1691438abd37356199958408233a747b14fc78ba32fbe8501b842d3290c591a565f589dbeec1c1e8b3dfe27de19002784c6c7020871fd07a5dd70e1003b6d1449255985c823464123085a00"
		txRaw       = "84a500818258201f55818892cc447cbf9fc27e04899ea98795538889555d3846a8071f4fdb75eb01018282581d70c4aab1955b120811d634e3a1b282ea090537d9e753842e8f46c280041a00200b2082583900712c77c7e146b95a569f2f7edf1dd81df2545edecb132701f17f84d4694c18049dcafc175d262c06eac9f52b86f205e38e8bfca6e6a545611a055e8308021a0002e908031a0152a319075820cb1b53bb62ee65e8ae893d04331dcc70d745298a32fcedf5ff9cc7a12d8471e3a0f5d90103a100a101a5616466766563746f726266611a0010c8e06173837828616464725f74657374317170636a63613738753972746a6b6a6b6e756868616863616d71776c793478287a376d6d39337866637037396c636634726666737671663877326c73743436663376716d34766e61781c6674736d6571746375773330373264653439673473737a333437377a61746662726964676562747881a26161827828766563746f725f7465737431766772677868347333356135706476306463347a6771333363726e33781934656d6e6b326537766e656e73663474657a7133746b6d396d616d1a000f4240"
		txWitness   = "84a500818258201f55818892cc447cbf9fc27e04899ea98795538889555d3846a8071f4fdb75eb01018282581d70c4aab1955b120811d634e3a1b282ea090537d9e753842e8f46c280041a00200b2082583900712c77c7e146b95a569f2f7edf1dd81df2545edecb132701f17f84d4694c18049dcafc175d262c06eac9f52b86f205e38e8bfca6e6a545611a055e8308021a0002e908031a0152a319075820cb1b53bb62ee65e8ae893d04331dcc70d745298a32fcedf5ff9cc7a12d8471e3a10081825820c73cd59dbfba2e07577ad69621e964d404c7bef56f69e1691438abd37356199958408233a747b14fc78ba32fbe8501b842d3290c591a565f589dbeec1c1e8b3dfe27de19002784c6c7020871fd07a5dd70e1003b6d1449255985c823464123085a00f5d90103a100a101a5616466766563746f726266611a0010c8e06173837828616464725f74657374317170636a63613738753972746a6b6a6b6e756868616863616d71776c793478287a376d6d39337866637037396c636634726666737671663877326c73743436663376716d34766e61781c6674736d6571746375773330373264653439673473737a333437377a61746662726964676562747881a26161827828766563746f725f7465737431766772677868347333356135706476306463347a6771333363726e33781934656d6e6b326537766e656e73663474657a7133746b6d396d616d1a000f4240"
	)

	skeyBytes, err := GetKeyBytes(skey)
	require.NoError(t, err)

	txRawBytes, err := hex.DecodeString(txRaw)
	require.NoError(t, err)

	wallet := NewWallet(skeyBytes, nil)

	txBuilder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(TestNetNetwork), eraName)
	require.NoError(t, err)

	defer txBuilder.Dispose()

	txWitnessBytes, err := txBuilder.CreateTxWitness(txRawBytes, wallet)
	require.NoError(t, err)

	require.Equal(t, witnessData, strings.TrimPrefix(hex.EncodeToString(txWitnessBytes), "8200"))

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	txHash, err := cliUtils.GetTxHash(txRawBytes)
	require.NoError(t, err)

	witness := TxWitnessRaw(txWitnessBytes)

	signature, vkey, err := witness.GetSignatureAndVKey()
	require.NoError(t, err)

	require.Equal(t, vkey, wallet.VerificationKey)

	txHashBytes, err := hex.DecodeString(txHash)
	require.NoError(t, err)

	require.NoError(t, VerifyMessage(txHashBytes, wallet.VerificationKey, signature))

	txFinal, err := txBuilder.AssembleTxWitnesses(txRawBytes, [][]byte{txWitnessBytes})
	require.NoError(t, err)

	require.Equal(t, txWitness, hex.EncodeToString(txFinal))
}

func TestCalculateMinUtxo(t *testing.T) {
	t.Parallel()

	token1, _ := NewTokenWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.4b6173685f546f6b656e", true)
	token2, _ := NewTokenWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.Route3", false)
	token3, _ := NewTokenWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.Route345", false)

	tokenAmount1 := NewTokenAmount(token1, 11_000_039)
	tokenAmount2 := NewTokenAmount(token2, 236_872_039)
	tokenAmount3 := NewTokenAmount(token3, 12_236_872_039)

	output := TxOutput{
		Addr:   "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u",
		Amount: uint64(1_000_000),
		Tokens: []TokenAmount{
			tokenAmount1, tokenAmount2, tokenAmount3,
		},
	}

	txBuilder, err := NewTxBuilderForEra(ResolveCardanoCliBinary(MainNetNetwork), eraName)
	require.NoError(t, err)

	defer txBuilder.Dispose()

	txBuilder.SetProtocolParameters(protocolParameters)

	minUtxo, err := txBuilder.CalculateMinUtxo(TxOutputWithRefScript{
		TxOutput: output,
	})
	require.NoError(t, err)

	require.Equal(t, uint64(1189560), minUtxo)

	output.Tokens[0].Amount = 2 // tokens amount does make a difference

	minUtxo, err = txBuilder.CalculateMinUtxo(TxOutputWithRefScript{
		TxOutput: output,
	})
	require.NoError(t, err)

	require.Equal(t, uint64(1172320), minUtxo)

	output.Tokens[1].Amount = 3 // tokens amount does make a difference

	minUtxo, err = txBuilder.CalculateMinUtxo(TxOutputWithRefScript{
		TxOutput: output,
	})
	require.NoError(t, err)

	require.Equal(t, uint64(1155080), minUtxo)

	output.Tokens = output.Tokens[:len(output.Tokens)-1]

	minUtxo, err = txBuilder.CalculateMinUtxo(TxOutputWithRefScript{
		TxOutput: output,
	})
	require.NoError(t, err)

	require.Equal(t, uint64(1077500), minUtxo)

	output.Tokens = nil

	minUtxo, err = txBuilder.CalculateMinUtxo(TxOutputWithRefScript{
		TxOutput: output,
	})
	require.NoError(t, err)

	require.Equal(t, uint64(849070), minUtxo)

	output.Amount = 3_600_000_348_100_893_234 // lovelace amount does not make a difference

	minUtxo, err = txBuilder.CalculateMinUtxo(TxOutputWithRefScript{
		TxOutput: output,
	})
	require.NoError(t, err)

	require.Equal(t, uint64(849070), minUtxo)
}
