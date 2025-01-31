package wallet

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	protocolParameters = []byte(`{"costModels":{"PlutusV1":[197209,0,1,1,396231,621,0,1,150000,1000,0,1,150000,32,2477736,29175,4,29773,100,29773,100,29773,100,29773,100,29773,100,29773,100,100,100,29773,100,150000,32,150000,32,150000,32,150000,1000,0,1,150000,32,150000,1000,0,8,148000,425507,118,0,1,1,150000,1000,0,8,150000,112536,247,1,150000,10000,1,136542,1326,1,1000,150000,1000,1,150000,32,150000,32,150000,32,1,1,150000,1,150000,4,103599,248,1,103599,248,1,145276,1366,1,179690,497,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,148000,425507,118,0,1,1,61516,11218,0,1,150000,32,148000,425507,118,0,1,1,148000,425507,118,0,1,1,2477736,29175,4,0,82363,4,150000,5000,0,1,150000,32,197209,0,1,1,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,150000,32,3345831,1,1],"PlutusV2":[205665,812,1,1,1000,571,0,1,1000,24177,4,1,1000,32,117366,10475,4,23000,100,23000,100,23000,100,23000,100,23000,100,23000,100,100,100,23000,100,19537,32,175354,32,46417,4,221973,511,0,1,89141,32,497525,14068,4,2,196500,453240,220,0,1,1,1000,28662,4,2,245000,216773,62,1,1060367,12586,1,208512,421,1,187000,1000,52998,1,80436,32,43249,32,1000,32,80556,1,57667,4,1000,10,197145,156,1,197145,156,1,204924,473,1,208896,511,1,52467,32,64832,32,65493,32,22558,32,16563,32,76511,32,196500,453240,220,0,1,1,69522,11687,0,1,60091,32,196500,453240,220,0,1,1,196500,453240,220,0,1,1,1159724,392670,0,2,806990,30482,4,1927926,82523,4,265318,0,4,0,85931,32,205665,812,1,1,41182,32,212342,32,31220,32,32696,32,43357,32,32247,32,38314,32,35892428,10,9462713,1021,10,38887044,32947,10]},"protocolVersion":{"major":7,"minor":0},"maxBlockHeaderSize":1100,"maxBlockBodySize":65536,"maxTxSize":16384,"txFeeFixed":155381,"txFeePerByte":44,"stakeAddressDeposit":0,"stakePoolDeposit":0,"minPoolCost":0,"poolRetireMaxEpoch":18,"stakePoolTargetNum":100,"poolPledgeInfluence":0,"monetaryExpansion":0.1,"treasuryCut":0.1,"collateralPercentage":150,"executionUnitPrices":{"priceMemory":0.0577,"priceSteps":0.0000721},"utxoCostPerByte":4310,"maxTxExecutionUnits":{"memory":16000000,"steps":10000000000},"maxBlockExecutionUnits":{"memory":80000000,"steps":40000000000},"maxCollateralInputs":3,"maxValueSize":5000,"extraPraosEntropy":null,"decentralization":null,"minUTxOValue":null}`)
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

	multiSigAddr, err := NewPolicyScriptAddress(TestNetNetwork, multisigPolicyID)
	require.NoError(t, err)

	multiSigFeeAddr, err := NewPolicyScriptAddress(TestNetNetwork, feeMultisigPolicyID)
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

	builder, err := NewTxBuilder(ResolveCardanoCliBinary(TestNetNetwork))
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

	assert.Equal(t, "84a50083825820098236134e0f2077a6434dd9d7727126fa8b3627bcab3ae030a194d46eded73e00825820d1fd0d772be7741d9bfaf0b037d02d2867a987ccba3e6ba2ee9aa2a861b7314502825820e99a5bde15aa05f24fcc04b7eabc1520d3397283b1ee720de9fe2653abbb0c9f00018382581d60244877c1aeefc7fd5405a6e14d927d91758d45e37c20fa2ac89cb1671a000f424082581d704aaad0f0626a8ce7b097497e542055b6520842ade881f980e002ae661a001e847682581d703ea4c4aef89a27f111e78464d7d6717b099f85ce27109ee9e5fbddec1a001a79bf021a00040ac103196dc0075820802e4d6f15ce98826886a5451e94855e77aae779cb341d3aab1e3bae4fb2f78da10182830304858200581c47344d5bd7b2fea56336ba789579705a944760032585ef64084c92db8200581c6837232854849427dae7c45892032d7ded136c5beb13c68fda635d878200581cd215701e2eb17c741b9d306cba553f9fbaaca1e12a5925a065b90fa88200581cf01018c1d8da54c2f557679243b09af1c4dd4d9c671512b01fa5f92b8200581cf0f4837b3a306752a2b3e52394168bc7391de3dce11364b723cc55cf830304858200581c06b4c7f5254d6395b527ac3de60c1d77194df7431d85fe55ca8f107d8200581c2368e8113bd5f32d713751791d29acee9e1b5a425b0454b963b2558b8200581c79df3577e4c7d7da04872c2182b8d8829d7b477912dbf35d89287c398200581ccba89c7084bf0ce4bf404346b668a7e83c8c9c250d1cafd8d8996e418200581cd6b67f93ffa4e2651271cc9bcdbdedb2539911266b534d9c163cba21f5d90103a100a200a36a6665655369676e65727305677369676e657273056474797065656d756c746904a26463697479684e6f76692053616464636f6d706845746865726e616c", hex.EncodeToString(txRaw))

	txHashUtil, err := cliUtils.GetTxHash(txRaw)
	require.NoError(t, err)

	require.Equal(t, "1b9298c51f4dc05c04cae37104124cfb76e9f98f04a7f6b8179cfe02913152ec", txHashUtil)
	require.Equal(t, txHash, txHashUtil)
}

func Test_TxBuilder_UpdateOutputAmountAndRemoveOutput(t *testing.T) {
	t.Parallel()

	builder, err := NewTxBuilder(ResolveCardanoCliBinary(TestNetNetwork))
	require.NoError(t, err)

	defer builder.Dispose()

	builder.AddOutputs(
		TxOutput{Addr: "0x1"},
		TxOutput{Addr: "0x2"},
		TxOutput{Addr: "0x3"},
		TxOutput{Addr: "0x4"},
	)

	require.Len(t, builder.outputs, 4)
	assert.Equal(t, uint64(0), builder.outputs[2].Amount)
	assert.Equal(t, uint64(0), builder.outputs[3].Amount)

	builder.UpdateOutputAmount(2, 200)
	builder.UpdateOutputAmount(-1, 500)

	assert.Equal(t, uint64(200), builder.outputs[2].Amount)
	assert.Equal(t, "0x3", builder.outputs[2].Addr)
	assert.Equal(t, uint64(500), builder.outputs[3].Amount)
	assert.Equal(t, "0x4", builder.outputs[3].Addr)

	builder.RemoveOutput(1)

	require.Len(t, builder.outputs, 3)
	assert.Equal(t, "0x1", builder.outputs[0].Addr)
	assert.Equal(t, uint64(0), builder.outputs[0].Amount)
	assert.Equal(t, "0x3", builder.outputs[1].Addr)
	assert.Equal(t, uint64(200), builder.outputs[1].Amount)
	assert.Equal(t, "0x4", builder.outputs[2].Addr)
	assert.Equal(t, uint64(500), builder.outputs[2].Amount)

	builder.RemoveOutput(0)

	require.Len(t, builder.outputs, 2)
	assert.Equal(t, "0x3", builder.outputs[0].Addr)
	assert.Equal(t, uint64(200), builder.outputs[0].Amount)
	assert.Equal(t, "0x4", builder.outputs[1].Addr)
	assert.Equal(t, uint64(500), builder.outputs[1].Amount)

	builder.RemoveOutput(1)

	require.Len(t, builder.outputs, 1)
	assert.Equal(t, "0x3", builder.outputs[0].Addr)
	assert.Equal(t, uint64(200), builder.outputs[0].Amount)

	builder.RemoveOutput(0)

	require.Len(t, builder.outputs, 0)
}

func Test_TxBuilder_CheckOutputs(t *testing.T) {
	t.Parallel()

	b, err := NewTxBuilder(ResolveCardanoCliBinary(TestNetNetwork))
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

func TestCreateTxWitness(t *testing.T) {
	t.Parallel()

	const (
		vkey        = "93eef6c081498a80b2dd7bb39654edbe7219c6262e102cf45c7105be0400bb7d"
		skey        = "a2f90889dd525397c2478e9e611c1790ca61b1edeebf73e1887f929585f8b2aa"
		witnessData = "82582093eef6c081498a80b2dd7bb39654edbe7219c6262e102cf45c7105be0400bb7d58405614996e84da65a3f774883a4cb29dc84667107078c96d2dac7186f587ea5faa6c027d235926f7deeffcf32dafcc2bafa90d680d727ddcc9ceb4a803cdd8030f"
	)

	vkeyBytes, _ := hex.DecodeString(vkey)
	skeyBytes, _ := hex.DecodeString(skey)
	wallet := NewWallet(vkeyBytes, skeyBytes)

	bytes, err := CreateTxWitness("8810020F", wallet)
	require.NoError(t, err)

	require.Equal(t, witnessData, hex.EncodeToString(bytes))
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
	txBuilder, err := NewTxBuilder(ResolveCardanoCliBinary(MainNetNetwork))
	require.NoError(t, err)
	defer txBuilder.Dispose()
	txBuilder.SetProtocolParameters(protocolParameters)
	minUtxo, err := txBuilder.CalculateMinUtxo(output)
	require.NoError(t, err)
	require.Equal(t, uint64(1189560), minUtxo)
	output.Tokens[0].Amount = 2 // tokens amount does make a difference
	minUtxo, err = txBuilder.CalculateMinUtxo(output)
	require.NoError(t, err)
	require.Equal(t, uint64(1172320), minUtxo)
	output.Tokens[1].Amount = 3 // tokens amount does make a difference
	minUtxo, err = txBuilder.CalculateMinUtxo(output)
	require.NoError(t, err)
	require.Equal(t, uint64(1155080), minUtxo)
	output.Tokens = output.Tokens[:len(output.Tokens)-1]
	minUtxo, err = txBuilder.CalculateMinUtxo(output)
	require.NoError(t, err)
	require.Equal(t, uint64(1077500), minUtxo)
	output.Tokens = nil
	minUtxo, err = txBuilder.CalculateMinUtxo(output)
	require.NoError(t, err)
	require.Equal(t, uint64(849070), minUtxo)
	output.Amount = 3_600_000_348_100_893_234 // lovelace amount does not make a difference
	minUtxo, err = txBuilder.CalculateMinUtxo(output)
	require.NoError(t, err)
	require.Equal(t, uint64(849070), minUtxo)
}
