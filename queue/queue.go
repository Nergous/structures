// Package queue provides a generic, slice-backed ring-buffer FIFO
// (first-in, first-out) queue.
//
// Create a queue with [New] or [NewWithCap]. Elements are added to the back
// with [Queue.Push] or [Queue.PushN] and removed from the front with
// [Queue.Pop] or [Queue.PopN]; [Queue.Peek] inspects the front without
// removing it.
//
// Push runs in amortized O(1), occasionally reallocating the backing buffer as
// it grows. Pop, Peek, Len, Cap, and IsEmpty run in O(1). Methods that copy or
// iterate over every element, such as All, Clone, and Slice, run in O(n).
//
// Elements leave the queue in the order they were added: [Queue.All] and
// [Queue.Slice] report them from front (oldest) to back (newest). Popped slots
// are reused by later pushes, so the buffer never slides or shrinks on its own.
//
// The zero value of Queue[T] is usable as an empty queue. A nil *Queue[T]
// behaves as an empty queue for every read-only method and for removal methods
// such as [Queue.Pop], so the zero pointer value is safe to query without first
// calling [New]. Methods that store, clear, reset, or resize data
// ([Queue.Push], [Queue.PushN], [Queue.Clear], [Queue.Reset], [Queue.Shrink],
// [Queue.Clip], [Queue.Grow]) require a non-nil receiver and panic on a nil
// one; see each method's documentation.
//
// A Queue is not safe for concurrent use: it performs no internal locking.
// Callers that share a queue across goroutines must provide their own
// synchronization.
package queue

import (
	"iter"

	"github.com/Nergous/structures/internal/ring"
	"github.com/Nergous/structures/internal/shared"
)

// Queue is a generic FIFO queue backed by a slice ring buffer. The front of the
// queue is the oldest element still present, i.e. the next one to be removed.
//
// The zero value is a valid empty queue. A nil *Queue[T] is treated as a valid
// empty queue by read-only and removal methods, but methods that store, clear,
// reset, or resize data require a non-nil receiver. Copying a Queue value
// shares its storage with the original; use [Queue.Clone] for an independent
// copy.
//
// Queue is not safe for concurrent use.
type Queue[T any] struct {
	buf ring.Buffer[T]
}

// New returns an empty queue with no preallocated capacity.
func New[T any]() *Queue[T] {
	return NewWithCap[T](0)
}

// NewWithCap returns an empty queue with a backing buffer preallocated for at
// least n elements. Use it when the maximum size is known up front to avoid
// reallocations while pushing. A negative n is treated as 0.
func NewWithCap[T any](n int) *Queue[T] {
	n = max(n, 0)

	return &Queue[T]{
		buf: ring.WithCap[T](n),
	}
}

// Len returns the number of elements currently in the queue. A nil receiver
// reports 0.
func (q *Queue[T]) Len() int {
	if q == nil {
		return 0
	}

	return q.buf.Len()
}

// Cap returns the capacity of the queue's backing buffer, i.e. the number of
// elements it can hold before the next reallocation. Popping elements does not
// reduce Cap; freed front slots are reused by later pushes. A nil receiver
// reports 0.
func (q *Queue[T]) Cap() int {
	if q == nil {
		return 0
	}

	return q.buf.Cap()
}

// IsEmpty reports whether the queue has no elements. A nil receiver reports
// true.
func (q *Queue[T]) IsEmpty() bool {
	return q == nil || q.buf.Len() == 0
}

// Peek returns the front element (the oldest, next to be removed) without
// removing it. The boolean result is false when the queue is empty, in which
// case the returned value is the zero value of T. It runs in O(1). A nil
// receiver is treated as empty and returns (zero, false).
func (q *Queue[T]) Peek() (T, bool) {
	if q == nil {
		var zero T
		return zero, false
	}

	return q.buf.Front()
}

// All returns an iterator over the queue's elements from front to back (FIFO
// order), without consuming them; the queue is left unchanged. The returned
// [iter.Seq] is a Go 1.23 range-over-func iterator, so it can be used directly
// in a range loop. Breaking out of the loop early stops iteration cleanly.
// Mutating the queue during iteration is undefined behavior. A nil receiver is
// treated as empty and yields nothing.
func (q *Queue[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if q == nil {
			return
		}

		first, second := q.buf.Segments()
		for _, v := range first {
			if !yield(v) {
				return
			}
		}

		for _, v := range second {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the queue. The two queues share
// no backing buffer, so mutating one does not affect the other. Elements are
// copied as-is: if T is a pointer or contains references, the pointed-to data
// is shared between the copies. The clone's capacity equals its length. It
// runs in O(n).
//
// A nil receiver clones to a new, usable empty queue.
func (q *Queue[T]) Clone() *Queue[T] {
	if q == nil {
		return New[T]()
	}

	return &Queue[T]{
		buf: q.buf.Clone(),
	}
}

// Slice returns a copy of the queue's elements from front to back (FIFO order),
// the same order [Queue.All] yields. The returned slice is independent of the
// queue, so callers may modify it freely. An empty queue returns nil; a nil
// receiver is treated as empty and likewise returns nil. It runs in O(n).
func (q *Queue[T]) Slice() []T {
	if q == nil || q.buf.Len() == 0 {
		return nil
	}

	out := make([]T, q.buf.Len())
	q.buf.CopyTo(out)

	return out
}

// Push adds v to the back of the queue. It runs in amortized O(1); the backing
// buffer may be reallocated to grow.
//
// Push requires a non-nil receiver and panics on a nil one: it must store the
// element somewhere. The zero value is usable, so [New] is optional.
func (q *Queue[T]) Push(v T) {
	shared.MustNotNil(q, "queue", "Push")
	q.buf.PushBack(v)
}

// PushN adds vs to the back of the queue in argument order, so the last
// argument ends up at the back. It grows the backing buffer at most once,
// making it cheaper than repeated [Queue.Push] calls. With no arguments it is a
// no-op. It runs in amortized O(k), where k is the number of elements pushed.
//
// PushN requires a non-nil receiver and panics on a nil one, even when called
// with no arguments: it must store the elements somewhere. The zero value is
// usable, so [New] is optional.
func (q *Queue[T]) PushN(vs ...T) {
	shared.MustNotNil(q, "queue", "PushN")
	q.buf.PushBackN(vs)
}

// Pop removes and returns the front element (the oldest, next to be removed).
// The boolean result is false when the queue is empty, in which case the
// returned value is the zero value of T. The vacated slot is zeroed so it no
// longer retains a reference to the removed element. It runs in O(1). A nil
// receiver is treated as empty and returns (zero, false).
func (q *Queue[T]) Pop() (T, bool) {
	if q == nil {
		var zero T
		return zero, false
	}

	return q.buf.PopFront()
}

// PopN removes and returns up to n elements from the front of the queue in FIFO
// order, so the first element of the result is the former front. If n is
// greater than Len, it drains the queue. If n is non-positive, or the queue is
// empty or nil, PopN returns nil. The vacated slots are zeroed so they no
// longer retain references, and the capacity is preserved. It runs in O(k),
// where k is the number of elements returned, and allocates the result slice
// once.
func (q *Queue[T]) PopN(n int) []T {
	if q == nil {
		return nil
	}

	return q.buf.PopFrontN(n)
}

// Clear removes all elements and releases the backing buffer, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To empty
// the queue while keeping its capacity for reuse, use [Queue.Reset] instead. It
// runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (q *Queue[T]) Clear() {
	shared.MustNotNil(q, "queue", "Clear")
	q.buf.Clear()
}

// Reset removes all elements but keeps the backing buffer for reuse, so Cap is
// preserved. The elements are zeroed so the buffer no longer retains references
// to them. To also release the backing buffer, use [Queue.Clear] instead. It
// runs in O(n).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (q *Queue[T]) Reset() {
	shared.MustNotNil(q, "queue", "Reset")
	q.buf.Reset()
}

// Shrink reduces the backing buffer to hold exactly Len elements, copying them
// into a new, right-sized buffer so the previous (larger) one becomes eligible
// for garbage collection immediately. Use it to reclaim memory after a queue
// that grew large has shrunk and will not grow back.
//
// Shrink runs in O(n) and allocates once; it is a no-op when the capacity
// already equals the length and the elements do not wrap around the end of the
// backing buffer. For the cheaper variant that avoids copying when the elements
// are contiguous, use [Queue.Clip]. Shrink is the counterpart of [Queue.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (q *Queue[T]) Shrink() {
	shared.MustNotNil(q, "queue", "Shrink")
	q.buf.Shrink()
}

// Clip reduces the reported capacity to the current length. When the elements
// are contiguous in the backing buffer, Clip reslices it without copying (see
// [slices.Clip]); when they wrap around its end, it copies them into a compact
// buffer. It runs in O(1) when it reslices and in O(n) when it copies. Because
// a resliced buffer keeps the original array alive, the unused memory is not
// returned to the garbage collector until a later growth reallocates. For an
// immediate, copying reclaim, use [Queue.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (q *Queue[T]) Clip() {
	shared.MustNotNil(q, "queue", "Clip")
	q.buf.Clip()
}

// Grow ensures the queue has spare capacity for at least n more elements
// without reallocating during subsequent pushes, growing the backing buffer if
// needed. It is the counterpart of [Queue.Shrink]. Grow runs in O(n) when it
// reallocates and is a no-op when n is non-positive or the capacity is already
// sufficient. It panics if the new capacity overflows.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (q *Queue[T]) Grow(n int) {
	shared.MustNotNil(q, "queue", "Grow")
	q.buf.Grow(n)
}
