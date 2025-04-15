package sendtx

import (
	"context"
	"errors"
	"fmt"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
)

type TxSender struct {
	minAmountToBridge uint64
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
	txBuilder, srcConfig, metadata, metadataRaw, outputLovelace, outputNativeToken, err := txSnd.initBridgingTx(
		srcChainID, dstChainID, senderAddr, receivers, bridgingFee, operationFee)
	if err != nil {
		return nil, nil, err
	}

	defer txBuilder.Dispose()

	txInfo, err := txSnd.createTx(
		ctx, txBuilder, srcConfig, senderAddr, srcConfig.MultiSigAddr, metadataRaw, outputLovelace, outputNativeToken)
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
	txBuilder, srcConfig, metadata, metadataRaw, outputLovelace, outputNativeToken, err := txSnd.initBridgingTx(
		srcChainID, dstChainID, senderAddr, receivers, bridgingFee, operationFee)
	if err != nil {
		return nil, nil, err
	}

	defer txBuilder.Dispose()

	txFeeInfo, err := txSnd.calculateFee(
		ctx, txBuilder, srcConfig, senderAddr, srcConfig.MultiSigAddr, metadataRaw, outputLovelace, outputNativeToken)
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
	outputLovelace uint64,
	outputNativeToken *cardanowallet.TokenAmount,
) (*TxInfo, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, fmt.Errorf("chain %s config not found", srcChainID)
	}

	txBuilder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, err
	}

	defer txBuilder.Dispose()

	outputLovelace, err = fixLovelaceOutput(
		txBuilder, &srcConfig, receiverAddr, outputNativeToken, outputLovelace)
	if err != nil {
		return nil, err
	}

	return txSnd.createTx(
		ctx, txBuilder, &srcConfig, senderAddr, receiverAddr, metadata, outputLovelace, outputNativeToken)
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
	senderAddr string,
	srcConfig *ChainConfig,
	dstConfig *ChainConfig,
	dstChainID string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (*BridgingRequestMetadata, error) {
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

func (txSnd *TxSender) GetBridgingFee(
	srcChainID string,
	dstChainID string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (uint64, error) {
	srcConfig, _, err := txSnd.getConfigs(srcChainID, dstChainID)
	if err != nil {
		return 0, err
	}

	txBuilder, _, _, bridgingFee, err := txSnd.prepareBridgingTx(
		srcConfig, dstChainID, receivers, bridgingFee, operationFee)
	if err != nil {
		return 0, err
	}

	defer txBuilder.Dispose()

	return bridgingFee, nil
}

func (txSnd *TxSender) initBridgingTx(
	srcChainID string,
	dstChainID string,
	senderAddr string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (
	*cardanowallet.TxBuilder, *ChainConfig,
	*BridgingRequestMetadata, []byte,
	uint64, *cardanowallet.TokenAmount, error,
) {
	srcConfig, dstConfig, err := txSnd.getConfigs(srcChainID, dstChainID)
	if err != nil {
		return nil, nil, nil, nil, 0, nil, err
	}

	txBuilder, outputLovelace, outputNativeToken, bridgingFee, err := txSnd.prepareBridgingTx(
		srcConfig, dstChainID, receivers, bridgingFee, operationFee)
	if err != nil {
		return nil, nil, nil, nil, 0, nil, err
	}

	metadata, err := txSnd.CreateMetadata(
		senderAddr, srcConfig, dstConfig, dstChainID, receivers, bridgingFee, operationFee)
	if err != nil {
		return nil, nil, nil, nil, 0, nil, err
	}

	metaDataRaw, err := metadata.Marshal()
	if err != nil {
		return nil, nil, nil, nil, 0, nil, err
	}

	return txBuilder, srcConfig, metadata, metaDataRaw, outputLovelace, outputNativeToken, nil
}

func (txSnd *TxSender) prepareBridgingTx(
	srcConfig *ChainConfig,
	dstChainID string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (
	*cardanowallet.TxBuilder,
	uint64, *cardanowallet.TokenAmount, uint64, error,
) {
	if err := checkFees(srcConfig, bridgingFee, operationFee); err != nil {
		return nil, 0, nil, 0, err
	}

	txBuilder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, 0, nil, 0, err
	}

	outputLovelace, feeDiff, outputNativeToken, err := getOutputsFromReceivers(
		txBuilder, srcConfig, dstChainID, receivers, bridgingFee+operationFee)
	if err != nil {
		return nil, 0, nil, 0, err
	}

	bridgingFee += feeDiff

	return txBuilder, outputLovelace, outputNativeToken, bridgingFee, nil
}

func (txSnd *TxSender) calculateFee(
	ctx context.Context,
	txBuilder *cardanowallet.TxBuilder,
	srcConfig *ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputLovelace uint64,
	outputNativeToken *cardanowallet.TokenAmount,
) (*TxFeeInfo, error) {
	_, changeMinUtxoAmount, err := txSnd.populateTxBuilder(
		ctx, txBuilder, srcConfig, senderAddr, receiverAddr, metadata, outputLovelace, outputNativeToken)
	if err != nil {
		return nil, err
	}

	fee, err := txBuilder.CalculateFee(1)
	if err != nil {
		return nil, err
	}

	return &TxFeeInfo{
		Fee:                 fee,
		ChangeMinUtxoAmount: changeMinUtxoAmount,
	}, nil
}

func (txSnd *TxSender) createTx(
	ctx context.Context,
	txBuilder *cardanowallet.TxBuilder,
	srcConfig *ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputLovelace uint64,
	outputNativeToken *cardanowallet.TokenAmount,
) (*TxInfo, error) {
	changeCurrencyLovelace, changeMinUtxoAmount, err := txSnd.populateTxBuilder(
		ctx, txBuilder, srcConfig, senderAddr, receiverAddr, metadata, outputLovelace, outputNativeToken)
	if err != nil {
		return nil, err
	}

	feeCurrencyLovelace, err := txBuilder.CalculateFee(1)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFee(feeCurrencyLovelace)

	change := changeCurrencyLovelace - feeCurrencyLovelace
	// handle overflow or insufficient amount
	if change != 0 && (change > changeCurrencyLovelace || change < changeMinUtxoAmount) {
		return nil,
			fmt.Errorf("insufficient remaining amount %d for fee %d, or minimum UTXO (%d) not satisfied",
				changeCurrencyLovelace, feeCurrencyLovelace, changeMinUtxoAmount)
	}

	if change != 0 {
		txBuilder.UpdateOutputAmount(-1, change)
	} else {
		txBuilder.RemoveOutput(-1)
	}

	txRaw, txHash, err := txBuilder.Build()
	if err != nil {
		return nil, err
	}

	return &TxInfo{
		TxRaw:               txRaw,
		TxHash:              txHash,
		ChangeMinUtxoAmount: changeMinUtxoAmount,
	}, nil
}

func (txSnd *TxSender) populateTxBuilder(
	ctx context.Context,
	txBuilder *cardanowallet.TxBuilder,
	config *ChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputLovelace uint64,
	outputNativeToken *cardanowallet.TokenAmount,
) (uint64, uint64, error) {
	if err := txSnd.populateProtocolParameters(ctx, txBuilder, config); err != nil {
		return 0, 0, err
	}

	var (
		outputNativeTokenAmounts  []cardanowallet.TokenAmount
		outputNativeTokenFullName string
	)

	if outputNativeToken != nil && outputNativeToken.Amount > 0 {
		outputNativeTokenFullName = outputNativeToken.String() // take the name used for maps
		outputNativeTokenAmounts = append(outputNativeTokenAmounts, *outputNativeToken)
	}

	utxos, err := infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]cardanowallet.Utxo, error) {
		return config.TxProvider.GetUtxos(ctx, senderAddr)
	}, txSnd.retryOptions...)
	if err != nil {
		return 0, 0, err
	}

	// calculate minUtxo for change output
	potentialChangeTokenCost, err := cardanowallet.GetMinUtxoForSumMap(
		txBuilder,
		senderAddr,
		cardanowallet.SubtractSumMaps(
			cardanowallet.GetUtxosSum(utxos),
			cardanowallet.GetTokensSumMap(outputNativeTokenAmounts...),
		))
	if err != nil {
		return 0, 0, err
	}

	srcChangeMinUtxo := max(config.MinUtxoValue, potentialChangeTokenCost)
	potentialFee := setOrDefault(config.PotentialFee, defaultPotentialFee)

	conditions := map[string]uint64{
		cardanowallet.AdaTokenName: outputLovelace + potentialFee + srcChangeMinUtxo,
	}

	if outputNativeToken != nil && outputNativeToken.Amount > 0 {
		conditions[outputNativeTokenFullName] = outputNativeToken.Amount
	}

	if txSnd.utxosTransformer != nil {
		utxos = txSnd.utxosTransformer.TransformUtxos(utxos)
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, txSnd.maxInputsPerTx, 1)
	if err != nil {
		return 0, 0, err
	}

	if txSnd.utxosTransformer != nil {
		txSnd.utxosTransformer.UpdateUtxos(inputs.Inputs)
	}

	if outputNativeToken != nil && outputNativeToken.Amount > 0 {
		inputs.Sum[outputNativeTokenFullName] -= outputNativeToken.Amount
		if inputs.Sum[outputNativeTokenFullName] == 0 {
			delete(inputs.Sum, outputNativeTokenFullName)
		}
	}

	outputRemainingTokens, err := cardanowallet.GetTokensFromSumMap(inputs.Sum)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create tokens from sum map. err: %w", err)
	}

	txBuilder.SetMetaData(metadata).
		SetTestNetMagic(config.TestNetMagic).
		AddInputs(inputs.Inputs...).
		AddOutputs(cardanowallet.TxOutput{
			Addr:   receiverAddr,
			Amount: outputLovelace,
			Tokens: outputNativeTokenAmounts,
		}, cardanowallet.TxOutput{
			Addr:   senderAddr,
			Tokens: outputRemainingTokens,
		})

	// populate ttl at the end because previous operations could take time
	if err := txSnd.populateTimeToLive(ctx, txBuilder, config); err != nil {
		return 0, 0, err
	}

	return inputs.Sum[cardanowallet.AdaTokenName] - outputLovelace, srcChangeMinUtxo, nil
}

func (txSnd *TxSender) populateProtocolParameters(
	ctx context.Context, txBuilder *cardanowallet.TxBuilder, srcConfig *ChainConfig,
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

	txBuilder.SetProtocolParameters(protocolParams)

	return nil
}

func (txSnd *TxSender) populateTimeToLive(
	ctx context.Context, txBuilder *cardanowallet.TxBuilder, config *ChainConfig,
) error {
	qtd, err := infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) (cardanowallet.QueryTipData, error) {
		return config.TxProvider.GetTip(ctx)
	}, txSnd.retryOptions...)
	if err != nil {
		return err
	}

	ttlSlotNumberInc := setOrDefault(config.TTLSlotNumberInc, defaultTTLSlotNumberInc)

	txBuilder.SetTimeToLive(qtd.Slot + ttlSlotNumberInc)

	return nil
}

func (txSnd *TxSender) getConfigs(
	srcChainID, dstChainID string,
) (*ChainConfig, *ChainConfig, error) {
	srcConfig, exists := txSnd.chainConfigMap[srcChainID]
	if !exists {
		return nil, nil, fmt.Errorf("source chain %s config not found", srcChainID)
	}

	dstConfig, exists := txSnd.chainConfigMap[dstChainID]
	if !exists {
		return nil, nil, fmt.Errorf("destination chain %s config not found", dstChainID)
	}

	return &srcConfig, &dstConfig, nil
}

func getOutputsFromReceivers(
	txBuilder *cardanowallet.TxBuilder, config *ChainConfig, dstChainID string,
	receivers []BridgingTxReceiver, initialOutputLovelace uint64,
) (outputLovelace uint64, feeDiff uint64, outputNativeToken *cardanowallet.TokenAmount, err error) {
	outputLovelaceBase, outputNativeTokenAmount := getOutputAmounts(receivers)
	outputLovelaceBase += initialOutputLovelace

	if outputNativeTokenAmount > 0 {
		nativeToken, err := getNativeToken(config.NativeTokens, dstChainID)
		if err != nil {
			return 0, 0, nil, err
		}

		token := cardanowallet.NewTokenAmount(nativeToken, outputNativeTokenAmount)
		outputNativeToken = &token
	}

	outputLovelace, err = fixLovelaceOutput(
		txBuilder, config, config.MultiSigAddr, outputNativeToken, outputLovelaceBase)
	if err != nil {
		return 0, 0, nil, err
	}

	if outputLovelace > outputLovelaceBase {
		feeDiff = outputLovelace - outputLovelaceBase
	}

	return outputLovelace, feeDiff, outputNativeToken, nil
}

func fixLovelaceOutput(
	txBuilder *cardanowallet.TxBuilder, config *ChainConfig,
	addr string, token *cardanowallet.TokenAmount, lovelaceOutputBase uint64,
) (uint64, error) {
	if token == nil {
		return lovelaceOutputBase, nil
	}

	// calculate min lovelace amount (min utxo) for receiver output
	calculatedMinUtxo, err := cardanowallet.GetMinUtxoForSumMap(
		txBuilder, addr, cardanowallet.GetTokensSumMap(*token))
	if err != nil {
		return 0, err
	}

	return max(lovelaceOutputBase, calculatedMinUtxo, config.MinUtxoValue), nil
}

func checkFees(config *ChainConfig, bridgingFee, operationFee uint64) error {
	if bridgingFee < config.MinBridgingFeeAmount {
		return fmt.Errorf("bridging fee is less than: %d", config.MinBridgingFeeAmount)
	}

	if operationFee < config.MinOperationFeeAmount {
		return fmt.Errorf("operation fee is less than: %d", config.MinOperationFeeAmount)
	}

	return nil
}

func getNativeToken(
	nativeTokenDsts []TokenExchangeConfig, dstChainID string,
) (cardanowallet.Token, error) {
	for _, cfg := range nativeTokenDsts {
		if cfg.DstChainID == dstChainID {
			token, err := cardanowallet.NewTokenWithFullName(cfg.TokenName, true)
			if err == nil {
				return token, nil
			}

			token, err = cardanowallet.NewTokenWithFullName(cfg.TokenName, false)
			if err == nil {
				return token, nil
			}

			return token, fmt.Errorf("invalid native token name: %w", err)
		}
	}

	return cardanowallet.Token{}, errors.New("native token name not specified")
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
