package gouroboros

import (
	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/Ethernal-Tech/cardano-infrastructure/wallet"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func createTx(blockHeader *indexer.BlockHeader, ledgerTx ledger.Transaction, indx uint32) (*indexer.Tx, error) {
	tx := &indexer.Tx{
		Indx:      indx,
		Hash:      indexer.NewHashFromHexString(ledgerTx.Hash()),
		Fee:       ledgerTx.Fee(),
		BlockSlot: blockHeader.Slot,
		BlockHash: blockHeader.Hash,
		Valid:     ledgerTx.IsValid(),
	}

	if metadata := ledgerTx.Metadata(); metadata != nil {
		tx.Metadata = metadata.Cbor()
	}

	if inputs := ledgerTx.Inputs(); len(inputs) > 0 {
		tx.Inputs = make([]*indexer.TxInputOutput, len(inputs))

		for j, inp := range inputs {
			// output will be set later by indexer
			tx.Inputs[j] = &indexer.TxInputOutput{
				Input: indexer.TxInput{
					Hash:  indexer.Hash(inp.Id()),
					Index: inp.Index(),
				},
			}
		}
	}

	if outputs := ledgerTx.Outputs(); len(outputs) > 0 {
		tx.Outputs = make([]*indexer.TxOutput, len(outputs))
		for j, out := range outputs {
			tx.Outputs[j] = createTxOutput(blockHeader.Slot, ledgerAddressToString(out.Address()), out)
		}
	}

	return tx, nil
}

func createTxOutput(slot uint64, addr string, txOut common.TransactionOutput) *indexer.TxOutput {
	var tokens []indexer.TokenAmount

	if assets := txOut.Assets(); assets != nil {
		policies := assets.Policies()
		tokens = make([]indexer.TokenAmount, 0, len(policies))

		for _, policyIDRaw := range policies {
			policyID := policyIDRaw.String()

			for _, asset := range assets.Assets(policyIDRaw) {
				tokens = append(tokens, indexer.TokenAmount{
					PolicyID: policyID,
					Name:     string(asset),
					Amount:   assets.Asset(policyIDRaw, asset),
				})
			}
		}
	}

	var (
		datum     []byte
		datumHash indexer.Hash
	)

	if tmp := txOut.Datum(); tmp != nil {
		datum = tmp.Cbor()
	}

	if tmp := txOut.DatumHash(); tmp != nil {
		datumHash = indexer.Hash(tmp.Bytes())
	}

	return &indexer.TxOutput{
		Slot:      slot,
		Address:   addr,
		Amount:    txOut.Amount(),
		Tokens:    tokens,
		Datum:     datum,
		DatumHash: datumHash,
	}
}

// ledgerAddressToString translates string representation of address to our wallet representation
// this will handle vector and other specific cases
func ledgerAddressToString(addr ledger.Address) string {
	ourAddr, err := wallet.NewCardanoAddress(addr.Bytes())
	if err != nil {
		return addr.String()
	}

	return ourAddr.String()
}
