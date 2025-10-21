package sendtx

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"

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
	txDto BridgingTxDto,
) (*TxInfo, *BridgingRequestMetadata, error) {
	preparedData, err := txSnd.prepareBridgingTx(ctx, txDto, true)
	if err != nil {
		return nil, nil, err
	}

	defer preparedData.TxBuilder.Dispose()

	genericDto, metadata, err := txSnd.createGenericTxDtoAndMetadata(txDto, preparedData)
	if err != nil {
		return nil, nil, err
	}

	txInfo, err := txSnd.createTx(ctx, preparedData.TxBuilder, genericDto)
	if err != nil {
		return nil, nil, err
	}

	return txInfo, metadata, nil
}

// CalculateBridgingTxFee returns calculated fee for bridging tx
func (txSnd *TxSender) CalculateBridgingTxFee(
	ctx context.Context,
	txDto BridgingTxDto,
) (*TxFeeInfo, *BridgingRequestMetadata, error) {
	preparedData, err := txSnd.prepareBridgingTx(ctx, txDto, true)
	if err != nil {
		return nil, nil, err
	}

	defer preparedData.TxBuilder.Dispose()

	genericDto, metadata, err := txSnd.createGenericTxDtoAndMetadata(txDto, preparedData)
	if err != nil {
		return nil, nil, err
	}

	txFeeInfo, err := txSnd.calculateFee(ctx, preparedData.TxBuilder, genericDto)
	if err != nil {
		return nil, nil, err
	}

	return txFeeInfo, metadata, nil
}

// CreateTxGeneric creates generic tx to one recipient and returns cbor of raw transaction data, tx hash and error
func (txSnd *TxSender) CreateTxGeneric(
	ctx context.Context,
	txDto GenericTxDto,
) (*TxInfo, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[txDto.SrcChainID]
	if !existsSrc {
		return nil, fmt.Errorf("chain %s config not found", txDto.SrcChainID)
	}

	txBuilder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, err
	}

	defer txBuilder.Dispose()

	if err := checkAddress(txDto.SenderAddr, txDto.SenderAddrPolicyScript, &srcConfig); err != nil {
		return nil, err
	}

	if err := txSnd.populateProtocolParameters(ctx, txBuilder, &srcConfig); err != nil {
		return nil, err
	}

	txDto.OutputLovelace, err = adjustLovelaceOutput(
		txBuilder, txDto.ReceiverAddr, txDto.OutputNativeTokens, srcConfig.MinUtxoValue, txDto.OutputLovelace)
	if err != nil {
		return nil, err
	}

	return txSnd.createTx(ctx, txBuilder, txDto)
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
	srcChainID string,
	dstChainID string,
	receivers []BridgingTxReceiver,
	bridgingFee uint64,
	operationFee uint64,
) (*BridgingRequestMetadata, error) {
	srcConfig, dstConfig, err := txSnd.getConfigs(srcChainID, dstChainID)
	if err != nil {
		return nil, err
	}

	txs := make([]BridgingRequestMetadataTransaction, len(receivers))

	for i, x := range receivers {
		if x.Addr == "" {
			return nil, fmt.Errorf("receiver %d address is empty", i)
		}

		switch x.BridgingType {
		case BridgingTypeNativeTokenOnSource:
			if x.Amount < dstConfig.MinUtxoValue {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, dstConfig.MinUtxoValue)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address:            AddrToMetaDataAddr(x.Addr),
				Amount:             x.Amount,
				IsNativeTokenOnSrc: metadataBoolTrue,
			}

		case BridgingTypeCurrencyOnSource:
			if x.Amount < srcConfig.MinUtxoValue {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, srcConfig.MinUtxoValue)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address: AddrToMetaDataAddr(x.Addr),
				Amount:  x.Amount,
			}
		default:
			if x.Amount < txSnd.minAmountToBridge {
				return nil, fmt.Errorf("amount for receiver %d is lower than %d", i, txSnd.minAmountToBridge)
			}

			txs[i] = BridgingRequestMetadataTransaction{
				Address: AddrToMetaDataAddr(x.Addr),
				Amount:  x.Amount,
			}
		}
	}

	return &BridgingRequestMetadata{
		BridgingTxType:     bridgingMetaDataType,
		DestinationChainID: dstChainID,
		SenderAddr:         AddrToMetaDataAddr(senderAddr),
		Transactions:       txs,
		BridgingFee:        bridgingFee,
		OperationFee:       operationFee,
	}, nil
}

func (txSnd *TxSender) GetBridgingFee(
	ctx context.Context,
	bridgingTxInput BridgingTxDto,
) (uint64, error) {
	data, err := txSnd.prepareBridgingTx(ctx, bridgingTxInput, false)
	if err != nil {
		return 0, err
	}

	defer data.TxBuilder.Dispose()

	return data.BridgingFee, nil
}

func (txSnd *TxSender) prepareBridgingTx(
	ctx context.Context,
	txDto BridgingTxDto,
	validateAddressData bool,
) (*bridgingTxPreparedData, error) {
	srcConfig, _, err := txSnd.getConfigs(txDto.SrcChainID, txDto.DstChainID)
	if err != nil {
		return nil, err
	}

	if validateAddressData {
		if err := checkAddress(txDto.SenderAddr, txDto.SenderAddrPolicyScript, srcConfig); err != nil {
			return nil, err
		}
	}

	if err := checkFees(srcConfig, txDto.BridgingFee, txDto.OperationFee); err != nil {
		return nil, err
	}

	txBuilder, err := cardanowallet.NewTxBuilder(srcConfig.CardanoCliBinary)
	if err != nil {
		return nil, err
	}

	if err := txSnd.populateProtocolParameters(ctx, txBuilder, srcConfig); err != nil {
		return nil, err
	}

	outputNativeTokens := ([]cardanowallet.TokenAmount)(nil)
	outputLovelaceBase, outputNativeTokenAmount := getOutputAmounts(txDto.Receivers)

	if outputNativeTokenAmount > 0 {
		nativeToken, err := getTokenFromTokenExchangeConfig(srcConfig.NativeTokens, txDto.DstChainID)
		if err != nil {
			return nil, err
		}

		outputNativeTokens = append(outputNativeTokens,
			cardanowallet.NewTokenAmount(nativeToken, outputNativeTokenAmount))
	}

	bridgingAddress := txDto.BridgingAddress
	if bridgingAddress == "" {
		bridgingAddress = srcConfig.MultiSigAddr
	}

	outputLovelaceBeforeAdditionalCharges, err := adjustLovelaceOutput(
		txBuilder, bridgingAddress, outputNativeTokens, srcConfig.MinUtxoValue, outputLovelaceBase)
	if err != nil {
		return nil, err
	}

	bridgingFee := txDto.BridgingFee
	outputLovelace := outputLovelaceBeforeAdditionalCharges + bridgingFee + txDto.OperationFee

	if outputLovelaceBeforeAdditionalCharges > outputLovelaceBase {
		bridgingFee += outputLovelaceBeforeAdditionalCharges - outputLovelaceBase
	}

	return &bridgingTxPreparedData{
		TxBuilder:          txBuilder,
		OutputLovelace:     outputLovelace,
		OutputNativeTokens: outputNativeTokens,
		BridgingAddress:    bridgingAddress,
		BridgingFee:        bridgingFee,
	}, nil
}

func (txSnd *TxSender) calculateFee(
	ctx context.Context,
	txBuilder *cardanowallet.TxBuilder,
	txDto GenericTxDto,
) (*TxFeeInfo, error) {
	data, err := txSnd.populateTxBuilder(ctx, txBuilder, txDto)
	if err != nil {
		return nil, err
	}

	witnessCount := 1
	if txDto.SenderAddrPolicyScript != nil {
		witnessCount = txDto.SenderAddrPolicyScript.GetCount()
	}

	fee, err := txBuilder.CalculateFee(witnessCount)
	if err != nil {
		return nil, err
	}

	return &TxFeeInfo{
		Fee:                 fee,
		ChangeMinUtxoAmount: data.ChangeMinUtxoAmount,
	}, nil
}

func (txSnd *TxSender) createTx(
	ctx context.Context,
	txBuilder *cardanowallet.TxBuilder,
	txDto GenericTxDto,
) (*TxInfo, error) {
	data, err := txSnd.populateTxBuilder(ctx, txBuilder, txDto)
	if err != nil {
		return nil, err
	}

	witnessCount := 1
	if txDto.SenderAddrPolicyScript != nil {
		witnessCount = txDto.SenderAddrPolicyScript.GetCount()
	}

	feeCurrencyLovelace, err := txBuilder.CalculateFee(witnessCount)
	if err != nil {
		return nil, err
	}

	txBuilder.SetFee(feeCurrencyLovelace)

	change := data.ChangeLovelace - feeCurrencyLovelace
	// handle overflow or insufficient amount
	if change != 0 && (change > data.ChangeLovelace || change < data.ChangeMinUtxoAmount) {
		return nil,
			fmt.Errorf("insufficient remaining amount %d for fee %d, or minimum UTXO (%d) not satisfied",
				data.ChangeLovelace, feeCurrencyLovelace, data.ChangeMinUtxoAmount)
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
		ChangeMinUtxoAmount: data.ChangeMinUtxoAmount,
		ChosenInputs:        data.ChosenInputs,
	}, nil
}

func (txSnd *TxSender) populateTxBuilder(
	ctx context.Context,
	txBuilder *cardanowallet.TxBuilder,
	txDto GenericTxDto,
) (*txBuilderPopulationData, error) {
	config := txSnd.chainConfigMap[txDto.SrcChainID]

	utxos, err := infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]cardanowallet.Utxo, error) {
		return config.TxProvider.GetUtxos(ctx, txDto.SenderAddr)
	}, txSnd.retryOptions...)
	if err != nil {
		return nil, err
	}

	// calculate minUtxo for change output
	potentialChangeTokenCost, err := cardanowallet.GetMinUtxoForSumMap(
		txBuilder,
		txDto.SenderAddr,
		cardanowallet.SubtractSumMaps(
			cardanowallet.GetUtxosSum(utxos),
			cardanowallet.GetTokensSumMap(txDto.OutputNativeTokens...),
		))
	if err != nil {
		return nil, err
	}

	srcChangeMinUtxo := max(config.MinUtxoValue, potentialChangeTokenCost)
	potentialFee := setOrDefault(config.PotentialFee, defaultPotentialFee)

	conditions := map[string]uint64{
		cardanowallet.AdaTokenName: txDto.OutputLovelace + potentialFee + srcChangeMinUtxo,
	}

	for _, token := range txDto.OutputNativeTokens {
		conditions[token.TokenName()] += token.Amount
	}

	if txSnd.utxosTransformer != nil && !reflect.ValueOf(txSnd.utxosTransformer).IsNil() {
		utxos = txSnd.utxosTransformer.TransformUtxos(utxos)
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, txSnd.maxInputsPerTx, 1)
	if err != nil {
		return nil, err
	}

	for _, token := range txDto.OutputNativeTokens {
		tokenName := token.TokenName()

		inputs.Sum[tokenName] -= token.Amount
		if inputs.Sum[tokenName] == 0 {
			delete(inputs.Sum, tokenName)
		}
	}

	outputRemainingTokens, err := cardanowallet.GetTokensFromSumMap(inputs.Sum)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokens from sum map. err: %w", err)
	}

	txBuilder.SetMetaData(txDto.Metadata).SetTestNetMagic(config.TestNetMagic)

	if txDto.SenderAddrPolicyScript != nil {
		txBuilder.AddInputsWithScript(txDto.SenderAddrPolicyScript, inputs.Inputs...)
	} else {
		txBuilder.AddInputs(inputs.Inputs...)
	}

	txBuilder.AddOutputs(cardanowallet.TxOutput{
		Addr:   txDto.ReceiverAddr,
		Amount: txDto.OutputLovelace,
		Tokens: txDto.OutputNativeTokens,
	}, cardanowallet.TxOutput{
		Addr:   txDto.SenderAddr,
		Amount: inputs.Sum[cardanowallet.AdaTokenName] - conditions[cardanowallet.AdaTokenName],
		Tokens: outputRemainingTokens,
	})

	// populate ttl at the end because previous operations could take time
	if err := txSnd.populateTimeToLive(ctx, txBuilder, &config); err != nil {
		return nil, err
	}

	return &txBuilderPopulationData{
		ChangeLovelace:      inputs.Sum[cardanowallet.AdaTokenName] - txDto.OutputLovelace,
		ChangeMinUtxoAmount: srcChangeMinUtxo,
		ChosenInputs:        inputs,
	}, nil
}

func (txSnd *TxSender) createGenericTxDtoAndMetadata(
	txDto BridgingTxDto,
	preparedData *bridgingTxPreparedData,
) (GenericTxDto, *BridgingRequestMetadata, error) {
	metadata, err := txSnd.CreateMetadata(
		txDto.SenderAddr,
		txDto.SrcChainID,
		txDto.DstChainID,
		txDto.Receivers,
		preparedData.BridgingFee,
		txDto.OperationFee,
	)
	if err != nil {
		return GenericTxDto{}, nil, err
	}

	metadataRaw, err := metadata.Marshal()
	if err != nil {
		return GenericTxDto{}, nil, err
	}

	return GenericTxDto{
		SrcChainID:             txDto.SrcChainID,
		SenderAddr:             txDto.SenderAddr,
		SenderAddrPolicyScript: txDto.SenderAddrPolicyScript,
		ReceiverAddr:           preparedData.BridgingAddress,
		Metadata:               metadataRaw,
		OutputLovelace:         preparedData.OutputLovelace,
		OutputNativeTokens:     preparedData.OutputNativeTokens,
	}, metadata, nil
}

func (txSnd *TxSender) populateProtocolParameters(
	ctx context.Context, txBuilder *cardanowallet.TxBuilder, config *ChainConfig,
) (err error) {
	protocolParams := config.ProtocolParameters
	if protocolParams == nil {
		protocolParams, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return config.TxProvider.GetProtocolParameters(ctx)
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

func adjustLovelaceOutput(
	txBuilder *cardanowallet.TxBuilder, addr string,
	tokens []cardanowallet.TokenAmount, defaultMinUtxo, lovelaceOutputBase uint64,
) (uint64, error) {
	if len(tokens) == 0 {
		return max(lovelaceOutputBase, defaultMinUtxo), nil
	}

	// calculate min lovelace amount (min utxo) for receiver output
	calculatedMinUtxo, err := cardanowallet.GetMinUtxoForSumMap(
		txBuilder, addr, cardanowallet.GetTokensSumMap(tokens...))
	if err != nil {
		return 0, err
	}

	return max(lovelaceOutputBase, calculatedMinUtxo, defaultMinUtxo), nil
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

func getTokenFromTokenExchangeConfig(
	nativeTokenDsts []TokenExchangeConfig, dstChainID string,
) (cardanowallet.Token, error) {
	for _, cfg := range nativeTokenDsts {
		if cfg.DstChainID == dstChainID {
			return cardanowallet.NewTokenWithFullNameTry(cfg.TokenName)
		}
	}

	return cardanowallet.Token{}, fmt.Errorf("native token name not specified for destination %s", dstChainID)
}

// getOutputAmounts returns amount needed for outputs in lovelace and native tokens
func getOutputAmounts(receivers []BridgingTxReceiver) (outputCurrencyLovelace uint64, outputNativeToken uint64) {
	for _, x := range receivers {
		if x.BridgingType == BridgingTypeNativeTokenOnSource {
			outputNativeToken += x.Amount // WSADA/WSAPEX to ADA/APEX
		} else {
			outputCurrencyLovelace += x.Amount // ADA/APEX to WSADA/WSAPEX or Reactor tokens
		}
	}

	return outputCurrencyLovelace, outputNativeToken
}

func checkAddress(
	addrStr string, policyScript *cardanowallet.PolicyScript, config *ChainConfig,
) error {
	addr, err := cardanowallet.NewCardanoAddressFromString(addrStr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	if policyScript != nil {
		policyID, err := cardanowallet.NewCliUtils(config.CardanoCliBinary).GetPolicyID(policyScript)
		if err != nil {
			return fmt.Errorf("failed to retrieve policy id: %w", err)
		}
		// address payment payload hash must be equal to policy id
		if hex.EncodeToString(addr.GetInfo().Payment.Payload[:]) != policyID {
			return fmt.Errorf("policy script does not belong to address: %s", addrStr)
		}
	}

	return nil
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
