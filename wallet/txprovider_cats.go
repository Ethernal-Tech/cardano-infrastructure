package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type TxProviderCats struct {
	url          string
	apiKey       string
	apiKeyHeader string
}

var _ ITxProvider = (*TxProviderCats)(nil)

func NewTxProviderCats(url string, apiKey string, apiKeyHeader string) *TxProviderCats {
	if apiKeyHeader == "" {
		apiKeyHeader = "x-api-key"
	}

	return &TxProviderCats{
		url:          url,
		apiKey:       apiKey,
		apiKeyHeader: apiKeyHeader,
	}
}

// Dispose implements ITxProvider.
func (c *TxProviderCats) Dispose() {}

// GetProtocolParameters implements ITxProvider.
func (c *TxProviderCats) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	response, err := executeHTTPCats[getProtocolParametersCats](
		ctx, c.getURL("retrieve", "protocol-params"), "GET", c.apiKeyHeader, c.apiKey, nil)
	if err != nil {
		return nil, err
	}

	return response.ProtocolParameters, nil
}

// GetSlot implements ITxProvider.
func (c *TxProviderCats) GetTip(ctx context.Context) (QueryTipData, error) {
	response, err := executeHTTPCats[getTipCats](
		ctx, c.getURL("retrieve", "tip"), "GET", c.apiKeyHeader, c.apiKey, nil)
	if err != nil {
		return QueryTipData{}, err
	}

	return response.Tip, nil
}

// GetUtxos implements ITxProvider.
func (c *TxProviderCats) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	response, err := executeHTTPCats[getUtxosResponseCats](
		ctx, c.getURL("retrieve", "utxo", addr), "GET", c.apiKeyHeader, c.apiKey, nil)
	if err != nil {
		return nil, err
	}

	return response.Utxos, nil
}

// Expects TxCborString
func (c *TxProviderCats) SubmitTx(ctx context.Context, txSigned []byte) error {
	_, err := executeHTTPCats[baseResponseCats](
		ctx, c.getURL("submit", "tx"), "POST", c.apiKeyHeader, c.apiKey, submitTxRequestCats{
			Data: txSigned,
		})

	return err
}

// EvaluateTx implements ITxProvider.
func (c *TxProviderCats) EvaluateTx(ctx context.Context, rawTx []byte) (QueryEvaluateTxData, error) {
	panic("unimplemented") //nolint:gocritic
}

func (b *TxProviderCats) GetStakeAddressInfo(ctx context.Context, stakeAddress string) (QueryStakeAddressInfo, error) {
	panic("unimplemented") //nolint:gocritic
}

func (b *TxProviderCats) GetStakePools(ctx context.Context) ([]string, error) {
	panic("unimplemented") //nolint:gocritic
}

func (c *TxProviderCats) GetTxByHash(ctx context.Context, hash string) (map[string]interface{}, error) {
	panic("not implemented") //nolint:gocritic
}

func (c *TxProviderCats) getURL(parts ...string) string {
	return strings.Join(append([]string{c.url}, parts...), "/")
}

func executeHTTPCats[T iresponsecats](
	ctx context.Context, url string, method string, apiKeyHeader string, apiKey string, request any,
) (result T, err error) {
	var body io.Reader

	if request != nil {
		queryBytes, err := json.Marshal(request)
		if err != nil {
			return result, err
		}

		body = bytes.NewBuffer(queryBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return result, err
	}

	req.Header.Set("Content-Type", "application/json")

	if apiKeyHeader != "" && apiKey != "" {
		req.Header.Set(apiKeyHeader, apiKey)
	}

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("status code: %d, error: %w", resp.StatusCode, err)
	} else if result.getError() != "" {
		return result, fmt.Errorf("status code: %d, error: %s", resp.StatusCode, result.getError())
	}

	return result, nil
}

type iresponsecats interface {
	getError() string
}

type baseResponseCats struct {
	Error string `json:"error,omitempty"`
}

func (b baseResponseCats) getError() string { return b.Error }

type submitTxRequestCats struct {
	Data []byte `json:"data"`
}

type getUtxosResponseCats struct {
	baseResponseCats
	Utxos []Utxo `json:"utxos"`
}

type getProtocolParametersCats struct {
	baseResponseCats
	ProtocolParameters []byte `json:"protocolParameters"`
}

type getTipCats struct {
	baseResponseCats
	Tip QueryTipData `json:"tip"`
}
