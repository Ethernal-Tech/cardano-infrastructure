package wallet

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyRegistrationCertificate(t *testing.T) {
	certificateBuilder := NewCardanoStakeCertBuilder()

	stakingVerificationKeyCbor := "5820f0e4bb5f8f62a880d7c4d9bd07a9d95263903c60e40f826f527374e886dab58b"
	stakingPubKeyBytes, err := DecodePublicKeyFromCBOR(stakingVerificationKeyCbor)
	require.NoError(t, err)

	stakeRegistrationCertBytes, err := certificateBuilder.CreateKeyStakeRegistrationCert(stakingPubKeyBytes)
	require.NoError(t, err)

	stakeRegistrationCertHex := hex.EncodeToString(stakeRegistrationCertBytes)
	require.Equal(t, stakeRegistrationCertHex, "82008200581ccefaa39286f49aa0566d46cc0ec619a5b3e2d9d6ad3fa58e1e64c49c")

	stakeRegistrationCertFile, err := certificateBuilder.CreateCertFile(stakeRegistrationCertBytes, "conway")
	require.NoError(t, err)
	fmt.Println(stakeRegistrationCertFile)

	require.Equal(t, stakeRegistrationCertFile.CborHex, stakeRegistrationCertHex)
	require.Equal(t, stakeRegistrationCertFile.Type, "CertificateConway")
}

func TestScriptRegistrationCertificate(t *testing.T) {
	keyHashes := []string{
		"6762a4577d1ee3fabd5cc45f6c42a8b8e47220de1f53687eedbc867b",
		"6a6f7866b8949847b8975ffa0fbea33b9ffb8aa61bf8c849af196cb2",
		"a0022c8bbfccd83786c4ee33a8a2e8b74a4c53f1875645c0a5224367",
		"a217149fec8641a8df68eca07b51da07c40819a1bd537d67fd73a5d4",
	}

	policyScript := NewPolicyScript(keyHashes, 3)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
	policyId, err := cliUtils.GetPolicyID(policyScript)
	require.NoError(t, err)

	certificateBuilder := NewCardanoStakeCertBuilder()

	stakeScriptHashBytes, err := hex.DecodeString(policyId)
	require.NoError(t, err)

	stakeRegistrationCertBytes, err := certificateBuilder.CreateScriptStakeRegistrationCert(stakeScriptHashBytes)
	require.NoError(t, err)

	stakeRegistrationCertHex := hex.EncodeToString(stakeRegistrationCertBytes)
	require.Equal(t, stakeRegistrationCertHex, "82008201581c781413c4fa573cce540adc0ba499fca9d48fe5fbdf6c03d9d7658b3b")

	stakeRegistrationCertFile, err := certificateBuilder.CreateCertFile(stakeRegistrationCertBytes, "shelley")
	require.NoError(t, err)

	require.Equal(t, stakeRegistrationCertFile.CborHex, stakeRegistrationCertHex)
	require.Equal(t, stakeRegistrationCertFile.Type, "CertificateShelley")
}

func TestKeyStakeDelegationCertificate(t *testing.T) {
	certificateBuilder := NewCardanoStakeCertBuilder()

	stakingVerificationKeyCbor := "5820f0e4bb5f8f62a880d7c4d9bd07a9d95263903c60e40f826f527374e886dab58b"
	stakingPubKeyBytes, err := DecodePublicKeyFromCBOR(stakingVerificationKeyCbor)
	require.NoError(t, err)

	poolId := "pool1knap9hldvhww0fjqew26sxkfjpj3c8tp8uuj7j3729lzqn9x70r"

	stakeDelegationCertBytes, err := certificateBuilder.CreateKeyStakeDelegationCert(stakingPubKeyBytes, poolId)
	require.NoError(t, err)

	stakeDelegationCertHex := hex.EncodeToString(stakeDelegationCertBytes)
	require.Equal(t, stakeDelegationCertHex, "83028200581ccefaa39286f49aa0566d46cc0ec619a5b3e2d9d6ad3fa58e1e64c49c581cb4fa12dfed65dce7a640cb95a81ac990651c1d613f392f4a3e517e20")

	stakeDelegationCertFile, err := certificateBuilder.CreateCertFile(stakeDelegationCertBytes, "conway")
	require.NoError(t, err)
	fmt.Println(stakeDelegationCertFile)

	require.Equal(t, stakeDelegationCertFile.CborHex, stakeDelegationCertHex)
	require.Equal(t, stakeDelegationCertFile.Type, "CertificateConway")
}

func TestScriptStakeDelegationCertificate(t *testing.T) {
	keyHashes := []string{
		"6762a4577d1ee3fabd5cc45f6c42a8b8e47220de1f53687eedbc867b",
		"6a6f7866b8949847b8975ffa0fbea33b9ffb8aa61bf8c849af196cb2",
		"a0022c8bbfccd83786c4ee33a8a2e8b74a4c53f1875645c0a5224367",
		"a217149fec8641a8df68eca07b51da07c40819a1bd537d67fd73a5d4",
	}

	policyScript := NewPolicyScript(keyHashes, 3)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
	policyId, err := cliUtils.GetPolicyID(policyScript)
	require.NoError(t, err)

	certificateBuilder := NewCardanoStakeCertBuilder()

	stakeScriptHashBytes, err := hex.DecodeString(policyId)
	require.NoError(t, err)

	poolId := "pool1hvsmu7l9c23ltrncj6lkgmr6ncth7s8tx67zyj2fxl8054xyjz6"

	stakeDelegationCertBytes, err := certificateBuilder.CreateScriptStakeDelegationCert(stakeScriptHashBytes, poolId)
	require.NoError(t, err)

	stakeDelegationCertHex := hex.EncodeToString(stakeDelegationCertBytes)
	require.Equal(t, stakeDelegationCertHex, "83028201581c781413c4fa573cce540adc0ba499fca9d48fe5fbdf6c03d9d7658b3b581cbb21be7be5c2a3f58e7896bf646c7a9e177f40eb36bc22494937cefa")

	stakeDelegationCertFile, err := certificateBuilder.CreateCertFile(stakeDelegationCertBytes, "shelley")
	require.NoError(t, err)

	require.Equal(t, stakeDelegationCertFile.CborHex, stakeDelegationCertHex)
	require.Equal(t, stakeDelegationCertFile.Type, "CertificateShelley")
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
