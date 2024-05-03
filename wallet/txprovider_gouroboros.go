package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"

	gouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
)

type TxProviderGoUroBoros struct {
	connection *gouroboros.Connection
}

var _ ITxProvider = (*TxProviderGoUroBoros)(nil)

func NewTxProviderGoUroBoros(
	networkMagic uint32, socketPath string, keepAlive bool,
) (*TxProviderGoUroBoros, error) {
	// create connection
	connection, err := gouroboros.NewConnection(
		gouroboros.WithNetworkMagic(networkMagic),
		gouroboros.WithNodeToNode(false),
		gouroboros.WithKeepAlive(keepAlive),
		gouroboros.WithFullDuplex(true),
	)
	if err != nil {
		return nil, err
	}

	// dial node -> connect to node
	if err := connection.Dial("unix", socketPath); err != nil {
		return nil, err
	}

	return &TxProviderGoUroBoros{
		connection: connection,
	}, nil
}

func (b *TxProviderGoUroBoros) Dispose() {
	_ = b.connection.Close() // log at least?
}

func (b *TxProviderGoUroBoros) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	protParams, err := b.connection.LocalStateQuery().Client.GetCurrentProtocolParams()
	if err != nil {
		return nil, err
	}

	return convertUroBorosProtocolParameters(protParams)
}

func (b *TxProviderGoUroBoros) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	address, err := ledger.NewAddress(addr)
	if err != nil {
		return nil, err
	}

	result, err := b.connection.LocalStateQuery().Client.GetUTxOByAddress([]ledger.Address{address})
	if err != nil {
		return nil, err
	}

	res := make([]Utxo, 0, len(result.Results))

	for key, val := range result.Results {
		res = append(res, Utxo{
			Hash:   key.Hash.String(),
			Index:  uint32(key.Idx),
			Amount: val.Amount(),
		})
	}

	return res, nil
}

func (b *TxProviderGoUroBoros) GetTip(ctx context.Context) (QueryTipData, error) {
	blockNum, err := b.connection.LocalStateQuery().Client.GetChainBlockNo()
	if err != nil {
		return QueryTipData{}, err
	}

	chainPoint, err := b.connection.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		return QueryTipData{}, err
	}

	epochNo, err := b.connection.LocalStateQuery().Client.GetEpochNo()
	if err != nil {
		return QueryTipData{}, err
	}

	return QueryTipData{
		Slot:  chainPoint.Slot,
		Hash:  hex.EncodeToString(chainPoint.Hash),
		Block: uint64(blockNum),
		Epoch: uint64(epochNo),
	}, nil
}

func (b *TxProviderGoUroBoros) SubmitTx(ctx context.Context, txSigned []byte) error {
	eraID, err := b.connection.LocalStateQuery().Client.GetCurrentEra()
	if err != nil {
		return err
	}

	return b.connection.LocalTxSubmission().Client.SubmitTx(uint16(eraID), txSigned)
}

func (b *TxProviderGoUroBoros) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	panic("not implemented") //nolint:gocritic
}

func convertUroBorosProtocolParameters(ps localstatequery.CurrentProtocolParamsResult) ([]byte, error) {
	switch v := ps.(type) {
	case ledger.BabbageProtocolParameters:
		priceMem, _ := v.ExecutionUnitPrices[0].Float64()
		priceSteps, _ := v.ExecutionUnitPrices[1].Float64()
		a0, _ := v.A0.Float64()
		rho, _ := v.Rho.Float64()
		tau, _ := v.Tau.Float64()
		resultJSON := map[string]interface{}{
			"extraPraosEntropy": nil,
			"decentralization":  nil,
			"protocolVersion": map[string]interface{}{
				"major": v.ProtocolMajor,
				"minor": v.ProtocolMinor,
			},
			"maxBlockHeaderSize":   v.MaxBlockHeaderSize,
			"maxBlockBodySize":     v.MaxBlockBodySize,
			"maxTxSize":            v.MaxTxSize,
			"txFeeFixed":           v.MinFeeB,
			"txFeePerByte":         v.MinFeeA,
			"stakeAddressDeposit":  v.KeyDeposit,
			"stakePoolDeposit":     v.PoolDeposit,
			"minPoolCost":          v.MinPoolCost,
			"poolRetireMaxEpoch":   v.MaxEpoch,
			"stakePoolTargetNum":   v.NOpt,
			"poolPledgeInfluence":  a0,
			"monetaryExpansion":    rho,
			"treasuryCut":          tau,
			"collateralPercentage": v.CollateralPercentage,
			"executionUnitPrices": map[string]interface{}{
				"priceMemory": priceMem,
				"priceSteps":  priceSteps,
			},
			"utxoCostPerByte": v.AdaPerUtxoByte,
			"minUTxOValue":    nil, // min_utxo? this was nil with cardano-cli
			"maxTxExecutionUnits": map[string]interface{}{
				"memory": v.MaxTxExecutionUnits[0],
				"steps":  v.MaxTxExecutionUnits[1],
			},
			"maxBlockExecutionUnits": map[string]interface{}{
				"memory": v.MaxBlockExecutionUnits[0],
				"steps":  v.MaxBlockExecutionUnits[1],
			},
			"maxCollateralInputs": v.MaxCollateralInputs,
			"maxValueSize":        v.MaxValueSize,
		}

		return json.Marshal(resultJSON)
	default:
		return nil, errors.New("invalid current protocol parameters")
	}
}
