package wallet

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTxInUtxos(t *testing.T) {
	t.Parallel()

	const existingHash = "0xfff03"

	var (
		ctx       = context.Background()
		errCustom = errors.New("custom error")
		utxos     = []Utxo{
			{Hash: existingHash},
		}
		mock = &txRetrieverMock{
			getUtxosFn: func(_ context.Context, addr string) ([]Utxo, error) {
				switch addr {
				case "a":
					return nil, errCustom
				default:
					return utxos, nil
				}
			},
		}
	)

	_, err := IsTxInUtxos(ctx, mock, "a", existingHash)
	require.ErrorIs(t, err, errCustom)

	res, err := IsTxInUtxos(ctx, mock, "b", "00dfg")
	require.NoError(t, err)
	require.False(t, res)

	res, err = IsTxInUtxos(ctx, mock, "b", existingHash)
	require.NoError(t, err)
	require.True(t, res)
}

func TestGetUtxosSum(t *testing.T) {
	t.Parallel()

	result := GetUtxosSum([]Utxo{
		{
			Amount: 200,
		},
		{
			Amount: 0,
			Tokens: []TokenAmount{
				NewTokenAmount("1", "1", 100),
				NewTokenAmount("2", "1", 400),
			},
		},
		{
			Amount: 300,
			Tokens: []TokenAmount{
				NewTokenAmount("3", "3", 20),
				NewTokenAmount("2", "1", 150),
			},
		},
	})

	require.Equal(t, 4, len(result))
	require.Equal(t, uint64(500), result[AdaTokenName])
	require.Equal(t, uint64(20), result["3.33"])
	require.Equal(t, uint64(550), result["2.31"])
	require.Equal(t, uint64(100), result["1.31"])
}

func TestGetOutputsSum(t *testing.T) {
	t.Parallel()

	const psHash = "29f8873beb52e126f207a2dfd50f7cff556806b5b4cba9834a7b26a8"

	// Kash_Token
	token1, err := NewTokenAmountWithFullName(fmt.Sprintf("%s.4b6173685f546f6b656e", psHash), 190, true)
	require.NoError(t, err)

	// Route3 token
	token2, err := NewTokenAmountWithFullName(fmt.Sprintf("%s.Route3", psHash), 720, false)
	require.NoError(t, err)

	result := GetOutputsSum([]TxOutput{
		NewTxOutput("", 200),
		NewTxOutput("", 300, NewTokenAmount("1", "2", 10), token1),
		NewTxOutput("", 100, token2),
		NewTxOutput("", 50, NewTokenAmount("1", "2", 30)),
	})

	require.Equal(t, 4, len(result))
	require.Equal(t, uint64(650), result[AdaTokenName])
	require.Equal(t, uint64(40), result["1.32"])
	require.Equal(t, uint64(720), result[fmt.Sprintf("%s.526f75746533", psHash)])
	require.Equal(t, uint64(190), result[fmt.Sprintf("%s.4b6173685f546f6b656e", psHash)])
}

type txRetrieverMock struct {
	getUtxosFn func(ctx context.Context, addr string) ([]Utxo, error)
}

func (m txRetrieverMock) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	return m.getUtxosFn(ctx, addr)
}
