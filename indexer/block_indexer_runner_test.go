package indexer

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestBlockIndexerRunner_CloseTerminates(t *testing.T) {
	handlerMock := NewBlockSyncerHandlerMock(1000, "ff")
	config := &BlockIndexerRunnerConfig{QueueChannelSize: 2}
	runner := NewBlockIndexerRunner(handlerMock, config, hclog.NewNullLogger())

	_, err := runner.Reset()
	require.NoError(t, err)

	<-time.After(time.Millisecond * 100)

	require.NoError(t, runner.Close())

	select {
	case <-runner.loopFinishedCh:
	case <-time.After(time.Millisecond * 200):
		t.Fatalf("timeout")
	}
}

func TestBlockIndexerRunner_runMainLoop(t *testing.T) {
	forward, backward, tries := int32(0), int32(0), int32(0)
	handlerMock := &BlockSyncerHandlerMock{
		RollForwardFn: func(_ BlockHeader, _ BlockTxsRetriever) error {
			newValue := atomic.AddInt32(&forward, 1)

			if newValue == 2 && atomic.AddInt32(&tries, 1) < 3 {
				atomic.AddInt32(&forward, -1)
				// return error if second item is called first two times
				return &processConfirmedBlockError{err: errors.New("dummy")}
			}

			return nil
		}, RollBackwardFuncFn: func(bp BlockPoint) error {
			newValue := atomic.AddInt32(&backward, 1)
			if newValue == 4 {
				return ErrBlockIndexerFatal
			}

			return nil
		},
	}
	config := &BlockIndexerRunnerConfig{QueueChannelSize: 2}
	runner := NewBlockIndexerRunner(handlerMock, config, hclog.NewNullLogger())
	runner.loopFinishedCh = make(chan struct{})
	ch := make(chan bool)

	runner.runMainLoop()

	go func() {
		<-runner.ErrorCh()
		ch <- true
	}()

	go func() {
		_ = runner.RollBackward(BlockPoint{BlockSlot: 1})
		_ = runner.RollBackward(BlockPoint{BlockSlot: 2})
		_ = runner.RollForward(BlockHeader{}, &BlockTxsRetrieverMock{})
		_ = runner.RollBackward(BlockPoint{BlockSlot: 3})
		_ = runner.RollForward(BlockHeader{}, &BlockTxsRetrieverMock{})
		_ = runner.RollForward(BlockHeader{}, &BlockTxsRetrieverMock{})
		_ = runner.RollBackward(BlockPoint{BlockSlot: 4})
	}()

	select {
	case <-ch:
	case <-time.After(time.Second * 2):
		t.Fatalf("timeout")
	}

	require.Equal(t, int32(3), forward)
	require.Equal(t, int32(4), backward)
	require.Equal(t, int32(3), tries)
}

func TestBlockIndexerRunner_Reset(t *testing.T) {
	lastNum := uint64(0)
	handlerMock := &BlockSyncerHandlerMock{
		ResetFn: func() (BlockPoint, error) {
			return BlockPoint{BlockSlot: atomic.LoadUint64(&lastNum)}, nil
		},
		RollForwardFn: func(bh BlockHeader, _ BlockTxsRetriever) error {
			if !atomic.CompareAndSwapUint64(&lastNum, bh.Number-1, bh.Number) {
				t.Fatalf("invalid block number")
			}

			return nil
		}, RollBackwardFuncFn: func(_ BlockPoint) error {
			return nil
		},
	}
	config := &BlockIndexerRunnerConfig{QueueChannelSize: 2000}
	runner := NewBlockIndexerRunner(handlerMock, config, hclog.NewNullLogger())

	_, _ = runner.Reset()

	go func() {
		<-time.After(time.Millisecond * 100)

		bp, err := runner.Reset()

		require.NoError(t, err)
		require.Greater(t, bp.BlockSlot, uint64(0))

		<-time.After(time.Millisecond * 100)

		runner.Close()
	}()

	go func() {
		for i := 1; i < 10000; i++ {
			_ = runner.RollForward(BlockHeader{Number: uint64(i)}, nil)
		}
	}()

	select {
	case <-runner.closeCh:
		require.Greater(t, lastNum, uint64(10))
	case <-time.After(time.Second * 2):
		t.Fatalf("timeout")
	}
}

func TestBlockIndexerRunner_Execute(t *testing.T) {
	defaultErr := errors.New("error")
	handlerMock := &BlockSyncerHandlerMock{}
	handlerMock.RollForwardFn = func(bh BlockHeader, btr BlockTxsRetriever) error {
		switch bh.Slot {
		case 1:
			return &processConfirmedBlockError{err: defaultErr}
		case 2:
			return defaultErr
		default:
			return nil
		}
	}
	runner := NewBlockIndexerRunner(handlerMock, &BlockIndexerRunnerConfig{}, hclog.NewNullLogger())

	t.Run("should break loop on stop and return true if processConfirmedBlockErr", func(t *testing.T) {
		stopLoopCh := make(chan struct{})
		close(stopLoopCh)

		require.True(t, runner.execute(blockIndexerRunnerQueueItem{BlockHeader: &BlockHeader{Slot: 1}}, stopLoopCh))
	})

	t.Run("should break loop and return false if normal error", func(t *testing.T) {
		require.False(t, runner.execute(blockIndexerRunnerQueueItem{BlockHeader: &BlockHeader{Slot: 2}}, nil))
	})
}
