package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Ethernal-Tech/cardano-infrastructure/wallet/bech32"
)

type AddressInfo struct {
	Address  string `json:"address"`
	Base16   string `json:"base16"`
	Encoding string `json:"encoding"`
	Era      string `json:"era"`
	Type     string `json:"type"`
}

type CliUtils struct {
	cardanoCliBinary string
	era              string
}

func NewCliUtils(cardanoCliBinary string) CliUtils {
	return NewCliUtilsForEra(cardanoCliBinary, DefaultEra)
}

func NewCliUtilsForEra(cardanoCliBinary string, era string) CliUtils {
	return CliUtils{
		cardanoCliBinary: cardanoCliBinary,
		era:              era,
	}
}

// GetPolicyScriptBaseAddress returns base address for policy script
func (cu CliUtils) GetPolicyScriptBaseAddress(
	testNetMagic uint, policyScript IPolicyScript, stakePolicyScript IPolicyScript,
) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-multisig-addr")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptFilePath, err := writeSerializableToFile(policyScript, baseDirectory, "ps.json")
	if err != nil {
		return "", err
	}

	stakePolicyScriptFilePath, err := writeSerializableToFile(stakePolicyScript, baseDirectory, "stake-ps.json")
	if err != nil {
		return "", err
	}

	args := []string{
		cu.era, "address", "build",
		"--payment-script-file", policyScriptFilePath,
		"--stake-script-file", stakePolicyScriptFilePath,
	}

	response, err := runCommand(cu.cardanoCliBinary, append(args, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetPolicyScriptEnterpriseAddress returns enterprise address for policy scripts
func (cu CliUtils) GetPolicyScriptEnterpriseAddress(
	testNetMagic uint, policyScript IPolicyScript,
) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-multisig-addr")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptFilePath, err := writeSerializableToFile(policyScript, baseDirectory, "ps.json")
	if err != nil {
		return "", err
	}

	args := []string{
		cu.era, "address", "build",
		"--payment-script-file", policyScriptFilePath,
	}

	response, err := runCommand(cu.cardanoCliBinary, append(args, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetPolicyScriptRewardAddress returns reward address for policy script
func (cu CliUtils) GetPolicyScriptRewardAddress(
	testNetMagic uint, policyScript IPolicyScript,
) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-reward-multisig-addr")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptFilePath, err := writeSerializableToFile(policyScript, baseDirectory, "ps.json")
	if err != nil {
		return "", err
	}

	args := []string{
		cu.era, "stake-address", "build",
		"--stake-script-file", policyScriptFilePath,
	}

	response, err := runCommand(cu.cardanoCliBinary, append(args, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetPolicyID returns policy id
func (cu CliUtils) GetPolicyID(policyScript IPolicyScript) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "ps-policy-id")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	policyScriptFilePath, err := writeSerializableToFile(policyScript, baseDirectory, "policy-script.json")
	if err != nil {
		return "", err
	}

	response, err := runCommand(cu.cardanoCliBinary, []string{
		cu.era, "transaction", "policyid", "--script-file", policyScriptFilePath,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(response, "\n"), nil
}

// GetAddressInfo returns address info if string representation for address is valid or error
func (cu CliUtils) GetAddressInfo(address string) (AddressInfo, error) {
	var ai AddressInfo

	res, err := runCommand(cu.cardanoCliBinary, []string{
		cu.era, "address", "info", "--address", address,
	})
	if err != nil {
		return ai, errors.Join(ErrInvalidAddressData, err)
	}

	if err := json.Unmarshal([]byte(strings.Trim(res, "\n")), &ai); err != nil {
		return ai, errors.Join(ErrInvalidAddressData, err)
	}

	return ai, nil
}

// GetWalletAddress returns address and stake address for wallet (if wallet is stake wallet)
func (cu CliUtils) GetWalletAddress(
	verificationKey, stakeVerificationKey []byte, testNetMagic uint,
) (addr string, stakeAddr string, err error) {
	bech32String, err := getBech32Key(verificationKey, "addr_vk")
	if err != nil {
		return "", "", err
	}

	// enterprise address
	if len(stakeVerificationKey) == 0 {
		addr, err = runCommand(cu.cardanoCliBinary, append([]string{
			cu.era, "address", "build",
			"--payment-verification-key", bech32String,
		}, getTestNetMagicArgs(testNetMagic)...))

		return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
	}

	bech32StakeString, err := getBech32Key(stakeVerificationKey, "stake_vk")
	if err != nil {
		return "", "", err
	}

	addr, err = runCommand(cu.cardanoCliBinary, append([]string{
		cu.era, "address", "build",
		"--payment-verification-key", bech32String,
		"--stake-verification-key", bech32StakeString,
	}, getTestNetMagicArgs(testNetMagic)...))
	if err != nil {
		return "", "", err
	}

	stakeAddr, err = runCommand(cu.cardanoCliBinary, append([]string{
		cu.era, "stake-address", "build",
		"--stake-verification-key", bech32StakeString,
	}, getTestNetMagicArgs(testNetMagic)...))

	return strings.Trim(addr, "\n"), strings.Trim(stakeAddr, "\n"), err
}

func (cu CliUtils) GetKeyHash(key []byte) (string, error) {
	bech32String, err := getBech32Key(key, "addr_vk")
	if err != nil {
		return "", err
	}

	resultKeyHash, err := runCommand(cu.cardanoCliBinary, []string{
		cu.era, "address", "key-hash",
		"--payment-verification-key", bech32String,
	})
	if err != nil {
		return "", err
	}

	return strings.Trim(resultKeyHash, "\n"), nil
}

// GetTxHash gets hash from transaction cbor slice
func (cu CliUtils) GetTxHash(txRaw []byte) (string, error) {
	baseDirectory, err := os.MkdirTemp("", "tx-hash-retriever")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(baseDirectory)

	realEraName, err := cu.GetRealEraName()
	if err != nil {
		return "", err
	}

	return cu.getTxHash(txRaw, baseDirectory, realEraName)
}

func (cu CliUtils) GetRealEraName() (string, error) {
	if strings.ToLower(cu.era) != "latest" {
		return strings.ToUpper(cu.era[:1]) + cu.era[1:], nil
	}

	list, err := runCommand(cu.cardanoCliBinary, []string{"--help"})
	if err != nil {
		return "", err
	}

	// Find the match
	matches := regexp.MustCompile(`Latest era commands \(([^)]+)\)`).FindStringSubmatch(list)
	if len(matches) >= 2 {
		// matches[0] = full match ("Latest era commands (Babbage/Conway)")
		// matches[1] = first captured group ("Babbage/Conway")
		return matches[1], nil
	}

	return "", errors.New("unknown era")
}

func (cu CliUtils) getTxHash(txRaw []byte, baseDirectory, eraName string) (string, error) {
	txFilePath := filepath.Join(baseDirectory, "tx.tmp")

	txBytes, err := transactionUnwitnessedRaw(txRaw).ToJSON(eraName)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(txFilePath, txBytes, FilePermission); err != nil {
		return "", err
	}

	args := []string{
		cu.era, "transaction", "txid",
		"--tx-body-file", txFilePath}

	res, err := runCommand(cu.cardanoCliBinary, args)
	if err != nil {
		return "", err
	}

	type txHashStruct struct {
		TxHash string `json:"txhash"`
	}

	var obj txHashStruct

	if err := json.Unmarshal([]byte(res), &obj); err == nil {
		return obj.TxHash, nil
	}

	return strings.TrimSpace(res), nil
}

func (cu CliUtils) CreateRegistrationCertificate(
	stakeAddress string, keyRegDepositAmount uint64,
) (cert *CardanoArtifact, err error) {
	baseDirectory, err := os.MkdirTemp("", "registration-cert")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(baseDirectory)

	certFilePath := filepath.Join(baseDirectory, "registration.cert")

	args := []string{
		cu.era, "stake-address", "registration-certificate",
		"--stake-address", stakeAddress,
		"--key-reg-deposit-amt", fmt.Sprintf("%d", keyRegDepositAmount),
		"--out-file", certFilePath}

	_, err = runCommand(cu.cardanoCliBinary, args)
	if err != nil {
		return nil, fmt.Errorf("failed to register certificate: %w", err)
	}

	bytes, err := os.ReadFile(certFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	if err := json.Unmarshal(bytes, &cert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate: %w", err)
	}

	return cert, nil
}

func (cu CliUtils) CreateDelegationCertificate(
	stakeAddress string, poolID string,
) (cert *CardanoArtifact, err error) {
	baseDirectory, err := os.MkdirTemp("", "delegation-cert")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(baseDirectory)

	certFilePath := filepath.Join(baseDirectory, "delegation.cert")

	args := []string{
		cu.era, "stake-address", "delegation-certificate",
		"--stake-address", stakeAddress,
		"--stake-pool-id", poolID,
		"--out-file", certFilePath}

	// On update to newer version this will fail because of the change:
	// delegation-certificate -> stake-delegation-certificate
	_, err = runCommand(cu.cardanoCliBinary, args)
	if err != nil {
		args[2] = "stake-delegation-certificate"
		_, err = runCommand(cu.cardanoCliBinary, args)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to delegate certificate: %w", err)
	}

	bytes, err := os.ReadFile(certFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	if err := json.Unmarshal(bytes, &cert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate: %w", err)
	}

	return cert, nil
}

func (cu CliUtils) CreateDeregistrationCertificate(
	stakeAddress string, depositAmount uint64,
) (cert *CardanoArtifact, err error) {
	baseDirectory, err := os.MkdirTemp("", "deregistration-cert")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(baseDirectory)

	certFilePath := filepath.Join(baseDirectory, "deregistration.cert")

	args := []string{
		cu.era, "stake-address", "deregistration-certificate",
		"--stake-address", stakeAddress,
		"--out-file", certFilePath,
	}

	// try without --key-reg-deposit-amt for cli before Conway era
	_, err = runCommand(cu.cardanoCliBinary, args)
	if err != nil && strings.Contains(err.Error(), "Missing: --key-reg-deposit-amt") {
		_, err = runCommand(cu.cardanoCliBinary, append(args,
			"--key-reg-deposit-amt", strconv.FormatUint(depositAmount, 10)))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create deregister certificate: %w", err)
	}

	bytes, err := os.ReadFile(certFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read deregister certificate: %w", err)
	}

	if err := json.Unmarshal(bytes, &cert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deregister certificate: %w", err)
	}

	return cert, nil
}

func getBech32Key(key []byte, prefix string) (string, error) {
	converted, err := bech32.ConvertBits(key, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("error converting bits: %w", err)
	}

	bech32String, err := bech32.Encode(prefix, converted)
	if err != nil {
		return "", fmt.Errorf("error encoding to Bech32: %w", err)
	}

	return bech32String, nil
}

// writeSerializableToFile writes a serializable object to a file
//
// fileName should always include the file extension
func writeSerializableToFile(ps ISerializable, baseDirectory, fileName string) (string, error) {
	bytes, err := ps.GetBytesJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal policy script: %w", err)
	}

	fullFilePath := filepath.Join(baseDirectory, fileName)
	if err := os.WriteFile(fullFilePath, bytes, FilePermission); err != nil {
		return "", fmt.Errorf("failed to save policy script: %w", err)
	}

	return fullFilePath, nil
}
