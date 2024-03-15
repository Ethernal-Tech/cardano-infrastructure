package indexerdb

import (
	"strings"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	bbolt "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/bbolt"
	leveldb "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/leveldb"
)

func NewDatabase(name string) core.Database {
	switch strings.ToLower(name) {
	case "leveldb":
		return &leveldb.LevelDbDatabase{}
	default:
		return &bbolt.BBoltDatabase{}
	}
}

func NewDatabaseInit(name string, filePath string) (core.Database, error) {
	db := NewDatabase(name)
	if err := db.Init(filePath); err != nil {
		return nil, err
	}

	return db, nil
}
