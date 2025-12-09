package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const draftTxFile = "tx.draft"

type TxInput struct {
	Hash  string `json:"hsh"`
	Index uint32 `json:"ind"`
}

func NewTxInput(hash string, index uint32) TxInput {
	return TxInput{
		Hash:  hash,
		Index: index,
	}
}

func (i TxInput) String() string {
	return fmt.Sprintf("%s#%d", i.Hash, i.Index)
}

type TxOutput struct {
	Addr   string        `json:"addr"`
	Amount uint64        `json:"amount"`
	Tokens []TokenAmount `json:"token,omitempty"`
}

func NewTxOutput(addr string, amount uint64, tokens ...TokenAmount) TxOutput {
	return TxOutput{
		Addr:   addr,
		Amount: amount,
		Tokens: tokens,
	}
}

func (o TxOutput) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s+%d", o.Addr, o.Amount))

	for _, token := range o.Tokens {
		sb.WriteRune('+')
		sb.WriteString(token.String())
	}

	return sb.String()
}

type TxBuilder struct {
	baseDirectory       string
	inputs              []txInputWithPolicyScript
	collateralInputs    []TxInput
	totalCollateral     uint64
	collateralOutputs   []TxOutput
	outputs             []TxOutputWithRefScript
	mints               txTokenMintInputs
	certificates        []txCertificateWithPolicyScript
	metadata            []byte
	protocolParameters  []byte
	timeToLive          uint64
	testNetMagic        uint
	fee                 uint64
	additionalWitnesses int
	withdrawalData      txWithdrawalDataPolicyScript
	plutusTokenMint     txPlutusTokenMintInputs
	era                 string
	realEraName         string
	cardanoCliBinary    string
}

func NewTxBuilder(cardanoCliBinary string) (*TxBuilder, error) {
	return NewTxBuilderForEra(cardanoCliBinary, DefaultEra)
}

func NewTxBuilderForEra(cardanoCliBinary string, era string) (*TxBuilder, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-txs")
	if err != nil {
		return nil, err
	}

	realEraName, err := NewCliUtilsForEra(cardanoCliBinary, era).GetRealEraName()
	if err != nil {
		return nil, err
	}

	return &TxBuilder{
		baseDirectory:    baseDirectory,
		cardanoCliBinary: cardanoCliBinary,
		realEraName:      realEraName,
		era:              era,
	}, nil
}

func (b *TxBuilder) Dispose() {
	os.RemoveAll(b.baseDirectory)
}

func (b *TxBuilder) SetTestNetMagic(testNetMagic uint) *TxBuilder {
	b.testNetMagic = testNetMagic

	return b
}

func (b *TxBuilder) SetFee(fee uint64) *TxBuilder {
	b.fee = fee

	return b
}

func (b *TxBuilder) SetTotalCollateral(totalCollateral uint64) *TxBuilder {
	b.totalCollateral = totalCollateral

	return b
}

func (b *TxBuilder) AddInputsWithScript(script IPolicyScript, inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, txInputWithPolicyScript{
			txInput:      inp,
			policyScript: script,
		})
	}

	return b
}

func (b *TxBuilder) AddInputsWithScripts(inputs []TxInput, scripts []IPolicyScript) *TxBuilder {
	cnt := len(inputs)
	if l := len(scripts); cnt > l {
		cnt = l
	}

	for i, inp := range inputs[:cnt] {
		b.inputs = append(b.inputs, txInputWithPolicyScript{
			txInput:      inp,
			policyScript: scripts[i],
		})
	}

	return b
}

func (b *TxBuilder) AddInputs(inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, txInputWithPolicyScript{
			txInput: inp,
		})
	}

	return b
}

func (b *TxBuilder) AddCollateralInputs(inputs []TxInput) *TxBuilder {
	b.collateralInputs = append(b.collateralInputs, inputs...)

	return b
}

func (b *TxBuilder) AddCollateralOutput(output TxOutput) *TxBuilder {
	b.collateralOutputs = append(b.collateralOutputs, output)

	return b
}

func (b *TxBuilder) AddOutputs(outputs ...TxOutput) *TxBuilder {
	for _, output := range outputs {
		b.outputs = append(b.outputs, TxOutputWithRefScript{
			TxOutput: output,
		})
	}

	return b
}

func (b *TxBuilder) AddOutputWithPlutusScript(script ICardanoArtifact, amount uint64) (*TxBuilder, string, error) {
	plutusAddr, err := b.GetPlutusScriptAddress(b.testNetMagic, script)
	if err != nil {
		return nil, "", err
	}

	b.outputs = append(b.outputs, TxOutputWithRefScript{
		TxOutput:     NewTxOutput(plutusAddr, amount),
		PlutusScript: script,
	})

	return b, plutusAddr, nil
}

func (b *TxBuilder) ReplaceOutput(index int, output TxOutput) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	b.outputs[index] = TxOutputWithRefScript{
		TxOutput: output,
	}

	return b
}

func (b *TxBuilder) UpdateOutputAmount(index int, amount uint64, tokenAmounts ...uint64) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	b.outputs[index].TxOutput.Amount = amount

	for i, amount := range tokenAmounts {
		if len(b.outputs[index].TxOutput.Tokens) > i {
			b.outputs[index].TxOutput.Tokens[i].Amount = amount
		}
	}

	return b
}

func (b *TxBuilder) RemoveOutput(index int) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	copy(b.outputs[index:], b.outputs[index+1:])
	b.outputs = b.outputs[:len(b.outputs)-1]

	return b
}

func (b *TxBuilder) UpdateCollateralOutputAmount(index int, amount uint64) *TxBuilder {
	if index < 0 {
		index = len(b.collateralOutputs) + index
	}

	b.collateralOutputs[index].Amount = amount

	return b
}

func (b *TxBuilder) AddTokenMints(
	policyScripts []IPolicyScript, tokens []TokenAmount,
) *TxBuilder {
	b.mints.tokens = append(b.mints.tokens, tokens...)
	b.mints.policyScripts = append(b.mints.policyScripts, policyScripts...)

	return b
}

func (b *TxBuilder) SetMetaData(metadata []byte) *TxBuilder {
	b.metadata = metadata

	return b
}

func (b *TxBuilder) SetProtocolParameters(protocolParameters []byte) *TxBuilder {
	b.protocolParameters = protocolParameters

	return b
}

func (b *TxBuilder) SetTimeToLive(timeToLive uint64) *TxBuilder {
	b.timeToLive = timeToLive

	return b
}

func (b *TxBuilder) SetWithdrawalData(
	stakeAddress string, rewardsAmount uint64, policyScript IPolicyScript,
) *TxBuilder {
	b.withdrawalData = txWithdrawalDataPolicyScript{
		stakeAddress: stakeAddress,
		rewardAmount: rewardsAmount,
		policyScript: policyScript,
	}

	return b
}

func (b *TxBuilder) SetExecutionUnitParams(cpu uint64, memory uint64) *TxBuilder {
	b.plutusTokenMint.CPU = cpu
	b.plutusTokenMint.Memory = memory

	return b
}

func (b *TxBuilder) AddCertificates(script IPolicyScript, certificates ...ICardanoArtifact) *TxBuilder {
	for _, cert := range certificates {
		b.certificates = append(b.certificates, txCertificateWithPolicyScript{
			certificate:  cert,
			policyScript: script,
		})
	}

	return b
}

func (b *TxBuilder) AddPlutusTokenMints(
	tokens []MintTokenAmount, txInReference TxInput, tokensPolicyID string,
) *TxBuilder {
	b.plutusTokenMint = txPlutusTokenMintInputs{
		tokens:         tokens,
		txInReference:  txInReference,
		tokensPolicyID: tokensPolicyID,
	}

	return b
}

func (b *TxBuilder) SetAdditionalWitnessCount(count int) *TxBuilder {
	b.additionalWitnesses = count

	return b
}

func (b *TxBuilder) CalculateFee(witnessCount int) (uint64, error) {
	if b.protocolParameters == nil {
		return 0, errors.New("protocol parameters not set")
	}

	protocolParamsFilePath := filepath.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, FilePermission); err != nil {
		return 0, err
	}

	if err := b.buildRawTx(protocolParamsFilePath, 0); err != nil {
		return 0, err
	}

	if witnessCount == 0 {
		for _, inp := range b.inputs {
			witnessCount += inp.GetWitnessCount()
		}

		witnessCount += getCertfificatesWitnessCount(b.certificates)
		witnessCount += b.withdrawalData.GetWitnessCount()
		witnessCount = max(witnessCount+b.additionalWitnesses, 1)
	}

	response, err := runCommand(b.cardanoCliBinary, append([]string{
		b.era, "transaction", "calculate-min-fee",
		"--tx-body-file", filepath.Join(b.baseDirectory, draftTxFile),
		"--tx-in-count", strconv.Itoa(len(b.inputs)),
		"--tx-out-count", strconv.Itoa(len(b.outputs)),
		"--witness-count", strconv.FormatUint(uint64(witnessCount), 10),
		"--protocol-params-file", protocolParamsFilePath,
	}, getTestNetMagicArgs(b.testNetMagic)...))
	if err != nil {
		return 0, err
	}

	type feeOutputStruct struct {
		Fee uint64 `json:"fee"`
	}

	var obj feeOutputStruct

	if err := json.Unmarshal([]byte(response), &obj); err == nil {
		return obj.Fee, nil
	}

	return strconv.ParseUint(strings.Split(response, " ")[0], 10, 64)
}

func (b *TxBuilder) CalculateMinUtxo(outputwithRefScript TxOutputWithRefScript) (uint64, error) {
	if b.protocolParameters == nil {
		return 0, errors.New("protocol parameters not set")
	}

	protocolParamsFilePath := filepath.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, FilePermission); err != nil {
		return 0, err
	}

	args := []string{
		b.era, "transaction", "calculate-min-required-utxo",
		"--protocol-params-file", protocolParamsFilePath,
		"--tx-out", outputwithRefScript.TxOutput.String(),
	}

	if outputwithRefScript.PlutusScript != nil {
		plutusScriptFilePath, err := writeSerializableToFile(outputwithRefScript.PlutusScript, b.baseDirectory, "ps.plutus")
		if err != nil {
			return 0, err
		}

		args = append(args, "--tx-out-reference-script-file", plutusScriptFilePath)
	}

	result, err := runCommand(b.cardanoCliBinary, args)
	if err != nil {
		return 0, err
	}

	result = strings.TrimPrefix(
		strings.TrimSpace(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(result)), AdaTokenName)), "coin ")

	return strconv.ParseUint(result, 0, 64)
}

func (b *TxBuilder) build(checkOutputs bool) ([]byte, string, error) {
	if b.protocolParameters == nil {
		return nil, "", errors.New("protocol parameters not set")
	}

	if checkOutputs {
		if err := b.CheckOutputs(); err != nil {
			return nil, "", err
		}

		if err := b.CheckCollateralOutputs(); err != nil {
			return nil, "", err
		}
	}

	protocolParamsFilePath := filepath.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, FilePermission); err != nil {
		return nil, "", err
	}

	if err := b.buildRawTx(protocolParamsFilePath, b.fee); err != nil {
		return nil, "", err
	}

	bytes, err := os.ReadFile(filepath.Join(b.baseDirectory, draftTxFile))
	if err != nil {
		return nil, "", err
	}

	txRaw, err := newTransactionUnwitnessedRawFromJSON(bytes)
	if err != nil {
		return nil, "", err
	}

	txHash, err := NewCliUtilsForEra(b.cardanoCliBinary, b.era).getTxHash(txRaw, b.baseDirectory, b.realEraName)
	if err != nil {
		return nil, "", err
	}

	return txRaw, txHash, nil
}

func (b *TxBuilder) UncheckedBuild() ([]byte, string, error) {
	return b.build(false)
}

func (b *TxBuilder) Build() ([]byte, string, error) {
	return b.build(true)
}

func (b *TxBuilder) CheckOutputs() error {
	var errs []error

	for i, x := range b.outputs {
		if x.TxOutput.Amount == 0 {
			errs = append(errs, fmt.Errorf("output (%s, %d) amount not specified", x.TxOutput.Addr, i))
		}
	}

	return errors.Join(errs...)
}

func (b *TxBuilder) CheckCollateralOutputs() error {
	var errs []error

	for i, x := range b.collateralOutputs {
		if x.Amount == 0 {
			errs = append(errs, fmt.Errorf("collateral return output (%s, %d) amount not specified", x.Addr, i))
		}
	}

	return errors.Join(errs...)
}

func (b *TxBuilder) buildRawTx(protocolParamsFilePath string, fee uint64) error {
	args := []string{
		b.era, "transaction", "build-raw",
		"--protocol-params-file", protocolParamsFilePath,
		"--fee", strconv.FormatUint(fee, 10),
		"--invalid-hereafter", strconv.FormatUint(b.timeToLive, 10),
		"--out-file", filepath.Join(b.baseDirectory, draftTxFile),
	}

	if b.metadata != nil {
		metaDataFilePath := filepath.Join(b.baseDirectory, "metadata.json")
		if err := os.WriteFile(metaDataFilePath, b.metadata, FilePermission); err != nil {
			return err
		}

		args = append(args, "--metadata-json-file", metaDataFilePath)
	}

	for i, cert := range b.certificates {
		if err := cert.Apply(&args, b.baseDirectory, i); err != nil {
			return err
		}
	}

	if err := b.withdrawalData.Apply(&args, b.baseDirectory); err != nil {
		return err
	}

	if err := b.mints.Apply(&args, b.baseDirectory); err != nil {
		return err
	}

	for i, inp := range b.inputs {
		if err := inp.Apply(&args, b.baseDirectory, i); err != nil {
			return err
		}
	}

	for _, inp := range b.collateralInputs {
		args = append(args, "--tx-in-collateral", inp.String())
	}

	if len(b.collateralOutputs) > 0 {
		args = append(args, "--tx-total-collateral", strconv.FormatUint(b.totalCollateral, 10))

		for _, cout := range b.collateralOutputs {
			args = append(args, "--tx-out-return-collateral", cout.String())
		}
	}

	if err := b.plutusTokenMint.Apply(&args); err != nil {
		return err
	}

	for i, out := range b.outputs {
		if err := out.Apply(&args, b.baseDirectory, i); err != nil {
			return err
		}
	}

	_, err := runCommand(b.cardanoCliBinary, args)

	return err
}

// SignTx signs tx and assembles all signatures in final tx
func (b *TxBuilder) SignTx(txRaw []byte, signers []ITxSigner) (res []byte, err error) {
	witnesses := make([][]byte, len(signers))
	for i, signer := range signers {
		witnesses[i], err = b.CreateTxWitness(txRaw, signer)
		if err != nil {
			return nil, err
		}
	}

	return b.AssembleTxWitnesses(txRaw, witnesses)
}

// CreateTxWitness signs transaction hash and creates witness cbor
func (b *TxBuilder) CreateTxWitness(txRaw []byte, signer ITxSigner) ([]byte, error) {
	outFilePath := filepath.Join(b.baseDirectory, "tx.wit")
	txFilePath := filepath.Join(b.baseDirectory, "tx.raw")
	signingKeyPath := filepath.Join(b.baseDirectory, "tx.skey")
	signingKey, _ := signer.GetSigningKeys()

	txBytes, err := transactionUnwitnessedRaw(txRaw).ToJSON(b.realEraName)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(txFilePath, txBytes, FilePermission); err != nil {
		return nil, err
	}

	var title string
	if len(signingKey) > KeySize {
		title = paymentExtendedSigningKeyShelley
	} else {
		title = paymentSigningKeyShelley
	}

	key, err := NewKeyFromBytes(title, "", signingKey)
	if err != nil {
		return nil, err
	}

	if err := key.WriteToFile(signingKeyPath); err != nil {
		return nil, err
	}

	args := append([]string{
		b.era, "transaction", "witness",
		"--signing-key-file", signingKeyPath,
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath},
		getTestNetMagicArgs(b.testNetMagic)...)

	if _, err = runCommand(b.cardanoCliBinary, args); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(outFilePath)
	if err != nil {
		return nil, err
	}

	return newTransactionWitnessedRawFromJSON(bytes)
}

// AssembleTxWitnesses assembles final signed transaction
func (b *TxBuilder) AssembleTxWitnesses(txRaw []byte, witnesses [][]byte) ([]byte, error) {
	outFilePath := filepath.Join(b.baseDirectory, "tx.sig")
	txFilePath := filepath.Join(b.baseDirectory, "tx.raw")
	witnessesFilePaths := make([]string, len(witnesses))

	for i, witness := range witnesses {
		witnessesFilePaths[i] = filepath.Join(b.baseDirectory, fmt.Sprintf("witness-%d", i+1))

		content, err := TxWitnessRaw(witness).ToJSON(b.realEraName)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(witnessesFilePaths[i], content, FilePermission); err != nil {
			return nil, err
		}
	}

	txBytes, err := transactionUnwitnessedRaw(txRaw).ToJSON(b.realEraName)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(txFilePath, txBytes, FilePermission); err != nil {
		return nil, err
	}

	args := []string{
		b.era, "transaction", "assemble",
		"--tx-body-file", txFilePath,
		"--out-file", outFilePath}

	for _, fp := range witnessesFilePaths {
		args = append(args, "--witness-file", fp)
	}

	if _, err = runCommand(b.cardanoCliBinary, args); err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(outFilePath)
	if err != nil {
		return nil, err
	}

	return newTransactionWitnessedRawFromJSON(bytes)
}

type txInputWithPolicyScript struct {
	txInput      TxInput
	policyScript IPolicyScript
}

func (txInputPS txInputWithPolicyScript) Apply(
	args *[]string, basePath string, indx int,
) error {
	*args = append(*args, "--tx-in", txInputPS.txInput.String())

	if txInputPS.policyScript != nil {
		filePath, err := writeSerializableToFile(txInputPS.policyScript, basePath, fmt.Sprintf("ps_%d.json", indx))
		if err != nil {
			return err
		}

		*args = append(*args, "--tx-in-script-file", filePath)
	}

	return nil
}

func (txInputPS txInputWithPolicyScript) GetWitnessCount() int {
	if txInputPS.policyScript != nil {
		return txInputPS.policyScript.GetCount()
	}

	return 1
}

type txTokenMintInputs struct {
	tokens        []TokenAmount
	policyScripts []IPolicyScript
}

func (txMint txTokenMintInputs) Apply(
	args *[]string, basePath string,
) error {
	if len(txMint.tokens) == 0 {
		return nil
	}

	var sb strings.Builder

	for _, token := range txMint.tokens {
		if sb.Len() > 0 {
			sb.WriteRune('+')
		}

		sb.WriteString(token.String())
	}

	*args = append(*args, "--mint", sb.String())

	for indx, policyScript := range txMint.policyScripts {
		policyFilePath, err := writeSerializableToFile(policyScript, basePath, fmt.Sprintf("ps_mint_%d.json", indx))
		if err != nil {
			return err
		}

		*args = append(*args, "--minting-script-file", policyFilePath)
	}

	return nil
}

type TxOutputWithRefScript struct {
	TxOutput     TxOutput
	PlutusScript ICardanoArtifact
}

func (txOutputPlutus TxOutputWithRefScript) Apply(
	args *[]string, basePath string, indx int,
) error {
	*args = append(*args, "--tx-out", txOutputPlutus.TxOutput.String())

	if txOutputPlutus.PlutusScript != nil {
		filePath, err := writeSerializableToFile(txOutputPlutus.PlutusScript, basePath, fmt.Sprintf("ps_%d.plutus", indx))
		if err != nil {
			return fmt.Errorf("failed to write Plutus script to file: %w", err)
		}

		*args = append(*args, "--tx-out-reference-script-file", filePath)
	}

	return nil
}

func (b *TxBuilder) GetPlutusScriptAddress(
	testNetMagic uint, plutusScript ICardanoArtifact,
) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-multisig-addr")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory for Plutus script: %w", err)
	}

	defer os.RemoveAll(baseDirectory)

	plutusScriptFilePath, err := writeSerializableToFile(plutusScript, baseDirectory, "ps.plutus")
	if err != nil {
		return "", fmt.Errorf("failed to write Plutus script to file: %w", err)
	}

	args := []string{
		b.era, "address", "build",
		"--payment-script-file", plutusScriptFilePath,
	}

	response, err := runCommand(b.cardanoCliBinary, append(args, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

type txCertificateWithPolicyScript struct {
	certificate  ICardanoArtifact
	policyScript IPolicyScript
}

func (txCert txCertificateWithPolicyScript) Apply(
	args *[]string, basePath string, index int,
) error {
	if txCert.certificate == nil {
		return nil
	}

	certificateFilePath, err := writeSerializableToFile(
		txCert.certificate,
		basePath,
		fmt.Sprintf("certificate_%d.cert", index),
	)
	if err != nil {
		return err
	}

	*args = append(*args, "--certificate-file", certificateFilePath)

	if txCert.policyScript == nil {
		return nil
	}

	policyFilePath, err := writeSerializableToFile(
		txCert.policyScript,
		basePath,
		fmt.Sprintf("cert_policy_%d.json", index),
	)
	if err != nil {
		return err
	}

	*args = append(*args, "--certificate-script-file", policyFilePath)

	return nil
}

func getCertfificatesWitnessCount(certificates []txCertificateWithPolicyScript) int {
	witnessCount := 0
	// In case we have multiple certificates for the same policy script
	// we don't need to multiply the witness number
	processedScripts := make([]IPolicyScript, 0, len(certificates))
	for _, cert := range certificates {
		alreadyProcessed := cert.policyScript != nil && slices.Contains(processedScripts, cert.policyScript)
		if !alreadyProcessed {
			witnessCount += cert.GetWitnessCount()
		}

		processedScripts = append(processedScripts, cert.policyScript)
	}

	return witnessCount
}

func (txCert txCertificateWithPolicyScript) GetWitnessCount() int {
	if txCert.policyScript != nil {
		return txCert.policyScript.GetCount()
	}

	if txCert.certificate != nil {
		return 1
	}

	return 0
}

type txWithdrawalDataPolicyScript struct {
	stakeAddress string
	rewardAmount uint64
	policyScript IPolicyScript
}

func (txWithdrawalData txWithdrawalDataPolicyScript) Apply(
	args *[]string, basePath string,
) error {
	if txWithdrawalData.stakeAddress == "" {
		return nil
	}

	*args = append(*args, "--withdrawal",
		fmt.Sprintf("%s+%d", txWithdrawalData.stakeAddress, txWithdrawalData.rewardAmount))

	if txWithdrawalData.policyScript == nil {
		return nil
	}

	policyFilePath, err := writeSerializableToFile(txWithdrawalData.policyScript, basePath, "policy_withdrawal.json")
	if err != nil {
		return err
	}

	*args = append(*args, "--withdrawal-script-file", policyFilePath)

	return nil
}

func (txWithdrawalData txWithdrawalDataPolicyScript) GetWitnessCount() int {
	if txWithdrawalData.policyScript != nil {
		return txWithdrawalData.policyScript.GetCount()
	}

	if txWithdrawalData.stakeAddress != "" {
		return 1
	}

	return 0
}

type MintTokenAmount struct {
	Token
	Amount     uint64
	IsNegative bool
}

func NewMintTokenAmount(token Token, amount uint64, isNegative bool) MintTokenAmount {
	return MintTokenAmount{
		Token:      token,
		Amount:     amount,
		IsNegative: isNegative,
	}
}

type txPlutusTokenMintInputs struct {
	tokens         []MintTokenAmount
	txInReference  TxInput
	CPU            uint64
	Memory         uint64
	tokensPolicyID string
}

func (txMint txPlutusTokenMintInputs) Apply(args *[]string) error {
	if len(txMint.tokens) == 0 {
		return nil
	}

	parts := make([]string, 0, len(txMint.tokens))
	populated := false

	for _, token := range txMint.tokens {
		sign := ""
		if token.IsNegative {
			sign = "-"
		}

		if token.Amount > 0 {
			parts = append(parts, fmt.Sprintf("%s%d %s", sign, token.Amount, token.String()))
			populated = true
		}
	}

	if populated {
		mintArgs := []string{
			"--mint", strings.Join(parts, " + "),
			"--mint-tx-in-reference", txMint.txInReference.String(),
			"--mint-plutus-script-v2",
			"--mint-reference-tx-in-redeemer-value", "0",
			"--mint-reference-tx-in-execution-units", fmt.Sprintf("(%d,%d)", txMint.CPU, txMint.Memory),
			"--policy-id", txMint.tokensPolicyID,
		}

		*args = append(*args, mintArgs...)
	}

	return nil
}
