package queues

import (
	"sync"
)

var (
	minShrink = 64
	baseLen   = 8
)

const growthFactor = 2

func shouldShrink(l, c int) (newCap int, ok bool) {
	newCap = l * growthFactor
	ok = l < c/4 && l > minShrink && l > baseLen
	return newCap, ok
}

// Queue represents a queue of elements.
// It is expected to automatically shrink its capacity when its length shrinks.
type Queue[T any] interface {
	// Len returns the amount of elements stored.
	Len() int
	// Dequeue returns the first element and removes it from the queue.
	// Callers are responsible to check if Len>0 before calling Dequeue.
	Dequeue() (t T)
	// Enqueue adds an element at the end of the queue.
	Enqueue(t T)
}

// Slice

var _ Queue[int] = &sliceQueue[int]{}

type sliceQueue[T any] []T

func (sq *sliceQueue[T]) Len() int {
	return len(*sq)
}

func (sq *sliceQueue[T]) checkShrink() {
	if nl, ok := shouldShrink(len(*sq), cap(*sq)); ok {
		n := make([]T, len(*sq), nl)
		copy(n, *sq)
		*sq = n
	}
}

func (sq *sliceQueue[T]) Dequeue() T {
	v := (*sq)[0]
	*sq = (*sq)[1:]
	sq.checkShrink()
	return v
}

func (sq *sliceQueue[T]) Enqueue(v T) {
	if *sq == nil {
		*sq = make([]T, 0, baseLen)
	}
	*sq = append(*sq, v)
}

// LinkedList

var _ Queue[int] = &linkedListQueue[int]{}

type elem[T any] struct {
	v    T
	next *elem[T]
}

type linkedListQueue[T any] struct {
	len  int
	head *elem[T]
	tail *elem[T]
}

func (sq *linkedListQueue[T]) Len() int {
	return sq.len
}

func (sq *linkedListQueue[T]) Dequeue() T {
	if sq.head == nil {
		panic("dequeue from empty queue")
	}
	sq.len--
	v := sq.head.v
	sq.head = sq.head.next
	if sq.head == nil {
		sq.tail = nil
	}
	return v
}

func (sq *linkedListQueue[T]) Enqueue(v T) {
	sq.len++
	var e elem[T] = elem[T]{v: v}
	if sq.tail == nil {
		sq.head = &e
		sq.tail = &e
		return
	}
	sq.tail.next = &e
	sq.tail = &e
}

// LinkedList with mempool

var _ Queue[int] = &linkedListPooledQueue[int]{}

type linkedListPooledQueue[T any] struct {
	len  int
	p    *sync.Pool
	head *elem[T]
	tail *elem[T]
}

func newPooled[T any]() *linkedListPooledQueue[T] {
	return &linkedListPooledQueue[T]{
		p: &sync.Pool{
			New: func() any {
				return &elem[T]{}
			},
		},
	}
}

func (sq *linkedListPooledQueue[T]) Len() int {
	return sq.len
}

func (sq *linkedListPooledQueue[T]) Dequeue() T {
	sq.len--
	if sq.head == nil {
		panic("pop from empty list")
	}
	oldHead := sq.head
	v := oldHead.v
	sq.head = oldHead.next
	sq.p.Put(oldHead)
	if sq.head == nil {
		sq.tail = nil
	}
	return v
}

func (sq *linkedListPooledQueue[T]) Enqueue(v T) {
	sq.len++
	e := sq.p.Get().(*elem[T])
	e.v = v
	e.next = nil
	if sq.tail == nil || sq.head == nil {
		sq.head = e
		sq.tail = e
		return
	}
	sq.tail.next = e
	sq.tail = e
}

// Chan

var _ Queue[int] = newChanQueue[int]()

func copyChan[T any](dst chan<- T, src chan T) {
	close(src)
	for v := range src {
		dst <- v
	}
}

type chanQueue[T any] chan T

func newChanQueue[T any]() *chanQueue[T] {
	c := chanQueue[T](make(chan T, baseLen))
	return &c
}

func (cq *chanQueue[T]) Len() int {
	return len(*cq)
}

func (cq *chanQueue[T]) checkShrink() {
	if nl, ok := shouldShrink(len(*cq), cap(*cq)); ok {
		n := make(chan T, nl)
		copyChan(n, *cq)
		*cq = n
	}
}

func (cq *chanQueue[T]) Dequeue() T {
	select {
	case v := <-*cq:
		cq.checkShrink()
		return v
	default:
		panic("dequeue from empty queue")
	}
}

func (cq *chanQueue[T]) Enqueue(v T) {
	select {
	case *cq <- v:
	default:
		n := make(chan T, cap(*cq)*growthFactor)
		copyChan(n, *cq)
		*cq = n
		n <- v
	}
}

// Ring

var _ Queue[int] = &ringQueue[int]{}

type ringQueue[T any] struct {
	first, l int
	buf      []T
}

func (sq *ringQueue[T]) Len() int {
	return sq.l
}

func (sq *ringQueue[T]) swapBuf(n []T) {
	if sq.first+sq.l > len(sq.buf) {
		skip := copy(n, sq.buf[sq.first:])
		copy(n[skip:], sq.buf[:sq.l-skip])
	} else {
		copy(n, sq.buf[sq.first:sq.first+sq.l])
	}
	sq.first = 0
	sq.buf = n
}

func (sq *ringQueue[T]) checkShrink() {
	nl, ok := shouldShrink(sq.l, len(sq.buf))
	if !ok {
		return
	}
	n := make([]T, nl, nl)
	sq.swapBuf(n)
}

func (sq *ringQueue[T]) Dequeue() T {
	if sq.l == 0 {
		panic("dequeue on empty queue")
	}
	v := sq.buf[sq.first]
	sq.first = (sq.first + 1) % len(sq.buf)
	sq.l--
	sq.checkShrink()
	return v
}

func (sq *ringQueue[T]) grow() {
	n := make([]T, max(growthFactor*len(sq.buf), baseLen))
	sq.swapBuf(n)
}

func (sq *ringQueue[T]) Enqueue(v T) {
	if sq.l+1 > len(sq.buf) {
		sq.grow()
	}
	sq.buf[(sq.first+sq.l)%len(sq.buf)] = v
	sq.l++
}

// Map

var _ Queue[int] = &mapQueue[int]{}

type mapQueue[T any] struct {
	first, last uint64
	mem         map[uint64]T
}

func newMapQueue[T any]() *mapQueue[T] {
	return &mapQueue[T]{mem: make(map[uint64]T)}
}

func (mq *mapQueue[T]) Len() int {
	return len(mq.mem)
}

func (mq *mapQueue[T]) Dequeue() T {
	if len(mq.mem) == 0 {
		panic("remove from empty map queue")
	}
	v := mq.mem[mq.first]
	delete(mq.mem, mq.first)
	mq.first++
	return v
}

func (mq *mapQueue[T]) Enqueue(v T) {
	mq.mem[mq.last] = v
	mq.last++
	if mq.last == mq.first {
		panic("this is impossible on modern machines")
	}
}
