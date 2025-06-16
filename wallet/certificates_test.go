package wallet

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistrationCertificate(t *testing.T) {
	certificateBuilder := NewCardanoStakeCertBuilder()

	stakingVerificationKeyCbor := "5820f0e4bb5f8f62a880d7c4d9bd07a9d95263903c60e40f826f527374e886dab58b"
	stakingPubKeyBytes, err := DecodePublicKeyFromCBOR(stakingVerificationKeyCbor)
	require.NoError(t, err)

	stakeRegistrationCert, err := certificateBuilder.CreateStakeRegistrationCert(stakingPubKeyBytes, KeyCredential)
	require.NoError(t, err)

	stakeRegistrationCertBytes, err := certificateBuilder.EncodeRegistrationCertificateToCBOR(stakeRegistrationCert)
	require.NoError(t, err)

	stakeRegistrationCertHex := hex.EncodeToString(stakeRegistrationCertBytes)
	require.Equal(t, stakeRegistrationCertHex, "82008200581ccefaa39286f49aa0566d46cc0ec619a5b3e2d9d6ad3fa58e1e64c49c")

	stakeRegistrationCertFile, err := certificateBuilder.CreateCertFile(stakeRegistrationCertBytes, "Conway")
	require.NoError(t, err)
	fmt.Println(stakeRegistrationCertFile)

	require.Equal(t, stakeRegistrationCertFile.CborHex, stakeRegistrationCertHex)
}

func TestDelegationCertificate(t *testing.T) {
	certificateBuilder := NewCardanoStakeCertBuilder()

	stakingVerificationKeyCbor := "5820f0e4bb5f8f62a880d7c4d9bd07a9d95263903c60e40f826f527374e886dab58b"
	stakingPubKeyBytes, err := DecodePublicKeyFromCBOR(stakingVerificationKeyCbor)
	require.NoError(t, err)

	poolId := "pool1knap9hldvhww0fjqew26sxkfjpj3c8tp8uuj7j3729lzqn9x70r"

	stakeDelegationCert, err := certificateBuilder.CreateStakeDelegationCert(stakingPubKeyBytes, poolId, KeyCredential)
	require.NoError(t, err)

	stakeDelegationCertBytes, err := certificateBuilder.EncodeDelegationCertificateToCBOR(stakeDelegationCert)
	require.NoError(t, err)

	stakeDelegationCertHex := hex.EncodeToString(stakeDelegationCertBytes)
	require.Equal(t, stakeDelegationCertHex, "83028200581ccefaa39286f49aa0566d46cc0ec619a5b3e2d9d6ad3fa58e1e64c49c581cb4fa12dfed65dce7a640cb95a81ac990651c1d613f392f4a3e517e20")

	stakeDelegationCertFile, err := certificateBuilder.CreateCertFile(stakeDelegationCertBytes, "Conway")
	require.NoError(t, err)
	fmt.Println(stakeDelegationCertFile)

	require.Equal(t, stakeDelegationCertFile.CborHex, stakeDelegationCertHex)
}

// Helper function to decode a public key from a CBOR hex string
func DecodePublicKeyFromCBOR(cborHex string) ([]byte, error) {
	// Remove the CBOR type prefix (first 2 bytes: 0x5820)
	// 0x58 = byte string with 1-byte length
	// 0x20 = 32 bytes length
	cborBytes, err := hex.DecodeString(cborHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	if len(cborBytes) < 2 {
		return nil, fmt.Errorf("CBOR data too short")
	}

	// Check for expected CBOR byte string prefix
	if cborBytes[0] != 0x58 || cborBytes[1] != 0x20 {
		return nil, fmt.Errorf("unexpected CBOR format: expected 5820 prefix, got %02x%02x", cborBytes[0], cborBytes[1])
	}

	// Extract the 32-byte public key (skip the 2-byte prefix)
	if len(cborBytes) != 34 {
		return nil, fmt.Errorf("unexpected CBOR length: expected 34 bytes, got %d", len(cborBytes))
	}

	return cborBytes[2:], nil
}
