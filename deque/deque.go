// Package deque provides a generic, slice-backed double-ended queue.
//
// Create a deque with [New] or [NewWithCap]. Elements can be added to either
// end with [Deque.PushFront], [Deque.PushBack], [Deque.PushFrontN], or
// [Deque.PushBackN], and removed from either end with [Deque.PopFront],
// [Deque.PopBack], [Deque.PopFrontN], or [Deque.PopBackN]. [Deque.Front] and
// [Deque.Back] inspect the ends without removing elements.
//
// Single-element pushes and pops run in amortized O(1) time. Front, Back, Len,
// Cap, and IsEmpty run in O(1). Methods that copy or iterate over every live
// element, such as All, Clone, Slice, Shrink, and Clip, run in O(n).
//
// The zero value of Deque[T] is usable as an empty deque. A nil *Deque[T]
// behaves as an empty deque for every read-only method and for empty-removal
// methods such as [Deque.PopFront], [Deque.PopBack], [Deque.PopFrontN], and
// [Deque.PopBackN], so the zero pointer value is safe to query without first
// calling [New]. Methods that need to store, clear, reset, shrink, or resize
// data require a non-nil receiver and panic on a nil one; see each method's
// documentation.
//
// A Deque is not safe for concurrent use: it performs no internal locking.
// Callers that share a deque across goroutines must provide their own
// synchronization.
package deque

import (
	"iter"

	"github.com/Nergous/structures/internal/ring"
	"github.com/Nergous/structures/internal/shared"
)

// Deque is a generic double-ended queue backed by a slice ring buffer. The
// front is the first element removed by [Deque.PopFront]; the back is the first
// element removed by [Deque.PopBack]. A Deque can be used as a FIFO queue, a
// LIFO stack, or a mixed-priority queue where either end is significant.
//
// The zero value is a valid empty deque. A nil *Deque[T] is treated as a valid
// empty deque by read-only and empty-removal methods, but methods that store or
// resize data require a non-nil receiver.
//
// Deque is not safe for concurrent use.
type Deque[T any] struct {
	buf ring.Buffer[T]
}

// New returns an empty deque with no preallocated capacity.
func New[T any]() *Deque[T] {
	return &Deque[T]{}
}

// NewWithCap returns an empty deque with a backing array preallocated for at
// least n elements. Use it when the maximum size is known up front to avoid
// reallocations while pushing. A negative n is clamped to zero.
func NewWithCap[T any](n int) *Deque[T] {
	return &Deque[T]{buf: ring.WithCap[T](n)}
}

// Len returns the number of elements currently in the deque. A nil receiver
// reports 0.
func (d *Deque[T]) Len() int {
	if d == nil {
		return 0
	}
	return d.buf.Len()
}

// Cap returns the capacity of the deque's backing ring buffer, i.e. the total
// number of elements it can hold before the next reallocation. Popping elements
// does not reduce Cap; freed front and back slots are reused by later pushes. A
// nil receiver reports 0.
func (d *Deque[T]) Cap() int {
	if d == nil {
		return 0
	}
	return d.buf.Cap()
}

// IsEmpty reports whether the deque has no elements. A nil receiver reports
// true.
func (d *Deque[T]) IsEmpty() bool {
	return d == nil || d.buf.Len() == 0
}

// Front returns the front element without removing it. The boolean result is
// false when the deque is empty, in which case the returned value is the zero
// value of T. A nil receiver is treated as empty and returns (zero, false).
func (d *Deque[T]) Front() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}
	return d.buf.Front()
}

// Back returns the back element without removing it. The boolean result is
// false when the deque is empty, in which case the returned value is the zero
// value of T. A nil receiver is treated as empty and returns (zero, false).
func (d *Deque[T]) Back() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}
	return d.buf.Back()
}

// All returns an iterator over the deque's elements from front to back, without
// consuming them; the deque is left unchanged. The returned [iter.Seq] is a Go
// 1.23 range-over-func iterator, so it can be used directly in a range loop.
// Breaking out of the loop early stops iteration cleanly. A nil receiver is
// treated as empty and yields no elements.
func (d *Deque[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if d == nil {
			return
		}
		first, second := d.buf.Segments()
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

// Clone returns an independent shallow copy of the deque. The two deques share
// no backing array, so mutating one does not affect the other. Elements are
// copied as-is: if T is a pointer or contains references, the pointed-to data is
// shared between the copies. Runs in O(n).
//
// A nil receiver, treated everywhere as an empty deque, clones to a new, usable
// empty deque (never nil), so the result can be pushed to without a further
// [New] call.
func (d *Deque[T]) Clone() *Deque[T] {
	if d == nil {
		return New[T]()
	}
	return &Deque[T]{buf: d.buf.Clone()}
}

// Slice returns a copy of the deque's elements from front to back, the same
// order [Deque.All] yields. The returned slice is independent of the deque:
// they share no backing array, so callers may modify it freely without
// affecting the deque, and vice versa. An empty deque returns nil; a nil
// receiver is treated as empty and likewise returns nil. Runs in O(n).
func (d *Deque[T]) Slice() []T {
	if d == nil || d.buf.Len() == 0 {
		return nil
	}

	out := make([]T, d.buf.Len())
	d.buf.CopyTo(out)
	return out
}

// PushFront adds v to the front of the deque. It runs in amortized O(1) time;
// the backing array may be reallocated to grow.
//
// PushFront requires a non-nil receiver and panics on a nil one: it must store
// the element somewhere. The zero value is usable, so [New] is optional.
func (d *Deque[T]) PushFront(v T) {
	shared.MustNotNil(d, "deque", "PushFront")
	d.buf.PushFront(v)
}

// PushBack adds v to the back of the deque. It runs in amortized O(1) time; the
// backing array may be reallocated to grow.
//
// PushBack requires a non-nil receiver and panics on a nil one: it must store
// the element somewhere. The zero value is usable, so [New] is optional.
func (d *Deque[T]) PushBack(v T) {
	shared.MustNotNil(d, "deque", "PushBack")
	d.buf.PushBack(v)
}

// PushFrontN adds vs to the front of the deque in argument order, so the first
// argument becomes the new front. It grows the backing array at most once,
// making it cheaper than repeated [Deque.PushFront] calls. With no arguments it
// is a no-op. Runs in amortized O(k) for k elements.
//
// PushFrontN requires a non-nil receiver and panics on a nil one, even when
// called with no arguments: it must store the elements somewhere. The zero
// value is usable, so [New] is optional.
func (d *Deque[T]) PushFrontN(vs ...T) {
	shared.MustNotNil(d, "deque", "PushFrontN")
	d.buf.PushFrontN(vs)
}

// PushBackN adds vs to the back of the deque in argument order, so the last
// argument becomes the new back. It grows the backing array at most once,
// making it cheaper than repeated [Deque.PushBack] calls. With no arguments it
// is a no-op. Runs in amortized O(k) for k elements.
//
// PushBackN requires a non-nil receiver and panics on a nil one, even when
// called with no arguments: it must store the elements somewhere. The zero
// value is usable, so [New] is optional.
func (d *Deque[T]) PushBackN(vs ...T) {
	shared.MustNotNil(d, "deque", "PushBackN")
	d.buf.PushBackN(vs)
}

// PopFront removes and returns the front element. The boolean result is false
// when the deque is empty, in which case the returned value is the zero value of
// T. The vacated slot is zeroed so it no longer retains a reference to the
// popped element, then the front is advanced. It runs in O(1) time.
//
// A nil receiver is treated as empty and returns (zero, false).
func (d *Deque[T]) PopFront() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}
	return d.buf.PopFront()
}

// PopBack removes and returns the back element. The boolean result is false
// when the deque is empty, in which case the returned value is the zero value of
// T. The vacated slot is zeroed so it no longer retains a reference to the
// popped element. It runs in O(1) time.
//
// A nil receiver is treated as empty and returns (zero, false).
func (d *Deque[T]) PopBack() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}
	return d.buf.PopBack()
}

// PopFrontN removes and returns up to n front elements in front-to-back order.
// If n is greater than Len, it drains the deque. Non-positive n, nil receivers,
// and empty deques return nil. The deque's capacity is preserved. Runs in O(k),
// where k is the number of elements returned.
func (d *Deque[T]) PopFrontN(n int) []T {
	if d == nil {
		return nil
	}
	return d.buf.PopFrontN(n)
}

// PopBackN removes and returns up to n back elements in removal order
// (back-to-front). If n is greater than Len, it drains the deque.
// Non-positive n, nil receivers, and empty deques return nil. The deque's
// capacity is preserved. Runs in O(k), where k is the number of elements
// returned.
func (d *Deque[T]) PopBackN(n int) []T {
	if d == nil {
		return nil
	}
	return d.buf.PopBackN(n)
}

// Shrink reduces the backing array to hold exactly Len elements, copying them
// into a new, right-sized array so the previous (larger) array becomes eligible
// for garbage collection immediately. Use it to reclaim memory after a deque
// that grew large has shrunk and will not grow back.
//
// Shrink runs in O(n) and allocates once; it is a no-op when the capacity
// already equals the length and live elements start at the front of the backing
// buffer. For the cheaper variant that can avoid copying when live elements are
// contiguous, use [Deque.Clip]. Shrink is the counterpart of [Deque.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Shrink() {
	shared.MustNotNil(d, "deque", "Shrink")
	d.buf.Shrink()
}

// Clip reduces the reported capacity to the current length, copying only when
// live elements wrap around the end of the backing buffer. Because a
// non-copying reslice may still retain the original array, unused memory is not
// guaranteed to be released immediately. For an immediate, always-copying
// reclaim, use [Deque.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Clip() {
	shared.MustNotNil(d, "deque", "Clip")
	d.buf.Clip()
}

// Grow ensures the deque has spare capacity for at least n more elements
// without reallocating during subsequent pushes, growing the backing array if
// needed. It is the counterpart of [Deque.Shrink]. Grow runs in O(n) when it
// reallocates and is a no-op when n is non-positive or the capacity is already
// sufficient.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Grow(n int) {
	shared.MustNotNil(d, "deque", "Grow")
	d.buf.Grow(n)
}

// Clear removes all elements and releases the backing array, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To empty
// the deque while keeping its capacity for reuse, use [Deque.Reset] instead.
//
// Clear requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Clear() {
	shared.MustNotNil(d, "deque", "Clear")
	d.buf.Clear()
}

// Reset removes all elements but keeps the backing array for reuse, so Cap is
// preserved. The elements are zeroed so the array no longer retains references
// to them. To also release the backing array, use [Deque.Clear] instead.
//
// Reset requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Reset() {
	shared.MustNotNil(d, "deque", "Reset")
	d.buf.Reset()
}
