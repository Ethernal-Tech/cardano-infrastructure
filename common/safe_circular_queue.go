package common

import (
	"sync"
)

type SafeCircularQueue[T any] struct {
	queue    CircularQueue[T]
	mutex    sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	closed   bool
}

// NewSafeCircularQueue initializes the SafeCircularQueue
func NewSafeCircularQueue[T any](capacity int) *SafeCircularQueue[T] {
	scq := &SafeCircularQueue[T]{
		queue: NewCircularQueue[T](capacity),
	}

	scq.notEmpty = sync.NewCond(&scq.mutex)
	scq.notFull = sync.NewCond(&scq.mutex)

	return scq
}

// Close signals the queue to stop accepting new items and unblocks any waiting goroutines.
func (scq *SafeCircularQueue[T]) Close() {
	scq.mutex.Lock()
	defer scq.mutex.Unlock()

	if !scq.closed {
		scq.closed = true
		scq.notEmpty.Broadcast() // Unblock all waiting goroutines
		scq.notFull.Broadcast()  // Unblock all waiting goroutines
	}
}

// Push adds an item to the queue if there’s space, otherwise waits until there is or until closed.
func (scq *SafeCircularQueue[T]) Push(item T) bool {
	scq.mutex.Lock()
	defer scq.mutex.Unlock()

	// Wait if the queue is full or if the queue is closed
	for scq.queue.IsFull() && !scq.closed {
		scq.notFull.Wait()
	}

	if scq.closed {
		return false
	}

	// Push item to the queue
	_ = scq.queue.Push(item)

	// Signal one waiting Pop operation (if any)
	scq.notEmpty.Signal()

	return true
}

// Pop removes and returns an item from the queue if it’s not empty
// otherwise waits until there is an item or until closed.
func (scq *SafeCircularQueue[T]) Pop() (result T, active bool) {
	scq.mutex.Lock()
	defer scq.mutex.Unlock()

	// Wait if the queue is empty or if the queue is closed
	for scq.queue.Len() == 0 && !scq.closed {
		scq.notEmpty.Wait()
	}

	if scq.closed {
		return result, false
	}

	// Signal one waiting Push operation (if any)
	defer scq.notFull.Signal()

	// Pop item from the queue
	return scq.queue.Pop(), true
}

// Reset clears all the items from the queu
func (scq *SafeCircularQueue[T]) Clear() {
	scq.mutex.Lock()
	defer scq.mutex.Unlock()

	scq.queue.ClearFrom(0)

	scq.notFull.Broadcast() // Unblock all waiting goroutines
}

func (scq *SafeCircularQueue[T]) IsEmpty() bool {
	scq.mutex.Lock()
	defer scq.mutex.Unlock()

	return scq.queue.Len() == 0
}
