package indexer

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

type BlockSyncerHandlerMock struct {
	BlockPoint         *BlockPoint
	RollForwardFn      func(blockHeader ledger.BlockHeader, getTxsFunc GetTxsFunc, tip chainsync.Tip) error
	RollBackwardFuncFn func(point common.Point, tip chainsync.Tip) error
}

func NewBlockSyncerHandlerMock(slot uint64, hash string) *BlockSyncerHandlerMock {
	bn := uint64(0)
	hashBytes, _ := hex.DecodeString(hash)

	if hash == ExistingPointHashStr {
		bn = ExistingPointBlockNum
	}

	return &BlockSyncerHandlerMock{
		BlockPoint: &BlockPoint{
			BlockSlot:   slot,
			BlockHash:   hashBytes,
			BlockNumber: bn,
		},
	}
}

func (hMock *BlockSyncerHandlerMock) RollBackwardFunc(point common.Point, tip chainsync.Tip) error {
	if hMock.RollBackwardFuncFn != nil {
		return hMock.RollBackwardFuncFn(point, tip)
	}

	return nil
}

func (hMock *BlockSyncerHandlerMock) RollForwardFunc(blockHeader ledger.BlockHeader, getTxsFunc GetTxsFunc, tip chainsync.Tip) error {
	if hMock.RollForwardFn != nil {
		return hMock.RollForwardFn(blockHeader, getTxsFunc, tip)
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
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.NotNil(t, err)
}

func TestSyncWrongNodeAddress(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "test",
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.NotNil(t, err)
}

func TestSyncWrongUnixNodeAddress(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "/" + NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.NotNil(t, err)
}

func TestSyncNonExistingSlot(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(NonExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "/" + NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.NotNil(t, err)
}

func TestSyncNonExistingHash(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, NonExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  "/" + NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.NotNil(t, err)
}

func TestSyncEmptyHash(t *testing.T) {
	t.Parallel()

	var emptyHash []byte

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, string(emptyHash))
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.Nil(t, err)
}

func TestSync(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	err := syncer.Sync()
	require.Nil(t, err)
}

func TestSyncWithExistingConnection(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
	)
	require.NoError(t, err)

	require.NoError(t, connection.Dial(ProtocolTCP, NodeAddress))

	syncer.connection = connection
	err = syncer.Sync()
	require.Nil(t, err)
}

func TestCloseWithConnectionNil(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	err := syncer.Close()
	require.Nil(t, err)
}

func TestCloseWithConnectionNotNil(t *testing.T) {
	t.Parallel()

	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	connection, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(NetworkMagic),
		ouroboros.WithNodeToNode(true),
		ouroboros.WithKeepAlive(true),
	)
	require.NoError(t, err)

	syncer.connection = connection

	err = syncer.Close()
	require.Nil(t, err)
}

func TestSyncRollForwardCalled(t *testing.T) {
	t.Parallel()

	called := false
	mockSyncerBlockHandler := NewBlockSyncerHandlerMock(ExistingPointSlot, ExistingPointHashStr)
	syncer := NewBlockSyncer(&BlockSyncerConfig{
		NetworkMagic: NetworkMagic,
		NodeAddress:  NodeAddress,
	}, mockSyncerBlockHandler, hclog.NewNullLogger())

	defer syncer.Close()

	mockSyncerBlockHandler.RollForwardFn = func(blockHeader ledger.BlockHeader, getTxsFunc GetTxsFunc, tip chainsync.Tip) error {
		t.Helper()

		_, err := getTxsFunc()
		require.True(t, err == nil || strings.Contains(err.Error(), "protocol is shutting down"))

		called = true

		return nil
	}

	err := syncer.Sync()
	require.Nil(t, err)

	time.Sleep(5 * time.Second)
	require.True(t, called)
}
