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
type resetRecorder struct {
	*indexer.BlockIndexer

	lock   sync.Mutex
	points []indexer.BlockPoint
}

func (r *resetRecorder) Reset() (indexer.BlockPoint, error) {
	point, err := r.BlockIndexer.Reset()

	r.lock.Lock()
	r.points = append(r.points, point)
	r.lock.Unlock()

	return point, err
}

func (r *resetRecorder) getPoints() []indexer.BlockPoint {
	r.lock.Lock()
	defer r.lock.Unlock()

	return append([]indexer.BlockPoint(nil), r.points...)
}

// Regression test against a real node (preprod) for the apex-web restart: the latest block in the database was
// rolled back by the network while the service was down, so the node does not know it. The first start attempt
// fails with "intersect not found". The second one must resume from an older confirmed block, not from
// config.StartingBlockPoint (which re-indexed everything after it when there were 20 or fewer confirmed blocks)
func TestSyncer_DeepRollback_Restart_ResumesFromOlderConfirmedBlock(t *testing.T) {
	t.Parallel()

	db := &indexerbbolt.BBoltDatabase{}
	require.NoError(t, db.Init(filepath.Join(t.TempDir(), "cardano.db")))

	defer db.Close()

	// the database from before the restart: a few real preprod blocks after the starting point and
	// the latest one which is unknown to the node
	confirmedBlocks := []indexer.BlockPoint{
		{BlockSlot: 2607260, BlockHash: indexer.NewHashFromHexString(
			"e1e889232db6afb27ba12c335eee9cb72d810b893fd90b5f39d8140cfe901eb6")},
		{BlockSlot: 2607282, BlockHash: indexer.NewHashFromHexString(
			"51e8550ef7e986696ac9ab0797107e354575a5467d6483e4cff6177fd10ee8dd")},
		{BlockSlot: 2607285, BlockHash: indexer.NewHashFromHexString(
			"d62046ec9e8cb3fc6f4fe943d76c7dbf422d181ba7d5c2e5127eca0d083bb362")},
		{BlockSlot: 2607291, BlockHash: indexer.NewHashFromHexString(
			"d3c5a98b514de04b51d9776f5bcdf0b51b79ba9d6c0b09fe4d2b3b000ee678c8")},
	}
	orphanedLatest := indexer.BlockPoint{
		BlockSlot: 2607300,
		BlockHash: indexer.NewHashFromHexString(nonExistingPointHashStr),
	}
	dbTx := db.OpenTx()

	for _, point := range append(confirmedBlocks, orphanedLatest) {
		dbTx.AddConfirmedBlock(&indexer.CardanoBlock{Slot: point.BlockSlot, Hash: point.BlockHash})
	}

	require.NoError(t, dbTx.SetLatestBlockPoint(&orphanedLatest).Execute())

	// same indexer config as apex-web cardano-api, startSlot/startBlockHash -> an existing preprod block
	startPoint := indexer.BlockPoint{
		BlockSlot: existingPointSlot,
		BlockHash: indexer.NewHashFromHexString(existingPointHashStr),
	}
	handler := &resetRecorder{
		BlockIndexer: indexer.NewBlockIndexer(&indexer.BlockIndexerConfig{
			StartingBlockPoint:     &startPoint,
			ConfirmationBlockCount: 0,
			AddressCheck:           indexer.AddressCheckOutputs,
			AddressesOfInterest:    []string{"addr_test_of_interest"},
			AllowDeepRollback:      true,
		}, func(*indexer.CardanoBlock, []*indexer.Tx) error { return nil }, db, hclog.NewNullLogger()),
	}

	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic:   networkMagic,
		NodeAddress:    nodeAddress,
		RestartDelay:   time.Millisecond * 10,
		SyncStartTries: 4,
	}, handler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NoError(t, syncer.Sync())

	// wait until the indexer stores a block it got from the node after the sync started
	require.Eventually(t, func() bool {
		p, err := db.GetLatestBlockPoint()

		return err == nil && p != nil && p.BlockSlot > confirmedBlocks[0].BlockSlot && *p != orphanedLatest
	}, 30*time.Second, 100*time.Millisecond, "the indexer did not process any block from the node")

	points := handler.getPoints()

	t.Logf("points returned by Reset (one per start attempt): %+v", points)

	require.Equal(t, []indexer.BlockPoint{orphanedLatest, confirmedBlocks[0]}, points,
		"the database point first, then the oldest confirmed block - never the starting point")
}
