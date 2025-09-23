package wallet

type ProtocolParametersVersion struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
}

func NewProtocolParametersVersion(major, minor uint64) ProtocolParametersVersion {
	return ProtocolParametersVersion{
		Major: major,
		Minor: minor,
	}
}

type ProtocolParametersMemorySteps struct {
	Memory uint64 `json:"memory"`
	Steps  uint64 `json:"steps"`
}

func NewProtocolParametersMemorySteps(memory, steps uint64) ProtocolParametersMemorySteps {
	return ProtocolParametersMemorySteps{
		Memory: memory,
		Steps:  steps,
	}
}

type ProtocolParametersPriceMemorySteps struct {
	PriceMemory float64 `json:"priceMemory"`
	PriceSteps  float64 `json:"priceSteps"`
}

func NewProtocolParametersPriceMemorySteps(memory, steps float64) ProtocolParametersPriceMemorySteps {
	return ProtocolParametersPriceMemorySteps{
		PriceMemory: memory,
		PriceSteps:  steps,
	}
}

type VotingThresholds struct {
	CommitteeNoConfidence float64 `json:"committeeNoConfidence"`
	CommitteeNormal       float64 `json:"committeeNormal"`
	HardForkInitiation    float64 `json:"hardForkInitiation"`
	MotionNoConfidence    float64 `json:"motionNoConfidence"`
	PPEconomicGroup       float64 `json:"ppEconomicGroup"`
	PPGovGroup            float64 `json:"ppGovGroup"`
	PPNetworkGroup        float64 `json:"ppNetworkGroup"`
	PPTechnicalGroup      float64 `json:"ppTechnicalGroup"`
	TreasuryWithdrawal    float64 `json:"treasuryWithdrawal"`
	UpdateToConstitution  float64 `json:"updateToConstitution"`
}

type PoolVotingThresholds struct {
	CommitteeNoConfidence float64 `json:"committeeNoConfidence"`
	CommitteeNormal       float64 `json:"committeeNormal"`
	HardForkInitiation    float64 `json:"hardForkInitiation"`
	MotionNoConfidence    float64 `json:"motionNoConfidence"`
	PPSecurityGroup       float64 `json:"ppSecurityGroup"`
}

type ProtocolParameters struct {
	CostModels             map[string][]int64                 `json:"costModels"`
	ProtocolVersion        ProtocolParametersVersion          `json:"protocolVersion"`
	MaxBlockHeaderSize     uint64                             `json:"maxBlockHeaderSize"`
	MaxBlockBodySize       uint64                             `json:"maxBlockBodySize"`
	MaxTxSize              uint64                             `json:"maxTxSize"`
	TxFeeFixed             uint64                             `json:"txFeeFixed"`
	TxFeePerByte           uint64                             `json:"txFeePerByte"`
	StakeAddressDeposit    uint64                             `json:"stakeAddressDeposit"`
	StakePoolDeposit       uint64                             `json:"stakePoolDeposit"`
	MinPoolCost            uint64                             `json:"minPoolCost"`
	PoolRetireMaxEpoch     uint64                             `json:"poolRetireMaxEpoch"`
	StakePoolTargetNum     uint64                             `json:"stakePoolTargetNum"`
	PoolPledgeInfluence    float64                            `json:"poolPledgeInfluence"`
	MonetaryExpansion      float64                            `json:"monetaryExpansion"`
	TreasuryCut            float64                            `json:"treasuryCut"`
	CollateralPercentage   uint64                             `json:"collateralPercentage"`
	ExecutionUnitPrices    ProtocolParametersPriceMemorySteps `json:"executionUnitPrices"`
	UtxoCostPerByte        uint64                             `json:"utxoCostPerByte"`
	MaxTxExecutionUnits    ProtocolParametersMemorySteps      `json:"maxTxExecutionUnits"`
	MaxBlockExecutionUnits ProtocolParametersMemorySteps      `json:"maxBlockExecutionUnits"`
	MaxCollateralInputs    uint64                             `json:"maxCollateralInputs"`
	MaxValueSize           uint64                             `json:"maxValueSize"`
	// conway
	PoolVotingThresholds       *PoolVotingThresholds `json:"poolVotingThresholds"`
	DRepVotingThresholds       *VotingThresholds     `json:"dRepVotingThresholds"`
	DRepActivity               *uint64               `json:"dRepActivity"`
	DRepDeposit                *uint64               `json:"dRepDeposit"`
	GovActionDeposit           *uint64               `json:"govActionDeposit"`
	GovActionLifetime          *uint64               `json:"govActionLifetime"`
	MinFeeRefScriptCostPerByte *float64              `json:"minFeeRefScriptCostPerByte"`
	CommitteeMaxTermLength     *uint64               `json:"committeeMaxTermLength"`
	CommitteeMinSize           *uint64               `json:"committeeMinSize"`
	// parameters required by cli for some reason but they are always nil
	ExtraPraosEntropy *uint64 `json:"extraPraosEntropy"`
	Decentralization  *uint64 `json:"decentralization"`
	MinUTxOValue      *uint64 `json:"minUTxOValue"`
}
