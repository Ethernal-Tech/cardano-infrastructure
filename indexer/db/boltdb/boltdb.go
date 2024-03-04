package indexerboltdb

import (
	"encoding/json"
	"fmt"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	"github.com/boltdb/bolt"
)

type BoltDatabase struct {
	db *bolt.DB
}

var (
	txOutputsBucket        = []byte("TXOuts")
	latestBlockPointBucket = []byte("LatestBlockPoint")
	processedTxsBucket     = []byte("ProcessedTxs")
	unprocessedTxsBucket   = []byte("UnprocessedTxs")

	defaultKey = []byte("default")
)

var _ core.Database = (*BoltDatabase)(nil)

func (bd *BoltDatabase) Init(filePath string) error {
	db, err := bolt.Open(filePath, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db: %v", err)
	}

	bd.db = db

	return db.Update(func(tx *bolt.Tx) error {
		for _, bn := range [][]byte{txOutputsBucket, latestBlockPointBucket, processedTxsBucket, unprocessedTxsBucket} {
			_, err := tx.CreateBucketIfNotExists(bn)
			if err != nil {
				return fmt.Errorf("could not bucket: %s, err: %v", string(bn), err)
			}
		}

		return nil
	})
}

func (bd *BoltDatabase) Close() error {
	return bd.db.Close()
}

func (bd *BoltDatabase) GetLatestBlockPoint() (*core.BlockPoint, error) {
	var result *core.BlockPoint

	if err := bd.db.View(func(tx *bolt.Tx) error {
		if data := tx.Bucket(latestBlockPointBucket).Get(defaultKey); len(data) > 0 {
			return json.Unmarshal(data, &result)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (bd *BoltDatabase) GetTxOutput(txInput core.TxInput) (result core.TxOutput, err error) {
	err = bd.db.View(func(tx *bolt.Tx) error {
		if data := tx.Bucket(txOutputsBucket).Get(txInput.Key()); len(data) > 0 {
			return json.Unmarshal(data, &result)
		}

		return nil
	})

	return result, err
}

func (bd *BoltDatabase) MarkConfirmedTxsProcessed(txs []*core.Tx) error {
	return bd.db.Update(func(tx *bolt.Tx) error {
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

func (bd *BoltDatabase) GetUnprocessedConfirmedTxs(maxCnt int) ([]*core.Tx, error) {
	var result []*core.Tx

	err := bd.db.View(func(tx *bolt.Tx) error {
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

func (bd *BoltDatabase) OpenTx() core.DbTransactionWriter {
	return &BoltDbTransactionWriter{
		db: bd.db,
	}
}
