package db

import (
	"strings"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	indexerbbolt "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/bbolt"
	indexerleveldb "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/leveldb"
)

func NewDatabase(name string) indexer.Database {
	switch strings.ToLower(name) {
	case "leveldb":
		return &indexerleveldb.LevelDbDatabase{}
	default:
		return &indexerbbolt.BBoltDatabase{}
	}
}

func NewDatabaseInit(name string, filePath string) (indexer.Database, error) {
	db := NewDatabase(name)
	if err := db.Init(filePath); err != nil {
		return nil, err
	}

	return db, nil
}
