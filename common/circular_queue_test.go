package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCircularQueue(t *testing.T) {
	t.Parallel()

	type Item struct {
		Val int
	}

	t.Run("push and pop pointer", func(t *testing.T) {
		t.Parallel()

		cq := NewCircularQueue[*Item](5)

		require.Nil(t, cq.Pop())
		require.Equal(t, 0, cq.Len())

		require.NoError(t, cq.Push(&Item{Val: 10}))
		require.NoError(t, cq.Push(&Item{Val: 20}))
		require.NoError(t, cq.Push(&Item{Val: 30}))
		require.NoError(t, cq.Push(&Item{Val: 40}))
		require.NoError(t, cq.Push(&Item{Val: 50}))
		require.Error(t, cq.Push(&Item{Val: 60}))

		require.Equal(t, &Item{Val: 10}, cq.Pop())
		require.Equal(t, 4, cq.Len())

		require.NoError(t, cq.Push(&Item{Val: 60}))
		require.Error(t, cq.Push(&Item{Val: 70}))

		for i := 0; i < 5; i++ {
			require.Equal(t, &Item{Val: 20 + i*10}, cq.Pop())
			require.Equal(t, 4-i, cq.Len())
		}

		for i := 0; i < 4; i++ {
			require.NoError(t, cq.Push(&Item{Val: 160 + i*10}))
		}

		for i := 0; i < 4; i++ {
			require.Equal(t, &Item{Val: 160 + i*10}, cq.Pop())
			require.Equal(t, 3-i, cq.Len())
		}
	})

	t.Run("push and pop struct", func(t *testing.T) {
		t.Parallel()

		cq2 := NewCircularQueue[Item](5)
		for i := 0; i < 5; i++ {
			require.NoError(t, cq2.Push(Item{Val: 160 + i*10}))
		}

		for i := 0; i < 8; i++ {
			require.Equal(t, Item{Val: 160 + i*10}, cq2.Pop())

			if i <= 2 {
				require.NoError(t, cq2.Push(Item{Val: 210 + i*10}))
			}
		}
	})

	t.Run("clear from", func(t *testing.T) {
		t.Parallel()

		cq := NewCircularQueue[*Item](5)
		for i := 0; i < 5; i++ {
			require.NoError(t, cq.Push(&Item{Val: 160 + i*10}))
		}

		require.NotNil(t, cq.Pop())
		require.NoError(t, cq.Push(&Item{Val: 160 + 5*10}))

		cq.ClearFrom(1)

		for i := 0; i < cq.size; i++ {
			pos := (cq.pos + i) % cq.size

			if i >= 1 {
				require.Nil(t, cq.items[pos])
			} else {
				require.Equal(t, &Item{Val: 170}, cq.items[pos])
			}
		}
	})

	t.Run("set count", func(t *testing.T) {
		t.Parallel()

		cq := NewCircularQueue[*Item](5)
		for i := 0; i < 5; i++ {
			require.NoError(t, cq.Push(&Item{Val: 160 + i*10}))
		}

		for i := 0; i < 5; i++ {
			cq.SetCount(5 - i - 1)

			require.Equal(t, 5-i-1, len(cq.ToList()))
		}
	})
}
