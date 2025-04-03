package indexerleveldb

import (
	"encoding/json"
	"errors"
	"fmt"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LevelDBDatabase struct {
	db *leveldb.DB
}

var (
	txOutputsBucket        = []byte("P1_")
	latestBlockPointBucket = []byte("P2_")
	processedTxsBucket     = []byte("P3_")
	unprocessedTxsBucket   = []byte("P4_")
	confirmedBlocks        = []byte("P5_")
)

var _ core.Database = (*LevelDBDatabase)(nil)

func (lvldb *LevelDBDatabase) Init(filePath string) error {
	db, err := leveldb.OpenFile(filePath, nil)
	if err != nil {
		return fmt.Errorf("could not open db: %w", err)
	}

	lvldb.db = db

	return nil
}

func (lvldb *LevelDBDatabase) Close() error {
	return lvldb.db.Close()
}

func (lvldb *LevelDBDatabase) GetLatestBlockPoint() (*core.BlockPoint, error) {
	var result *core.BlockPoint

	bytes, err := lvldb.db.Get(latestBlockPointBucket, nil)
	if err != nil {
		return nil, processNotFoundErr(err)
	}

	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (lvldb *LevelDBDatabase) GetTxOutput(txInput core.TxInput) (result core.TxOutput, err error) {
	bytes, err := lvldb.db.Get(bucketKey(txOutputsBucket, txInput.Key()), nil)
	if err != nil {
		return result, processNotFoundErr(err)
	}

	err = json.Unmarshal(bytes, &result)

	return result, err
}

func (lvldb *LevelDBDatabase) MarkConfirmedTxsProcessed(txs []*core.Tx) error {
	batch := new(leveldb.Batch)

	for _, tx := range txs {
		bytes, err := json.Marshal(tx)
		if err != nil {
			return fmt.Errorf("could not marshal tx: %w", err)
		}

		batch.Put(bucketKey(processedTxsBucket, tx.Key()), bytes)
		batch.Delete(bucketKey(unprocessedTxsBucket, tx.Key()))
	}

	return lvldb.db.Write(batch, &opt.WriteOptions{
		NoWriteMerge: false,
		Sync:         true,
	})
}

func (lvldb *LevelDBDatabase) GetUnprocessedConfirmedTxs(maxCnt int) ([]*core.Tx, error) {
	var result []*core.Tx

	iter := lvldb.db.NewIterator(util.BytesPrefix(unprocessedTxsBucket), nil)
	defer iter.Release()

	for iter.Next() {
		var tx *core.Tx

		if err := json.Unmarshal(iter.Value(), &tx); err != nil {
			return nil, err
		}

		result = append(result, tx)
		if maxCnt > 0 && len(result) == maxCnt {
			break
		}
	}

	return result, iter.Error()
}

func (lvldb *LevelDBDatabase) GetLatestConfirmedBlocks(maxCnt int) ([]*core.CardanoBlock, error) {
	var result []*core.CardanoBlock

	iter := lvldb.db.NewIterator(util.BytesPrefix(confirmedBlocks), nil)
	defer iter.Release()

	for ok := iter.Last(); ok; ok = iter.Prev() {
		var block *core.CardanoBlock

		if err := json.Unmarshal(iter.Value(), &block); err != nil {
			return nil, err
		}

		result = append(result, block)
		if maxCnt > 0 && len(result) == maxCnt {
			break
		}
	}

	return result, iter.Error()
}

func (lvldb *LevelDBDatabase) GetConfirmedBlocksFrom(slotNumber uint64, maxCnt int) ([]*core.CardanoBlock, error) {
	var result []*core.CardanoBlock

	iter := lvldb.db.NewIterator(util.BytesPrefix(confirmedBlocks), nil)
	defer iter.Release()

	for ok := iter.Seek(bucketKey(confirmedBlocks, core.SlotNumberToKey(slotNumber))); ok; ok = iter.Next() {
		var block *core.CardanoBlock

		if err := json.Unmarshal(iter.Value(), &block); err != nil {
			return nil, err
		}

		result = append(result, block)
		if maxCnt > 0 && len(result) == maxCnt {
			break
		}
	}

	return result, iter.Error()
}

func (lvldb *LevelDBDatabase) GetAllTxOutputs(address string, onlyNotUsed bool) ([]*core.TxInputOutput, error) {
	var result []*core.TxInputOutput

	iter := lvldb.db.NewIterator(util.BytesPrefix(txOutputsBucket), nil)
	defer iter.Release()

	for iter.Next() {
		var output core.TxOutput

		if err := json.Unmarshal(iter.Value(), &output); err != nil {
			return nil, err
		}

		if output.Address != address || (onlyNotUsed && output.IsUsed) {
			continue
		}

		input, _ := core.NewTxInputFromBytes(iter.Key())

		result = append(result, &core.TxInputOutput{
			Input:  input,
			Output: output,
		})
	}

	return core.SortTxInputOutputs(result), nil
}

func (lvldb *LevelDBDatabase) OpenTx() core.DBTransactionWriter {
	return NewLevelDBTransactionWriter(lvldb.db)
}

func bucketKey(bucket []byte, key []byte) []byte {
	const separator = "_#_"

	outputKey := make([]byte, len(bucket)+len(separator)+len(key))
	copy(outputKey, bucket)
	copy(outputKey[len(bucket):], []byte(separator))
	copy(outputKey[len(bucket)+len(separator):], key)

	return outputKey
}

func processNotFoundErr(err error) error {
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil
	}

	return err
}
