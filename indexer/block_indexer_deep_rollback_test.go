package indexer_test

import (
	"path/filepath"
	"testing"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	indexerbbolt "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/bbolt"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

// Regression tests: with AllowDeepRollback, a second Reset without any block in between (the syncer calls Reset
// on every start attempt, see gouroboros.BlockSyncerImpl.syncExecute) discarded the latest point from the
// database and resumed from config.StartingBlockPoint when the database held 20 or fewer confirmed blocks.
// The setup mirrors apex-web cardano-api: no confirmations, deep rollbacks allowed, real bbolt database.

var deepRollbackStartPoint = indexer.BlockPoint{BlockSlot: 1_000, BlockHash: indexer.Hash{0xAA, 0x01}}

func newDeepRollbackIndexer(t *testing.T, db indexer.Database) *indexer.BlockIndexer {
	t.Helper()

	startPoint := deepRollbackStartPoint

	return indexer.NewBlockIndexer(&indexer.BlockIndexerConfig{
		StartingBlockPoint:     &startPoint,
		ConfirmationBlockCount: 0,
		AddressCheck:           indexer.AddressCheckOutputs,
		AddressesOfInterest:    []string{"addr_test_of_interest"},
		AllowDeepRollback:      true,
	}, func(*indexer.CardanoBlock, []*indexer.Tx) error { return nil }, db, hclog.NewNullLogger())
}

func newDeepRollbackDB(t *testing.T) *indexerbbolt.BBoltDatabase {
	t.Helper()

	db := &indexerbbolt.BBoltDatabase{}
	require.NoError(t, db.Init(filepath.Join(t.TempDir(), "cardano.db")))
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// blockPoint returns the point of the i-th block fed by rollForwardBlocks
func blockPoint(i int) indexer.BlockPoint {
	return indexer.BlockPoint{
		BlockSlot: deepRollbackStartPoint.BlockSlot + uint64(i)*20,
		BlockHash: indexer.Hash{0xBB, byte(i)},
	}
}

// rollForwardBlocks feeds cnt blocks after the starting point, as the node would after a successful sync start
func rollForwardBlocks(t *testing.T, bi *indexer.BlockIndexer, cnt int) indexer.BlockPoint {
	t.Helper()

	retriever := &indexer.BlockTxsRetrieverMock{
		RetrieveFn: func(indexer.BlockHeader) ([]*indexer.Tx, error) { return nil, nil },
	}

	for i := 1; i <= cnt; i++ {
		point := blockPoint(i)
		require.NoError(t, bi.RollForward(indexer.BlockHeader{
			Slot: point.BlockSlot, Hash: point.BlockHash, Number: uint64(i),
		}, retriever))
	}

	return blockPoint(cnt)
}

func startIndexing(t *testing.T, bi *indexer.BlockIndexer, cnt int) indexer.BlockPoint {
	t.Helper()

	_, err := bi.Reset()
	require.NoError(t, err)
	require.NoError(t, bi.RollBackward(deepRollbackStartPoint)) // intersect found

	return rollForwardBlocks(t, bi, cnt)
}

func TestBlockIndexer_DeepRollback_Restart_DoesNotResumeFromStartingPoint(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)

	// the service ran for a while and indexed 10 blocks
	latestInDB := startIndexing(t, newDeepRollbackIndexer(t, db), 10)

	dbPoint, err := db.GetLatestBlockPoint()
	require.NoError(t, err)
	require.Equal(t, latestInDB, *dbPoint)

	// service restart: a fresh indexer on the same database
	restarted := newDeepRollbackIndexer(t, db)

	// start attempt 1 uses the latest point from the database...
	point, err := restarted.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)

	// ...and fails before any block or rollback arrives (intersect not found, connection dropped, ...)
	// start attempt 2 uses the oldest confirmed block, as there are fewer than 20 before the latest one
	point, err = restarted.Reset()
	require.NoError(t, err)
	require.NotEqual(t, deepRollbackStartPoint, point)
	require.Equal(t, blockPoint(1), point)
}

func TestBlockIndexer_DeepRollback_Reconnect_DoesNotResumeFromStartingPoint(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 10)

	// the connection drops while the indexer is running: the syncer reconnects and calls Reset
	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)

	// the reconnect attempt fails before any block or rollback arrives, the syncer retries
	point, err = bi.Reset()
	require.NoError(t, err)
	require.NotEqual(t, deepRollbackStartPoint, point)
	require.Equal(t, blockPoint(1), point)
}

// With only the latest block confirmed there is nothing older: the database point is kept
func TestBlockIndexer_DeepRollback_Reconnect_SingleBlock_KeepsLatestPoint(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 1)

	for range 3 {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, latestInDB, point)
	}
}

// With more than 20 confirmed blocks the first retry resumes from the 20th block before the latest one
func TestBlockIndexer_DeepRollback_Reconnect_EnoughBlocks_ResumesFromOlderBlock(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 25)

	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)

	point, err = bi.Reset()
	require.NoError(t, err)
	require.Equal(t, blockPoint(5), point)
}

// If the block 20 back is unknown to the node (abandoned fork), every further failed start goes twice as far
// back instead of retrying the same block forever
func TestBlockIndexer_DeepRollback_RepeatedFailedStarts_GoFurtherBack(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 100)

	expected := []indexer.BlockPoint{
		latestInDB,     // the latest one
		blockPoint(80), // 20 back
		blockPoint(60), // 40 back
		blockPoint(20), // 80 back
		blockPoint(1),  // 160 back -> the oldest one
		blockPoint(1),
	}

	for i, exp := range expected {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, exp, point, "start attempt %d", i+1)
	}

	// a received block ends the failed starts: the next reset uses the latest point again
	require.NoError(t, bi.RollBackward(blockPoint(1)))

	latestInDB = rollForwardBlocks(t, bi, 100)

	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)
}
