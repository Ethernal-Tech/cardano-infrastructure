package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTokenCostSum(t *testing.T) {
	t.Parallel()

	token1, _ := NewTokenAmountWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.4b6173685f546f6b656e", 11_000_039, true)
	token2, _ := NewTokenAmountWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.Route3", 236_872_039, false)
	token3, _ := NewTokenAmountWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.Route345", 12_236_872_039, false)

	txBuilder, err := NewTxBuilder(ResolveCardanoCliBinary(MainNetNetwork))
	require.NoError(t, err)

	defer txBuilder.Dispose()

	txBuilder.SetProtocolParameters(protocolParameters)

	address := "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u"
	utxos := []Utxo{
		{
			Amount: 100_000_000,
		},
		{
			Amount: 0,
			Tokens: []TokenAmount{
				token1,
				token2,
			},
		},
		{
			Amount: 3_000_000,
			Tokens: []TokenAmount{
				token3,
			},
		},
	}

	result, err := GetTokenCostSum(txBuilder, address, utxos)
	require.NoError(t, err)
	require.Equal(t, uint64(1293000), result)

	utxos[1].Tokens[0].Amount = 1 // changing token amount will change the output

	result, err = GetTokenCostSum(txBuilder, address, utxos)
	require.NoError(t, err)
	require.Equal(t, uint64(1275760), result)

	utxos[2].Tokens[0].Amount = 3 // changing token amount will change the output

	result, err = GetTokenCostSum(txBuilder, address, utxos)
	require.NoError(t, err)
	require.Equal(t, uint64(1241280), result)

	utxos[0].Amount = 3
	utxos[1].Amount = 300_021_416_931_256_900 // changing lovelace amounts won't make any difference

	result, err = GetTokenCostSum(txBuilder, address, utxos)
	require.NoError(t, err)
	require.Equal(t, uint64(1241280), result)
}

func TestCreateTxOutputChange(t *testing.T) {
	t.Parallel()

	token1, _ := NewTokenAmountWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.4b6173685f546f6b656e", 200, true)
	token2, _ := NewTokenAmountWithFullName("29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8.Route3", 300, false)
	address := "addr_test1vqjysa7p4mhu0l25qknwznvj0kghtr29ud7zp732ezwtzec0w8g3u"

	t.Run("invalid amount", func(t *testing.T) {
		t.Parallel()

		_, err := CreateTxOutputChange(TxOutput{}, map[string]uint64{
			AdaTokenName: 100,
		}, map[string]uint64{
			AdaTokenName: 101,
		})
		require.ErrorContains(t, err, "invalid amount:")
	})

	t.Run("invalid token amount", func(t *testing.T) {
		t.Parallel()

		_, err := CreateTxOutputChange(TxOutput{}, map[string]uint64{
			AdaTokenName:       102,
			token1.TokenName(): 105,
			token2.TokenName(): 105,
		}, map[string]uint64{
			AdaTokenName:       101,
			token1.TokenName(): 106,
			token2.TokenName(): 105,
		})
		require.ErrorContains(t, err, "invalid token amount:")
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		res, err := CreateTxOutputChange(TxOutput{
			Addr:   address,
			Amount: 100,
			Tokens: []TokenAmount{
				token1,
				token2,
			},
		}, map[string]uint64{
			AdaTokenName:       400,
			token1.TokenName(): 500,
			token2.TokenName(): 600,
		}, map[string]uint64{
			AdaTokenName:       10,
			token1.TokenName(): 20,
			token2.TokenName(): 30,
		})
		require.NoError(t, err)
		require.Equal(t, TxOutput{
			Addr:   address,
			Amount: 490,
			Tokens: []TokenAmount{
				NewTokenAmount(token1.PolicyID, token1.Name, 680),
				NewTokenAmount(token2.PolicyID, token2.Name, 870),
			},
		}, res)
	})
}
