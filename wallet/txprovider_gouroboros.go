package wallet

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	gouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/gouroboros/protocol/localtxsubmission"
	"github.com/fxamacker/cbor/v2"
	"github.com/hashicorp/go-hclog"
)

type txProviderGoUroBorosConfig struct {
	networkMagic    uint32
	socketPath      string
	keepAlive       bool
	txSubmitTimeout time.Duration
	acquireTimeout  time.Duration
	logger          hclog.Logger
}

type TxProviderGoUroBorosOption func(*txProviderGoUroBorosConfig)

func WithTxProviderGoUroBorosKeepAlive(keepAlive bool) TxProviderGoUroBorosOption {
	return func(c *txProviderGoUroBorosConfig) {
		c.keepAlive = keepAlive
	}
}

func WithTxProviderGoUroBorosTxSubmitTimeout(txSubmitTimeout time.Duration) TxProviderGoUroBorosOption {
	return func(c *txProviderGoUroBorosConfig) {
		c.txSubmitTimeout = txSubmitTimeout
	}
}

func WithTxProviderGoUroBorosAcquireTimeout(acquireTimeout time.Duration) TxProviderGoUroBorosOption {
	return func(c *txProviderGoUroBorosConfig) {
		c.acquireTimeout = acquireTimeout
	}
}

func WithTxProviderGoUroBorosLogger(logger hclog.Logger) TxProviderGoUroBorosOption {
	return func(c *txProviderGoUroBorosConfig) {
		c.logger = logger
	}
}

type TxProviderGoUroBoros struct {
	config txProviderGoUroBorosConfig

	connection *gouroboros.Connection
	closeCh    chan struct{}
	errChan    chan error
	lock       sync.Mutex

	lastAcquiredTime time.Time
}

var _ ITxProvider = (*TxProviderGoUroBoros)(nil)

func NewTxProviderGoUroBoros(
	networkMagic uint32, socketPath string, options ...TxProviderGoUroBorosOption,
) *TxProviderGoUroBoros {
	config := txProviderGoUroBorosConfig{
		networkMagic:    networkMagic,
		socketPath:      socketPath,
		keepAlive:       true,
		txSubmitTimeout: 5 * time.Second,
		acquireTimeout:  2 * time.Second,
		logger:          hclog.NewNullLogger(),
	}

	for _, op := range options {
		op(&config)
	}

	txProvider := &TxProviderGoUroBoros{
		closeCh: make(chan struct{}),
		config:  config,
	}

	go txProvider.loop()

	return txProvider
}

func (b *TxProviderGoUroBoros) Dispose() {
	close(b.closeCh)

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.connection != nil {
		b.connection.Close()
	}
}

func (b *TxProviderGoUroBoros) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	conn, err := b.getConnection()
	if err != nil {
		return nil, err
	}

	if err := b.acquire(); err != nil {
		return nil, err
	}

	protParams, err := conn.LocalStateQuery().Client.GetCurrentProtocolParams()
	if err != nil {
		return nil, err
	}

	return convertUroBorosProtocolParameters(protParams)
}

func (b *TxProviderGoUroBoros) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	conn, err := b.getConnection()
	if err != nil {
		return nil, err
	}

	if err := b.acquire(); err != nil {
		return nil, err
	}

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
	conn, err := b.getConnection()
	if err != nil {
		return QueryTipData{}, err
	}

	if err := b.acquire(); err != nil {
		return QueryTipData{}, err
	}

	blockNum, err := conn.LocalStateQuery().Client.GetChainBlockNo()
	if err != nil {
		return QueryTipData{}, err
	}

	chainPoint, err := conn.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		return QueryTipData{}, err
	}

	epochNo, err := conn.LocalStateQuery().Client.GetEpochNo()
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
		return fmt.Errorf("could not parse transaction to determine type: %w", err)
	}

	conn, err := b.getConnection()
	if err != nil {
		return err
	}

	return conn.LocalTxSubmission().Client.SubmitTx(uint16(txType), txSigned) //nolint:gosec
}

func (b *TxProviderGoUroBoros) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	panic("not implemented") //nolint:gocritic
}

func (b *TxProviderGoUroBoros) getConnection() (*gouroboros.Connection, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.connection == nil {
		b.config.logger.Debug("new connection created", "time", time.Now().UTC())

		b.errChan = make(chan error) // create new channel because old one is closed

		conn, err := createGoUroBorosConnection(
			b.config.networkMagic, b.config.socketPath,
			b.config.keepAlive, b.config.txSubmitTimeout, b.errChan)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve connection: %w", err)
		}

		b.connection = conn
	}

	return b.connection, nil
}

func (b *TxProviderGoUroBoros) acquire() error {
	b.lock.Lock()
	currentTime := time.Now().UTC()
	isTimeOut := b.config.acquireTimeout == 0 || currentTime.Sub(b.lastAcquiredTime) > b.config.acquireTimeout

	if isTimeOut {
		b.lastAcquiredTime = currentTime
	}

	b.lock.Unlock()

	if isTimeOut {
		b.config.logger.Debug("new point acquired", "time", currentTime)

		return b.connection.LocalStateQuery().Client.Acquire(nil)
	}

	return nil
}

func (b *TxProviderGoUroBoros) loop() {
	for {
		select {
		case <-b.closeCh:
			return // close routine
		case <-b.errChan:
			b.lock.Lock()
			b.connection = nil
			b.lock.Unlock()
		}
	}
}

func createGoUroBorosConnection(
	networkMagic uint32, socketPath string, keepAlive bool, txSubmitTimeout time.Duration, errChan chan error,
) (*gouroboros.Connection, error) {
	connection, err := gouroboros.NewConnection(
		gouroboros.WithNetworkMagic(networkMagic),
		gouroboros.WithNodeToNode(false),
		gouroboros.WithKeepAlive(keepAlive),
		gouroboros.WithErrorChan(errChan),
		gouroboros.WithLocalTxSubmissionConfig(
			localtxsubmission.NewConfig(
				localtxsubmission.WithTimeout(txSubmitTimeout),
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
