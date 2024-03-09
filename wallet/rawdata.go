package wallet

import (
	"encoding/hex"
	"encoding/json"

	"github.com/fxamacker/cbor/v2"
)

var (
	witnessJsonType       = "TxWitness BabbageEra"
	witnessJsonDesc       = "Key Witness ShelleyEra"
	txUnwitnessedJsonType = "Unwitnessed Tx BabbageEra"
	txUnwitnessedJsonDesc = "Ledger Cddl Format"
	txWitnessedJsonType   = "Witnessed Tx BabbageEra"
	txWitnessedJsonDesc   = "Ledger Cddl Format"
)

type TransactionUnwitnessedRaw []byte

func NewTransactionUnwitnessedRawFromJson(bytes []byte) (TransactionUnwitnessedRaw, error) {
	var data map[string]interface{}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	// a little hack so we have always correct witness key and description for json (cardano-cli can return error otherwise)
	txUnwitnessedJsonType = data["type"].(string)
	txUnwitnessedJsonDesc = data["description"].(string)

	return hex.DecodeString(data["cborHex"].(string))
}

func (tx TransactionUnwitnessedRaw) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        txUnwitnessedJsonType,
		"description": txUnwitnessedJsonDesc,
		"cborHex":     hex.EncodeToString(tx),
	})
}

type TransactionWitnessedRaw []byte

func NewTransactionWitnessedRawFromJson(bytes []byte) (TransactionWitnessedRaw, error) {
	var data map[string]interface{}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	// a little hack so we have always correct witness key and description for json (cardano-cli can return error otherwise)
	txWitnessedJsonType = data["type"].(string)
	txWitnessedJsonDesc = data["description"].(string)

	return hex.DecodeString(data["cborHex"].(string))
}

func (tx TransactionWitnessedRaw) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        txWitnessedJsonType,
		"description": txWitnessedJsonDesc,
		"cborHex":     hex.EncodeToString(tx),
	})
}

type TxWitnessRaw []byte // cbor slice of bytes

func (w TxWitnessRaw) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        witnessJsonType,
		"description": witnessJsonDesc,
		"cborHex":     hex.EncodeToString(w),
	})
}

func (w TxWitnessRaw) GetSignatureAndVKey() ([]byte, []byte, error) {
	var signatureWitness [2][]byte // Use the appropriate type for your CBOR structure

	if err := cbor.Unmarshal(w, &signatureWitness); err != nil {
		return nil, nil, err
	}

	return signatureWitness[1], signatureWitness[0], nil
}
