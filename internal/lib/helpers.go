package lib

import "fmt"

func Assert(condition bool, msg string, args ...any) {
	if !condition {
		panic(fmt.Sprintf(msg, args...))
	}
}

type FIFO[T any] struct {
	FIFOState[T]
}

type FIFOState[T any] struct {
	Elements []T
	Head     int
	Tail     int
	Count    int
}

func (f *FIFO[T]) Init(size int) {
	f.Elements = make([]T, size)
}

func (f *FIFO[T]) Push(e T) {
	if f.Count == len(f.Elements) {
		return
	}

	f.Elements[f.Tail] = e
	f.Tail = (f.Tail + 1) % len(f.Elements)
	f.Count++
}

func (f *FIFO[T]) Pop() (T, bool) {
	if f.Count == 0 {
		var zero T
		return zero, false
	}

	e := f.Elements[f.Head]
	f.Head = (f.Head + 1) % len(f.Elements)
	f.Count--

	return e, true
}

func (f *FIFO[T]) Peek(idx int) T {
	return f.Elements[idx]
}

func (f *FIFO[T]) Replace(idx int, e T) {
	f.Elements[idx] = e
}

func (f *FIFO[T]) Clear() {
	f.Count = 0
	f.Head = 0
	f.Tail = 0
}

func (f *FIFO[T]) GetCount() int {
	return f.Count
}
