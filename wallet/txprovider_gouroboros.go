package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	gouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
	"github.com/fxamacker/cbor/v2"
)

type TxProviderGoUroBoros struct {
	connection      *gouroboros.Connection
	networkMagic    uint32
	socketPath      string
	keepAlive       bool
	txSubmitTimeout time.Duration
}

var _ ITxProvider = (*TxProviderGoUroBoros)(nil)

func NewTxProviderGoUroBoros(
	networkMagic uint32, socketPath string, keepAlive bool, txSubmitTimeout time.Duration,
) (*TxProviderGoUroBoros, error) {
	connection, err := createGoUroBorosConnection(networkMagic, socketPath, keepAlive, txSubmitTimeout)
	if err != nil {
		return nil, err
	}

	return &TxProviderGoUroBoros{
		connection:      connection,
		networkMagic:    networkMagic,
		socketPath:      socketPath,
		keepAlive:       keepAlive,
		txSubmitTimeout: txSubmitTimeout,
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
	// create connection every time - otherwise wont work. WHY?!?!
	conn, err := createGoUroBorosConnection(
		b.networkMagic, b.socketPath, b.keepAlive, b.txSubmitTimeout)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	address, err := getLedgerAddress(addr)
	if err != nil {
		return nil, err
	}

	result, err := conn.LocalStateQuery().Client.GetUTxOByAddress([]ledger.Address{address})
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
	txType, err := ledger.DetermineTransactionType(txSigned)
	if err != nil {
		return fmt.Errorf("could not parse transaction to determine type: %s", err)
	}

	return b.connection.LocalTxSubmission().Client.SubmitTx(uint16(txType), txSigned)
}

func (b *TxProviderGoUroBoros) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	panic("not implemented") //nolint:gocritic
}

func createGoUroBorosConnection(
	networkMagic uint32, socketPath string, keepAlive bool, txSubmitTimeout time.Duration,
) (*gouroboros.Connection, error) {
	connection, err := gouroboros.NewConnection(
		gouroboros.WithNetworkMagic(networkMagic),
		gouroboros.WithNodeToNode(false),
		gouroboros.WithKeepAlive(keepAlive),
		gouroboros.WithLocalTxSubmissionConfig(
			localtxsubmission.NewConfig(
				localtxsubmission.WithTimeout(txSubmitTimeout*time.Second),
			)),
	)
	if err != nil {
		return nil, err
	}

	// dial node -> connect to node
	if err := connection.Dial("unix", socketPath); err != nil {
		return nil, err
	}

	return connection, nil
}

func getLedgerAddress(raw string) (addr ledger.Address, err error) {
	addrBase, err := NewAddress(raw)
	if err != nil {
		return addr, err
	}

	cborBytes, err := cbor.Marshal(addrBase.Bytes())
	if err != nil {
		return addr, err
	}

	err = addr.UnmarshalCBOR(cborBytes)

	return addr, err
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
