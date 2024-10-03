package wallet

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWalletFromMnemonics(t *testing.T) {
	const (
		mnemonic = "famous drink primary tree harsh fiscal skill space lift pact cruise custom long joke mask clerk pulse sword better thunder naive kick gasp jazz"
		network  = MainNetNetwork
	)

	w, err := NewWalletFromMnemonic(
		ResolveCardanoCliBinary(network),
		ResolveCardanoAddressBinary(),
		mnemonic, 0)
	require.NoError(t, err)

	baseAddr, err := NewBaseAddress(network, w.GetVerificationKey(), w.GetStakeVerificationKey())
	require.NoError(t, err)

	eaddr, err := NewEnterpriseAddress(network, w.GetVerificationKey())
	require.NoError(t, err)

	stakeAddr, err := NewRewardAddress(network, w.GetStakeVerificationKey())
	require.NoError(t, err)

	fmt.Println(stakeAddr)
	fmt.Println(eaddr)
	fmt.Println(baseAddr)
	fmt.Println(hex.EncodeToString(w.GetVerificationKey()))
	fmt.Println(hex.EncodeToString(w.GetStakeVerificationKey()))
}
