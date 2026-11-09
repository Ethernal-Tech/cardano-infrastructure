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
// Only a start the node rejects (IntersectNotFound) may now move the point back, and never to the start point.
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

	// ...which the node rejects (rolled back while the service was down)
	restarted.IntersectNotFound(point)

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

	// the tip was orphaned in the meantime, the node rejects it and the syncer retries
	bi.IntersectNotFound(point)

	point, err = bi.Reset()
	require.NoError(t, err)
	require.NotEqual(t, deepRollbackStartPoint, point)
	require.Equal(t, blockPoint(1), point)
}

// Failed starts the node did not reject (dial, handshake, TLS, dropped connection) must not move the
// checkpoint back, no matter how many of them happen in a row
func TestBlockIndexer_DeepRollback_ConnectionFailures_KeepLatestPoint(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 100)

	for i := range 10 {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, latestInDB, point, "start attempt %d", i+1)
	}

	// the same after a restart
	restarted := newDeepRollbackIndexer(t, db)

	for i := range 10 {
		point, err := restarted.Reset()
		require.NoError(t, err)
		require.Equal(t, latestInDB, point, "start attempt %d after restart", i+1)
	}
}

// With more than 20 confirmed blocks the first rejection resumes from the 20th block before the latest one
func TestBlockIndexer_DeepRollback_Rejected_EnoughBlocks_ResumesFromOlderBlock(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 25)

	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)

	bi.IntersectNotFound(point)

	point, err = bi.Reset()
	require.NoError(t, err)
	require.Equal(t, blockPoint(5), point)
}

// If the block 20 back is unknown to the node too (abandoned fork), every further rejection goes twice as far
// back instead of retrying the same block forever. Connection failures in between do not go further back
func TestBlockIndexer_DeepRollback_RepeatedRejections_GoFurtherBack(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 100)

	expected := []indexer.BlockPoint{
		latestInDB,             // the latest one
		blockPoint(80),         // 20 back
		blockPoint(60),         // 40 back
		blockPoint(20),         // 80 back
		blockPoint(1),          // 160 back -> the oldest one
		deepRollbackStartPoint, // older than every stored block
		deepRollbackStartPoint, // nothing older
	}

	for i, exp := range expected {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, exp, point, "start attempt %d", i+1)

		// a connection failure retries the same point
		point, err = bi.Reset()
		require.NoError(t, err)
		require.Equal(t, exp, point, "retry of start attempt %d", i+1)

		bi.IntersectNotFound(point)
	}

	// a received block ends the rejections: the next reset uses the latest point again
	require.NoError(t, bi.RollBackward(deepRollbackStartPoint))

	latestInDB = rollForwardBlocks(t, bi, 100)

	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)
}

// Blocks newer than the checkpoint stay in the database after a deep rollback (here 2160 of them, the maximum
// look-back). They must not hide the older confirmed blocks: every rejection has to go further back than the
// checkpoint, down to the oldest block, instead of returning the rejected checkpoint again
func TestBlockIndexer_DeepRollback_RetainedNewerBlocks_DoNotHideOlderBlocks(t *testing.T) {
	t.Parallel()

	const (
		checkpointSlot = 100
		lastSlot       = checkpointSlot + 2160
	)

	slotPoint := func(slot uint64) indexer.BlockPoint {
		return indexer.BlockPoint{BlockSlot: slot, BlockHash: indexer.Hash{0xCC, byte(slot >> 8), byte(slot)}}
	}

	db := newDeepRollbackDB(t)
	dbTx := db.OpenTx()

	for slot := uint64(1); slot <= lastSlot; slot++ {
		point := slotPoint(slot)
		dbTx.AddConfirmedBlock(&indexer.CardanoBlock{Slot: point.BlockSlot, Hash: point.BlockHash})
	}

	checkpoint := slotPoint(checkpointSlot)
	require.NoError(t, dbTx.SetLatestBlockPoint(&checkpoint).Execute())

	bi := newDeepRollbackIndexer(t, db)
	expected := []indexer.BlockPoint{
		checkpoint,
		slotPoint(80), // 20 back
		slotPoint(60), // 40 back
		slotPoint(20), // 80 back
		slotPoint(1),  // 160 back -> the oldest one
		slotPoint(1),
	}

	for i, exp := range expected {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, exp, point, "start attempt %d", i+1)

		bi.IntersectNotFound(point)
	}
}

// A young database whose blocks were all abandoned by the network (e.g. a few tip blocks indexed without
// confirmations, then orphaned): once every stored block is rejected, syncing must fall back to the starting
// point, which is older than all of them, instead of retrying the oldest rejected block forever
func TestBlockIndexer_DeepRollback_AllStoredBlocksRejected_FallsBackToStartingPoint(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 3)

	expected := []indexer.BlockPoint{
		latestInDB,             // rejected
		blockPoint(1),          // the oldest stored block, rejected
		deepRollbackStartPoint, // older than every stored block
	}

	for i, exp := range expected {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, exp, point, "start attempt %d", i+1)

		bi.IntersectNotFound(point)
	}

	// the starting point is rejected too (wrong config or the node is behind it): it is retried, the indexer
	// never goes back to a newer rejected block
	for i := range 3 {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, deepRollbackStartPoint, point, "retry %d", i+1)

		bi.IntersectNotFound(point)
	}
}

// A single stored block which the node rejects: the starting point is the only older point
func TestBlockIndexer_DeepRollback_SingleBlockRejected_FallsBackToStartingPoint(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 1)

	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)

	bi.IntersectNotFound(point)

	point, err = bi.Reset()
	require.NoError(t, err)
	require.Equal(t, deepRollbackStartPoint, point)
}

// When the node accepts an older point it rolls back to it first. That point must be stored, so a connection
// that drops before the next block does not make the indexer start again from the rejected database point
func TestBlockIndexer_DeepRollback_FoundIntersection_IsStored(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 30)

	point, err := bi.Reset()
	require.NoError(t, err)
	require.Equal(t, latestInDB, point)

	bi.IntersectNotFound(point)

	point, err = bi.Reset()
	require.NoError(t, err)
	require.Equal(t, blockPoint(10), point)

	// the node found the intersection and rolls back to it, then the connection drops
	require.NoError(t, bi.RollBackward(point))

	dbPoint, err := db.GetLatestBlockPoint()
	require.NoError(t, err)
	require.Equal(t, blockPoint(10), *dbPoint)

	point, err = bi.Reset()
	require.NoError(t, err)
	require.Equal(t, blockPoint(10), point)
}

// A node that is behind rejects every recent point although there is no fork. Rollbacks are never deeper than
// 2160 blocks, so the indexer must stop going back there and retry, not re-index from the starting point
func TestBlockIndexer_DeepRollback_LaggingNode_StopsAtRollbackLimit(t *testing.T) {
	t.Parallel()

	db := newDeepRollbackDB(t)
	bi := newDeepRollbackIndexer(t, db)
	latestInDB := startIndexing(t, bi, 2500)

	expected := []indexer.BlockPoint{
		latestInDB,
		blockPoint(2480), // 20 back
		blockPoint(2460), // 40 back
		blockPoint(2420), // 80 back
		blockPoint(2340), // 160 back
		blockPoint(2180), // 320 back
		blockPoint(1860), // 640 back
		blockPoint(1220), // 1280 back
		blockPoint(340),  // 2160 back, the limit
		blockPoint(340),
		blockPoint(340),
	}

	for i, exp := range expected {
		point, err := bi.Reset()
		require.NoError(t, err)
		require.Equal(t, exp, point, "start attempt %d", i+1)

		bi.IntersectNotFound(point)
	}
}
