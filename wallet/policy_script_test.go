package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyScript(t *testing.T) {
	t.Parallel()

	var (
		err       error
		wallets   = [6]*Wallet{}
		keyHashes = [6]string{}
	)

	for i := range wallets {
		wallets[i], err = GenerateWallet(true)
		require.NoError(t, err)

		keyHashes[i], err = GetKeyHash(wallets[i].VerificationKey)
		require.NoError(t, err)
	}

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))

	ps := NewPolicyScript(keyHashes[:4], 4)
	psStake := NewPolicyScript(keyHashes[4:], 1)
	psDifferentOrder := NewPolicyScript(append([]string{}, keyHashes[1], keyHashes[0], keyHashes[3], keyHashes[2]), 4)

	policyID, err := cliUtils.GetPolicyID(ps)
	require.NoError(t, err)

	policyIDDifferentOrder, err := cliUtils.GetPolicyID(psDifferentOrder)
	require.NoError(t, err)

	require.Equal(t, policyID, policyIDDifferentOrder)

	policyIDStake, err := cliUtils.GetPolicyID(psStake)
	require.NoError(t, err)

	cliAddrStake, err := cliUtils.GetPolicyScriptBaseAddress(MainNetProtocolMagic, ps, psStake)
	require.NoError(t, err)

	addrStake, err := NewPolicyScriptBaseAddress(MainNetNetwork, policyID, policyIDStake)
	require.NoError(t, err)

	require.Equal(t, cliAddrStake, addrStake.String())

	cliAddr, err := cliUtils.GetPolicyScriptEnterpriseAddress(MainNetProtocolMagic, ps)
	require.NoError(t, err)

	addr, err := NewPolicyScriptEnterpriseAddress(MainNetNetwork, policyID)
	require.NoError(t, err)

	require.Equal(t, cliAddr, addr.String())
}

func TestPolicyScript_SpecificKeysAllPermutations(t *testing.T) {
	t.Parallel()

	var (
		allPublicKeys = []string{
			"5d767d06a9426bafd31eae25122b586fb6cac32efcee60c94bf8f43faddb8f5b",
			"f5c69c8a0bb63016068d4683dab19af0b0833158ed7a5ed91bd328d0939f3173",
			"2983addc84a6032feeb8870a9f74308e4a5446779bf44a8a790be3fb266e1abd",
			"001c4e9e6493675a3f380d1749d203a5aabb92b217b28b918a1b0aea6b8981b0",
		}
		permute func(int, []int, []bool, func([]string))
	)

	// permute generates all permutations of the input slice and appends them to the result.
	permute = func(n int, current []int, used []bool, execute func(keys []string)) {
		if len(current) == n {
			keys := make([]string, n)
			for i, x := range current {
				keys[i] = allPublicKeys[x]
			}

			execute(keys)

			return
		}

		for i := 0; i < n; i++ {
			if used[i] {
				continue
			}
			// Mark the element as used
			used[i] = true
			current = append(current, i)

			// Recursively generate permutations
			permute(n, current, used, execute)

			// Unmark the element and remove it from the current permutation
			used[i] = false
			current = current[:len(current)-1]
		}
	}

	permute(len(allPublicKeys), []int{}, make([]bool, len(allPublicKeys)), func(keys []string) {
		cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))
		hashes := make([]string, len(keys))

		for i, k := range keys {
			bytes, err := hex.DecodeString(k)
			require.NoError(t, err)

			h, err := GetKeyHash(bytes)
			require.NoError(t, err)

			hashes[i] = h
		}

		ps := NewPolicyScript(hashes, (len(hashes)*2+2)/3)

		policyID, err := cliUtils.GetPolicyID(ps)
		require.NoError(t, err)

		cliAddr, err := cliUtils.GetPolicyScriptEnterpriseAddress(TestNetProtocolMagic, ps)
		require.NoError(t, err)

		addr, err := NewPolicyScriptEnterpriseAddress(TestNetNetwork, policyID)
		require.NoError(t, err)

		require.Equal(t, cliAddr, addr.String())
	})
}

func TestPolicyScript_GetCount(t *testing.T) {
	hashes := []string{"123", "245", "459", "129"}
	adminHash := "888"
	ps := NewPolicyScript(hashes, len(hashes)*2/3+1)

	assert.Equal(t, 4, (&PolicyScript{
		Type: "any",
		Scripts: []PolicyScript{
			{
				Type:    "sig",
				KeyHash: adminHash,
			},
			*ps,
		},
	}).GetCount())

	assert.Equal(t, 5, (&PolicyScript{
		Type: "all",
		Scripts: []PolicyScript{
			{
				Type:    "sig",
				KeyHash: adminHash,
			},
			*ps,
		},
	}).GetCount())
}

func TestGetPolicyScriptRewardAddress(t *testing.T) {
	t.Parallel()

	var (
		err       error
		wallets   = [6]*Wallet{}
		keyHashes = [6]string{}
	)

	for i := range wallets {
		wallets[i], err = GenerateWallet(true)
		require.NoError(t, err)

		keyHashes[i], err = GetKeyHash(wallets[i].VerificationKey)
		require.NoError(t, err)
	}

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(MainNetNetwork))

	ps := NewPolicyScript(keyHashes[:4], 4)

	cliAddr, err := cliUtils.GetPolicyScriptRewardAddress(MainNetProtocolMagic, ps)
	require.NoError(t, err)

	pid, err := cliUtils.GetPolicyID(ps)
	require.NoError(t, err)

	addr, err := NewPolicyScriptRewardAddress(MainNetNetwork, pid)
	require.NoError(t, err)

	require.Equal(t, cliAddr, addr.String())
}

func TestDifferentSlots(t *testing.T) {
	t.Parallel()

	var (
		err              error
		paymentKeyHashes = [4]string{
			"0fb340e2fc18865fbf406dce76f743de13c46d2eb91d6e87e6eb63c6",
			"41b46f772b622e7e5bc8970d128faccb7a457c610a48d514801a0411",
			"5282885af1f234cb9407f05b120f2eb06872f297864ca9066a657011",
			"6a2f73455484b658c168c18ed54222d189e7e746ec3dc2d8d8891e42",
		}
		stakeKeyHashes = [4]string{
			"30356731c6f4d92598732163a68d9dcec7c386075d5da4f1dca5724d",
			"794eb34ded015c701fcf7b6ec4e0476e3dc2054a8831f636361680c9",
			"8d2f93fdc4dbe32b1cb6951a441f081d2d111cb4a4c79a69f27d00a9",
			"9f584550989f8a6cd6ce152b1c34661a764e0237200359e0f553d7db",
		}
	)

	cliUtils := NewCliUtils(ResolveCardanoCliBinary(TestNetNetwork))

	ps0 := NewPolicyScript(paymentKeyHashes[:], 3)
	psStake0 := NewPolicyScript(stakeKeyHashes[:], 3)

	ps1 := NewPolicyScript(paymentKeyHashes[:], 3, WithAfter(1))
	psStake1 := NewPolicyScript(stakeKeyHashes[:], 3, WithAfter(1))

	ps2 := NewPolicyScript(paymentKeyHashes[:], 3, WithAfter(2))
	psStake2 := NewPolicyScript(stakeKeyHashes[:], 3, WithAfter(2))

	ps0Bytes, err := ps0.GetBytesJSON()
	require.NoError(t, err)
	ps1Bytes, err := ps1.GetBytesJSON()
	require.NoError(t, err)
	ps2Bytes, err := ps2.GetBytesJSON()
	require.NoError(t, err)
	require.NotEqual(t, ps0Bytes, ps1Bytes)
	require.NotEqual(t, ps1Bytes, ps2Bytes)

	psStake0Bytes, err := psStake0.GetBytesJSON()
	require.NoError(t, err)
	psStake1Bytes, err := psStake1.GetBytesJSON()
	require.NoError(t, err)
	psStake2Bytes, err := psStake2.GetBytesJSON()
	require.NoError(t, err)
	require.NotEqual(t, psStake0Bytes, psStake1Bytes)
	require.NotEqual(t, psStake1Bytes, psStake2Bytes)

	addr0, err := cliUtils.GetPolicyScriptBaseAddress(TestNetProtocolMagic, ps0, psStake0)
	require.NoError(t, err)
	require.Equal(t, "addr_test1xqdt3kene0l87agrdcsn7jzspfrj83h5svgmaw8rnzzva644n47f76yle0p2r8dzdz0elefvtaju8v79ddahutcg790s37mp24", addr0)

	addr0Stake, err := cliUtils.GetPolicyScriptRewardAddress(TestNetProtocolMagic, psStake0)
	require.NoError(t, err)
	require.Equal(t, "stake_test17z6e6lyldz0uhs4pnk3x38ulu5k97ewrk0zkk7m79uy0zhcp9x067", addr0Stake)

	addr1, err := cliUtils.GetPolicyScriptBaseAddress(TestNetProtocolMagic, ps1, psStake1)
	require.NoError(t, err)
	require.NotEqual(t, addr0, addr1)
	require.Equal(t, "addr_test1xrp6fzzexfw2zusddj2fqftgv3dct300rqy5jyjkdljrgw3kqjmhg3z3nhhp3z7c4v40vt0d6ayns0cz8p9ccvplalwqhldpzk", addr1)

	addr1Stake, err := cliUtils.GetPolicyScriptRewardAddress(TestNetProtocolMagic, psStake1)
	require.NoError(t, err)
	require.Equal(t, "stake_test17qmqfdm5g3gemmsc30v2k2hk9hkawjfc8uprsjuvxql7lhq8tp4wd", addr1Stake)
}
