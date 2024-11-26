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
	MinUTxODefaultValue = uint64(1_344_798) // 1_000_000
	draftTxFile         = "tx.draft"
)

type TxInput struct {
	Hash  string `json:"hsh"`
	Index uint32 `json:"ind"`
}

func (i TxInput) String() string {
	return fmt.Sprintf("%s#%d", i.Hash, i.Index)
}

type TxInputWithPolicyScript struct {
	Input        TxInput
	PolicyScript IPolicyScript
}

type TxOutput struct {
	Addr        string `json:"addr"`
	Amount      uint64 `json:"amount"`
	Token       string `json:"token"`
	TokenAmount uint64 `json:"tokenamount"`
}

func (o TxOutput) String() string {
	return fmt.Sprintf("%s+%d", o.Addr, o.Amount)
}

func (o TxOutput) StringMint() string {
	token := fmt.Sprintf("%d ", o.TokenAmount) + o.Token
	return fmt.Sprintf("%s+%d+%s", o.Addr, o.Amount, token)
}

type TxBuilder struct {
	baseDirectory      string
	inputs             []TxInputWithPolicyScript
	outputs            []TxOutput
	metadata           []byte
	protocolParameters []byte
	timeToLive         uint64
	testNetMagic       uint
	minOutputAmount    uint64
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

func (b *TxBuilder) SetMinOutputAmount(minOutputAmount uint64) *TxBuilder {
	b.minOutputAmount = minOutputAmount

	return b
}

func (b *TxBuilder) AddInputsWithScript(script IPolicyScript, inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			Input:        inp,
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
			Input:        inp,
			PolicyScript: scripts[i],
		})
	}

	return b
}

func (b *TxBuilder) AddInputs(inputs ...TxInput) *TxBuilder {
	for _, inp := range inputs {
		b.inputs = append(b.inputs, TxInputWithPolicyScript{
			Input: inp,
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

func (b *TxBuilder) UpdateNTOutputAmount(index int, amount uint64) *TxBuilder {
	if index < 0 {
		index = len(b.outputs) + index
	}

	b.outputs[index].TokenAmount = amount

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

func (b *TxBuilder) CalculateFeeMint(witnessCount int) (uint64, error) {
	if b.protocolParameters == nil {
		return 0, errors.New("protocol parameters not set")
	}

	protocolParamsFilePath := filepath.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, FilePermission); err != nil {
		return 0, err
	}

	if err := b.buildRawTxMint(protocolParamsFilePath, 0); err != nil {
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

func (b *TxBuilder) CalculateFeeNativeToken(witnessCount int) (uint64, error) {
	if b.protocolParameters == nil {
		return 0, errors.New("protocol parameters not set")
	}

	protocolParamsFilePath := filepath.Join(b.baseDirectory, "protocol-parameters.json")
	if err := os.WriteFile(protocolParamsFilePath, b.protocolParameters, FilePermission); err != nil {
		return 0, err
	}

	if err := b.buildRawNativeTokenTx(protocolParamsFilePath, 0); err != nil {
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

func (b *TxBuilder) BuildNativeToken() ([]byte, string, error) {
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

	if err := b.buildRawNativeTokenTx(protocolParamsFilePath, b.fee); err != nil {
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

func (b *TxBuilder) BuildMint() ([]byte, string, error) {
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

	if err := b.buildRawTxMint(protocolParamsFilePath, b.fee); err != nil {
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
	minAmount := b.minOutputAmount
	if minAmount == 0 {
		minAmount = MinUTxODefaultValue
	}

	var errs []error

	for i, x := range b.outputs {
		if x.Amount < minAmount {
			errs = append(errs,
				fmt.Errorf("output (%d, %s) has insufficient amount %d", i, x.Addr, x.Amount))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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

	for i, inp := range b.inputs {
		args = append(args, "--tx-in", inp.Input.String())

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

func (b *TxBuilder) buildRawNativeTokenTx(protocolParamsFilePath string, fee uint64) error {
	// tokenName := "54657374746F6B656E"

	// a := []string{
	// 	"transaction", "policyid",
	// 	"--script-file", "/home/dejan/dev/sandbox/test/policy" + "prime" + ".script",
	// }
	// policyId, err := runCommand("cardano-cli", a)
	// if err != nil {
	// 	fmt.Printf("DN_LOG_TAG ERROR creating PolicyID %+v\n", err)
	// }

	// a := []string{
	// 	"transaction", "policyid",
	// 	"--script-file", b.baseDirectory + "/policyprime.script",
	// }
	// policyId, err := runCommand("cardano-cli", a)
	// if err != nil {
	// 	fmt.Printf("DN_LOG_TAG ERROR creating PolicyID %+v\n", err)

	// 	return err
	// }

	policyId, err := b.GetPolicyID("prime")
	if err != nil {
		return err
	}

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

	for i, inp := range b.inputs {
		args = append(args, "--tx-in", inp.Input.String())

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

	fmt.Printf("\n\nDN_LOG_TAG b.baseDirectory: %s\n\n", b.baseDirectory)

	for _, out := range b.outputs {
		if out.TokenAmount > 0 {
			token := policyId[:len(policyId)-1] + "." + out.Token

			outToken := fmt.Sprintf("%d ", out.TokenAmount) + token
			outStr := fmt.Sprintf("%s+%d+%s", out.Addr, out.Amount, outToken)
			args = append(args, "--tx-out", outStr)

		} else {
			args = append(args, "--tx-out", out.String())
		}
	}

	_, err = runCommand(b.cardanoCliBinary, args)

	return err
}

func (b *TxBuilder) buildRawTxMint(protocolParamsFilePath string, fee uint64) error {
	// tokenName := "54657374746F6B656E"

	// DN_TODO: Pass policyID with the name
	// a := []string{
	// 	"transaction", "policyid",
	// 	"--script-file", b.baseDirectory + "/policyprime.script",
	// }
	// policyId, err := runCommand("cardano-cli", a)
	// if err != nil {
	// 	fmt.Printf("DN_LOG_TAG ERROR creating PolicyID %+v\n", err)

	// 	return err
	// }

	policyId, err := b.GetPolicyID("prime")
	if err != nil {
		return err
	}

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

	for i, inp := range b.inputs {
		args = append(args, "--tx-in", inp.Input.String())

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
		if out.TokenAmount > 0 {
			token := policyId[:len(policyId)-1] + "." + out.Token
			fmt.Printf("DN_LOG_TAG mint token: %+v\n", token)

			mintToken := fmt.Sprintf("%d ", out.TokenAmount) + token
			args = append(args, "--mint", mintToken)
			// DN_TODO: FIX policy path
			// args = append(args, "--minting-script-file", b.baseDirectory+"/policy_prime.script")
			args = append(args, "--minting-script-file", "/home/dejan/dev/sandbox/test/policy_prime.script")

			// tokenOut := out.StringMint()

			fmt.Printf("tokenOut MINT: %s\n", mintToken)

			args = append(args, "--tx-out", fmt.Sprintf("%s+%d+%s", out.Addr, out.Amount, mintToken))

		} else if out.Amount > 0 {
			args = append(args, "--tx-out", out.String())
		} else {
			fmt.Println("ERROR: output with 0 amount")
		}
	}

	_, err = runCommand(b.cardanoCliBinary, args)

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

func (b *TxBuilder) GetPolicyID(chainID string) (string, error) {
	a := []string{
		"transaction", "policyid",
		// "--script-file", b.baseDirectory + "/policy_" + chainID + ".script",
		"--script-file", "/home/dejan/dev/sandbox/test/policy_" + chainID + ".script",
	}
	policyId, err := runCommand("cardano-cli", a)
	if err != nil {
		fmt.Printf("DN_LOG_TAG ERROR getting PolicyID %+v\n", err)

		return "", err
	}

	return policyId, nil
}

func (b *TxBuilder) GetBaseDir() string {
	return b.baseDirectory
}
