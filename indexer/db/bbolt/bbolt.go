package indexerbbolt

import (
	"encoding/json"
	"fmt"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"go.etcd.io/bbolt"
)

type BBoltDatabase struct {
	db *bbolt.DB
}

var (
	txOutputsBucket        = []byte("TXOuts")
	latestBlockPointBucket = []byte("LatestBlockPoint")
	processedTxsBucket     = []byte("ProcessedTxs")
	unprocessedTxsBucket   = []byte("UnprocessedTxs")
	confirmedBlocks        = []byte("confirmedBlocks")

	defaultKey = []byte("default")
)

var _ core.Database = (*BBoltDatabase)(nil)

func (bd *BBoltDatabase) Init(filePath string) error {
	db, err := bbolt.Open(filePath, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db: %v", err)
	}

	bd.db = db

	return db.Update(func(tx *bbolt.Tx) error {
		for _, bn := range [][]byte{txOutputsBucket, latestBlockPointBucket, processedTxsBucket, unprocessedTxsBucket, confirmedBlocks} {
			_, err := tx.CreateBucketIfNotExists(bn)
			if err != nil {
				return fmt.Errorf("could not bucket: %s, err: %v", string(bn), err)
			}
		}

		return nil
	})
}

func (bd *BBoltDatabase) Close() error {
	return bd.db.Close()
}

func (bd *BBoltDatabase) GetLatestBlockPoint() (*core.BlockPoint, error) {
	var result *core.BlockPoint

	if err := bd.db.View(func(tx *bbolt.Tx) error {
		if data := tx.Bucket(latestBlockPointBucket).Get(defaultKey); len(data) > 0 {
			return json.Unmarshal(data, &result)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (bd *BBoltDatabase) GetTxOutput(txInput core.TxInput) (result core.TxOutput, err error) {
	err = bd.db.View(func(tx *bbolt.Tx) error {
		if data := tx.Bucket(txOutputsBucket).Get(txInput.Key()); len(data) > 0 {
			return json.Unmarshal(data, &result)
		}

		return nil
	})

	return result, err
}

func (bd *BBoltDatabase) MarkConfirmedTxsProcessed(txs []*core.Tx) error {
	return bd.db.Update(func(tx *bbolt.Tx) error {
		for _, cardTx := range txs {
			if err := tx.Bucket(unprocessedTxsBucket).Delete(cardTx.Key()); err != nil {
				return fmt.Errorf("could not remove from unprocessed blocks: %v", err)
			}

			bytes, err := json.Marshal(cardTx)
			if err != nil {
				return fmt.Errorf("could not marshal block: %v", err)
			}

			if err := tx.Bucket(processedTxsBucket).Put(cardTx.Key(), bytes); err != nil {
				return fmt.Errorf("could not move to processed blocks: %v", err)
			}
		}

		return nil
	})
}

func (bd *BBoltDatabase) GetUnprocessedConfirmedTxs(maxCnt int) ([]*core.Tx, error) {
	var result []*core.Tx

	err := bd.db.View(func(tx *bbolt.Tx) error {
		cursor := tx.Bucket(unprocessedTxsBucket).Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var cardTx *core.Tx

			if err := json.Unmarshal(v, &cardTx); err != nil {
				return err
			}

			result = append(result, cardTx)
			if maxCnt > 0 && len(result) == maxCnt {
				break
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bd *BBoltDatabase) GetLatestConfirmedBlocks(maxCnt int) ([]*core.CardanoBlock, error) {
	var result []*core.CardanoBlock

	err := bd.db.View(func(tx *bbolt.Tx) error {
		cursor := tx.Bucket(confirmedBlocks).Cursor()

		for k, v := cursor.Last(); k != nil; k, v = cursor.Prev() {
			var block *core.CardanoBlock

			if err := json.Unmarshal(v, &block); err != nil {
				return err
			}

			result = append(result, block)
			if maxCnt > 0 && len(result) == maxCnt {
				break
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bd *BBoltDatabase) GetConfirmedBlocksFrom(slotNumber uint64, maxCnt int) ([]*core.CardanoBlock, error) {
	var result []*core.CardanoBlock

	err := bd.db.View(func(tx *bbolt.Tx) error {
		cursor := tx.Bucket(confirmedBlocks).Cursor()

		for k, v := cursor.Seek(core.SlotNumberToKey(slotNumber)); k != nil; k, v = cursor.Next() {
			var block *core.CardanoBlock

			if err := json.Unmarshal(v, &block); err != nil {
				return err
			}

			result = append(result, block)
			if maxCnt > 0 && len(result) == maxCnt {
				break
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bd *BBoltDatabase) OpenTx() core.DbTransactionWriter {
	return &BBoltTransactionWriter{
		db: bd.db,
	}
}
