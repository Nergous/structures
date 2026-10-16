// Package deque provides a generic, slice-backed ring-buffer double-ended
// queue.
//
// Create a deque with [New] or [NewWithCap]. Elements are added to either end
// with [Deque.PushFront], [Deque.PushBack], [Deque.PushFrontN], or
// [Deque.PushBackN] and removed from either end with [Deque.PopFront],
// [Deque.PopBack], [Deque.PopFrontN], or [Deque.PopBackN]; [Deque.Front] and
// [Deque.Back] inspect the ends without removing elements.
//
// PushFront and PushBack run in amortized O(1), occasionally reallocating the
// backing buffer as it grows. PopFront, PopBack, Front, Back, Len, Cap, and
// IsEmpty run in O(1). Methods that copy or iterate over every element, such as
// All, Backward, Clone, and Slice, run in O(n).
//
// [Deque.All] and [Deque.Slice] report the elements from front to back, and
// [Deque.Backward] reports them from back to front. Popped slots at either end
// are reused by later pushes, so the buffer never slides or shrinks on its own.
//
// The zero value of Deque[T] is usable as an empty deque. A nil *Deque[T]
// behaves as an empty deque for every read-only method and for removal methods
// such as [Deque.PopFront], so the zero pointer value is safe to query without
// first calling [New]. Methods that store, clear, reset, or resize data
// ([Deque.PushFront], [Deque.PushBack], [Deque.PushFrontN], [Deque.PushBackN],
// [Deque.Clear], [Deque.Reset], [Deque.Shrink], [Deque.Clip], [Deque.Grow])
// require a non-nil receiver and panic on a nil one; see each method's
// documentation.
//
// A Deque is not safe for concurrent use: it performs no internal locking.
// Callers that share a deque across goroutines must provide their own
// synchronization.
package deque

import (
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/ring"
	"github.com/Nergous/structures/internal/shared"
)

// Deque is a generic double-ended queue backed by a slice ring buffer. The
// front is the element removed by [Deque.PopFront] and the back is the element
// removed by [Deque.PopBack].
//
// The zero value is a valid empty deque. A nil *Deque[T] is treated as a valid
// empty deque by read-only and removal methods, but methods that store, clear,
// reset, or resize data require a non-nil receiver. Copying a Deque value
// shares its storage with the original; use [Deque.Clone] for an independent
// copy.
//
// Deque is not safe for concurrent use.
type Deque[T any] struct {
	buf ring.Buffer[T]
}

// New returns an empty deque with no preallocated capacity.
func New[T any]() *Deque[T] {
	return NewWithCap[T](0)
}

// NewWithCap returns an empty deque with a backing buffer preallocated for at
// least n elements. Use it when the maximum size is known up front to avoid
// reallocations while pushing. A negative n is treated as 0.
func NewWithCap[T any](n int) *Deque[T] {
	n = max(n, 0)

	return &Deque[T]{
		buf: ring.WithCap[T](n),
	}
}

// Len returns the number of elements currently in the deque. A nil receiver
// reports 0.
func (d *Deque[T]) Len() int {
	if d == nil {
		return 0
	}

	return d.buf.Len()
}

// Cap returns the capacity of the deque's backing buffer, i.e. the number of
// elements it can hold before the next reallocation. Popping elements does not
// reduce Cap; freed front and back slots are reused by later pushes. A nil
// receiver reports 0.
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
// value of T. It runs in O(1). A nil receiver is treated as empty and returns
// (zero, false).
func (d *Deque[T]) Front() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}

	return d.buf.Front()
}

// Back returns the back element without removing it. The boolean result is
// false when the deque is empty, in which case the returned value is the zero
// value of T. It runs in O(1). A nil receiver is treated as empty and returns
// (zero, false).
func (d *Deque[T]) Back() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}

	return d.buf.Back()
}

// All returns an iterator over the deque's elements from front to back,
// without consuming them; the deque is left unchanged. The returned [iter.Seq]
// is a Go 1.23 range-over-func iterator, so it can be used directly in a range
// loop. Breaking out of the loop early stops iteration cleanly. Mutating the
// deque during iteration is undefined behavior. A nil receiver is treated as
// empty and yields nothing.
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

// Backward returns an iterator over the deque's elements from back to front,
// without consuming them; the deque is left unchanged. The returned [iter.Seq]
// is a Go 1.23 range-over-func iterator, so it can be used directly in a range
// loop. Breaking out of the loop early stops iteration cleanly. Mutating the
// deque during iteration is undefined behavior. A nil receiver is treated as
// empty and yields nothing.
func (d *Deque[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		if d == nil {
			return
		}

		first, second := d.buf.Segments()
		for _, v := range slices.Backward(second) {
			if !yield(v) {
				return
			}
		}

		for _, v := range slices.Backward(first) {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the deque. The two deques share
// no backing buffer, so mutating one does not affect the other. Elements are
// copied as-is: if T is a pointer or contains references, the pointed-to data
// is shared between the copies. The clone's capacity equals its length. It
// runs in O(n).
//
// A nil receiver clones to a new, usable empty deque.
func (d *Deque[T]) Clone() *Deque[T] {
	if d == nil {
		return New[T]()
	}

	return &Deque[T]{
		buf: d.buf.Clone(),
	}
}

// Slice returns a copy of the deque's elements from front to back, the same
// order [Deque.All] yields. The returned slice is independent of the deque, so
// callers may modify it freely. An empty deque returns nil; a nil receiver is
// treated as empty and likewise returns nil. It runs in O(n).
func (d *Deque[T]) Slice() []T {
	if d == nil || d.buf.Len() == 0 {
		return nil
	}

	out := make([]T, d.buf.Len())
	d.buf.CopyTo(out)

	return out
}

// PushFront adds v to the front of the deque. It runs in amortized O(1); the
// backing buffer may be reallocated to grow.
//
// PushFront requires a non-nil receiver and panics on a nil one: it must store
// the element somewhere. The zero value is usable, so [New] is optional.
func (d *Deque[T]) PushFront(v T) {
	shared.MustNotNil(d, "deque", "PushFront")
	d.buf.PushFront(v)
}

// PushBack adds v to the back of the deque. It runs in amortized O(1); the
// backing buffer may be reallocated to grow.
//
// PushBack requires a non-nil receiver and panics on a nil one: it must store
// the element somewhere. The zero value is usable, so [New] is optional.
func (d *Deque[T]) PushBack(v T) {
	shared.MustNotNil(d, "deque", "PushBack")
	d.buf.PushBack(v)
}

// PushFrontN adds vs to the front of the deque in argument order, so the first
// argument becomes the new front. It grows the backing buffer at most once,
// making it cheaper than repeated [Deque.PushFront] calls. With no arguments it
// is a no-op. It runs in amortized O(k), where k is the number of elements
// pushed.
//
// PushFrontN requires a non-nil receiver and panics on a nil one, even when
// called with no arguments: it must store the elements somewhere. The zero
// value is usable, so [New] is optional.
func (d *Deque[T]) PushFrontN(vs ...T) {
	shared.MustNotNil(d, "deque", "PushFrontN")
	d.buf.PushFrontN(vs)
}

// PushBackN adds vs to the back of the deque in argument order, so the last
// argument becomes the new back. It grows the backing buffer at most once,
// making it cheaper than repeated [Deque.PushBack] calls. With no arguments it
// is a no-op. It runs in amortized O(k), where k is the number of elements
// pushed.
//
// PushBackN requires a non-nil receiver and panics on a nil one, even when
// called with no arguments: it must store the elements somewhere. The zero
// value is usable, so [New] is optional.
func (d *Deque[T]) PushBackN(vs ...T) {
	shared.MustNotNil(d, "deque", "PushBackN")
	d.buf.PushBackN(vs)
}

// PopFront removes and returns the front element. The boolean result is false
// when the deque is empty, in which case the returned value is the zero value
// of T. The vacated slot is zeroed so it no longer retains a reference to the
// removed element. It runs in O(1). A nil receiver is treated as empty and
// returns (zero, false).
func (d *Deque[T]) PopFront() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}

	return d.buf.PopFront()
}

// PopBack removes and returns the back element. The boolean result is false
// when the deque is empty, in which case the returned value is the zero value
// of T. The vacated slot is zeroed so it no longer retains a reference to the
// removed element. It runs in O(1). A nil receiver is treated as empty and
// returns (zero, false).
func (d *Deque[T]) PopBack() (T, bool) {
	if d == nil {
		var zero T
		return zero, false
	}

	return d.buf.PopBack()
}

// PopFrontN removes and returns up to n elements from the front of the deque
// in front-to-back order, so the first element of the result is the former
// front. If n is greater than Len, it drains the deque. If n is non-positive,
// or the deque is empty or nil, PopFrontN returns nil. The vacated slots are
// zeroed so they no longer retain references, and the capacity is preserved.
// It runs in O(k), where k is the number of elements returned, and allocates
// the result slice once.
func (d *Deque[T]) PopFrontN(n int) []T {
	if d == nil {
		return nil
	}

	return d.buf.PopFrontN(n)
}

// PopBackN removes and returns up to n elements from the back of the deque in
// back-to-front order, so the first element of the result is the former back.
// If n is greater than Len, it drains the deque. If n is non-positive, or the
// deque is empty or nil, PopBackN returns nil. The vacated slots are zeroed so
// they no longer retain references, and the capacity is preserved. It runs in
// O(k), where k is the number of elements returned, and allocates the result
// slice once.
func (d *Deque[T]) PopBackN(n int) []T {
	if d == nil {
		return nil
	}

	return d.buf.PopBackN(n)
}

// Clear removes all elements and releases the backing buffer, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To empty
// the deque while keeping its capacity for reuse, use [Deque.Reset] instead. It
// runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Clear() {
	shared.MustNotNil(d, "deque", "Clear")
	d.buf.Clear()
}

// Reset removes all elements but keeps the backing buffer for reuse, so Cap is
// preserved. The elements are zeroed so the buffer no longer retains references
// to them. To also release the backing buffer, use [Deque.Clear] instead. It
// runs in O(n).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Reset() {
	shared.MustNotNil(d, "deque", "Reset")
	d.buf.Reset()
}

// Shrink reduces the backing buffer to hold exactly Len elements, copying them
// into a new, right-sized buffer so the previous (larger) one becomes eligible
// for garbage collection immediately. Use it to reclaim memory after a deque
// that grew large has shrunk and will not grow back.
//
// Shrink runs in O(n) and allocates once; it is a no-op when the capacity
// already equals the length and the elements do not wrap around the end of the
// backing buffer. For the cheaper variant that avoids copying when the elements
// are contiguous, use [Deque.Clip]. Shrink is the counterpart of [Deque.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Shrink() {
	shared.MustNotNil(d, "deque", "Shrink")
	d.buf.Shrink()
}

// Clip reduces the reported capacity to the current length. When the elements
// are contiguous in the backing buffer, Clip reslices it without copying (see
// [slices.Clip]); when they wrap around its end, it copies them into a compact
// buffer. It runs in O(1) when it reslices and in O(n) when it copies. Because
// a resliced buffer keeps the original array alive, the unused memory is not
// returned to the garbage collector until a later growth reallocates. For an
// immediate, copying reclaim, use [Deque.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Clip() {
	shared.MustNotNil(d, "deque", "Clip")
	d.buf.Clip()
}

// Grow ensures the deque has spare capacity for at least n more elements
// without reallocating during subsequent pushes, growing the backing buffer if
// needed. It is the counterpart of [Deque.Shrink]. Grow runs in O(n) when it
// reallocates and is a no-op when n is non-positive or the capacity is already
// sufficient. It panics if the new capacity overflows.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (d *Deque[T]) Grow(n int) {
	shared.MustNotNil(d, "deque", "Grow")
	d.buf.Grow(n)
}
