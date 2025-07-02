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
		var tokens []TokenAmount

		if assets := val.Assets(); assets != nil {
			policies := assets.Policies()
			tokens = make([]TokenAmount, 0, len(policies))

			for _, policyIDRaw := range policies {
				policyID := policyIDRaw.String()

				for _, asset := range assets.Assets(policyIDRaw) {
					tokens = append(tokens, TokenAmount{
						Token:  NewToken(policyID, string(asset)),
						Amount: assets.Asset(policyIDRaw, asset),
					})
				}
			}
		}

		res = append(res, Utxo{
			Hash:   key.Hash.String(),
			Index:  uint32(key.Idx),
			Amount: val.Amount(),
			Tokens: tokens,
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

// GetStakePools implements ITxProvider.
func (b *TxProviderGoUroBoros) GetStakePools(ctx context.Context) ([]string, error) {
	panic("unimplemented")
}

// GetStakeAddressInfo implements ITxProvider.
func (b *TxProviderGoUroBoros) GetStakeAddressInfo(ctx context.Context, stakeAddress string) (QueryStakeAddressInfo, error) {
	panic("unimplemented")
}

func (b *TxProviderGoUroBoros) getConnection() (*gouroboros.Connection, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.connection == nil {
		b.config.logger.Debug("new connection created")

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
		b.config.logger.Debug("new point acquired")

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
	addrBase, err := NewCardanoAddressFromString(raw)
	if err != nil {
		return addr, err
	}

	cborBytes, err := cbor.Marshal(addrBase.GetBytes())
	if err != nil {
		return addr, err
	}

	err = addr.UnmarshalCBOR(cborBytes)

	return addr, err
}

func convertUroBorosProtocolParameters(ps localstatequery.CurrentProtocolParamsResult) ([]byte, error) {
	switch gupp := ps.(type) {
	case ledger.BabbageProtocolParameters:
		priceMem, _ := gupp.ExecutionCosts.MemPrice.Float64()
		priceSteps, _ := gupp.ExecutionCosts.StepPrice.Float64()
		a0, _ := gupp.A0.Float64()
		rho, _ := gupp.Rho.Float64()
		tau, _ := gupp.Tau.Float64()

		pp := ProtocolParameters{
			ProtocolVersion:      NewProtocolParametersVersion(uint64(gupp.ProtocolMajor), uint64(gupp.ProtocolMinor)),
			MaxBlockHeaderSize:   uint64(gupp.MaxBlockHeaderSize),
			MaxBlockBodySize:     uint64(gupp.MaxBlockBodySize),
			MaxTxSize:            uint64(gupp.MaxTxSize),
			TxFeeFixed:           uint64(gupp.MinFeeB),
			TxFeePerByte:         uint64(gupp.MinFeeA),
			StakeAddressDeposit:  uint64(gupp.KeyDeposit),
			StakePoolDeposit:     uint64(gupp.PoolDeposit),
			MinPoolCost:          gupp.MinPoolCost,
			PoolRetireMaxEpoch:   uint64(gupp.MaxEpoch),
			StakePoolTargetNum:   uint64(gupp.NOpt),
			PoolPledgeInfluence:  a0,
			MonetaryExpansion:    rho,
			TreasuryCut:          tau,
			CollateralPercentage: uint64(gupp.CollateralPercentage),
			ExecutionUnitPrices:  NewProtocolParametersPriceMemorySteps(priceMem, priceSteps),
			UtxoCostPerByte:      gupp.AdaPerUtxoByte,
			MaxTxExecutionUnits: NewProtocolParametersMemorySteps(
				uint64(gupp.MaxTxExUnits.Mem), uint64(gupp.MaxTxExUnits.Steps)),
			MaxBlockExecutionUnits: NewProtocolParametersMemorySteps(
				uint64(gupp.MaxBlockExUnits.Mem), uint64(gupp.MaxBlockExUnits.Steps)),
			MaxCollateralInputs: uint64(gupp.MaxCollateralInputs),
			MaxValueSize:        uint64(gupp.MaxValueSize),
			CostModels:          map[string][]int64{},
		}

		for scriptIndx, values := range gupp.CostModels {
			pp.CostModels[fmt.Sprintf("PlutusV%d", scriptIndx+1)] = values
		}

		return json.Marshal(pp)
	default:
		return nil, errors.New("invalid current protocol parameters")
	}
}
