package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fxamacker/cbor/v2"
)

const (
	KeyHashSize = 28
	KeySize     = 32

	StakeSigningKeyShelley          = "StakeSigningKeyShelley_ed25519"
	StakeSigningKeyShelleyDesc      = "Stake Signing Key"
	StakeVerificationKeyShelley     = "StakeVerificationKeyShelley_ed25519"
	StakeVerificationKeyShelleyDesc = "Stake Verification Key"

	PaymentSigningKeyShelley          = "PaymentSigningKeyShelley_ed25519"
	PaymentSigningKeyShelleyDesc      = "Payment Signing Key"
	PaymentVerificationKeyShelley     = "PaymentVerificationKeyShelley_ed25519"
	PaymentVerificationKeyShelleyDesc = "Payment Verification Key"
)

type Wallet struct {
	VerificationKey      []byte `json:"vkey"`
	SigningKey           []byte `json:"skey"`
	StakeVerificationKey []byte `json:"vstake"`
	StakeSigningKey      []byte `json:"sstake"`
}

func NewWallet(verificationKey []byte, signingKey []byte) *Wallet {
	return &Wallet{
		VerificationKey: PadKeyToSize(verificationKey),
		SigningKey:      PadKeyToSize(signingKey),
	}
}

func NewStakeWallet(verificationKey []byte, signingKey []byte,
	stakeVerificationKey []byte, stakeSigningKey []byte) *Wallet {
	return &Wallet{
		StakeVerificationKey: PadKeyToSize(stakeVerificationKey),
		StakeSigningKey:      PadKeyToSize(stakeSigningKey),
		VerificationKey:      PadKeyToSize(verificationKey),
		SigningKey:           PadKeyToSize(signingKey),
	}
}

func (w Wallet) GetVerificationKey() []byte {
	return w.VerificationKey
}

func (w Wallet) GetSigningKey() []byte {
	return w.SigningKey
}

func (w Wallet) GetStakeVerificationKey() []byte {
	return w.StakeVerificationKey
}

func (w Wallet) GetStakeSigningKey() []byte {
	return w.StakeSigningKey
}

type Key struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Hex         string `json:"cborHex"`
}

func NewKey(filePath string) (Key, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return Key{}, err
	}

	var key Key

	if err := json.Unmarshal(bytes, &key); err != nil {
		return Key{}, err
	}

	return key, nil
}

func NewKeyFromBytes(keyType string, desc string, bytes []byte) (Key, error) {
	cborBytes, err := cbor.Marshal(PadKeyToSize(bytes))
	if err != nil {
		return Key{}, err
	}

	return Key{
		Type:        keyType,
		Description: desc,
		Hex:         hex.EncodeToString(cborBytes),
	}, nil
}

func (k Key) GetKeyBytes() ([]byte, error) {
	return GetKeyBytes(k.Hex)
}

func (k Key) WriteToFile(filePath string) error {
	bytes, err := json.Marshal(k)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, bytes, FilePermission); err != nil {
		return err
	}

	return nil
}

func PadKeyToSize(key []byte) []byte {
	// If the key is already 32 bytes long, return it as is
	if len(key) == KeySize {
		return key
	}

	// If the key is shorter than 32 bytes, pad with leading zeroes
	if len(key) < KeySize {
		return append(make([]byte, KeySize-len(key)), key...)
	}

	// If the key is longer than 32 bytes, truncate it to 32 bytes
	return key[:KeySize]
}

// NewWalletFromMnemonic creates wallet from menomonics
func NewWalletFromMnemonic(
	cardanoCliBinary, cardanoAddressBinary, mnemonic string, num int,
) (IWallet, error) {
	baseDirectory, err := os.MkdirTemp("", "mnemonics")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(baseDirectory)

	runCommandAddr := func(inputFile string, outputFile string, args ...string) error {
		res, err := runCommand(cardanoAddressBinary, args, inputFile)
		if err != nil {
			return err
		}

		return os.WriteFile(outputFile, []byte(res), 0750)
	}

	getSigningKey := func(inputFile, outputFile string, isStake bool) ([]byte, error) {
		args := []string{"key", "convert-cardano-address-key"}
		if isStake {
			args = append(args, "--shelley-stake-key")
		} else {
			args = append(args, "--shelley-payment-key")
		}

		args = append(args, "--signing-key-file", inputFile, "--out-file", outputFile)

		if _, err := runCommand(cardanoCliBinary, args); err != nil {
			return nil, err
		}

		key, err := NewKey(outputFile)
		if err != nil {
			return nil, err
		}

		return key.GetKeyBytes()
	}

	mnemonicFilePath := filepath.Join(baseDirectory, "mnemonic")
	rootXskFilePath := filepath.Join(baseDirectory, "cardano_root.xsk")
	paymentXskFilePath := filepath.Join(baseDirectory, "payment.xsk")
	stakeXskFilePath := filepath.Join(baseDirectory, "stake.xsk")
	paymentKeyFilePath := filepath.Join(baseDirectory, "payment.skey")
	stakeKeyFilePath := filepath.Join(baseDirectory, "stake.skey")

	if err := os.WriteFile(mnemonicFilePath, []byte(mnemonic), 0750); err != nil {
		return nil, err
	}

	if err := runCommandAddr(
		mnemonicFilePath, rootXskFilePath,
		"key", "from-recovery-phrase", "Shelley"); err != nil {
		return nil, err
	}

	if err := runCommandAddr(
		rootXskFilePath, paymentXskFilePath, "key", "child",
		fmt.Sprintf("1852H/1815H/0H/0/%d", num)); err != nil {
		return nil, err
	}

	if err := runCommandAddr(
		rootXskFilePath, stakeXskFilePath, "key", "child",
		fmt.Sprintf("1852H/1815H/0H/2/%d", num)); err != nil {
		return nil, err
	}

	paymentSigningKey, err := getSigningKey(paymentXskFilePath, paymentKeyFilePath, false)
	if err != nil {
		return nil, err
	}

	stakeSigningKey, err := getSigningKey(stakeXskFilePath, stakeKeyFilePath, true)
	if err != nil {
		return nil, err
	}

	return NewStakeWallet(
		GetVerificationKeyFromSigningKey(paymentSigningKey), paymentSigningKey,
		GetVerificationKeyFromSigningKey(stakeSigningKey), stakeSigningKey), nil
}
