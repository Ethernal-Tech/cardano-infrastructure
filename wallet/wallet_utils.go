package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/Ethernal-Tech/cardano-infrastructure/wallet/bech32"
	"github.com/fxamacker/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

var ErrInvalidSignature = errors.New("invalid signature")

// GenerateKeyPair generates ed25519 (signing key, verifying) key pair
func GenerateKeyPair() ([]byte, []byte, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := io.ReadFull(rand.Reader, seed); err != nil {
		return nil, nil, err
	}

	return seed, GetVerificationKeyFromSigningKey(seed), nil
}

// GetVerificationKeyFromSigningKey retrieves verification/public key from signing/private key
func GetVerificationKeyFromSigningKey(signingKey []byte) []byte {
	return ed25519.NewKeyFromSeed(signingKey).Public().(ed25519.PublicKey) //nolint:forcetypeassert
}

// VerifyWitness verifies if txHash is signed by witness
func VerifyWitness(txHash string, witness []byte) error {
	txHashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return err
	}

	signature, vKey, err := TxWitnessRaw(witness).GetSignatureAndVKey()
	if err != nil {
		return err
	}

	return VerifyMessage(txHashBytes, vKey, signature)
}

// SignMessage signs message
func SignMessage(signingKey, verificationKey, message []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error: %v", r)
		}
	}()

	privateKey := make([]byte, len(signingKey)+len(verificationKey))

	copy(privateKey, signingKey)
	copy(privateKey[len(signingKey):], verificationKey)

	result = ed25519.Sign(privateKey, message)

	return
}

// VerifyMessage verifies message with verificationKey and signature
func VerifyMessage(message, verificationKey, signature []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error: %v", r)
		}
	}()

	if !ed25519.Verify(verificationKey, message, signature) {
		err = ErrInvalidSignature
	}

	return
}

// GetKeyHashBytes gets Cardano key hash from arbitrary bytes
func GetKeyHashBytes(bytes []byte) ([]byte, error) {
	hasher, err := blake2b.New(KeyHashSize, nil)
	if err != nil {
		return nil, err
	}

	if _, err := hasher.Write(bytes); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}

// GetKeyHash gets Cardano key hash string from arbitrary key
func GetKeyHash(bytes []byte) (string, error) {
	bytes, err := GetKeyHashBytes(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// PadKeyToSize pads key to 32 bytes
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

// GetKeyBytes extracts the original key bytes from a given string. Supported formats:
// - Hex + CBOR encoded string: Attempts to decode the key assuming it is hex-encoded,
// - Bech32 encoded keys: Handles formats like addr_vk, addr_sk, stake_vk, stake_sk
func GetKeyBytes(key string) (result []byte, err error) {
	if bytes, err := hex.DecodeString(key); err == nil {
		if err := cbor.Unmarshal(bytes, &result); err != nil {
			return nil, err
		}
	} else if _, result, err = bech32.DecodeToBase256(key); err != nil {
		return nil, err
	}

	return PadKeyToSize(result), nil
}
