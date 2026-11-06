package gouroboros

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	indexerbbolt "github.com/Ethernal-Tech/cardano-infrastructure/indexer/db/bbolt"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

// resetRecorder records every point the block indexer returns from Reset, i.e. every point the syncer
// asks the node to intersect with
// and every point the syncer reports as rejected by the node
type resetRecorder struct {
	*indexer.BlockIndexer

	// called after Reset, before the syncer asks the node to intersect
	afterReset func(attempt int)

	lock     sync.Mutex
	points   []indexer.BlockPoint
	rejected []indexer.BlockPoint
}

func (r *resetRecorder) Reset() (indexer.BlockPoint, error) {
	point, err := r.BlockIndexer.Reset()

	r.lock.Lock()
	r.points = append(r.points, point)
	attempt := len(r.points)
	r.lock.Unlock()

	if r.afterReset != nil {
		r.afterReset(attempt)
	}

	return point, err
}

func (r *resetRecorder) IntersectNotFound(point indexer.BlockPoint) {
	r.lock.Lock()
	r.rejected = append(r.rejected, point)
	r.lock.Unlock()

	r.BlockIndexer.IntersectNotFound(point)
}

func (r *resetRecorder) getPoints() ([]indexer.BlockPoint, []indexer.BlockPoint) {
	r.lock.Lock()
	defer r.lock.Unlock()

	return append([]indexer.BlockPoint(nil), r.points...), append([]indexer.BlockPoint(nil), r.rejected...)
}

// realPreprodBlocks are consecutive preprod blocks after existingPointSlot
var realPreprodBlocks = []indexer.BlockPoint{
	{BlockSlot: 2607260, BlockHash: indexer.NewHashFromHexString(
		"e1e889232db6afb27ba12c335eee9cb72d810b893fd90b5f39d8140cfe901eb6")},
	{BlockSlot: 2607282, BlockHash: indexer.NewHashFromHexString(
		"51e8550ef7e986696ac9ab0797107e354575a5467d6483e4cff6177fd10ee8dd")},
	{BlockSlot: 2607285, BlockHash: indexer.NewHashFromHexString(
		"d62046ec9e8cb3fc6f4fe943d76c7dbf422d181ba7d5c2e5127eca0d083bb362")},
	{BlockSlot: 2607291, BlockHash: indexer.NewHashFromHexString(
		"d3c5a98b514de04b51d9776f5bcdf0b51b79ba9d6c0b09fe4d2b3b000ee678c8")},
}

// newDeepRollbackTestSetup opens a database holding the given confirmed blocks (the last one is the latest
// point) and creates a block indexer with the apex-web cardano-api config around it
func newDeepRollbackTestSetup(
	t *testing.T, blocks []indexer.BlockPoint,
) (*indexerbbolt.BBoltDatabase, *resetRecorder) {
	t.Helper()

	db := &indexerbbolt.BBoltDatabase{}
	require.NoError(t, db.Init(filepath.Join(t.TempDir(), "cardano.db")))
	t.Cleanup(func() { _ = db.Close() })

	dbTx := db.OpenTx()

	for _, point := range blocks {
		dbTx.AddConfirmedBlock(&indexer.CardanoBlock{Slot: point.BlockSlot, Hash: point.BlockHash})
	}

	require.NoError(t, dbTx.SetLatestBlockPoint(&blocks[len(blocks)-1]).Execute())

	// startSlot/startBlockHash -> an existing preprod block
	startPoint := indexer.BlockPoint{
		BlockSlot: existingPointSlot,
		BlockHash: indexer.NewHashFromHexString(existingPointHashStr),
	}

	return db, &resetRecorder{
		BlockIndexer: indexer.NewBlockIndexer(&indexer.BlockIndexerConfig{
			StartingBlockPoint:     &startPoint,
			ConfirmationBlockCount: 0,
			AddressCheck:           indexer.AddressCheckOutputs,
			AddressesOfInterest:    []string{"addr_test_of_interest"},
			AllowDeepRollback:      true,
		}, func(*indexer.CardanoBlock, []*indexer.Tx) error { return nil }, db, hclog.NewNullLogger()),
	}
}

// waitForBlockAfter waits until the indexer stores a block it got from the node after the given slot
func waitForBlockAfter(t *testing.T, db *indexerbbolt.BBoltDatabase, slot uint64) {
	t.Helper()

	require.Eventually(t, func() bool {
		p, err := db.GetLatestBlockPoint()

		return err == nil && p != nil && p.BlockSlot > slot
	}, 30*time.Second, 100*time.Millisecond, "the indexer did not process any block from the node")
}

// Regression test against a real node (preprod) for the apex-web restart: the latest block in the database was
// rolled back by the network while the service was down, so the node does not know it. The first start attempt
// is rejected with "intersect not found". The second one must resume from an older confirmed block, not from
// config.StartingBlockPoint (which re-indexed everything after it when there were 20 or fewer confirmed blocks)
func TestSyncer_DeepRollback_Restart_ResumesFromOlderConfirmedBlock(t *testing.T) {
	t.Parallel()

	orphanedLatest := indexer.BlockPoint{
		BlockSlot: 2607300,
		BlockHash: indexer.NewHashFromHexString(nonExistingPointHashStr),
	}
	db, handler := newDeepRollbackTestSetup(t, append(append([]indexer.BlockPoint{}, realPreprodBlocks...), orphanedLatest))

	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic:   networkMagic,
		NodeAddress:    nodeAddress,
		RestartDelay:   time.Millisecond * 10,
		SyncStartTries: 4,
	}, handler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NoError(t, syncer.Sync())
	waitForBlockAfter(t, db, realPreprodBlocks[0].BlockSlot)

	points, rejected := handler.getPoints()

	t.Logf("points returned by Reset (one per start attempt): %+v", points)
	t.Logf("points rejected by the node: %+v", rejected)

	require.Equal(t, []indexer.BlockPoint{orphanedLatest}, rejected)
	require.Equal(t, []indexer.BlockPoint{orphanedLatest, realPreprodBlocks[0]}, points,
		"the database point first, then the oldest confirmed block - never the starting point")
}

// Start attempts that fail for any other reason than a rejection by the node (here the connection is closed
// after Reset, before the node answers) must retry the latest point from the database, not move back
func TestSyncer_DeepRollback_ConnectionFailure_RetriesLatestPoint(t *testing.T) {
	t.Parallel()

	const failedAttempts = 2

	latest := realPreprodBlocks[len(realPreprodBlocks)-1]
	db, handler := newDeepRollbackTestSetup(t, realPreprodBlocks)

	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic:   networkMagic,
		NodeAddress:    nodeAddress,
		RestartDelay:   time.Millisecond * 10,
		SyncStartTries: failedAttempts + 2,
	}, handler, hclog.NewNullLogger())

	defer syncer.Close()

	// syncExecute holds the syncer lock while calling Reset, so the connection is closed directly
	handler.afterReset = func(attempt int) {
		if attempt <= failedAttempts {
			_ = syncer.connection.Close()
		}
	}

	require.NoError(t, syncer.Sync())
	waitForBlockAfter(t, db, latest.BlockSlot)

	points, rejected := handler.getPoints()

	t.Logf("points returned by Reset (one per start attempt): %+v", points)
	t.Logf("points rejected by the node: %+v", rejected)

	require.Empty(t, rejected, "a closed connection is not a rejection by the node")
	require.Equal(t, []indexer.BlockPoint{latest, latest, latest}, points,
		"every attempt must use the latest point from the database")
}
