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

type BridgingTxChainConfig struct {
	MultiSigAddr        string
	TestNetMagic        uint
	TTLSlotNumberInc    uint64
	PotentialFee        uint64
	MinUtxoValue        uint64
	NativeTokenFullName string
	ExchangeRate        float64
	ProtocolParameters  []byte
}

type BridgingTxReceiver struct {
	Addr   string `json:"addr"`
	Amount uint64 `json:"amount"`
}

type BridgingTxSender struct {
	cardanoCliBinary   string
	txProviderSrc      cardanowallet.ITxProvider
	txUtxoRetrieverDst cardanowallet.IUTxORetriever
	chainConfigMap     map[string]BridgingTxChainConfig
	bridgingFeeAmount  uint64
	maxInputsPerTx     int
}

func NewBridgingTxSender(
	cardanoCliBinary string,
	txProviderSrc cardanowallet.ITxProvider,
	txUtxoRetrieverDst cardanowallet.IUTxORetriever,
	bridgingFeeAmount uint64,
	maxInputsPerTx int,
	chainConfigMap map[string]BridgingTxChainConfig,
) *BridgingTxSender {
	return &BridgingTxSender{
		cardanoCliBinary:   cardanoCliBinary,
		txProviderSrc:      txProviderSrc,
		txUtxoRetrieverDst: txUtxoRetrieverDst,
		bridgingFeeAmount:  bridgingFeeAmount,
		maxInputsPerTx:     maxInputsPerTx,
		chainConfigMap:     chainConfigMap,
	}
}

// CreateTx creates tx and returns cbor of raw transaction data, tx hash and error
func (bts *BridgingTxSender) CreateTx(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	bridgingType BridgingType,
	senderAddr string,
	receivers []BridgingTxReceiver,
) ([]byte, string, error) {
	// first try with exact sum
	raw, hash, err := bts.createTx(ctx, srcChainID, dstChainID, bridgingType, senderAddr, receivers, false)
	if err == nil {
		return raw, hash, nil
	}

	return bts.createTx(ctx, srcChainID, dstChainID, bridgingType, senderAddr, receivers, true)
}

func (bts *BridgingTxSender) SendTx(
	ctx context.Context, txRaw []byte, cardanoWallet cardanowallet.ITxSigner,
) error {
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
		return true, bts.txProviderSrc.SubmitTx(ctx, txSigned)
	})

	return err
}

func (bts *BridgingTxSender) WaitForTx(
	ctx context.Context, receivers []cardanowallet.TxOutput, tokenName string,
) error {
	errs := make([]error, len(receivers))
	wg := sync.WaitGroup{}

	for i, x := range receivers {
		wg.Add(1)

		go func(idx int, recv cardanowallet.TxOutput) {
			defer wg.Done()

			_, errs[idx] = infracommon.WaitForAmount(
				ctx, new(big.Int).SetUint64(recv.Amount), func(ctx context.Context) (*big.Int, error) {
					utxos, err := bts.txUtxoRetrieverDst.GetUtxos(ctx, recv.Addr)
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

func (bts *BridgingTxSender) createMetadata(
	dstChainID, senderAddr string, bridgingType BridgingType,
	srcConfig, dstConfig BridgingTxChainConfig,
	receivers []BridgingTxReceiver, bridgingFeeAmount BridgingRequestMetadataCurrencyInfo,
) ([]byte, error) {
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
					DestAmount: mul(srcConfig.MinUtxoValue, dstConfig.ExchangeRate),
					SrcAmount:  srcConfig.MinUtxoValue,
				},
			}
		case BridgingTypeCurrencyOnSource:
			metadataObj.Transactions[i] = BridgingRequestMetadataTransaction{
				Address: infracommon.SplitString(x.Addr, splitStringLength),
				Amount:  x.Amount,
				Additional: &BridgingRequestMetadataCurrencyInfo{
					DestAmount: dstConfig.MinUtxoValue,
					SrcAmount:  mul(dstConfig.MinUtxoValue, srcConfig.ExchangeRate),
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

func (bts *BridgingTxSender) createTx(
	ctx context.Context,
	srcChainID string,
	dstChainID string,
	bridgingType BridgingType,
	senderAddr string,
	receivers []BridgingTxReceiver,
	exactSumNotAllowed bool,
) ([]byte, string, error) {
	srcConfig, existsSrc := bts.chainConfigMap[srcChainID]
	dstConfig, existsDst := bts.chainConfigMap[dstChainID]

	if !existsSrc || !existsDst {
		return nil, "", fmt.Errorf("src %s or dst %s chain config not found", srcChainID, dstChainID)
	}

	queryTip, protocolParams, utxos, err := bts.GetDynamicParameters(ctx, srcConfig, senderAddr)
	if err != nil {
		return nil, "", err
	}

	neededSumCurrencyLovelace, neededSumNativeToken, feeOnSrcCurrencyLovelace := getNeededAmounts(
		bridgingType, srcConfig, dstConfig, bts.bridgingFeeAmount, receivers)

	metadata, err := bts.createMetadata(
		dstChainID, senderAddr, bridgingType, srcConfig, dstConfig, receivers, BridgingRequestMetadataCurrencyInfo{
			SrcAmount:  feeOnSrcCurrencyLovelace,
			DestAmount: bts.bridgingFeeAmount,
		})
	if err != nil {
		return nil, "", err
	}

	potentialFee := setOrDefault(srcConfig.PotentialFee, defaultPotentialFee)
	ttlSlotNumberInc := setOrDefault(srcConfig.TTLSlotNumberInc, defaultTTLSlotNumberInc)

	lovelaceExactSumModificator := uint64(0)
	// do not satisfy exact sum for lovelace if there is a native tokens involed or exact sum is not allowed
	if exactSumNotAllowed || neededSumNativeToken != 0 {
		lovelaceExactSumModificator = srcConfig.MinUtxoValue
	}

	outputNativeTokens := []cardanowallet.TokenAmount(nil)
	conditions := map[string]AmountCondition{
		cardanowallet.AdaTokenName: {
			Exact:   neededSumCurrencyLovelace + potentialFee + lovelaceExactSumModificator,
			AtLeast: neededSumCurrencyLovelace + potentialFee + srcConfig.MinUtxoValue,
		},
	}

	if neededSumNativeToken != 0 {
		conditions[srcConfig.NativeTokenFullName] = AmountCondition{
			Exact:   neededSumNativeToken,
			AtLeast: neededSumNativeToken,
		}
	}

	inputs, err := GetUTXOsForAmounts(utxos, conditions, bts.maxInputsPerTx)
	if err != nil {
		return nil, "", err
	}

	if neededSumNativeToken != 0 {
		nativeToken, err := getNativeToken(srcConfig.NativeTokenFullName, neededSumNativeToken)
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
			Addr:   srcConfig.MultiSigAddr,
			Amount: neededSumCurrencyLovelace,
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

	feeCurrencyLovelace, err := builder.CalculateFee(0)
	if err != nil {
		return nil, "", err
	}

	builder.SetFee(feeCurrencyLovelace)

	inputsSumCurrencyLovelace := inputs.Sum[cardanowallet.AdaTokenName]
	change := inputsSumCurrencyLovelace - neededSumCurrencyLovelace - feeCurrencyLovelace
	// handle overflow or insufficient amount
	if change != 0 && (change > inputsSumCurrencyLovelace || change < srcConfig.MinUtxoValue) {
		return []byte{}, "", fmt.Errorf("insufficient amount %d for %d or min utxo not satisfied",
			inputsSumCurrencyLovelace, neededSumCurrencyLovelace+feeCurrencyLovelace)
	}

	if change != 0 {
		builder.UpdateOutputAmount(-1, change)
	} else {
		builder.RemoveOutput(-1)
	}

	return builder.Build()
}

func (bts BridgingTxSender) GetDynamicParameters(
	ctx context.Context, srcConfig BridgingTxChainConfig, addr string,
) (qtd cardanowallet.QueryTipData, protocolParams []byte, utxos []cardanowallet.Utxo, err error) {
	protocolParams = srcConfig.ProtocolParameters
	if protocolParams == nil {
		protocolParams, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]byte, error) {
			return bts.txProviderSrc.GetProtocolParameters(ctx)
		})
		if err != nil {
			return
		}
	}

	qtd, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) (cardanowallet.QueryTipData, error) {
		return bts.txProviderSrc.GetTip(ctx)
	})
	if err != nil {
		return
	}

	utxos, err = infracommon.ExecuteWithRetry(ctx, func(ctx context.Context) ([]cardanowallet.Utxo, error) {
		return bts.txProviderSrc.GetUtxos(ctx, addr)
	})

	return qtd, protocolParams, utxos, err
}

func getNeededAmounts(
	bridgingType BridgingType, srcConfig, dstConfig BridgingTxChainConfig,
	bridgingFeeAmount uint64, receivers []BridgingTxReceiver,
) (neededSumCurrencyLovelace uint64, neededSumNativeToken uint64, feeOnSrcCurrencyLovelace uint64) {
	feeOnSrcCurrencyLovelace = mul(bridgingFeeAmount, srcConfig.ExchangeRate) // fee is always paid in lovelace
	neededSumCurrencyLovelace = feeOnSrcCurrencyLovelace

	for _, x := range receivers {
		switch bridgingType {
		case BridgingTypeNativeTokenOnSource:
			neededSumNativeToken += x.Amount
			neededSumCurrencyLovelace += srcConfig.MinUtxoValue // NOTE: is this good -> shell we count only once for multisig?
		case BridgingTypeCurrencyOnSource:
			neededSumNativeToken += mul(dstConfig.MinUtxoValue, srcConfig.ExchangeRate)
			neededSumCurrencyLovelace += x.Amount
		default:
			neededSumCurrencyLovelace += x.Amount
		}
	}

	return
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

func setOrDefault(val, def uint64) uint64 {
	if val == 0 {
		return def
	}

	return val
}
