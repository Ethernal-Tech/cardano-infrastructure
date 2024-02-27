package indexerdb

import (
	"strings"

	core "github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	boltdb "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/boltdb"
	leveldb "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/leveldb"
)

func NewDatabase(name string) core.Database {
	switch strings.ToLower(name) {
	case "leveldb":
		return &leveldb.LevelDbDatabase{}
	default:
		return &boltdb.BoltDatabase{}
	}
}

func NewDatabaseInit(name string, filePath string) (core.Database, error) {
	db := NewDatabase(name)
	if err := db.Init(filePath); err != nil {
		return nil, err
	}

	return db, nil
}
