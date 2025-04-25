package gouroboros

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ethernal-Tech/cardano-infrastructure/indexer"
	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger/byron"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

const (
	nodeAddress             = "preprod-node.play.dev.cardano.org:3001"
	networkMagic            = 1
	existingPointSlot       = uint64(2607239)
	existingPointHashStr    = "34c36a9eb7228ca529e91babcf2215be29ce2a65b609540b483abc4520848d19"
	nonExistingPointSlot    = uint64(2607240)
	nonExistingPointHashStr = "34c36a9eb7228ca529e91babcf2215be29ce2a65b609540b483abc4520848d20"
)

func TestNewBlockSyncer(t *testing.T) {
	t.Parallel()

	var logger hclog.Logger

	syncer := NewBlockSyncer(&BlockSyncerConfig{}, &BlockSyncerHandlerMock{}, logger)
	require.NotNil(t, syncer)
}

func TestSyncer_Sync_WrongMagic(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(existingPointSlot, existingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: 71,
		NodeAddress:  nodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncer_Sync_WrongNodeAddress(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(existingPointSlot, existingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  "test",
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncer_Sync_WrongUnixNodeAddress(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(existingPointSlot, existingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  "/" + nodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncer_Sync_NonExistingSlot(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(nonExistingPointSlot, existingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  "/" + nodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncer_Sync_NonExistingHash(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(existingPointSlot, nonExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  "/" + nodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncer_Sync_ZeroSlot(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(0, "")

	defer syncer.Close()

	require.Nil(t, syncer.Sync())
}

func TestSyncer_Sync_Valid(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(existingPointSlot, existingPointHashStr)

	defer syncer.Close()

	require.Nil(t, syncer.Sync())
}

func TestSyncer_Sync_ExistingConnection(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(existingPointSlot, existingPointHashStr)

	defer syncer.Close()

	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(networkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
	)
	require.NoError(t, err)

	require.NoError(t, connection.Dial(ProtocolTCP, nodeAddress))

	syncer.connection = connection

	require.Nil(t, syncer.Sync())
}

func TestSyncer_Close_ConnectionNil(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(existingPointSlot, existingPointHashStr)

	require.Nil(t, syncer.Close())
}

func TestSyncer_Close_ConnectionNotNil(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(existingPointSlot, existingPointHashStr)

	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(networkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
	)
	require.NoError(t, err)

	require.NoError(t, connection.Dial(ProtocolTCP, nodeAddress))

	syncer.connection = connection

	require.Nil(t, syncer.Close())
}

func TestSyncer_RollForward_Valid(t *testing.T) {
	t.Parallel()

	called := uint64(1)
	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(existingPointSlot, existingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  nodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	mockSyncerBlockHandler.RollForwardFn = func(bh indexer.BlockHeader, txsRetriever indexer.BlockTxsRetriever) error {
		t.Helper()

		_, err := txsRetriever.GetBlockTransactions(bh)
		require.True(t, err == nil || strings.Contains(err.Error(), "protocol is shutting down"))

		atomic.StoreUint64(&called, 1)

		return nil
	}

	require.Nil(t, syncer.Sync())

	time.Sleep(5 * time.Second)
	require.True(t, atomic.LoadUint64(&called) == uint64(1))
}

func TestSyncer_RollForwardCallback_ConnectionNil(t *testing.T) {
	t.Parallel()

	syncer := NewBlockSyncer(&BlockSyncerConfig{}, nil, hclog.NewNullLogger())

	err := syncer.rollForwardCallback(chainsync.CallbackContext{}, 10, byron.ByronMainBlockHeader{}, chainsync.Tip{})
	require.NotNil(t, err)
}

func TestSyncer_Sync_ConnectionIsClosed(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(existingPointSlot, existingPointHashStr)
	syncer.Close()

	require.NoError(t, syncer.syncExecute())
	require.Nil(t, syncer.connection)

	require.NoError(t, syncer.Sync())
	require.Nil(t, syncer.connection)
}

func TestSyncer_ErrorHandler(t *testing.T) {
	t.Parallel()

	const Good = 0x9689

	t.Run("syncer closed", func(t *testing.T) {
		t.Parallel()

		errCh := make(chan error, 1)
		waitCh := make(chan int, 1)
		syncer := getTestSyncer(existingPointSlot, existingPointHashStr)
		syncer.config.RestartOnError = true

		go func() {
			syncer.errorHandler(errCh)
			waitCh <- Good
		}()

		syncer.Close()

		select {
		case value := <-waitCh:
			require.Equal(t, Good, value)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout")
		}
	})

	t.Run("error channel closed", func(t *testing.T) {
		t.Parallel()

		errCh := make(chan error, 1)
		waitCh := make(chan int, 1)
		syncer := getTestSyncer(existingPointSlot, existingPointHashStr)
		syncer.config.RestartOnError = true

		go func() {
			syncer.errorHandler(errCh)
			waitCh <- Good
		}()

		close(errCh)

		select {
		case value := <-waitCh:
			require.Equal(t, Good, value)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout")
		}
	})

	t.Run("error non fatal RestartOnError false", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}
		testErr := errors.New("test error")
		errCh := make(chan error, 1)
		isOk := false
		syncer := getTestSyncer(existingPointSlot, existingPointHashStr)

		wg.Add(2)

		go func() {
			defer wg.Done()

			syncer.errorHandler(errCh)
		}()

		go func() {
			defer wg.Done()

			v := <-syncer.ErrorCh()
			isOk = errors.Is(v, testErr)
		}()

		errCh <- testErr

		wg.Wait()

		require.True(t, isOk)
	})

	t.Run("error non fatal RestartOnError true - try sync again", func(t *testing.T) {
		t.Parallel()

		testErr := errors.New("test error")
		wg := sync.WaitGroup{}
		errCh := make(chan error, 1)
		isOk := false
		syncer := getTestSyncer(existingPointSlot, existingPointHashStr)
		syncer.config.RestartOnError = true
		syncer.config.NodeAddress = "invalid node address"

		wg.Add(2)

		go func() {
			defer wg.Done()

			syncer.errorHandler(errCh)
		}()

		go func() {
			defer wg.Done()

			v := <-syncer.ErrorCh()
			isOk = v != nil && strings.Contains(v.Error(), "missing port")
		}()

		errCh <- testErr

		wg.Wait()

		require.True(t, isOk)
	})

	t.Run("close during re-sync", func(t *testing.T) {
		t.Parallel()

		testErr := errors.New("test error")
		wg := sync.WaitGroup{}
		waitCh := make(chan struct{}, 1)
		errCh := make(chan error, 1)
		syncer := getTestSyncer(existingPointSlot, existingPointHashStr)
		syncer.config.RestartOnError = true
		syncer.config.RestartDelay = time.Second * 100

		wg.Add(2)

		go func() {
			defer wg.Done()

			syncer.errorHandler(errCh)
		}()

		go func() {
			defer wg.Done()

			time.Sleep(1 * time.Second)

			syncer.Close()
		}()

		go func() {
			<-syncer.ErrorCh()
			waitCh <- struct{}{}
		}()

		errCh <- testErr

		wg.Wait()

		select {
		case <-waitCh:
			t.Fatalf("timeout expected")
		case <-time.After(4 * time.Second):
		}
	})
}

func getTestSyncer(pointSlot uint64, pointHash string) *BlockSyncerImpl {
	return NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: networkMagic,
		NodeAddress:  nodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, NewBlockSyncerHandlerMock(pointSlot, pointHash), hclog.NewNullLogger())
}

type BlockSyncerHandlerMock struct {
	BlockPoint         *indexer.BlockPoint
	RollForwardFn      func(indexer.BlockHeader, indexer.BlockTxsRetriever) error
	RollBackwardFuncFn func(indexer.BlockPoint) error
}

var _ indexer.BlockSyncerHandler = (*BlockSyncerHandlerMock)(nil)

func NewBlockSyncerHandlerMock(slot uint64, hash string) *BlockSyncerHandlerMock {
	return &BlockSyncerHandlerMock{
		BlockPoint: &indexer.BlockPoint{
			BlockSlot: slot,
			BlockHash: indexer.NewHashFromHexString(hash),
		},
	}
}

func (hMock *BlockSyncerHandlerMock) RollBackward(point indexer.BlockPoint) error {
	if hMock.RollBackwardFuncFn != nil {
		return hMock.RollBackwardFuncFn(point)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) RollForward(
	blockHeader indexer.BlockHeader, txsRetriever indexer.BlockTxsRetriever,
) error {
	if hMock.RollForwardFn != nil {
		return hMock.RollForwardFn(blockHeader, txsRetriever)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) Reset() (indexer.BlockPoint, error) {
	if hMock.BlockPoint == nil {
		return indexer.BlockPoint{}, errors.New("error sync block point")
	}

	return *hMock.BlockPoint, nil
}
