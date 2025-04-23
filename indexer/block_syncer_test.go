package indexer

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

type BlockTxsRetrieverMock struct {
	RetrieveFn func(blockHeader ledger.BlockHeader) ([]ledger.Transaction, error)
}

func (bt *BlockTxsRetrieverMock) GetBlockTransactions(blockHeader ledger.BlockHeader) ([]ledger.Transaction, error) {
	return bt.RetrieveFn(blockHeader)
}

type BlockSyncerHandlerMock struct {
	BlockPoint         *BlockPoint
	RollForwardFn      func(ledger.BlockHeader, BlockTxsRetriever) error
	RollBackwardFuncFn func(common.Point) error
}

func NewBlockSyncerHandlerMock(slot uint64, hash string) *BlockSyncerHandlerMock {
	bn := uint64(0)
	if hash == ExistingPointHashStr {
		bn = ExistingPointBlockNum
	}

	return &BlockSyncerHandlerMock{
		BlockPoint: &BlockPoint{
			BlockSlot:   slot,
			BlockHash:   NewHashFromHexString(hash),
			BlockNumber: bn,
		},
	}
}

func (hMock *BlockSyncerHandlerMock) RollBackwardFunc(point common.Point) error {
	if hMock.RollBackwardFuncFn != nil {
		return hMock.RollBackwardFuncFn(point)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) RollForwardFunc(
	blockHeader ledger.BlockHeader, txsRetriever BlockTxsRetriever,
) error {
	if hMock.RollForwardFn != nil {
		return hMock.RollForwardFn(blockHeader, txsRetriever)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) Reset() (BlockPoint, error) {
	if hMock.BlockPoint == nil {
		return BlockPoint{}, errors.New("error sync block point")
	}

	return *hMock.BlockPoint, nil
}

const (
	NodeAddress             = "preprod-node.play.dev.cardano.org:3001"
	NetworkMagic            = 1
	ExistingPointSlot       = uint64(2607239)
	ExistingPointHashStr    = "34c36a9eb7228ca529e91babcf2215be29ce2a65b609540b483abc4520848d19"
	NonExistingPointSlot    = uint64(2607240)
	NonExistingPointHashStr = "34c36a9eb7228ca529e91babcf2215be29ce2a65b609540b483abc4520848d20"
	ExistingPointBlockNum   = 125819
)

func TestNewBlockSyncer(t *testing.T) {
	t.Parallel()

	var logger hclog.Logger

	syncer := NewBlockSyncer(&BlockSyncerConfig{}, &BlockSyncerHandlerMock{}, logger)
	require.NotNil(t, syncer)
}

func TestSyncWrongMagic(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: 71,
		NodeAddress:  NodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncWrongNodeAddress(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "test",
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncWrongUnixNodeAddress(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "/" + NodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncNonExistingSlot(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(NonExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "/" + NodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncNonExistingHash(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, NonExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "/" + NodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	require.NotNil(t, syncer.Sync())
}

func TestSyncZeroSlot(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(0, "")

	defer syncer.Close()

	require.Nil(t, syncer.Sync())
}

func TestSync(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)

	defer syncer.Close()

	require.Nil(t, syncer.Sync())
}

func TestSyncWithExistingConnection(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)

	defer syncer.Close()

	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
	)
	require.NoError(t, err)

	require.NoError(t, connection.Dial(ProtocolTCP, NodeAddress))

	syncer.connection = connection

	require.Nil(t, syncer.Sync())
}

func TestCloseWithConnectionNil(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)

	require.Nil(t, syncer.Close())
}

func TestCloseWithConnectionNotNil(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)

	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
	)
	require.NoError(t, err)

	require.NoError(t, connection.Dial(ProtocolTCP, NodeAddress))

	syncer.connection = connection

	require.Nil(t, syncer.Close())
}

func TestSyncRollForwardCalled(t *testing.T) {
	t.Parallel()

	called := uint64(1)
	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	mockSyncerBlockHandler.RollForwardFn = func(bh ledger.BlockHeader, txsRetriever BlockTxsRetriever) error {
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

func TestSync_ConnectionIsClosed(t *testing.T) {
	t.Parallel()

	syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)
	syncer.Close()

	require.NoError(t, syncer.syncExecute())
	require.Nil(t, syncer.connection)

	require.NoError(t, syncer.Sync())
	require.Nil(t, syncer.connection)
}

func TestSync_errorHandler(t *testing.T) {
	t.Parallel()

	const Good = 0x9689

	t.Run("syncer closed", func(t *testing.T) {
		t.Parallel()

		errCh := make(chan error, 1)
		waitCh := make(chan int, 1)
		syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)
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
		syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)
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
		syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)

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
		syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)
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
		syncer := getTestSyncer(ExistingPointSlot, ExistingPointHashStr)
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
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
		RestartDelay: time.Millisecond * 10,
	}, NewBlockSyncerHandlerMock(pointSlot, pointHash), hclog.NewNullLogger())
}
