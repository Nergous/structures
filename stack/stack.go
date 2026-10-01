// Package stack provides a generic, slice-backed LIFO (last-in, first-out) stack.
//
// Create a stack with [New] or [NewWithCap]. Elements are added with
// [Stack.Push] or [Stack.PushN] and removed from the top with [Stack.Pop] or
// [Stack.PopN]; [Stack.Peek] inspects the top without removing it.
//
// Push runs in amortized O(1) time, occasionally reallocating the backing
// array as it grows. Pop, Peek, Len, Cap and IsEmpty run in O(1).
//
// The zero value of Stack[T] is usable as an empty stack. A nil *Stack[T]
// behaves as an empty stack for every read-only method and for empty-removal
// methods such as [Stack.Pop] and [Stack.PopN], so the zero pointer value is
// safe to query without first calling [New]. Methods that need to store or
// resize data ([Stack.Push], [Stack.PushN], [Stack.Grow], [Stack.Clear],
// [Stack.Reset], [Stack.Shrink], [Stack.Clip]) require a non-nil receiver and
// panic on a nil one; see each method's documentation.
//
// A Stack is not safe for concurrent use: it performs no internal locking.
// Callers that share a stack across goroutines must provide their own
// synchronization.
package stack

import (
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/shared"
)

// Stack is a generic LIFO stack backed by a slice. The top of the stack is the
// most recently pushed element.
//
// The zero value is a valid empty stack. A nil *Stack[T] is treated as a valid
// empty stack by read-only and empty-removal methods, but methods that store or
// resize data require a non-nil receiver.
//
// Stack is not safe for concurrent use.
type Stack[T any] struct {
	data []T
}

// New returns an empty stack with no preallocated capacity.
func New[T any]() *Stack[T] {
	return &Stack[T]{}
}

// NewWithCap returns an empty stack with a backing array preallocated for at
// least n elements. Use it when the maximum size is known up front to avoid
// reallocations while pushing. A negative n is clamped to zero.
func NewWithCap[T any](n int) *Stack[T] {
	return &Stack[T]{data: make([]T, 0, max(n, 0))}
}

// Len returns the number of elements currently on the stack. A nil receiver
// reports 0.
func (s *Stack[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.data)
}

// Cap returns the capacity of the stack's backing array, i.e. the number of
// elements it can hold before the next reallocation. A nil receiver reports 0.
func (s *Stack[T]) Cap() int {
	if s == nil {
		return 0
	}
	return cap(s.data)
}

// IsEmpty reports whether the stack has no elements. A nil receiver reports
// true.
func (s *Stack[T]) IsEmpty() bool {
	return s == nil || len(s.data) == 0
}

// Peek returns the top element without removing it. The boolean result is false
// when the stack is empty, in which case the returned value is the zero value
// of T. A nil receiver is treated as empty and returns (zero, false).
func (s *Stack[T]) Peek() (T, bool) {
	if s == nil || len(s.data) == 0 {
		var zero T
		return zero, false
	}

	return s.data[len(s.data)-1], true
}

// All returns an iterator over the stack's elements from top to bottom (LIFO
// order), without consuming them; the stack is left unchanged. The returned
// [iter.Seq] is a Go 1.23 range-over-func iterator, so it can be used directly
// in a range loop. Breaking out of the loop early stops iteration cleanly. A nil
// receiver is treated as empty and yields no elements.
func (s *Stack[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s == nil {
			return
		}
		for i := len(s.data) - 1; i >= 0; i-- {
			if !yield(s.data[i]) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the stack. The two stacks share
// no backing array, so mutating one (pushing, popping, clearing) does not affect
// the other. Elements are copied as-is: if T is a pointer or contains
// references, the pointed-to data is shared between the copies. Runs in O(n).
//
// A nil receiver, treated everywhere as an empty stack, clones to a new, usable
// empty stack (never nil), so the result can be pushed to without a further
// [New] call.
func (s *Stack[T]) Clone() *Stack[T] {
	if s == nil {
		return New[T]()
	}

	return &Stack[T]{
		data: append([]T(nil), s.data...),
	}
}

// Slice returns a copy of the stack's elements from top to bottom (LIFO), the
// same order [Stack.All] yields. The returned slice is independent of the stack:
// they share no backing array, so callers may modify it freely without affecting
// the stack, and vice versa. An empty stack returns nil; a nil receiver is
// treated as empty and likewise returns nil. Runs in O(n).
func (s *Stack[T]) Slice() []T {
	if s == nil || len(s.data) == 0 {
		return nil
	}

	out := slices.Clone(s.data)
	slices.Reverse(out)
	return out
}

// Push adds v to the top of the stack. It runs in amortized O(1) time; the
// backing array may be reallocated to grow.
//
// Push requires a non-nil receiver and panics on a nil one: it must store the
// element somewhere. The zero value is usable, so [New] is optional.
func (s *Stack[T]) Push(v T) {
	shared.MustNotNil(s, "stack", "Push")
	s.data = append(s.data, v)
}

// PushN adds vs to the top of the stack in argument order, so the last argument
// ends up on top. It grows the backing array at most once, making it cheaper
// than repeated [Stack.Push] calls. With no arguments it is a no-op. Runs in
// amortized O(k) for k elements.
//
// PushN requires a non-nil receiver and panics on a nil one, even when called
// with no arguments: it must store the elements somewhere. The zero value is
// usable, so [New] is optional.
func (s *Stack[T]) PushN(vs ...T) {
	shared.MustNotNil(s, "stack", "PushN")
	s.data = append(s.data, vs...)
}

// Pop removes and returns the top element. The boolean result is false when the
// stack is empty, in which case the returned value is the zero value of T. The
// vacated slot is zeroed so it no longer retains a reference to the popped
// element. It runs in O(1) time. A nil receiver is treated as empty: there is
// nothing to remove, so it returns (zero, false) without panicking.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if s == nil {
		return zero, false
	}

	l := len(s.data) - 1
	if l < 0 {
		return zero, false
	}

	value := s.data[l]
	s.data[l] = zero
	s.data = s.data[:l]

	return value, true
}

// PopN removes and returns up to n elements from the top of the stack in pop
// order, so the first element of the result is the former top. If n is greater
// than Len, it drains the stack. If n is non-positive, or the stack is empty or
// nil, PopN returns nil. The vacated slots are zeroed so they no longer retain
// references, and the capacity is preserved. Runs in O(k), where k is the
// number of elements returned.
func (s *Stack[T]) PopN(n int) []T {
	if s == nil || n <= 0 || len(s.data) == 0 {
		return nil
	}

	l := len(s.data)
	n = min(n, l)
	out := make([]T, n)
	for i := range out {
		out[i] = s.data[l-1-i]
	}
	clear(s.data[l-n:])
	s.data = s.data[:l-n]
	return out
}

// Clear removes all elements and releases the backing array, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To empty
// the stack while keeping its capacity for reuse, use [Stack.Reset] instead.
//
// Clear requires a non-nil receiver and panics on a nil one.
func (s *Stack[T]) Clear() {
	shared.MustNotNil(s, "stack", "Clear")
	s.data = nil
}

// Reset removes all elements but keeps the backing array for reuse, so Cap is
// preserved. The elements are zeroed so the array no longer retains references
// to them. To also release the backing array, use [Stack.Clear] instead.
//
// Reset requires a non-nil receiver and panics on a nil one.
func (s *Stack[T]) Reset() {
	shared.MustNotNil(s, "stack", "Reset")
	clear(s.data)
	s.data = s.data[:0]
}

// Shrink reduces the backing array to hold exactly Len elements, copying them
// into a new, right-sized array so the previous (larger) array becomes eligible
// for garbage collection immediately. Use it to reclaim memory after a stack
// that grew large has shrunk and will not grow back.
//
// Shrink runs in O(n) and allocates once; it is a no-op when the capacity
// already equals the length. For the cheaper, non-copying variant that only
// reslices and lets memory be reclaimed lazily on the next growth, use
// [Stack.Clip]. Shrink is the counterpart of [Stack.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (s *Stack[T]) Shrink() {
	shared.MustNotNil(s, "stack", "Shrink")
	s.data = shared.ShrinkSlice(s.data)
}

// Clip reduces the reported capacity to the current length without copying, by
// reslicing the backing array (see [slices.Clip]). It runs in O(1) and does not
// allocate. Because the backing array is retained, the unused tail memory is not
// returned to the garbage collector until a later growth reallocates. For an
// immediate, copying reclaim, use [Stack.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (s *Stack[T]) Clip() {
	shared.MustNotNil(s, "stack", "Clip")
	s.data = slices.Clip(s.data)
}

// Grow ensures the stack has spare capacity for at least n more elements without
// reallocating during subsequent pushes, growing the backing array if needed. It
// is the counterpart of [Stack.Shrink]. Grow runs in O(n) when it reallocates and
// is a no-op when n is non-positive or the capacity is already sufficient. It
// panics if the new capacity overflows, the same condition as [slices.Grow].
//
// Grow requires a non-nil receiver and panics on a nil one.
func (s *Stack[T]) Grow(n int) {
	shared.MustNotNil(s, "stack", "Grow")
	if n <= 0 {
		return
	}

	s.data = slices.Grow(s.data, n)
}
