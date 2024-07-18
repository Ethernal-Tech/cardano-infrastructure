package common

import (
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSafeCircularQueue_Parallel(t *testing.T) {
	t.Parallel()

	const (
		pushCount = 200
		capacity  = 5
	)

	captured1, captured2 := []int{}, []int{}

	queue := NewSafeCircularQueue[int](capacity)

	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()

		for {
			value, active := queue.Pop()
			if !active {
				return
			}

			captured1 = append(captured1, value)
		}
	}()

	go func() {
		defer wg.Done()

		for {
			value, active := queue.Pop()
			if !active {
				return
			}

			captured2 = append(captured2, value)
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < pushCount; i++ {
			if !queue.Push(i) {
				return
			}
		}
	}()

	// Simulate closing the queue
	go func() {
		time.Sleep(2 * time.Second)

		queue.Close()
	}()

	wg.Wait()

	captured1 = append(captured1, captured2...)

	sort.Ints(captured1)

	require.Len(t, captured1, pushCount)

	for i, v := range captured1 {
		require.Equal(t, i, v)
	}
}

func TestSafeCircularQueue_WriteThanRead(t *testing.T) {
	t.Parallel()

	const capacity = 20

	captured := []int{}
	flag := int64(0)
	queue := NewSafeCircularQueue[int](capacity)
	ch := make(chan struct{})

	for i := 0; i < capacity; i++ {
		if !queue.Push(i + 1) {
			return
		}
	}

	go func() {
		ch <- struct{}{}

		queue.Push(capacity + 1)
		require.Greater(t, atomic.LoadInt64(&flag), int64(0))
	}()

	<-ch

	time.Sleep(time.Millisecond * 100)

	for i := 0; i < capacity+1; i++ {
		value, active := queue.Pop()
		if !active {
			return
		}

		atomic.AddInt64(&flag, 1)

		captured = append(captured, value)
	}

	sort.Ints(captured)

	require.Len(t, captured, capacity+1)

	for i, v := range captured {
		require.Equal(t, i+1, v)
	}
}

func TestSafeCircularQueue_Clear(t *testing.T) {
	queue := NewSafeCircularQueue[int](2)

	queue.Push(100)
	queue.Clear()

	require.True(t, queue.IsEmpty())
}
