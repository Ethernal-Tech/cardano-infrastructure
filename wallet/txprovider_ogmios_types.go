package wallet

import (
	"encoding/json"
	"strconv"
	"strings"
)

type queryLedgerStateUtxoParams struct {
	Addresses []string `json:"addresses"`
}

type queryLedgerStateUtxo struct {
	Jsonrpc string                     `json:"jsonrpc"`
	Method  string                     `json:"method"`
	Params  queryLedgerStateUtxoParams `json:"params"`
	ID      interface{}                `json:"id"`
}

type queryLedgerStateUtxoResponseResultTransaction struct {
	ID string `json:"id"`
}

type queryLedgerStateUtxoResponseResultValueAda struct {
	Lovelace uint `json:"lovelace"`
}

type queryLedgerStateUtxoResponseResultValue struct {
	Ada queryLedgerStateUtxoResponseResultValueAda `json:"ada"`
}

type queryLedgerStateUtxoResponseResult struct {
	Transaction queryLedgerStateUtxoResponseResultTransaction `json:"transaction"`
	Index       uint                                          `json:"index"`
	Address     string                                        `json:"address"`
	Value       queryLedgerStateUtxoResponseResultValue       `json:"value"`
}

type queryLedgerStateUtxoResponse struct {
	Jsonrpc string                               `json:"jsonrpc"`
	Method  string                               `json:"method"`
	Result  []queryLedgerStateUtxoResponseResult `json:"result"`
	ID      interface{}                          `json:"id"`
}

// Define the types for the protocol parameters response
type queryLedgerStateProtocolParametersResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  struct {
		MinFeeCoefficient uint `json:"minFeeCoefficient"`
		MinFeeConstant    struct {
			Ada struct {
				Lovelace uint `json:"lovelace"`
			} `json:"ada"`
		} `json:"minFeeConstant"`
		MaxBlockBodySize struct {
			Bytes uint `json:"bytes"`
		} `json:"maxBlockBodySize"`
		MaxBlockHeaderSize struct {
			Bytes uint `json:"bytes"`
		} `json:"maxBlockHeaderSize"`
		MaxTransactionSize struct {
			Bytes uint `json:"bytes"`
		} `json:"maxTransactionSize"`
		StakeCredentialDeposit struct {
			Ada struct {
				Lovelace uint `json:"lovelace"`
			} `json:"ada"`
		} `json:"stakeCredentialDeposit"`
		StakePoolDeposit struct {
			Ada struct {
				Lovelace uint `json:"lovelace"`
			} `json:"ada"`
		} `json:"stakePoolDeposit"`
		StakePoolRetirementEpochBound uint   `json:"stakePoolRetirementEpochBound"`
		DesiredNumberOfStakePools     uint   `json:"desiredNumberOfStakePools"`
		StakePoolPledgeInfluence      string `json:"stakePoolPledgeInfluence"`
		MonetaryExpansion             string `json:"monetaryExpansion"`
		TreasuryExpansion             string `json:"treasuryExpansion"`
		MinStakePoolCost              struct {
			Ada struct {
				Lovelace uint `json:"lovelace"`
			} `json:"ada"`
		} `json:"minStakePoolCost"`
		MinUtxoDepositConstant struct {
			Ada struct {
				Lovelace uint `json:"lovelace"`
			} `json:"ada"`
		} `json:"minUtxoDepositConstant"`
		MinUtxoDepositCoefficient uint              `json:"minUtxoDepositCoefficient"`
		PlutusCostModels          map[string][]uint `json:"plutusCostModels"`
		ScriptExecutionPrices     struct {
			Memory string `json:"memory"`
			CPU    string `json:"cpu"`
		} `json:"scriptExecutionPrices"`
		MaxExecutionUnitsPerTransaction struct {
			Memory uint `json:"memory"`
			CPU    uint `json:"cpu"`
		} `json:"maxExecutionUnitsPerTransaction"`
		MaxExecutionUnitsPerBlock struct {
			Memory uint `json:"memory"`
			CPU    uint `json:"cpu"`
		} `json:"maxExecutionUnitsPerBlock"`
		MaxValueSize struct {
			Bytes uint `json:"bytes"`
		} `json:"maxValueSize"`
		CollateralPercentage uint `json:"collateralPercentage"`
		MaxCollateralInputs  uint `json:"maxCollateralInputs"`
		Version              struct {
			Major uint `json:"major"`
			Minor uint `json:"minor"`
		} `json:"version"`
	} `json:"result"`
	ID interface{} `json:"id"`
}

type queryLedgerStateTipResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  struct {
		Slot uint   `json:"slot"`
		ID   string `json:"id"`
	} `json:"result"`
	ID interface{} `json:"id"`
}

type queryLedgerStateProtocolParameters queryLedgerState
type queryLedgerStateTip queryLedgerState

type queryLedgerState struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      interface{} `json:"id"`
}

type submitTransactionParamsTransaction struct {
	CBOR string `json:"cbor"`
}

type submitTransactionParams struct {
	Transaction submitTransactionParamsTransaction `json:"transaction"`
}

type submitTransaction struct {
	Jsonrpc string                  `json:"jsonrpc"`
	Method  string                  `json:"method"`
	Params  submitTransactionParams `json:"params"`
	ID      interface{}             `json:"id"`
}

// Define the types for the submit transaction response
type submitTransactionResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Result  struct {
		Transaction struct {
			ID string `json:"id"`
		} `json:"transaction"`
	} `json:"result"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			MissingScripts []string `json:"missingScripts"`
		} `json:"data"`
	} `json:"error"`
	ID interface{} `json:"id"`
}

func convertProtocolParametersOgmios(params queryLedgerStateProtocolParametersResponse) ([]byte, error) {
	poolPledgeInfluence, _ := strconv.ParseFloat(strings.TrimSpace(params.Result.StakePoolPledgeInfluence), 64)
	monetaryExpansion, _ := strconv.ParseFloat(strings.TrimSpace(params.Result.MonetaryExpansion), 64)
	treasuryCut, _ := strconv.ParseFloat(strings.TrimSpace(params.Result.TreasuryExpansion), 64)
	priceSteps, _ := strconv.ParseFloat(strings.TrimSpace(params.Result.ScriptExecutionPrices.CPU), 64)
	priceMemory, _ := strconv.ParseFloat(strings.TrimSpace(params.Result.ScriptExecutionPrices.Memory), 64)

	resultJSON := map[string]interface{}{
		"extraPraosEntropy": nil,
		"decentralization":  nil,
		"protocolVersion": map[string]interface{}{
			"major": params.Result.Version.Major,
			"minor": params.Result.Version.Minor,
		},
		"maxBlockHeaderSize":   params.Result.MaxBlockHeaderSize.Bytes,
		"maxBlockBodySize":     params.Result.MaxBlockBodySize.Bytes,
		"maxTxSize":            params.Result.MaxTransactionSize.Bytes,
		"txFeeFixed":           params.Result.MinFeeConstant.Ada.Lovelace,
		"txFeePerByte":         params.Result.MinFeeCoefficient,
		"stakeAddressDeposit":  params.Result.StakeCredentialDeposit.Ada.Lovelace,
		"stakePoolDeposit":     params.Result.StakePoolDeposit.Ada.Lovelace,
		"minPoolCost":          params.Result.MinStakePoolCost.Ada.Lovelace,
		"poolRetireMaxEpoch":   params.Result.StakePoolRetirementEpochBound,
		"stakePoolTargetNum":   params.Result.DesiredNumberOfStakePools,
		"poolPledgeInfluence":  poolPledgeInfluence,
		"monetaryExpansion":    monetaryExpansion,
		"treasuryCut":          treasuryCut,
		"collateralPercentage": params.Result.CollateralPercentage,
		"executionUnitPrices": map[string]interface{}{
			"priceMemory": priceMemory,
			"priceSteps":  priceSteps,
		},
		"utxoCostPerByte": params.Result.MinUtxoDepositCoefficient, // coins_per_utxo_size ?
		"minUTxOValue":    nil,                                     // min_utxo? this was nil with cardano-cli
		"maxTxExecutionUnits": map[string]interface{}{
			"memory": params.Result.MaxExecutionUnitsPerTransaction.Memory,
			"steps":  params.Result.MaxExecutionUnitsPerTransaction.CPU,
		},
		"maxBlockExecutionUnits": map[string]interface{}{
			"memory": params.Result.MaxExecutionUnitsPerBlock.Memory,
			"steps":  params.Result.MaxExecutionUnitsPerBlock.CPU,
		},
		"maxCollateralInputs": params.Result.MaxCollateralInputs,
		"maxValueSize":        params.Result.MaxValueSize.Bytes,
	}

	//nolint
	// TODO: "costModels": "PlutusV1" ...

	return json.Marshal(resultJSON)
}
