package sendtx

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	infracommon "github.com/Ethernal-Tech/cardano-infrastructure/common"
	cardanowallet "github.com/Ethernal-Tech/cardano-infrastructure/wallet"
)

type BridgingType byte

const (
	BridgingTypeNormal BridgingType = iota
	BridgingTypeNativeTokenOnSource
	BridgingTypeCurrencyOnSource

	defaultPotentialFee     = 250_000
	defaultTTLSlotNumberInc = 500
	splitStringLength       = 40
)

type ChainConfig struct {
	CardanoCliBinary    string
	TxProvider          cardanowallet.ITxProvider
	MultiSigAddr        string
	TestNetMagic        uint
	TTLSlotNumberInc    uint64
	MinUtxoValue        uint64
	NativeTokenFullName string // policyID.hex(name)
	ExchangeRate        map[string]float64
	ProtocolParameters  []byte
}

type BridgingTxReceiver struct {
	BridgingType BridgingType `json:"type"`
	Addr         string       `json:"addr"`
	Amount       uint64       `json:"amount"`
}

type TxSender struct {
	bridgingFeeAmount uint64
	minAmountToBridge uint64
	potentialFee      uint64
	maxInputsPerTx    int
	chainConfigMap    map[string]ChainConfig
	retryOptions      []infracommon.RetryConfigOption
}

func NewTxSender(
	bridgingFeeAmount uint64,
	minAmountToBridge uint64,
	potentialFee uint64,
	maxInputsPerTx int,
	chainConfigMap map[string]ChainConfig,
	retryOptions ...infracommon.RetryConfigOption,
) *TxSender {
	return &TxSender{
		bridgingFeeAmount: bridgingFeeAmount,
		minAmountToBridge: minAmountToBridge,
		maxInputsPerTx:    maxInputsPerTx,
		potentialFee:      potentialFee,
		chainConfigMap:    chainConfigMap,
		retryOptions:      retryOptions,
	}
}

// CreateBridgingTx creates bridging tx and returns cbor of raw transaction data, tx hash and error
func (txSnd *TxSender) CreateBridgingTx(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	senderAddr string,
	receivers []BridgingTxReceiver,
) ([]byte, string, *BridgingRequestMetadata, error) {
	metadata, err := txSnd.CreateMetadata(senderAddr, srcChainID, dstChainID, receivers)
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
) (uint64, error) {
	metadata, err := txSnd.CreateMetadata(senderAddr, srcChainID, dstChainID, receivers)
	if err != nil {
		return 0, err
	}

	srcConfig := txSnd.chainConfigMap[srcChainID]
	outputCurrencyLovelace, outputNativeToken := GetOutputAmounts(metadata)

	metaDataRaw, err := metadata.Marshal()
	if err != nil {
		return 0, err
	}

	return txSnd.calculateFee(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metaDataRaw, outputCurrencyLovelace, outputNativeToken)
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

func (txSnd *TxSender) WaitForTx(
	ctx context.Context, chainID string, receivers []BridgingTxReceiver, tokenName string,
) error {
	chainConfig, existsSrc := txSnd.chainConfigMap[chainID]
	if !existsSrc {
		return fmt.Errorf("%s chain config not found", chainID)
	}

	errs := make([]error, len(receivers))
	wg := sync.WaitGroup{}

	for i, x := range receivers {
		wg.Add(1)

		go func(idx int, recv BridgingTxReceiver) {
			defer wg.Done()

			_, errs[idx] = infracommon.WaitForAmount(
				ctx, new(big.Int).SetUint64(recv.Amount), func(ctx context.Context) (*big.Int, error) {
					utxos, err := chainConfig.TxProvider.GetUtxos(ctx, recv.Addr)
					if err != nil {
						return nil, err
					}

					return new(big.Int).SetUint64(cardanowallet.GetUtxosSum(utxos)[tokenName]), nil
				})
		}(i, x)
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (txSnd *TxSender) CreateMetadata(
	senderAddr string, srcChainID, dstChainID string, receivers []BridgingTxReceiver,
) (*BridgingRequestMetadata, error) {
	srcConfig, existsSrc := txSnd.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, fmt.Errorf("source chain %s config not found", srcChainID)
	}

	dstConfig, existsDst := txSnd.chainConfigMap[dstChainID]
	if !existsDst {
		return nil, fmt.Errorf("destination chain %s config not found", dstChainID)
	}

	exchangeRateOnDst := setOrDefault(srcConfig.ExchangeRate[dstChainID], 1)
	exchangeRateOnSrc := 1.0 / exchangeRateOnDst
	feeSrcCurrencyLovelaceAmount := mul(txSnd.bridgingFeeAmount, exchangeRateOnSrc)
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

	feeDstCurrencyLovelaceAmount := txSnd.bridgingFeeAmount

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

	potentialFee := setOrDefault(txSnd.potentialFee, defaultPotentialFee)
	ttlSlotNumberInc := setOrDefault(srcConfig.TTLSlotNumberInc, defaultTTLSlotNumberInc)

	outputNativeTokens := []cardanowallet.TokenAmount(nil)
	conditions := map[string]uint64{
		cardanowallet.AdaTokenName: outputCurrencyLovelace + potentialFee + srcConfig.MinUtxoValue,
	}

	if outputNativeToken != 0 {
		conditions[srcConfig.NativeTokenFullName] = outputNativeToken
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, txSnd.maxInputsPerTx, 1)
	if err != nil {
		return nil, err
	}

	if outputNativeToken != 0 {
		inputs.Sum[srcConfig.NativeTokenFullName] -= outputNativeToken
		if inputs.Sum[srcConfig.NativeTokenFullName] == 0 {
			delete(inputs.Sum, srcConfig.NativeTokenFullName)
		}

		nativeToken, err := cardanowallet.NewTokenAmountWithFullName(
			srcConfig.NativeTokenFullName, outputNativeToken, true)
		if err == nil {
			return nil, err
		}

		outputNativeTokens = []cardanowallet.TokenAmount{nativeToken}
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

func mul(a uint64, b float64) uint64 {
	return uint64(float64(a) * b)
}

func setOrDefault[T comparable](val, def T) T {
	var zero T

	if val == zero {
		return def
	}

	return val
}
