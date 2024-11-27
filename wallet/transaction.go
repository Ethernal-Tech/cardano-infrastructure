package wallet

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	draftTxFile = "tx.draft"
)

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

type TxInputWithPolicyScript struct {
	TxInput
	PolicyScript IPolicyScript
}

func NewTxInputWithPolicyScript(hash string, index uint32, ps IPolicyScript) TxInputWithPolicyScript {
	return TxInputWithPolicyScript{
		TxInput:      NewTxInput(hash, index),
		PolicyScript: ps,
	}
}

type TxOutput struct {
	Addr   string         `json:"addr"`
	Amount uint64         `json:"amount"`
	Tokens []ITokenAmount `json:"token,omitempty"`
}

func NewTxOutput(addr string, amount uint64, tokens ...ITokenAmount) TxOutput {
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
		sb.WriteString(fmt.Sprintf("+%d %s", token.TokenAmount(), token.TokenName()))
	}

	return sb.String()
}

type TxTokenAmount struct {
	PolicyID string `json:"pid"`
	Name     string `json:"name"`
	Amount   uint64 `json:"amount"`
}

func NewTxTokenAmount(policyID string, name string, amount uint64) *TxTokenAmount {
	return &TxTokenAmount{
		PolicyID: policyID,
		Name:     name,
		Amount:   amount,
	}
}

func NewTxTokenAmountWithFullName(name string, amount uint64) (*TxTokenAmount, error) {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("name should have two parts but instead has: %d", len(parts))
	}

	return &TxTokenAmount{
		PolicyID: parts[0],
		Name:     parts[1],
		Amount:   amount,
	}, nil
}

func (tt TxTokenAmount) TokenName() string {
	return fmt.Sprintf("%s.%s", tt.PolicyID, tt.Name)
}

func (tt TxTokenAmount) TokenAmount() uint64 {
	return tt.Amount
}

func (tt *TxTokenAmount) UpdateAmount(amount uint64) {
	tt.Amount = amount
}

func (tt TxTokenAmount) String() string {
	return fmt.Sprintf("%d %s.%s", tt.Amount, tt.PolicyID, tt.Name)
}

type TxTokenAmountWithPolicyScript struct {
	tokenAmount  ITokenAmount
	policyScript IPolicyScript
}

func (tt TxTokenAmountWithPolicyScript) Token() ITokenAmount {
	return tt.tokenAmount
}

func (tt TxTokenAmountWithPolicyScript) PolicyScript() IPolicyScript {
	return tt.policyScript
}

func NewTxTokenAmountWithPolicyScript(
	cardanoCliBinary string, name string, amount uint64, policyScript IPolicyScript,
) (*TxTokenAmountWithPolicyScript, error) {
	pid, err := NewCliUtils(cardanoCliBinary).GetPolicyID(policyScript)
	if err != nil {
		return nil, err
	}

	return &TxTokenAmountWithPolicyScript{
		tokenAmount:  NewTxTokenAmount(pid, name, amount),
		policyScript: policyScript,
	}, nil
}

type TxBuilder struct {
	baseDirectory      string
	inputs             []TxInputWithPolicyScript
	outputs            []TxOutput
	tokenMints         []ITokenAmountWithPolicyScript
	metadata           []byte
	protocolParameters []byte
	timeToLive         uint64
	testNetMagic       uint
	fee                uint64
	cardanoCliBinary   string
}

func NewTxBuilder(cardanoCliBinary string) (*TxBuilder, error) {
	baseDirectory, err := os.MkdirTemp("", "cardano-txs")
	if err != nil {
		return nil, err
	}

	return &TxBuilder{
		baseDirectory:    baseDirectory,
		cardanoCliBinary: cardanoCliBinary,
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

func (b *TxBuilder) AddInputsWithScript(script IPolicyScript, inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			TxInput:      inp,
			PolicyScript: script,
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
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			TxInput:      inp,
			PolicyScript: scripts[i],
		})
	}

	return b
}

func (b *TxBuilder) AddInputs(inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			TxInput: inp,
		})
	}

	return b
}

func (b *TxBuilder) AddOutputs(outputs ...TxOutput) *TxBuilder {
	b.outputs = append(b.outputs, outputs...)

	return b
}

func (b *TxBuilder) UpdateOutputAmount(index int, amount uint64) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	b.outputs[index].Amount = amount

	return b
}

func (b *TxBuilder) UpdateOutputTokensAmounts(index int, amounts ...uint64) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	for i, amount := range amounts {
		if len(b.outputs[index].Tokens) > i {
			b.outputs[index].Tokens[i].UpdateAmount(amount)
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

func (b *TxBuilder) AddTokenMints(addr string, amount uint64, tokenMints ...ITokenAmountWithPolicyScript) *TxBuilder {
	b.tokenMints = append(b.tokenMints, tokenMints...)
	// outputs should be updated too
	tokens := make([]ITokenAmount, len(tokenMints))
	for i, tokenMint := range tokenMints {
		tokens[i] = tokenMint.Token()
	}

	b.outputs = append(b.outputs, TxOutput{
		Addr:   addr,
		Amount: amount,
		Tokens: tokens,
	})

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
			if inp.PolicyScript != nil {
				witnessCount += inp.PolicyScript.GetCount()
			}
		}

		if witnessCount == 0 {
			witnessCount = 1
		}
	}

	feeOutput, err := runCommand(b.cardanoCliBinary, append([]string{
		"transaction", "calculate-min-fee",
		"--tx-body-file", filepath.Join(b.baseDirectory, draftTxFile),
		"--tx-in-count", strconv.Itoa(len(b.inputs)),
		"--tx-out-count", strconv.Itoa(len(b.outputs)),
		"--witness-count", strconv.FormatUint(uint64(witnessCount), 10),
		"--protocol-params-file", protocolParamsFilePath,
	}, getTestNetMagicArgs(b.testNetMagic)...))
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(strings.Split(feeOutput, " ")[0], 10, 64)
}

func (b *TxBuilder) Build() ([]byte, string, error) {
	if b.protocolParameters == nil {
		return nil, "", errors.New("protocol parameters not set")
	}

	if err := b.CheckOutputs(); err != nil {
		return nil, "", err
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

	txRaw, err := NewTransactionUnwitnessedRawFromJSON(bytes)
	if err != nil {
		return nil, "", err
	}

	txHash, err := NewCliUtils(b.cardanoCliBinary).getTxHash(txRaw, b.baseDirectory)
	if err != nil {
		return nil, "", err
	}

	return txRaw, txHash, nil
}

func (b *TxBuilder) CheckOutputs() error {
	var errs []error

	for i, x := range b.outputs {
		if x.Amount == 0 {
			errs = append(errs, fmt.Errorf("output (%s, %d) amount not specified", x.Addr, i))
		}
	}

	return errors.Join(errs...)
}

func (b *TxBuilder) buildRawTx(protocolParamsFilePath string, fee uint64) error {
	args := []string{
		"transaction", "build-raw",
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

	if len(b.tokenMints) > 0 {
		args = append(args, "--mint", getTokensStrings(b.tokenMints))

		for i, tokenMint := range b.tokenMints {
			policyScriptJSON, err := tokenMint.PolicyScript().GetPolicyScriptJSON()
			if err != nil {
				return err
			}

			policyFilePath := filepath.Join(b.baseDirectory, fmt.Sprintf("policy_mint_%d.json", i))
			if err := os.WriteFile(policyFilePath, policyScriptJSON, FilePermission); err != nil {
				return err
			}

			args = append(args, "--minting-script-file", policyFilePath)
		}
	}

	for i, inp := range b.inputs {
		args = append(args, "--tx-in", inp.String())

		if inp.PolicyScript != nil {
			policyScriptJSON, err := inp.PolicyScript.GetPolicyScriptJSON()
			if err != nil {
				return err
			}

			policyFilePath := filepath.Join(b.baseDirectory, fmt.Sprintf("policy_%d.json", i))
			if err := os.WriteFile(policyFilePath, policyScriptJSON, FilePermission); err != nil {
				return err
			}

			args = append(args, "--tx-in-script-file", policyFilePath)
		}
	}

	for _, out := range b.outputs {
		args = append(args, "--tx-out", out.String())
	}

	_, err := runCommand(b.cardanoCliBinary, args)

	return err
}

// AssembleTxWitnesses assembles final signed transaction
func (b *TxBuilder) AssembleTxWitnesses(txRaw []byte, witnesses [][]byte) ([]byte, error) {
	outFilePath := filepath.Join(b.baseDirectory, "tx.sig")
	txFilePath := filepath.Join(b.baseDirectory, "tx.raw")
	witnessesFilePaths := make([]string, len(witnesses))

	for i, witness := range witnesses {
		witnessesFilePaths[i] = filepath.Join(b.baseDirectory, fmt.Sprintf("witness-%d", i+1))

		content, err := TxWitnessRaw(witness).ToJSON()
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(witnessesFilePaths[i], content, FilePermission); err != nil {
			return nil, err
		}
	}

	txBytes, err := TransactionUnwitnessedRaw(txRaw).ToJSON()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(txFilePath, txBytes, FilePermission); err != nil {
		return nil, err
	}

	args := []string{
		"transaction", "assemble",
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

	return NewTransactionWitnessedRawFromJSON(bytes)
}

func getTokensStrings(tokens []ITokenAmountWithPolicyScript) string {
	var sb strings.Builder

	for _, token := range tokens {
		if sb.Len() > 0 {
			sb.WriteRune('+')
		}

		sb.WriteString(token.Token().String())
	}

	return sb.String()
}
