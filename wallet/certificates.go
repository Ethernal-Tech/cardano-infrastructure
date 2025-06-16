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

// CreateStakeRegistrationCert creates a stake registration certificate
func (b *CardanoStakeCertBuilder) CreateStakeRegistrationCert(stakingPubKeyBytes []byte, credentialType CredentialType) (StakeRegistrationCert, error) {
	stakeKeyHash, err := GetKeyHashBytes(stakingPubKeyBytes)
	if err != nil {
		return StakeRegistrationCert{}, nil
	}

	stakeCredential := StakeCredential{
		Type: credentialType,
		Hash: stakeKeyHash,
	}

	return StakeRegistrationCert{
		Type:       StakeRegistration,
		Credential: stakeCredential,
	}, nil
}

// CreateStakeDelegationCert creates a stake delegation certificate
func (b *CardanoStakeCertBuilder) CreateStakeDelegationCert(stakingPubKeyBytes []byte, poolId string, credentialType CredentialType) (StakeDelegationCert, error) {
	stakeKeyHash, err := GetKeyHashBytes(stakingPubKeyBytes)
	if err != nil {
		return StakeDelegationCert{}, err
	}

	stakeCredential := StakeCredential{
		Type: credentialType,
		Hash: stakeKeyHash,
	}

	poolKeyHashBytes, err := b.decodePoolKeyHashFromBech32(poolId)
	if err != nil {
		return StakeDelegationCert{}, err
	}

	return StakeDelegationCert{
		Type:        StakeDelegation,
		Credential:  stakeCredential,
		PoolKeyHash: poolKeyHashBytes,
	}, nil
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

		credType, ok := credential[0].(uint64)
		if !ok {
			return nil, fmt.Errorf("invalid credential type format")
		}

		credHash, ok := credential[1].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid credential hash format")
		}

		stakeRegistrationCert := StakeRegistrationCert{
			Type: StakeRegistration,
			Credential: StakeCredential{
				Type: CredentialType(credType),
				Hash: credHash,
			},
		}
		return b.createStakeRegistrationCertFile(stakeRegistrationCert, era)
	} else if decoded[0].(uint64) == StakeDelegation {
		credential, ok := decoded[1].([]interface{})
		if !ok || len(credential) != 2 {
			return nil, fmt.Errorf("invalid credential format")
		}

		credType, ok := credential[0].(uint64)
		if !ok {
			return nil, fmt.Errorf("invalid credential type format")
		}

		credHash, ok := credential[1].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid credential hash format")
		}

		if len(decoded) < 3 {
			return nil, fmt.Errorf("invalid delegation certificate: missing pool key hash")
		}

		poolKeyHash, ok := decoded[2].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid pool key hash format")
		}

		stakeDelegationCert := StakeDelegationCert{
			Type: StakeDelegation,
			Credential: StakeCredential{
				Type: CredentialType(credType),
				Hash: credHash,
			},
			PoolKeyHash: poolKeyHash,
		}
		return b.createStakeDelegationCertFile(stakeDelegationCert, era)
	}

	return nil, fmt.Errorf("invalid certificate type: %d", decoded[0].(uint64))
}

// CreateStakeRegistrationCertFile creates a stake registration certificate in a file
func (b *CardanoStakeCertBuilder) createStakeRegistrationCertFile(stakeRegistrationCert StakeRegistrationCert, era string) (*CertificateFile, error) {
	stakeRegistrationCertBytes, err := b.EncodeRegistrationCertificateToCBOR(stakeRegistrationCert)
	if err != nil {
		return nil, fmt.Errorf("failed to encode stake registration certificate to CBOR: %w", err)
	}

	certType := "CertificateShelley"
	if era == "Conway" || era == "latest" {
		certType = "CertificateConway"
	}

	return &CertificateFile{
		Type:        certType,
		Description: "Stake Address Registration Certificate",
		CborHex:     hex.EncodeToString(stakeRegistrationCertBytes),
	}, nil
}

// CreateStakeDelegationCertFile creates a stake delegation certificate in a file
func (b *CardanoStakeCertBuilder) createStakeDelegationCertFile(stakeDelegationCert StakeDelegationCert, era string) (*CertificateFile, error) {
	stakeDelegationCertBytes, err := b.EncodeDelegationCertificateToCBOR(stakeDelegationCert)
	if err != nil {
		return nil, fmt.Errorf("failed to encode stake delegation certificate to CBOR: %w", err)
	}

	certType := "CertificateShelley"
	if era == "Conway" || era == "latest" {
		certType = "CertificateConway"
	}

	return &CertificateFile{
		Type:        certType,
		Description: "Stake Address Delegation Certificate",
		CborHex:     hex.EncodeToString(stakeDelegationCertBytes),
	}, nil
}

// EncodeRegistrationCertificateToCBOR encodes a stake registration certificate to CBOR
func (b *CardanoStakeCertBuilder) EncodeRegistrationCertificateToCBOR(cert StakeRegistrationCert) ([]byte, error) {
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
func (b *CardanoStakeCertBuilder) EncodeDelegationCertificateToCBOR(cert StakeDelegationCert) ([]byte, error) {
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
