package bridgingtx

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

type ExchangeRateConfig struct {
	RateOnDst         float64
	MinUtxoValueOnDst uint64
}

type BridgingTxChainConfig struct {
	TxProvider          cardanowallet.ITxProvider
	MultiSigAddr        string
	TestNetMagic        uint
	TTLSlotNumberInc    uint64
	PotentialFee        uint64
	MinUtxoValue        uint64
	NativeTokenFullName string
	ExchangeRate        map[string]ExchangeRateConfig
	ProtocolParameters  []byte
}

type BridgingTxReceiver struct {
	Addr   string `json:"addr"`
	Amount uint64 `json:"amount"`
}

type BridgingTxSender struct {
	cardanoCliBinary  string
	chainConfigMap    map[string]BridgingTxChainConfig
	bridgingFeeAmount uint64
	maxInputsPerTx    int
	retryOptions      []infracommon.RetryConfigOption
}

func NewBridgingTxSender(
	cardanoCliBinary string,
	bridgingFeeAmount uint64,
	maxInputsPerTx int,
	chainConfigMap map[string]BridgingTxChainConfig,
	retryOptions ...infracommon.RetryConfigOption,
) *BridgingTxSender {
	return &BridgingTxSender{
		cardanoCliBinary:  cardanoCliBinary,
		bridgingFeeAmount: bridgingFeeAmount,
		maxInputsPerTx:    maxInputsPerTx,
		chainConfigMap:    chainConfigMap,
		retryOptions:      retryOptions,
	}
}

// CreateBridgingTx creates bridging tx and returns cbor of raw transaction data, tx hash and error
func (bts *BridgingTxSender) CreateBridgingTx(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	bridgingType BridgingType,
	senderAddr string,
	receivers []BridgingTxReceiver,
) ([]byte, string, error) {
	srcConfig, existsSrc := bts.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, "", fmt.Errorf("chain %s config not found", srcChainID)
	}

	outputCurrencyLovelace, outputNativeToken, feeOnSrcCurrencyLovelace := bts.getOutputAmounts(
		srcConfig, dstChainID, bridgingType, receivers)

	metadata, err := bts.createMetadata(
		senderAddr, srcConfig, dstChainID, bridgingType, receivers, BridgingRequestMetadataCurrencyInfo{
			SrcAmount:  feeOnSrcCurrencyLovelace,
			DestAmount: bts.bridgingFeeAmount,
		})
	if err != nil {
		return nil, "", err
	}

	// first try with exact sum
	raw, hash, err := bts.createTx(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metadata, outputCurrencyLovelace, outputNativeToken, false)
	if err == nil {
		return raw, hash, nil
	}

	// then without
	return bts.createTx(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metadata, outputCurrencyLovelace, outputNativeToken, true)
}

// CreateTxGeneric creates generic tx to one recipient and returns cbor of raw transaction data, tx hash and error
func (bts *BridgingTxSender) CreateTxGeneric(
	ctx context.Context,
	srcChainID string,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
) ([]byte, string, error) {
	srcConfig, existsSrc := bts.chainConfigMap[srcChainID]
	if !existsSrc {
		return nil, "", fmt.Errorf("chain %s config not found", srcChainID)
	}

	// first try with exact sum
	raw, hash, err := bts.createTx(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metadata, outputCurrencyLovelace, outputNativeToken, false)
	if err == nil {
		return raw, hash, nil
	}

	// then without
	return bts.createTx(
		ctx, srcConfig, senderAddr, srcConfig.MultiSigAddr, metadata, outputCurrencyLovelace, outputNativeToken, true)
}

func (bts *BridgingTxSender) SubmitTx(
	ctx context.Context, chainID string, txRaw []byte, cardanoWallet cardanowallet.ITxSigner,
) error {
	chainConfig, existsSrc := bts.chainConfigMap[chainID]
	if !existsSrc {
		return fmt.Errorf("%s chain config not found", chainID)
	}

	builder, err := cardanowallet.NewTxBuilder(bts.cardanoCliBinary)
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
	}, bts.retryOptions...)

	return err
}

func (bts *BridgingTxSender) WaitForTx(
	ctx context.Context, chainID string, receivers []BridgingTxReceiver, tokenName string,
) error {
	chainConfig, existsSrc := bts.chainConfigMap[chainID]
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

					sum := cardanowallet.GetUtxosSum(utxos)

					return new(big.Int).SetUint64(sum[tokenName]), nil
				})
		}(i, x)
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (bts *BridgingTxSender) createTx(
	ctx context.Context,
	srcConfig BridgingTxChainConfig,
	senderAddr string,
	receiverAddr string,
	metadata []byte,
	outputCurrencyLovelace uint64,
	outputNativeToken uint64,
	exactSumNotAllowed bool,
) ([]byte, string, error) {
	queryTip, protocolParams, utxos, err := bts.getDynamicParameters(ctx, srcConfig, senderAddr)
	if err != nil {
		return nil, "", err
	}

	potentialFee := setOrDefault(srcConfig.PotentialFee, defaultPotentialFee)
	ttlSlotNumberInc := setOrDefault(srcConfig.TTLSlotNumberInc, defaultTTLSlotNumberInc)

	lovelaceExactSumModificator := uint64(0)
	// do not satisfy exact sum for lovelace if there is a native tokens involed or exact sum is not allowed
	if exactSumNotAllowed || outputNativeToken != 0 {
		lovelaceExactSumModificator = srcConfig.MinUtxoValue
	}

	outputNativeTokens := []cardanowallet.TokenAmount(nil)
	conditions := map[string]AmountCondition{
		cardanowallet.AdaTokenName: {
			Exact:   outputCurrencyLovelace + potentialFee + lovelaceExactSumModificator,
			AtLeast: outputCurrencyLovelace + potentialFee + srcConfig.MinUtxoValue,
		},
	}

	if outputNativeToken != 0 {
		conditions[srcConfig.NativeTokenFullName] = AmountCondition{
			Exact:   outputNativeToken,
			AtLeast: outputNativeToken,
		}
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, bts.maxInputsPerTx)
	if err != nil {
		return nil, "", err
	}

	if outputNativeToken != 0 {
		nativeToken, err := getNativeToken(srcConfig.NativeTokenFullName, outputNativeToken)
		if err != nil {
			return nil, "", err
		}

		tokenFullName := nativeToken.TokenName()
		outputNativeTokens = []cardanowallet.TokenAmount{nativeToken}

		inputs.Sum[tokenFullName] -= nativeToken.Amount
		if inputs.Sum[tokenFullName] == 0 {
			delete(inputs.Sum, tokenFullName)
		}
	}

	outputRemainingTokens, err := cardanowallet.GetTokensFromSumMap(inputs.Sum)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create tokens from sum map. err: %w", err)
	}

	outputs := []cardanowallet.TxOutput{
		{
			Addr:   receiverAddr,
			Amount: outputCurrencyLovelace,
			Tokens: outputNativeTokens,
		},
		{
			Addr:   senderAddr,
			Tokens: outputRemainingTokens,
		},
	}

	builder, err := cardanowallet.NewTxBuilder(bts.cardanoCliBinary)
	if err != nil {
		return nil, "", err
	}

	defer builder.Dispose()

	builder.SetMetaData(metadata).
		SetProtocolParameters(protocolParams).
		SetTimeToLive(queryTip.Slot + ttlSlotNumberInc).
		SetTestNetMagic(srcConfig.TestNetMagic).
		AddInputs(inputs.Inputs...).
		AddOutputs(outputs...)

	feeCurrencyLovelace, err := builder.CalculateFee(1)
	if err != nil {
		return nil, "", err
	}

	builder.SetFee(feeCurrencyLovelace)

	inputsSumCurrencyLovelace := inputs.Sum[cardanowallet.AdaTokenName]
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

func (bts BridgingTxSender) getDynamicParameters(
	ctx context.Context, srcConfig BridgingTxChainConfig, addr string,
) (qtd cardanowallet.QueryTipData, protocolParams []byte, utxos []cardanowallet.Utxo, err error) {
	protocolParams = srcConfig.ProtocolParameters
	if protocolParams == nil {
		protocolParams, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return srcConfig.TxProvider.GetProtocolParameters(ctx)
		}, bts.retryOptions...)
		if err != nil {
			return
		}
	}

	qtd, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) (cardanowallet.QueryTipData, error) {
		return srcConfig.TxProvider.GetTip(ctx)
	}, bts.retryOptions...)
	if err != nil {
		return
	}

	utxos, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]cardanowallet.Utxo, error) {
		return srcConfig.TxProvider.GetUtxos(ctx, addr)
	}, bts.retryOptions...)

	return qtd, protocolParams, utxos, err
}

func (bts *BridgingTxSender) createMetadata(
	senderAddr string, srcConfig BridgingTxChainConfig, dstChainID string, bridgingType BridgingType,
	receivers []BridgingTxReceiver, bridgingFeeAmount BridgingRequestMetadataCurrencyInfo,
) ([]byte, error) {
	minUtxoOnDst := setOrDefault(srcConfig.ExchangeRate[dstChainID].MinUtxoValueOnDst, srcConfig.MinUtxoValue)
	exchangeRateOnDst := setOrDefault(srcConfig.ExchangeRate[dstChainID].RateOnDst, 1)
	exchangeRateOnSrc := 1.0 / exchangeRateOnDst
	metadataObj := BridgingRequestMetadata{
		BridgingTxType:     bridgingMetaDataType,
		DestinationChainID: dstChainID,
		SenderAddr:         infracommon.SplitString(senderAddr, splitStringLength),
		Transactions:       make([]BridgingRequestMetadataTransaction, len(receivers)),
		FeeAmount:          bridgingFeeAmount,
	}

	for i, x := range receivers {
		switch bridgingType {
		case BridgingTypeNativeTokenOnSource:
			metadataObj.Transactions[i] = BridgingRequestMetadataTransaction{
				Address:            infracommon.SplitString(x.Addr, splitStringLength),
				Amount:             x.Amount,
				IsNativeTokenOnSrc: true,
				Additional: &BridgingRequestMetadataCurrencyInfo{
					DestAmount: mul(srcConfig.MinUtxoValue, exchangeRateOnDst),
					SrcAmount:  srcConfig.MinUtxoValue,
				},
			}
		case BridgingTypeCurrencyOnSource:
			metadataObj.Transactions[i] = BridgingRequestMetadataTransaction{
				Address: infracommon.SplitString(x.Addr, splitStringLength),
				Amount:  x.Amount,
				Additional: &BridgingRequestMetadataCurrencyInfo{
					DestAmount: minUtxoOnDst,
					SrcAmount:  mul(minUtxoOnDst, exchangeRateOnSrc),
				},
			}
		default:
			metadataObj.Transactions[i] = BridgingRequestMetadataTransaction{
				Address: infracommon.SplitString(x.Addr, splitStringLength),
				Amount:  x.Amount,
			}
		}
	}

	return metadataObj.Marshal()
}

func (bts *BridgingTxSender) getOutputAmounts(
	srcConfig BridgingTxChainConfig, dstChainID string, bridgingType BridgingType, receivers []BridgingTxReceiver,
) (outputCurrencyLovelace uint64, outputNativeToken uint64, feeOnSrcCurrencyLovelace uint64) {
	minUtxoOnDst := setOrDefault(srcConfig.ExchangeRate[dstChainID].MinUtxoValueOnDst, srcConfig.MinUtxoValue)
	exchangeRateOnDst := setOrDefault(srcConfig.ExchangeRate[dstChainID].RateOnDst, 1)
	exchangeRateOnSrc := 1.0 / exchangeRateOnDst
	feeOnSrcCurrencyLovelace = mul(bts.bridgingFeeAmount, exchangeRateOnSrc) // fee is always paid in lovelace
	outputCurrencyLovelace = feeOnSrcCurrencyLovelace

	for _, x := range receivers {
		switch bridgingType {
		case BridgingTypeNativeTokenOnSource:
			outputNativeToken += x.Amount
			outputCurrencyLovelace += srcConfig.MinUtxoValue // NOTE: is this good -> shell we count only once for multisig?
		case BridgingTypeCurrencyOnSource:
			outputNativeToken += mul(minUtxoOnDst, exchangeRateOnSrc)
			outputCurrencyLovelace += x.Amount
		default:
			outputCurrencyLovelace += x.Amount
		}
	}

	return outputCurrencyLovelace, outputNativeToken, feeOnSrcCurrencyLovelace
}

func mul(a uint64, b float64) uint64 {
	return uint64(float64(a) * b)
}

func getNativeToken(fullName string, amount uint64) (cardanowallet.TokenAmount, error) {
	if r, err := cardanowallet.NewTokenAmountWithFullName(fullName, amount, true); err == nil {
		return r, nil
	}

	return cardanowallet.NewTokenAmountWithFullName(fullName, amount, false)
}

func setOrDefault[T comparable](val, def T) T {
	var zero T

	if val == zero {
		return def
	}

	return val
}
