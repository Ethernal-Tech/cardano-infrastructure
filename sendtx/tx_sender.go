package sendtx

import (
	"context"
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

	defaultPotentialFee     = 250_000
	defaultMaxInputsPerTx   = 16
	defaultTTLSlotNumberInc = 500
	splitStringLength       = 40
)

type BridgingTxReceiver struct {
	BridgingType BridgingType `json:"type"`
	Addr         string       `json:"addr"`
	Amount       uint64       `json:"amount"`
}

type TxSender struct {
	minAmountToBridge uint64
	potentialFee      uint64
	maxInputsPerTx    int
	chainConfigMap    map[string]ChainConfig
	retryOptions      []infracommon.RetryConfigOption
	utxosTransformer  IUtxosTransformer
}

type TxSenderOption func(*TxSender)

func NewTxSender(
	chainConfigMap map[string]ChainConfig,
	options ...TxSenderOption,
) *TxSender {
	txSnd := &TxSender{
		chainConfigMap: chainConfigMap,
		potentialFee:   defaultPotentialFee,
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
	exchangeRate ExchangeRate,
) ([]byte, string, *BridgingRequestMetadata, error) {
	metadata, err := txSnd.CreateMetadata(senderAddr, srcChainID, dstChainID, receivers, exchangeRate)
	if err != nil {
		return nil, "", nil, err
	}

	srcConfig := txSnd.chainConfigMap[srcChainID]
	outputCurrencyLovelace, outputNativeToken := GetOutputAmounts(metadata)

	metaDataRaw, err := metadata.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	txRaw, txHash, err := txSnd.createTx(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metaDataRaw, outputCurrencyLovelace, outputNativeToken)
	if err != nil {
		return nil, "", nil, err
	}

	return txRaw, txHash, metadata, nil
}

// CalculateBridgingTxFee returns calculated fee for bridging tx
func (txSnd *TxSender) CalculateBridgingTxFee(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	senderAddr string,
	receivers []BridgingTxReceiver,
	exchangeRate ExchangeRate,
) (uint64, *BridgingRequestMetadata, error) {
	metadata, err := txSnd.CreateMetadata(senderAddr, srcChainID, dstChainID, receivers, exchangeRate)
	if err != nil {
		return 0, nil, err
	}

	srcConfig := txSnd.chainConfigMap[srcChainID]
	outputCurrencyLovelace, outputNativeToken := GetOutputAmounts(metadata)

	metaDataRaw, err := metadata.Marshal()
	if err != nil {
		return 0, nil, err
	}

	fee, err := txSnd.calculateFee(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metaDataRaw, outputCurrencyLovelace, outputNativeToken)
	if err != nil {
		return 0, nil, err
	}

	return fee, metadata, nil
}

// CreateTxGeneric creates generic tx to one recipient and returns cbor of raw transaction data, tx hash and error
func (txSnd *TxSender) CreateTxGeneric(
	ctx context.Context,
	srcChainID string,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
) ([]byte, string, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, "", fmt.Errorf("chain %s config not found", srcChainID)
	}

	return txSnd.createTx(
		ctx, srcConfig, senderAddr, receiverAddr, metadata, outputCurrencyLovelace, outputNativeToken)
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
	senderAddr string, srcChainID, dstChainID string, receivers []BridgingTxReceiver, exchangeRate ExchangeRate,

) (*BridgingRequestMetadata, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, fmt.Errorf("source chain %s config not found", srcChainID)
	}

	dstConfig, existsDst := txSnd.chainConfigMap[dstChainID]
	if !existsDst {
		return nil, fmt.Errorf("destination chain %s config not found", dstChainID)
	}

	exchangeRateOnSrc := exchangeRate.Get(dstChainID, srcChainID)
	exchangeRateOnDst := exchangeRate.Get(srcChainID, dstChainID)
	feeSrcCurrencyLovelaceAmount := mul(dstConfig.BridgingFeeAmount, exchangeRateOnSrc)
	srcCurrencyLovelaceSum := feeSrcCurrencyLovelaceAmount
	txs := make([]BridgingRequestMetadataTransaction, len(receivers))

	for i, x := range receivers {
		switch x.BridgingType {
		case BridgingTypeNativeTokenOnSource:
			if x.Amount < dstConfig.MinUtxoValue {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, dstConfig.MinUtxoValue)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address:            infracommon.SplitString(x.Addr, splitStringLength),
				Amount:             x.Amount,
				IsNativeTokenOnSrc: metadataBoolTrue,
			}
		case BridgingTypeCurrencyOnSource:
			if x.Amount < srcConfig.MinUtxoValue {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, srcConfig.MinUtxoValue)
			}

			srcAdditionalInfo := mul(dstConfig.MinUtxoValue, exchangeRateOnSrc)
			srcCurrencyLovelaceSum += srcAdditionalInfo + x.Amount
			txs[i] = BridgingRequestMetadataTransaction{
				Address: infracommon.SplitString(x.Addr, splitStringLength),
				Amount:  x.Amount,
				Additional: &BridgingRequestMetadataCurrencyInfo{
					DestAmount: dstConfig.MinUtxoValue,
					SrcAmount:  srcAdditionalInfo,
				},
			}
		default:
			if x.Amount < txSnd.minAmountToBridge {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, txSnd.minAmountToBridge)
			}

			srcCurrencyLovelaceSum += x.Amount
			txs[i] = BridgingRequestMetadataTransaction{
				Address: infracommon.SplitString(x.Addr, splitStringLength),
				Amount:  x.Amount,
			}
		}
	}

	feeDstCurrencyLovelaceAmount := dstConfig.BridgingFeeAmount

	if srcCurrencyLovelaceSum < srcConfig.MinUtxoValue {
		feeSrcCurrencyLovelaceAmount += srcConfig.MinUtxoValue - srcCurrencyLovelaceSum
		feeDstCurrencyLovelaceAmount += mul(srcConfig.MinUtxoValue-srcCurrencyLovelaceSum, exchangeRateOnDst)
	}

	return &BridgingRequestMetadata{
		BridgingTxType:     bridgingMetaDataType,
		DestinationChainID: dstChainID,
		SenderAddr:         infracommon.SplitString(senderAddr, splitStringLength),
		Transactions:       txs,
		FeeAmount: BridgingRequestMetadataCurrencyInfo{
			SrcAmount:  feeSrcCurrencyLovelaceAmount,
			DestAmount: feeDstCurrencyLovelaceAmount,
		},
	}, nil
}

func (txSnd *TxSender) calculateFee(ctx context.Context,
	srcConfig ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
) (uint64, error) {
	builder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return 0, err
	}

	defer builder.Dispose()

	_, err = txSnd.populateTxBuilder(
		ctx, builder, srcConfig, senderAddr, receiverAddr, metadata, outputCurrencyLovelace, outputNativeToken)
	if err != nil {
		return 0, err
	}

	return builder.CalculateFee(1)
}

func (txSnd *TxSender) createTx(
	ctx context.Context,
	srcConfig ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
) ([]byte, string, error) {
	builder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, "", err
	}

	defer builder.Dispose()

	inputsSum, err := txSnd.populateTxBuilder(
		ctx, builder, srcConfig, senderAddr, receiverAddr, metadata, outputCurrencyLovelace, outputNativeToken)
	if err != nil {
		return nil, "", err
	}

	feeCurrencyLovelace, err := builder.CalculateFee(1)
	if err != nil {
		return nil, "", err
	}

	builder.SetFee(feeCurrencyLovelace)

	inputsSumCurrencyLovelace := inputsSum[cardanowallet.AdaTokenName]
	change := inputsSumCurrencyLovelace - outputCurrencyLovelace - feeCurrencyLovelace
	// handle overflow or insufficient amount
	if change != 0 && (change > inputsSumCurrencyLovelace || change < srcConfig.MinUtxoValue) {
		return []byte{}, "", fmt.Errorf("insufficient amount %d for %d or min utxo not satisfied",
			inputsSumCurrencyLovelace, outputCurrencyLovelace+feeCurrencyLovelace)
	}

	if change != 0 {
		builder.UpdateOutputAmount(-1, change)
	} else {
		builder.RemoveOutput(-1)
	}

	return builder.Build()
}

func (txSnd *TxSender) populateTxBuilder(
	ctx context.Context,
	builder *cardanowallet.TxBuilder,
	srcConfig ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
) (map[string]uint64, error) {
	queryTip, protocolParams, utxos, err := txSnd.getDynamicParameters(ctx, srcConfig, senderAddr)
	if err != nil {
		return nil, err
	}

	ttlSlotNumberInc := setOrDefault(srcConfig.TTLSlotNumberInc, defaultTTLSlotNumberInc)

	outputNativeTokens := []cardanowallet.TokenAmount(nil)
	conditions := map[string]uint64{
		cardanowallet.AdaTokenName: outputCurrencyLovelace + txSnd.potentialFee + srcConfig.MinUtxoValue,
	}
	srcNativeTokenFullName := srcConfig.NativeToken.String()

	if outputNativeToken != 0 {
		conditions[srcNativeTokenFullName] = outputNativeToken
	}

	if txSnd.utxosTransformer != nil {
		utxos = txSnd.utxosTransformer.TransformUtxos(utxos)
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, txSnd.maxInputsPerTx, 1)
	if err != nil {
		return nil, err
	}

	if txSnd.utxosTransformer != nil {
		txSnd.utxosTransformer.UpdateUtxos(inputs.Inputs)
	}

	if outputNativeToken != 0 {
		inputs.Sum[srcNativeTokenFullName] -= outputNativeToken
		if inputs.Sum[srcNativeTokenFullName] == 0 {
			delete(inputs.Sum, srcNativeTokenFullName)
		}

		outputNativeTokens = []cardanowallet.TokenAmount{
			cardanowallet.NewTokenAmount(srcConfig.NativeToken, outputNativeToken),
		}
	}

	outputRemainingTokens, err := cardanowallet.GetTokensFromSumMap(inputs.Sum)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokens from sum map. err: %w", err)
	}

	builder.SetMetaData(metadata).
		SetProtocolParameters(protocolParams).
		SetTimeToLive(queryTip.Slot+ttlSlotNumberInc).
		SetTestNetMagic(srcConfig.TestNetMagic).
		AddInputs(inputs.Inputs...).
		AddOutputs(cardanowallet.TxOutput{
			Addr:   receiverAddr,
			Amount: outputCurrencyLovelace,
			Tokens: outputNativeTokens,
		}, cardanowallet.TxOutput{
			Addr:   senderAddr,
			Tokens: outputRemainingTokens,
		})

	return inputs.Sum, nil
}

func (txSnd TxSender) getDynamicParameters(
	ctx context.Context, srcConfig ChainConfig, addr string,
) (qtd cardanowallet.QueryTipData, protocolParams []byte, utxos []cardanowallet.Utxo, err error) {
	protocolParams = srcConfig.ProtocolParameters
	if protocolParams == nil {
		protocolParams, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return srcConfig.TxProvider.GetProtocolParameters(ctx)
		}, txSnd.retryOptions...)
		if err != nil {
			return
		}
	}

	qtd, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) (cardanowallet.QueryTipData, error) {
		return srcConfig.TxProvider.GetTip(ctx)
	}, txSnd.retryOptions...)
	if err != nil {
		return
	}

	utxos, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]cardanowallet.Utxo, error) {
		return srcConfig.TxProvider.GetUtxos(ctx, addr)
	}, txSnd.retryOptions...)

	return qtd, protocolParams, utxos, err
}

func WithUtxosTransformer(utxosTransformer IUtxosTransformer) TxSenderOption {
	return func(txSnd *TxSender) {
		txSnd.utxosTransformer = utxosTransformer
	}
}

func WithPotentialFee(potentialFee uint64) TxSenderOption {
	return func(txSnd *TxSender) {
		txSnd.potentialFee = potentialFee
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
