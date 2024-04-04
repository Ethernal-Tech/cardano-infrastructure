package indexerleveldb

import (
	"encoding/json"
	"errors"
	"fmt"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type txOperation func(*leveldb.DB, *leveldb.Batch) error

type LevelDbTransactionWriter struct {
	db         *leveldb.DB
	operations []txOperation
}

var _ core.DbTransactionWriter = (*LevelDbTransactionWriter)(nil)

func NewLevelDbTransactionWriter(db *leveldb.DB) *LevelDbTransactionWriter {
	return &LevelDbTransactionWriter{
		db: db,
	}
}

func (tw *LevelDbTransactionWriter) SetLatestBlockPoint(point *core.BlockPoint) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(db *leveldb.DB, batch *leveldb.Batch) error {
		bytes, err := json.Marshal(point)
		if err != nil {
			return fmt.Errorf("could not marshal latest block point: %w", err)
		}

		batch.Put(latestBlockPointBucket, bytes)

		return nil
	})

	return tw
}

func (tw *LevelDbTransactionWriter) AddTxOutputs(txOutputs []*core.TxInputOutput) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(db *leveldb.DB, batch *leveldb.Batch) error {
		for _, inpOut := range txOutputs {
			bytes, err := json.Marshal(inpOut.Output)
			if err != nil {
				return fmt.Errorf("could not marshal tx output: %w", err)
			}

			batch.Put(bucketKey(txOutputsBucket, inpOut.Input.Key()), bytes)
		}

		return nil
	})

	return tw
}

func (tw *LevelDbTransactionWriter) AddConfirmedBlock(block *core.CardanoBlock) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(db *leveldb.DB, batch *leveldb.Batch) error {
		bytes, err := json.Marshal(block)
		if err != nil {
			return fmt.Errorf("could not marshal confirmed block: %w", err)
		}

		batch.Put(bucketKey(confirmedBlocks, block.Key()), bytes)

		return nil
	})

	return tw
}

func (tw *LevelDbTransactionWriter) AddConfirmedTxs(txs []*core.Tx) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(db *leveldb.DB, batch *leveldb.Batch) error {
		for _, tx := range txs {
			bytes, err := json.Marshal(tx)
			if err != nil {
				return fmt.Errorf("could not marshal confirmed tx: %w", err)
			}

			batch.Put(bucketKey(unprocessedTxsBucket, tx.Key()), bytes)
		}

		return nil
	})

	return tw
}

func (tw *LevelDbTransactionWriter) RemoveTxOutputs(txInputs []*core.TxInput, softDelete bool) core.DbTransactionWriter {
	tw.operations = append(tw.operations, func(db *leveldb.DB, batch *leveldb.Batch) error {
		for _, inp := range txInputs {
			key := bucketKey(txOutputsBucket, inp.Key())

			if !softDelete {
				batch.Delete(key)

				continue
			}

			data, err := db.Get(key, &opt.ReadOptions{
				DontFillCache: true,
			})
			if err != nil {
				if errors.Is(err, leveldb.ErrNotFound) {
					continue
				}

				return err
			}

			var result core.TxOutput

			if err := json.Unmarshal(data, &result); err != nil {
				return fmt.Errorf("soft delete unmarshal utxo error: %w", err)
			}

			result.IsUsed = true

			bytes, err := json.Marshal(result)
			if err != nil {
				return fmt.Errorf("soft delete marshal utxo error: %w", err)
			}

			batch.Put(key, bytes)
		}

		return nil
	})

	return tw
}

func (tw *LevelDbTransactionWriter) Execute() error {
	defer func() {
		tw.operations = nil
	}()

	batch := new(leveldb.Batch)

	for _, op := range tw.operations {
		if err := op(tw.db, batch); err != nil {
			return err
		}
	}

	return tw.db.Write(batch, &opt.WriteOptions{
		NoWriteMerge: false,
		Sync:         true,
	})
}
