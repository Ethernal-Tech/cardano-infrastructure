package wallet

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Ethernal-Tech/cardano-infrastructure/wallet/bech32"
	"github.com/fxamacker/cbor/v2"
)

// Certificate types
const (
	StakeRegistration   = 0
	StakeDeregistration = 1
	StakeDelegation     = 2
)

// Credential types
type CredentialType uint8

const (
	KeyCredential    CredentialType = 0
	ScriptCredential CredentialType = 1
)

// StakeCredential represents a stake credential (key hash or script hash)
type StakeCredential struct {
	_    struct{}       `cbor:",toarray"`
	Type CredentialType // 0 for key, 1 for script
	Hash []byte         // 28 bytes
}

// StakeRegistrationCert represents a stake registration certificate
type StakeRegistrationCert struct {
	Type       uint8           // Certificate type (0 for stake registration)
	Credential StakeCredential // Stake credential
}

// Certificate represents a generic certificate that can be either registration or delegation
type Certificate struct {
	_           struct{} `cbor:",toarray"`
	Type        uint8
	Credential  []interface{} `cbor:",toarray"` // This will be [cred_type, key_hash]
	PoolKeyHash []byte
}

// StakeDelegationCert represents a stake delegation certificate
type StakeDelegationCert struct {
	Type        uint8           // Certificate type (2 for stake delegation)
	Credential  StakeCredential // Stake credential
	PoolKeyHash []byte          // Pool key hash (28 bytes)
}

// StakeRegistrationCert represents a stake registration certificate
type CertificateFile struct {
	Type        string // Dependent on the cardano era
	Description string // Always the same: Stake Address Registration Certificate
	CborHex     string // CBOR encoded stake registration certificate
}

func (c *CertificateFile) ToJSON() []byte {
	return []byte(fmt.Sprintf(`{
		"type": "%s",
		"description": "%s",
		"cborHex": "%s"
	}`, c.Type, c.Description, c.CborHex))
}

// CardanoStakeCertBuilder handles creating stake registration certificates
type CardanoStakeCertBuilder struct{}

// NewCardanoStakeCertBuilder creates a new certificate builder
func NewCardanoStakeCertBuilder() *CardanoStakeCertBuilder {
	return &CardanoStakeCertBuilder{}
}

// CreateKeyStakeRegistrationCert creates a stake registration certificate for stake address
//
// Returns the CBOR encoded certificate
func (b *CardanoStakeCertBuilder) CreateKeyStakeRegistrationCert(stakePubKeyBytes []byte) ([]byte, error) {
	stakeKeyHash, err := GetKeyHashBytes(stakePubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get key hash: %w", err)
	}

	return b.createStakeRegistrationCert(stakeKeyHash, KeyCredential)
}

// CreateScriptStakeRegistrationCert creates a stake registration certificate for script address
//
// Returns the CBOR encoded certificate
func (b *CardanoStakeCertBuilder) CreateScriptStakeRegistrationCert(stakeScriptHashBytes []byte) ([]byte, error) {
	return b.createStakeRegistrationCert(stakeScriptHashBytes, ScriptCredential)
}

// createStakeRegistrationCert creates a stake registration certificate
func (b *CardanoStakeCertBuilder) createStakeRegistrationCert(keyHashBytes []byte, credentialType CredentialType) ([]byte, error) {
	stakeCredential := StakeCredential{
		Type: credentialType,
		Hash: keyHashBytes,
	}

	certificate := StakeRegistrationCert{
		Type:       StakeRegistration,
		Credential: stakeCredential,
	}

	// Encode to CBOR
	cborData, err := b.encodeRegistrationCertificateToCBOR(certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate to CBOR: %w", err)
	}

	return cborData, nil
}

// CreateKeyStakeDelegationCert creates a stake delegation certificate for stake address
//
// Returns the CBOR encoded certificate
func (b *CardanoStakeCertBuilder) CreateKeyStakeDelegationCert(stakingPubKeyBytes []byte, poolId string) ([]byte, error) {
	stakeKeyHashBytes, err := GetKeyHashBytes(stakingPubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get key hash: %w", err)
	}

	return b.createStakeDelegationCert(stakeKeyHashBytes, poolId, KeyCredential)
}

// CreateScriptStakeDelegationCert creates a stake delegation certificate for script address
//
// Returns the CBOR encoded certificate
func (b *CardanoStakeCertBuilder) CreateScriptStakeDelegationCert(stakeScriptHashBytes []byte, poolId string) ([]byte, error) {
	return b.createStakeDelegationCert(stakeScriptHashBytes, poolId, ScriptCredential)
}

// createStakeDelegationCert creates a stake delegation certificate
//
// Returns the CBOR encoded certificate
func (b *CardanoStakeCertBuilder) createStakeDelegationCert(keyHashBytes []byte, poolId string, credentialType CredentialType) ([]byte, error) {

	stakeCredential := StakeCredential{
		Type: credentialType,
		Hash: keyHashBytes,
	}

	poolKeyHashBytes, err := b.decodePoolKeyHashFromBech32(poolId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode pool key hash: %w", err)
	}

	certificate := StakeDelegationCert{
		Type:        StakeDelegation,
		Credential:  stakeCredential,
		PoolKeyHash: poolKeyHashBytes,
	}

	// Encode to CBOR
	cborData, err := b.encodeDelegationCertificateToCBOR(certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate to CBOR: %w", err)
	}

	return cborData, nil
}

func (b *CardanoStakeCertBuilder) CreateCertFile(cert []byte, era string) (*CertificateFile, error) {
	// Decode from cbor into a generic array
	var decoded []interface{}
	if err := cbor.Unmarshal(cert, &decoded); err != nil {
		return nil, fmt.Errorf("failed to decode certificate from CBOR: %w", err)
	}

	if decoded[0].(uint64) == StakeRegistration {
		credential, ok := decoded[1].([]interface{})
		if !ok || len(credential) != 2 {
			return nil, fmt.Errorf("invalid credential format")
		}

		return b.createCertFile(cert, "Registration", era)
	} else if decoded[0].(uint64) == StakeDelegation {
		credential, ok := decoded[1].([]interface{})
		if !ok || len(credential) != 2 {
			return nil, fmt.Errorf("invalid credential format")
		}

		return b.createCertFile(cert, "Delegation", era)
	}

	return nil, fmt.Errorf("invalid certificate type: %d", decoded[0].(uint64))
}

// CreateStakeRegistrationCertFile creates a stake registration certificate in a file
func (b *CardanoStakeCertBuilder) createCertFile(cert []byte, certificateType string, era string) (*CertificateFile, error) {
	certType := "CertificateShelley"
	if era == "conway" || era == "latest" {
		certType = "CertificateConway"
	}

	return &CertificateFile{
		Type:        certType,
		Description: fmt.Sprintf("Stake Address %s Certificate", certificateType),
		CborHex:     hex.EncodeToString(cert),
	}, nil
}

// EncodeRegistrationCertificateToCBOR encodes a stake registration certificate to CBOR
func (b *CardanoStakeCertBuilder) encodeRegistrationCertificateToCBOR(cert StakeRegistrationCert) ([]byte, error) {
	// CBOR structure for stake registration certificate:
	// [cert_type, [cred_type, key_hash]]

	// Create the credential array [cred_type, key_hash]
	credential := []interface{}{
		cert.Credential.Type,
		cert.Credential.Hash,
	}

	// Create the certificate array [cert_type, credential]
	certificate := []interface{}{
		cert.Type,
		credential,
	}

	// Encode to CBOR
	cborData, err := cbor.Marshal(certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate to CBOR: %w", err)
	}

	return cborData, nil
}

// EncodeDelegationCertificateToCBOR encodes a stake delegation certificate to CBOR
func (b *CardanoStakeCertBuilder) encodeDelegationCertificateToCBOR(cert StakeDelegationCert) ([]byte, error) {
	// CBOR structure for stake delegation certificate:
	// [cert_type, [cred_type, key_hash], pool_key_hash]

	// Create the credential array [cred_type, key_hash]
	credential := []interface{}{
		cert.Credential.Type,
		cert.Credential.Hash,
	}

	// Create the certificate array [cert_type, credential, pool_key_hash]
	certificate := []interface{}{
		cert.Type,
		credential,
		cert.PoolKeyHash,
	}

	// Encode to CBOR
	cborData, err := cbor.Marshal(certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to encode delegation certificate to CBOR: %w", err)
	}

	return cborData, nil
}

// decodePoolKeyHashFromBech32 extracts pool key hash from a bech32 pool ID
func (b *CardanoStakeCertBuilder) decodePoolKeyHashFromBech32(poolId string) ([]byte, error) {
	// Must be a bech32 pool ID
	if !strings.HasPrefix(poolId, "pool1") {
		return nil, fmt.Errorf("invalid pool ID: must start with 'pool1' or be 56-char hex")
	}

	hrp, data, err := bech32.Decode(poolId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bech32: %w", err)
	}

	if hrp != "pool" {
		return nil, fmt.Errorf("invalid pool ID prefix: expected 'pool', got '%s'", hrp)
	}

	// Convert 5-bit data to 8-bit bytes
	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bits: %w", err)
	}

	if len(converted) != 28 {
		return nil, fmt.Errorf("invalid pool key hash length: expected 28 bytes, got %d", len(converted))
	}

	return converted, nil
}
