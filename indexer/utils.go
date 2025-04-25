package indexer

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// SortTxInputOutputs sorts a slice of TxInputOutput pointers based on the following priority:
//  1. Inputs with lower Output.Slot values come first —
//     this is critical because inputs added earlier must be processed first
//  2. If Slot values are equal, inputs are sorted lexicographically by their Input.Hash
//  3. If both Slot and Hash are equal, inputs are sorted by Input.Index
//
// The returned slice reflects this ordering and ensures deterministic processing of inputs
// in the correct chronological order.
func SortTxInputOutputs(txInputsOutputs []*TxInputOutput) []*TxInputOutput {
	sort.Slice(txInputsOutputs, func(i, j int) bool {
		first, second := txInputsOutputs[i], txInputsOutputs[j]

		if first.Output.Slot != second.Output.Slot {
			return first.Output.Slot < second.Output.Slot
		}

		if cmp := bytes.Compare(first.Input.Hash[:], second.Input.Hash[:]); cmp != 0 {
			return cmp < 0
		}

		return first.Input.Index < second.Input.Index
	})

	return txInputsOutputs
}

// SlotNumberToKey converts a slot number to a byte array of size 8
func SlotNumberToKey(slotNumber uint64) []byte {
	bytes := make([]byte, 8)

	binary.BigEndian.PutUint64(bytes, slotNumber)

	return bytes
}

func getTxHashes(txs []*Tx) []Hash {
	if len(txs) == 0 {
		return nil
	}

	result := make([]Hash, len(txs))
	for i, tx := range txs {
		result[i] = tx.Hash
	}

	return result
}

func getTxOutputs(txs []*Tx, addressesOfInterest map[string]bool) (res []*TxInputOutput) {
	for _, tx := range txs {
		for outIndex, txOut := range tx.Outputs {
			if len(addressesOfInterest) == 0 || addressesOfInterest[txOut.Address] {
				res = append(res, &TxInputOutput{
					Input: TxInput{
						Hash:  tx.Hash,
						Index: uint32(outIndex), //nolint:gosec
					},
					Output: *txOut,
				})
			}
		}
	}

	return res
}

func getTxInputs(txs []*Tx, addressesOfInterest map[string]bool) (res []*TxInput) {
	for _, tx := range txs {
		for _, inp := range tx.Inputs {
			if len(addressesOfInterest) == 0 || addressesOfInterest[inp.Output.Address] {
				res = append(res, &inp.Input)
			}
		}
	}

	return res
}
