package sendtx

import (
	"context"
	"errors"
	"fmt"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
)

type IUtxosTransformer interface {
	TransformUtxos(utxos []cardanowallet.Utxo) []cardanowallet.Utxo
	UpdateUtxos([]cardanowallet.TxInput)
}

type BridgingType byte

const (
	BridgingTypeNormal BridgingType = iota
	BridgingTypeNativeTokenOnSource
	BridgingTypeCurrencyOnSource

	defaultPotentialFee     = 400_000
	defaultMaxInputsPerTx   = 50
	defaultTTLSlotNumberInc = 500
)

type TokenExchangeConfig struct {
	DstChainID string `json:"dstChainID"`
	TokenName  string `json:"tokenName"`
}

type ChainConfig struct {
	CardanoCliBinary      string
	TxProvider            cardanowallet.ITxProvider
	MultiSigAddr          string
	TestNetMagic          uint
	TTLSlotNumberInc      uint64
	MinUtxoValue          uint64
	NativeTokens          []TokenExchangeConfig
	MinBridgingFeeAmount  uint64
	MinOperationFeeAmount uint64
	PotentialFee          uint64
	ProtocolParameters    []byte
}

type BridgingTxReceiver struct {
	BridgingType BridgingType `json:"type"`
	Addr         string       `json:"addr"`
	Amount       uint64       `json:"amount"`
}

type TxSender struct {
	minAmountToBridge uint64
	maxInputsPerTx    int
	chainConfigMap    map[string]ChainConfig
	retryOptions      []infracommon.RetryConfigOption
	utxosTransformer  IUtxosTransformer
}

type TxInfo struct {
	TxRaw                 []byte
	TxHash                string
	ChangeMinUtxoAmount   uint64
	ReceiverMinUtxoAmount uint64
}

type TxFeeInfo struct {
	Fee                   uint64
	ChangeMinUtxoAmount   uint64
	ReceiverMinUtxoAmount uint64
}

type TxSenderOption func(*TxSender)

func NewTxSender(
	chainConfigMap map[string]ChainConfig,
	options ...TxSenderOption,
) *TxSender {
	txSnd := &TxSender{
		chainConfigMap: chainConfigMap,
		maxInputsPerTx: defaultMaxInputsPerTx,
	}

	for _, config := range chainConfigMap {
		txSnd.minAmountToBridge = max(txSnd.minAmountToBridge, config.MinUtxoValue)
	}

	for _, opt := range options {
		opt(txSnd)
	}

	return txSnd
}

// CreateBridgingTx creates bridging tx and returns cbor of raw transaction data, tx hash and error
func (txSnd *TxSender) CreateBridgingTx(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	senderAddr string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (*TxInfo, *BridgingRequestMetadata, error) {
	metadata, err := txSnd.CreateMetadata(
		ctx, senderAddr, srcChainID, dstChainID, receivers, bridgingFee, operationFee)
	if err != nil {
		return nil, nil, err
	}

	srcConfig := txSnd.chainConfigMap[srcChainID]
	outputCurrencyLovelace, outputNativeToken := metadata.GetOutputAmounts()
	srcNativeTokenFullName := getNativeTokenNameForDstChainID(srcConfig.NativeTokens, dstChainID)

	metaDataRaw, err := metadata.Marshal()
	if err != nil {
		return nil, nil, err
	}

	txInfo, err := txSnd.createTx(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr,
		metaDataRaw, outputCurrencyLovelace, outputNativeToken, srcNativeTokenFullName)
	if err != nil {
		return nil, nil, err
	}

	return txInfo, metadata, nil
}

// CalculateBridgingTxFee returns calculated fee for bridging tx
func (txSnd *TxSender) CalculateBridgingTxFee(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	senderAddr string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (*TxFeeInfo, *BridgingRequestMetadata, error) {
	metadata, err := txSnd.CreateMetadata(
		ctx, senderAddr, srcChainID, dstChainID, receivers, bridgingFee, operationFee)
	if err != nil {
		return nil, nil, err
	}

	srcConfig := txSnd.chainConfigMap[srcChainID]
	outputCurrencyLovelace, outputNativeTokenAmount := metadata.GetOutputAmounts()
	srcNativeTokenFullName := getNativeTokenNameForDstChainID(srcConfig.NativeTokens, dstChainID)

	metaDataRaw, err := metadata.Marshal()
	if err != nil {
		return nil, nil, err
	}

	txFeeInfo, err := txSnd.calculateFee(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr,
		metaDataRaw, outputCurrencyLovelace, outputNativeTokenAmount, srcNativeTokenFullName)
	if err != nil {
		return nil, nil, err
	}

	return txFeeInfo, metadata, nil
}

// CreateTxGeneric creates generic tx to one recipient and returns cbor of raw transaction data, tx hash and error
func (txSnd *TxSender) CreateTxGeneric(
	ctx context.Context,
	srcChainID string,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeTokenAmount uint64,
	srcNativeTokenFullName string,
) (*TxInfo, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, fmt.Errorf("chain %s config not found", srcChainID)
	}

	return txSnd.createTx(
		ctx, srcConfig, senderAddr, receiverAddr, metadata,
		outputCurrencyLovelace, outputNativeTokenAmount, srcNativeTokenFullName)
}

func (txSnd *TxSender) SubmitTx(
	ctx context.Context, chainID string, txRaw []byte, cardanoWallet cardanowallet.ITxSigner,
) error {
	chainConfig, existsSrc := txSnd.chainConfigMap[chainID]
	if !existsSrc {
		return fmt.Errorf("%s chain config not found", chainID)
	}

	builder, err := cardanowallet.NewTxBuilder(chainConfig.CardanoCliBinary)
	if err != nil {
		return err
	}

	defer builder.Dispose()

	txSigned, err := builder.SignTx(txRaw, []cardanowallet.ITxSigner{cardanoWallet})
	if err != nil {
		return err
	}

	_, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) (bool, error) {
		return true, chainConfig.TxProvider.SubmitTx(ctx, txSigned)
	}, txSnd.retryOptions...)

	return err
}

func (txSnd *TxSender) CreateMetadata(
	ctx context.Context,
	senderAddr string,
	srcChainID string,
	dstChainID string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (*BridgingRequestMetadata, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, fmt.Errorf("source chain %s config not found", srcChainID)
	}

	dstConfig, existsDst := txSnd.chainConfigMap[dstChainID]
	if !existsDst {
		return nil, fmt.Errorf("destination chain %s config not found", dstChainID)
	}

	if bridgingFee < srcConfig.MinBridgingFeeAmount {
		return nil, fmt.Errorf("bridging fee is less than: %d", dstConfig.MinBridgingFeeAmount)
	}

	if operationFee < srcConfig.MinOperationFeeAmount {
		return nil, fmt.Errorf("operation fee is less than: %d", dstConfig.MinOperationFeeAmount)
	}

	txs := make([]BridgingRequestMetadataTransaction, len(receivers))

	for i, x := range receivers {
		switch x.BridgingType {
		case BridgingTypeNativeTokenOnSource:
			if x.Amount < dstConfig.MinUtxoValue {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, dstConfig.MinUtxoValue)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address:            addrToMetaDataAddr(x.Addr),
				Amount:             x.Amount,
				IsNativeTokenOnSrc: metadataBoolTrue,
			}

		case BridgingTypeCurrencyOnSource:
			if x.Amount < srcConfig.MinUtxoValue {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, srcConfig.MinUtxoValue)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address: addrToMetaDataAddr(x.Addr),
				Amount:  x.Amount,
			}
		default:
			if x.Amount < txSnd.minAmountToBridge {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, txSnd.minAmountToBridge)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address: addrToMetaDataAddr(x.Addr),
				Amount:  x.Amount,
			}
		}
	}

	return &BridgingRequestMetadata{
		BridgingTxType:     bridgingMetaDataType,
		DestinationChainID: dstChainID,
		SenderAddr:         addrToMetaDataAddr(senderAddr),
		Transactions:       txs,
		BridgingFee:        bridgingFee,
		OperationFee:       operationFee,
	}, nil
}

func (txSnd *TxSender) calculateFee(ctx context.Context,
	srcConfig ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeTokenAmount uint64,
	srcNativeTokenFullName string,
) (*TxFeeInfo, error) {
	builder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, err
	}

	defer builder.Dispose()

	_, changeMinUtxoAmount, receiverMinUtxoAmount, err := txSnd.populateTxBuilder(
		ctx, builder, srcConfig,
		senderAddr, receiverAddr,
		metadata, outputCurrencyLovelace,
		outputNativeTokenAmount, srcNativeTokenFullName)
	if err != nil {
		return nil, err
	}

	fee, err := builder.CalculateFee(1)
	if err != nil {
		return nil, err
	}

	return &TxFeeInfo{
		Fee:                   fee,
		ChangeMinUtxoAmount:   changeMinUtxoAmount,
		ReceiverMinUtxoAmount: receiverMinUtxoAmount,
	}, nil
}

func (txSnd *TxSender) createTx(
	ctx context.Context,
	srcConfig ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
	srcNativeTokenFullName string,
) (*TxInfo, error) {
	builder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, err
	}

	defer builder.Dispose()

	changeCurrencyLovelace, changeMinUtxoAmount, receiverMinUtxoAmount, err := txSnd.populateTxBuilder(
		ctx, builder, srcConfig,
		senderAddr, receiverAddr,
		metadata, outputCurrencyLovelace,
		outputNativeToken, srcNativeTokenFullName)
	if err != nil {
		return nil, err
	}

	feeCurrencyLovelace, err := builder.CalculateFee(1)
	if err != nil {
		return nil, err
	}

	builder.SetFee(feeCurrencyLovelace)

	change := changeCurrencyLovelace - feeCurrencyLovelace
	// handle overflow or insufficient amount
	if change != 0 && (change > changeCurrencyLovelace || change < changeMinUtxoAmount) {
		return nil,
			fmt.Errorf("insufficient remaining amount %d for fee %d, or minimum UTXO (%d) not satisfied",
				changeCurrencyLovelace, feeCurrencyLovelace, changeMinUtxoAmount)
	}

	if change != 0 {
		builder.UpdateOutputAmount(-1, change)
	} else {
		builder.RemoveOutput(-1)
	}

	txRaw, txHash, err := builder.Build()
	if err != nil {
		return nil, err
	}

	return &TxInfo{
		TxRaw:                 txRaw,
		TxHash:                txHash,
		ChangeMinUtxoAmount:   changeMinUtxoAmount,
		ReceiverMinUtxoAmount: receiverMinUtxoAmount,
	}, nil
}

func (txSnd *TxSender) populateTxBuilder(
	ctx context.Context,
	builder *cardanowallet.TxBuilder,
	srcConfig ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeTokenAmount uint64,
	srcNativeTokenFullName string,
) (uint64, uint64, uint64, error) {
	if err := txSnd.populateProtocolParameters(ctx, builder, srcConfig); err != nil {
		return 0, 0, 0, err
	}

	var (
		srcNativeTokenOutputs []cardanowallet.TokenAmount
		srcReceiverMinUtxo    uint64
	)

	if outputNativeTokenAmount != 0 {
		srcNativeToken, err := getNativeToken(srcNativeTokenFullName)
		if err != nil {
			return 0, 0, 0, err
		}

		srcNativeTokenFullName = srcNativeToken.String() // take the name used for maps
		srcNativeTokenOutputs = []cardanowallet.TokenAmount{
			cardanowallet.NewTokenAmount(srcNativeToken, outputNativeTokenAmount),
		}

		// calculate min lovelace amount (min utxo) for receiver output
		srcReceiverMinUtxo, err = cardanowallet.GetMinUtxoForSumMap(
			builder, receiverAddr, cardanowallet.GetTokensSumMap(srcNativeTokenOutputs...))
		if err != nil {
			return 0, 0, 0, err
		}

		outputCurrencyLovelace = max(outputCurrencyLovelace, srcReceiverMinUtxo, srcConfig.MinUtxoValue)
	}

	srcReceiverMinUtxo = max(srcConfig.MinUtxoValue, srcReceiverMinUtxo)

	utxos, err := infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]cardanowallet.Utxo, error) {
		return srcConfig.TxProvider.GetUtxos(ctx, senderAddr)
	}, txSnd.retryOptions...)
	if err != nil {
		return 0, 0, 0, err
	}

	// calculate minUtxo for change output
	potentialChangeTokenCost, err := cardanowallet.GetMinUtxoForSumMap(
		builder,
		senderAddr,
		cardanowallet.SubtractTokensFromSumMap(cardanowallet.GetUtxosSum(utxos), srcNativeTokenOutputs))
	if err != nil {
		return 0, 0, 0, err
	}

	srcChangeMinUtxo := max(srcConfig.MinUtxoValue, potentialChangeTokenCost)
	potentialFee := setOrDefault(srcConfig.PotentialFee, defaultPotentialFee)

	conditions := map[string]uint64{
		cardanowallet.AdaTokenName: outputCurrencyLovelace + potentialFee + srcChangeMinUtxo,
	}
	if outputNativeTokenAmount != 0 {
		conditions[srcNativeTokenFullName] = outputNativeTokenAmount
	}

	if txSnd.utxosTransformer != nil {
		utxos = txSnd.utxosTransformer.TransformUtxos(utxos)
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, txSnd.maxInputsPerTx, 1)
	if err != nil {
		return 0, 0, 0, err
	}

	if txSnd.utxosTransformer != nil {
		txSnd.utxosTransformer.UpdateUtxos(inputs.Inputs)
	}

	if outputNativeTokenAmount != 0 {
		inputs.Sum[srcNativeTokenFullName] -= outputNativeTokenAmount
		if inputs.Sum[srcNativeTokenFullName] == 0 {
			delete(inputs.Sum, srcNativeTokenFullName)
		}
	}

	outputRemainingTokens, err := cardanowallet.GetTokensFromSumMap(inputs.Sum)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create tokens from sum map. err: %w", err)
	}

	builder.SetMetaData(metadata).
		SetTestNetMagic(srcConfig.TestNetMagic).
		AddInputs(inputs.Inputs...).
		AddOutputs(cardanowallet.TxOutput{
			Addr:   receiverAddr,
			Amount: outputCurrencyLovelace,
			Tokens: srcNativeTokenOutputs,
		}, cardanowallet.TxOutput{
			Addr:   senderAddr,
			Tokens: outputRemainingTokens,
		})

	// populate ttl at the end because previous operations could take time
	if err := txSnd.populateTimeToLive(ctx, builder, srcConfig); err != nil {
		return 0, 0, 0, err
	}

	return inputs.Sum[cardanowallet.AdaTokenName] - outputCurrencyLovelace, srcChangeMinUtxo, srcReceiverMinUtxo, nil
}

func (txSnd *TxSender) populateProtocolParameters(
	ctx context.Context, builder *cardanowallet.TxBuilder, srcConfig ChainConfig,
) (err error) {
	protocolParams := srcConfig.ProtocolParameters
	if protocolParams == nil {
		protocolParams, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return srcConfig.TxProvider.GetProtocolParameters(ctx)
		}, txSnd.retryOptions...)
		if err != nil {
			return err
		}
	}

	builder.SetProtocolParameters(protocolParams)

	return nil
}

func (txSnd *TxSender) populateTimeToLive(
	ctx context.Context, builder *cardanowallet.TxBuilder, srcConfig ChainConfig,
) error {
	qtd, err := infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) (cardanowallet.QueryTipData, error) {
		return srcConfig.TxProvider.GetTip(ctx)
	}, txSnd.retryOptions...)
	if err != nil {
		return err
	}

	ttlSlotNumberInc := setOrDefault(srcConfig.TTLSlotNumberInc, defaultTTLSlotNumberInc)

	builder.SetTimeToLive(qtd.Slot + ttlSlotNumberInc)

	return nil
}

func getNativeTokenNameForDstChainID(
	nativeTokenDsts []TokenExchangeConfig, dstChainID string,
) string {
	for _, nativeTokenDst := range nativeTokenDsts {
		if nativeTokenDst.DstChainID == dstChainID {
			return nativeTokenDst.TokenName
		}
	}

	return ""
}

func getNativeToken(fullName string) (token cardanowallet.Token, err error) {
	if fullName == "" {
		return token, errors.New("native token name not specified")
	}

	token, err = cardanowallet.NewTokenWithFullName(fullName, true)
	if err == nil {
		return token, nil
	}

	token, err = cardanowallet.NewTokenWithFullName(fullName, false)
	if err != nil {
		return token, fmt.Errorf("invalid native token name: %w", err)
	}

	return token, nil
}

func WithUtxosTransformer(utxosTransformer IUtxosTransformer) TxSenderOption {
	return func(txSnd *TxSender) {
		txSnd.utxosTransformer = utxosTransformer
	}
}

func WithMaxInputsPerTx(maxInputsPerTx int) TxSenderOption {
	return func(txSnd *TxSender) {
		txSnd.maxInputsPerTx = maxInputsPerTx
	}
}

func WithRetryOptions(retryOptions []infracommon.RetryConfigOption) TxSenderOption {
	return func(txSnd *TxSender) {
		txSnd.retryOptions = retryOptions
	}
}
