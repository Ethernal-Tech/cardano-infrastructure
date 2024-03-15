package indexerbbolt

import (
	"encoding/json"
	"fmt"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"

	"go.etcd.io/bbolt"
)

type txOperation func(tx *bbolt.Tx) error

type BBoltTransactionWriter struct {
	db         *bbolt.DB
	operations []txOperation
}

var _ core.DbTransactionWriter = (*BBoltTransactionWriter)(nil)

func (tw *BBoltTransactionWriter) SetLatestBlockPoint(point *core.BlockPoint) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(tx *bbolt.Tx) error {
		bytes, err := json.Marshal(point)
		if err != nil {
			return fmt.Errorf("could not marshal latest block point: %v", err)
		}

		if err = tx.Bucket(latestBlockPointBucket).Put(defaultKey, bytes); err != nil {
			return fmt.Errorf("latest block point write error: %v", err)
		}

		return nil
	})

	return tw
}

func (tw *BBoltTransactionWriter) AddTxOutputs(txOutputs []*core.TxInputOutput) core.DbTransactionWriter {
	if len(txOutputs) == 0 {
		return tw
	}

	tw.operations = append(tw.operations, func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(txOutputsBucket)

		for _, inpOut := range txOutputs {
			bytes, err := json.Marshal(inpOut.Output)
			if err != nil {
				return fmt.Errorf("could not marshal tx output: %v", err)
			}

			if err = bucket.Put(inpOut.Input.Key(), bytes); err != nil {
				return fmt.Errorf("tx output write error: %v", err)
			}
		}

		return nil
	})

	return tw
}

func (tw *BBoltTransactionWriter) AddConfirmedTxs(txs []*core.Tx) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(tx *bbolt.Tx) error {
		for _, cardTx := range txs {
			bytes, err := json.Marshal(cardTx)
			if err != nil {
				return fmt.Errorf("could not marshal confirmed tx: %v", err)
			}

			if err = tx.Bucket(unprocessedTxsBucket).Put(cardTx.Key(), bytes); err != nil {
				return fmt.Errorf("confirmed tx write error: %v", err)
			}
		}

		return nil
	})

	return tw
}

func (tw *BBoltTransactionWriter) RemoveTxOutputs(txInputs []*core.TxInput, softDelete bool) core.DbTransactionWriter {
	if len(txInputs) == 0 {
		return tw
	}

	tw.operations = append(tw.operations, func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(txOutputsBucket)

		for _, inp := range txInputs {
			key := inp.Key()

			if !softDelete {
				if err := bucket.Delete(key); err != nil {
					return fmt.Errorf("delete utxo error: %v", err)
				}
			} else if data := bucket.Get(key); len(data) > 0 {
				var result core.TxOutput

				if err := json.Unmarshal(data, &result); err != nil {
					return fmt.Errorf("soft delete unmarshal utxo error: %v", err)
				}

				result.IsUsed = true

				bytes, err := json.Marshal(result)
				if err != nil {
					return fmt.Errorf("soft delete marshal utxo error: %v", err)
				}

				if err := bucket.Put(key, bytes); err != nil {
					return fmt.Errorf("soft delete put utxo error: %v", err)
				}
			}
		}

		return nil
	})

	return tw
}

func (tw *BBoltTransactionWriter) Execute() error {
	defer func() {
		tw.operations = nil
	}()

	return tw.db.Update(func(tx *bbolt.Tx) error {
		for _, op := range tw.operations {
			if err := op(tx); err != nil {
				return err
			}
		}

		return nil
	})
}
