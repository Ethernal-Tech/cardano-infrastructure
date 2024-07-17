package common

import "fmt"

type CircularQueue[T any] struct {
	items []T
	count int
	size  int
	pos   int
}

func NewCircularQueue[T any](size int) CircularQueue[T] {
	return CircularQueue[T]{
		items: make([]T, size),
		size:  size,
	}
}

func (cq *CircularQueue[T]) Push(item T) error {
	if cq.count == cq.size {
		return fmt.Errorf("queue is already populated with %d items", cq.count)
	}

	cq.items[(cq.pos+cq.count)%cq.size] = item
	cq.count++

	return nil
}

func (cq *CircularQueue[T]) Pop() T {
	var def T

	if cq.count == 0 {
		return def
	}

	result := cq.items[cq.pos]
	cq.items[cq.pos] = def
	cq.pos = (cq.pos + 1) % cq.size
	cq.count--

	return result
}

func (cq *CircularQueue[T]) Peek() T {
	if cq.count == 0 {
		var def T

		return def
	}

	result := cq.items[cq.pos]

	return result
}

func (cq CircularQueue[T]) Len() int {
	return cq.count
}

func (cq CircularQueue[T]) IsFull() bool {
	return cq.count == cq.size
}

func (cq *CircularQueue[T]) SetCount(cnt int) {
	cq.count = min(cnt, cq.size)
}

func (cq *CircularQueue[T]) ClearFrom(from int) {
	var (
		def    T
		oldLen = cq.count
	)

	for i := max(0, from); i < oldLen; i++ {
		pos := (cq.pos + i) % cq.size

		cq.items[pos] = def
		cq.count--
	}
}

func (cq *CircularQueue[T]) Find(handler func(t T) bool) int {
	for i := 0; i < cq.count; i++ {
		pos := (cq.pos + i) % cq.size

		if handler(cq.items[pos]) {
			return i
		}
	}

	return -1
}

func (cq *CircularQueue[T]) ToList() []T {
	lst := make([]T, cq.count)

	for i := 0; i < cq.count; i++ {
		pos := (cq.pos + i) % cq.size
		lst[i] = cq.items[pos]
	}

	return lst
}
