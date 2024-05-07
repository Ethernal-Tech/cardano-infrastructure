package wallet

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type OgmiosProvider struct {
	url string
}

var _ ITxProvider = (*OgmiosProvider)(nil)

func NewOgmiosProvider(url string) *OgmiosProvider {
	return &OgmiosProvider{
		url: url,
	}
}

// Dispose implements ITxProvider.
func (o *OgmiosProvider) Dispose() {}

// GetProtocolParameters implements ITxProvider.
func (o *OgmiosProvider) GetProtocolParameters(ctx context.Context) ([]byte, error) {
	query := queryLedgerStateProtocolParameters{
		Jsonrpc: "2.0",
		Method:  "queryLedgerState/protocolParameters",
		ID:      nil,
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, getErrorFromResponse(resp)
	}

	var responseData queryLedgerStateProtocolParametersResponse
	// Unmarshal the JSON into the struct
	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}

	return convertProtocolParametersOgmios(responseData)
}

// GetSlot implements ITxProvider.
func (o *OgmiosProvider) GetTip(ctx context.Context) (QueryTipData, error) {
	queryBlockHeight := queryNetworkBlockHeight{
		Jsonrpc: "2.0",
		Method:  "queryNetwork/blockHeight",
		ID:      nil,
	}

	queryBytes, err := json.Marshal(queryBlockHeight)
	if err != nil {
		return QueryTipData{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return QueryTipData{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	respBlockHeight, err := new(http.Client).Do(req)
	if err != nil {
		return QueryTipData{}, err
	}
	defer respBlockHeight.Body.Close()

	if respBlockHeight.StatusCode != http.StatusOK {
		return QueryTipData{}, getErrorFromResponse(respBlockHeight)
	}

	var blockHeightResponseData queryNetworkBlockHeightResponse
	if err = json.NewDecoder(respBlockHeight.Body).Decode(&blockHeightResponseData); err != nil {
		return QueryTipData{}, err
	}

	query := queryLedgerStateTip{
		Jsonrpc: "2.0",
		Method:  "queryLedgerState/tip",
		ID:      nil,
	}

	queryBytes, err = json.Marshal(query)
	if err != nil {
		return QueryTipData{}, err
	}

	req, err = http.NewRequestWithContext(ctx, "POST", o.url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return QueryTipData{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return QueryTipData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return QueryTipData{}, getErrorFromResponse(resp)
	}

	// Unmarshal the JSON into the struct
	var responseData queryLedgerStateTipResponse

	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return QueryTipData{}, err
	}

	return QueryTipData{
		Block:           blockHeightResponseData.Result,
		Epoch:           0,
		Era:             "",
		Hash:            responseData.Result.ID,
		Slot:            uint64(responseData.Result.Slot),
		SlotInEpoch:     0,
		SlotsToEpochEnd: 0,
		SyncProgress:    "",
	}, nil
}

// GetUtxos implements ITxProvider.
func (o *OgmiosProvider) GetUtxos(ctx context.Context, addr string) ([]Utxo, error) {
	query := queryLedgerStateUtxo{
		Jsonrpc: "2.0",
		Method:  "queryLedgerState/utxo",
		Params: queryLedgerStateUtxoParams{
			Addresses: []string{addr},
		},
		ID: nil,
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, getErrorFromResponse(resp)
	}

	var responseData queryLedgerStateUtxoResponse
	// Unmarshal the JSON into the struct
	if err = json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}

	var retVal = make([]Utxo, len(responseData.Result))
	for i, utxo := range responseData.Result {
		retVal[i] = Utxo{
			Hash:   utxo.Transaction.ID,
			Index:  uint32(utxo.Index),
			Amount: uint64(utxo.Value.Ada.Lovelace),
		}
	}

	return retVal, nil
}

// Expects TxCborString
func (o *OgmiosProvider) SubmitTx(ctx context.Context, txSigned []byte) error {
	txCborString := hex.EncodeToString(txSigned)

	requestBody := submitTransaction{
		Jsonrpc: "2.0",
		Method:  "submitTransaction",
		Params: submitTransactionParams{
			Transaction: submitTransactionParamsTransaction{
				CBOR: txCborString,
			},
		},
		ID: nil,
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getErrorFromResponse(resp)
	}

	return nil
}
