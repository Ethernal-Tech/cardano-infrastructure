package db

import (
	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	indexerbbolt "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/bbolt"
)

func NewDatabaseInit(name string, filePath string) (indexer.Database, error) {
	// currently name is not used because only bbolt is supported
	db := &indexerbbolt.BBoltDatabase{}
	if err := db.Init(filePath); err != nil {
		return nil, err
	}

	return db, nil
}
