package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

const (
	paymentExtendedSigningKeyShelley = "PaymentExtendedSigningKeyShelley_ed25519_bip32"
	paymentSigningKeyShelley         = "PaymentSigningKeyShelley_ed25519"
	witnessJSONDesc                  = "Key Witness ShelleyEra"
	ledgerCddlFormatDesc             = "Ledger Cddl Format"
)

type TxWitnessRaw []byte // cbor slice of bytes

func (w TxWitnessRaw) ToJSON(era string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":        fmt.Sprintf("TxWitness %sEra", era),
		"description": witnessJSONDesc,
		"cborHex":     hex.EncodeToString(w),
	})
}

func (w TxWitnessRaw) GetSignatureAndVKey() ([]byte, []byte, error) {
	bytes := w
	// skip prefix bytes for conway
	if len(bytes) >= 2 && bytes[0] == 0x82 && bytes[1] == 0x00 {
		bytes = bytes[2:]
	}

	var signatureWitness [2][]byte // Use the appropriate type for your CBOR structure

	if err := cbor.Unmarshal(bytes, &signatureWitness); err != nil {
		return nil, nil, err
	}

	return signatureWitness[1], signatureWitness[0], nil
}

type transactionUnwitnessedRaw []byte

func newTransactionUnwitnessedRawFromJSON(bytes []byte) (transactionUnwitnessedRaw, error) {
	var data map[string]any

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	return hex.DecodeString(data["cborHex"].(string)) //nolint:forcetypeassert
}

func (tx transactionUnwitnessedRaw) ToJSON(era string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":        fmt.Sprintf("Unwitnessed Tx %sEra", era),
		"description": ledgerCddlFormatDesc,
		"cborHex":     hex.EncodeToString(tx),
	})
}

type transactionWitnessedRaw []byte

func newTransactionWitnessedRawFromJSON(bytes []byte) (transactionWitnessedRaw, error) {
	var data map[string]any

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	return hex.DecodeString(data["cborHex"].(string)) //nolint:forcetypeassert
}

func (tx transactionWitnessedRaw) ToJSON(era string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":        fmt.Sprintf("Witnessed Tx %sEra", era),
		"description": ledgerCddlFormatDesc,
		"cborHex":     hex.EncodeToString(tx),
	})
}
