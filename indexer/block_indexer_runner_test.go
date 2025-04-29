package indexer

import (
	"sync/atomic"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func TestBlockIndexerRunner_CloseTerminates(t *testing.T) {
	handlerMock := &BlockSyncerHandlerMock{}
	config := &BlockIndexerRunnerConfig{QueueChannelSize: 2}
	runner := NewBlockIndexerRunner(handlerMock, config, hclog.NewNullLogger())
	ch := make(chan bool)

	runner.Start()

	go func() {
		<-runner.closeCh
		ch <- true
	}()

	require.NoError(t, runner.Close())
	require.True(t, <-ch)
}

func TestBlockIndexerRunner_Start(t *testing.T) {
	forward, backward := uint64(0), uint64(0)
	handlerMock := &BlockSyncerHandlerMock{
		RollForwardFn: func(_ BlockHeader, _ BlockTxsRetriever) error {
			atomic.AddUint64(&forward, 1)

			return nil
		}, RollBackwardFuncFn: func(bp BlockPoint) error {
			newValue := atomic.AddUint64(&backward, 1)
			if newValue == 4 {
				return ErrBlockIndexerFatal
			}

			return nil
		},
	}
	config := &BlockIndexerRunnerConfig{QueueChannelSize: 2}
	runner := NewBlockIndexerRunner(handlerMock, config, hclog.NewNullLogger())
	ch := make(chan bool)

	runner.Start()

	go func() {
		<-runner.errorCh
		ch <- true
	}()

	runner.RollBackward(BlockPoint{})
	runner.RollBackward(BlockPoint{})
	runner.RollForward(BlockHeader{}, &BlockTxsRetrieverMock{})
	runner.RollBackward(BlockPoint{})
	runner.RollForward(BlockHeader{}, &BlockTxsRetrieverMock{})
	runner.RollForward(BlockHeader{}, &BlockTxsRetrieverMock{})
	runner.RollBackward(BlockPoint{})

	require.True(t, <-ch)
	require.Equal(t, uint64(3), forward)
	require.Equal(t, uint64(4), backward)
}
