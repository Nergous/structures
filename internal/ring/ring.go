// Package ring implements the growable ring buffer shared by the queue and
// deque packages.
//
// A [Buffer] stores elements in a circular slice. It supports pushing and
// popping at both ends in amortized O(1) time. The buffer never panics on an
// empty state: removals and inspections report false instead. Nil-receiver
// handling and public documentation belong to the packages that wrap it.
//
// The zero Buffer is empty and ready to use. Buffer is not safe for concurrent
// use.
package ring

// Buffer is a growable ring buffer. The live elements start at head and wrap
// around the end of data. Capacity equals len(data), so every slot of data is
// usable.
type Buffer[T any] struct {
	data []T
	head int
	len  int
}

// WithCap returns an empty buffer with room for at least n elements. A
// non-positive n returns the zero Buffer.
func WithCap[T any](n int) Buffer[T] {
	if n <= 0 {
		return Buffer[T]{}
	}
	return Buffer[T]{data: make([]T, n)}
}

// Len returns the number of live elements.
func (b *Buffer[T]) Len() int { return b.len }

// Cap returns the number of elements the buffer can hold before it grows.
func (b *Buffer[T]) Cap() int { return len(b.data) }

// Data returns the backing array. It is exposed for white-box tests of memory
// behavior and must not be used to read or modify live elements.
func (b *Buffer[T]) Data() []T { return b.data }

// Front returns the first element, or (zero, false) when the buffer is empty.
func (b *Buffer[T]) Front() (T, bool) {
	if b.len == 0 {
		var zero T
		return zero, false
	}
	return b.data[b.head], true
}

// Back returns the last element, or (zero, false) when the buffer is empty.
func (b *Buffer[T]) Back() (T, bool) {
	if b.len == 0 {
		var zero T
		return zero, false
	}
	return b.data[b.index(b.len-1)], true
}

// Segments returns the live elements as up to two contiguous slices in
// front-to-back order. The second slice is nil when the elements do not wrap.
// The slices alias the backing array and are valid until the next mutation.
func (b *Buffer[T]) Segments() (first, second []T) {
	if b.len == 0 {
		return nil, nil
	}
	end := b.head + b.len
	if end <= len(b.data) {
		return b.data[b.head:end], nil
	}
	return b.data[b.head:], b.data[:end-len(b.data)]
}

// CopyTo copies the live elements, front to back, into dst and returns the
// number copied.
func (b *Buffer[T]) CopyTo(dst []T) int {
	first, second := b.Segments()
	n := copy(dst, first)
	return n + copy(dst[n:], second)
}

// Clone returns an independent copy whose capacity equals its length.
func (b *Buffer[T]) Clone() Buffer[T] {
	c := Buffer[T]{data: make([]T, b.len), len: b.len}
	b.CopyTo(c.data)
	return c
}

// PushBack appends v at the back, growing the buffer when it is full.
func (b *Buffer[T]) PushBack(v T) {
	if b.len == len(b.data) {
		b.growFor(b.len + 1)
	}
	b.data[b.index(b.len)] = v
	b.len++
}

// PushFront prepends v at the front, growing the buffer when it is full.
func (b *Buffer[T]) PushFront(v T) {
	if b.len == len(b.data) {
		b.growFor(b.len + 1)
	}
	b.head--
	if b.head < 0 {
		b.head = len(b.data) - 1
	}
	b.data[b.head] = v
	b.len++
}

// PushBackN appends vs in order, growing the buffer at most once.
func (b *Buffer[T]) PushBackN(vs []T) {
	if len(vs) == 0 {
		return
	}
	b.growFor(b.len + len(vs))
	for i, v := range vs {
		b.data[b.index(b.len+i)] = v
	}
	b.len += len(vs)
}

// PushFrontN prepends vs so that vs[0] becomes the new front, growing the
// buffer at most once.
func (b *Buffer[T]) PushFrontN(vs []T) {
	if len(vs) == 0 {
		return
	}
	b.growFor(b.len + len(vs))
	b.head -= len(vs)
	if b.head < 0 {
		b.head += len(b.data)
	}
	for i, v := range vs {
		b.data[b.index(i)] = v
	}
	b.len += len(vs)
}

// PopFront removes and returns the first element, or (zero, false) when the
// buffer is empty. The vacated slot is zeroed.
func (b *Buffer[T]) PopFront() (T, bool) {
	var zero T
	if b.len == 0 {
		return zero, false
	}
	v := b.data[b.head]
	b.data[b.head] = zero
	b.head++
	if b.head == len(b.data) {
		b.head = 0
	}
	b.len--
	if b.len == 0 {
		b.head = 0
	}
	return v, true
}

// PopBack removes and returns the last element, or (zero, false) when the
// buffer is empty. The vacated slot is zeroed.
func (b *Buffer[T]) PopBack() (T, bool) {
	var zero T
	if b.len == 0 {
		return zero, false
	}
	i := b.index(b.len - 1)
	v := b.data[i]
	b.data[i] = zero
	b.len--
	if b.len == 0 {
		b.head = 0
	}
	return v, true
}

// PopFrontN removes up to n elements from the front and returns them in
// front-to-back order. It returns nil when n is not positive or the buffer is
// empty.
func (b *Buffer[T]) PopFrontN(n int) []T {
	if n <= 0 || b.len == 0 {
		return nil
	}
	n = min(n, b.len)
	out := make([]T, n)
	for i := range out {
		out[i], _ = b.PopFront()
	}
	return out
}

// PopBackN removes up to n elements from the back and returns them in removal
// (back-to-front) order. It returns nil when n is not positive or the buffer is
// empty.
func (b *Buffer[T]) PopBackN(n int) []T {
	if n <= 0 || b.len == 0 {
		return nil
	}
	n = min(n, b.len)
	out := make([]T, n)
	for i := range out {
		out[i], _ = b.PopBack()
	}
	return out
}

// Grow makes room for at least n more elements. It is a no-op when n is not
// positive or the capacity already suffices.
func (b *Buffer[T]) Grow(n int) {
	if n <= 0 {
		return
	}
	b.growFor(b.len + n)
}

// Shrink copies the live elements into a new array of exactly Len elements so
// the old array can be collected. An empty buffer drops its array. It is a
// no-op when the buffer is already right-sized and starts at index 0.
func (b *Buffer[T]) Shrink() {
	if b.len == 0 {
		b.data = nil
		b.head = 0
		return
	}
	if len(b.data) == b.len && b.head == 0 {
		return
	}
	trimmed := make([]T, b.len)
	b.CopyTo(trimmed)
	b.data = trimmed
	b.head = 0
}

// Clip reduces the capacity to Len. Contiguous elements are resliced without
// copying; wrapped elements are copied into a compact array. An empty buffer
// drops its array.
func (b *Buffer[T]) Clip() {
	if b.len == 0 {
		b.data = nil
		b.head = 0
		return
	}
	if len(b.data) == b.len && b.head == 0 {
		return
	}
	if end := b.head + b.len; end <= len(b.data) {
		b.data = b.data[b.head:end:end]
		b.head = 0
		return
	}
	clipped := make([]T, b.len)
	b.CopyTo(clipped)
	b.data = clipped
	b.head = 0
}

// Clear removes all elements and releases the backing array.
func (b *Buffer[T]) Clear() {
	b.data = nil
	b.head = 0
	b.len = 0
}

// Reset removes all elements, zeroes them, and keeps the backing array.
func (b *Buffer[T]) Reset() {
	first, second := b.Segments()
	clear(first)
	clear(second)
	b.head = 0
	b.len = 0
}

// index maps a logical offset from the front to a position in data. offset must
// be less than the capacity.
func (b *Buffer[T]) index(offset int) int {
	i := b.head + offset
	if i >= len(b.data) {
		i -= len(b.data)
	}
	return i
}

// growFor reallocates so that the capacity is at least need, preserving order
// and moving the front to index 0. It does nothing when need already fits.
func (b *Buffer[T]) growFor(need int) {
	if need <= len(b.data) {
		return
	}
	grown := make([]T, nextCap(len(b.data), need))
	b.CopyTo(grown)
	b.data = grown
	b.head = 0
}

// nextCap returns the capacity to allocate when current is too small for need:
// it doubles below 1024 elements and grows by a quarter above that, starting
// from 1.
func nextCap(current, need int) int {
	if current <= 0 {
		current = 1
	}
	for current < need {
		if current < 1024 {
			current *= 2
			continue
		}
		current += current / 4
	}
	return current
}
