package wallet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidCardanoAddress(t *testing.T) {
	t.Parallel()

	addresses := []string{
		"addr_test1vp4l5ka8jaqe32kygjemg6g745lxrn0mem7fxvuarrazmesdyntms",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sk9t664",
		"stake1uy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwccdjgq9n",
		"addr1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmsf689u8",
		"addr_test1wpkr0wd9ggr3zmfs7a2pg845jld95nvjjzg4mnr0ewqmzmef689u8",
		"addr1qy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwcet0gxxtr902v0rt2whvwfzawz66kgjpu35zu82k8znca3sket664",
		"stake1uy4h5rr93jh4x83448tk8y3whpddtyfq7g6pwr4tr3fuwccdjgq9n",
	}
	results := []bool{
		true,
		true,
		true,
		true,
		false,
		false,
		false,
		true,
	}

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	for i, addr := range addresses {
		ai, err := cliUtils.GetAddressInfo(addr)
		if results[i] {
			if strings.Contains(ai.Address, "stake") {
				assert.Equal(t, ai.Type, "stake")
			} else {
				assert.Equal(t, ai.Type, "payment")
			}

			assert.NoError(t, err)
		} else {
			require.ErrorContains(t, err, ErrInvalidAddressData.Error())
		}
	}
}

func TestRegistrationCertificate(t *testing.T) {
	keyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}

	policyScript := NewPolicyScript(keyHashes, 3)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
	policyID, err := cliUtils.GetPolicyID(policyScript)
	require.NoError(t, err)

	stakeAddress, err := NewPolicyScriptRewardAddress(MainNetNetwork, policyID)
	require.NoError(t, err)

	stakeRegistrationCert, err := cliUtils.CreateRegistrationCertificate(stakeAddress.String(), 0)
	require.NoError(t, err)

	require.Equal(t, "CertificateShelley", stakeRegistrationCert.Type)
	require.Equal(t, "Stake Address Registration Certificate", stakeRegistrationCert.Description)
	require.Equal(t, "82008201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f", stakeRegistrationCert.CborHex)
}

func TestDelegationCertificate(t *testing.T) {
	keyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}
	poolID := "pool1ttxrlraudm8msm88x4pjz75xqwrug2qmkw2tfgfr7ddjgqfa43q"

	policyScript := NewPolicyScript(keyHashes, 3)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
	policyID, err := cliUtils.GetPolicyID(policyScript)
	require.NoError(t, err)

	stakeAddress, err := NewPolicyScriptRewardAddress(MainNetNetwork, policyID)
	require.NoError(t, err)

	stakeRegistrationCert, err := cliUtils.CreateDelegationCertificate(stakeAddress.String(), poolID)
	require.NoError(t, err)

	require.Equal(t, "CertificateShelley", stakeRegistrationCert.Type)
	require.Equal(t, "Stake Delegation Certificate", stakeRegistrationCert.Description)
	require.Equal(t, "83028201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f581c5acc3f8fbc6ecfb86ce73543217a860387c4281bb394b4a123f35b24", stakeRegistrationCert.CborHex)
}

func TestDeregistrationCertificate(t *testing.T) {
	keyHashes := []string{
		"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
		"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
		"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
		"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
	}

	policyScript := NewPolicyScript(keyHashes, 3)
	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
	policyID, err := cliUtils.GetPolicyID(policyScript)
	require.NoError(t, err)

	stakeAddress, err := NewPolicyScriptRewardAddress(MainNetNetwork, policyID)
	require.NoError(t, err)

	stakeRegistrationCert, err := cliUtils.CreateDeregistrationCertificate(stakeAddress.String())
	require.NoError(t, err)

	require.Equal(t, "CertificateShelley", stakeRegistrationCert.Type)
	require.Equal(t, "Stake Address Deregistration Certificate", stakeRegistrationCert.Description)
	require.Equal(t, "82018201581cb59d7c9f689fcbc2a19da2689f9fe52c5f65c3b3c56b7b7e2f08f15f", stakeRegistrationCert.CborHex)
}
