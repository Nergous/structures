// Package binheap provides a generic, slice-backed binary heap.
//
// A heap keeps its least element, as defined by its comparator, on top.
// Throughout this documentation "least" always means least according to the
// heap's comparator: for a max-heap built with [NewMax] it is the largest value.
//
// Heaps are created with one of three constructor families:
//
//   - [New], [NewWithCap], and [From] build min-heaps for [cmp.Ordered] types
//     using [cmp.Compare].
//   - [NewMax], [NewMaxWithCap], and [FromMax] build max-heaps for
//     [cmp.Ordered] types, so callers never hand-reverse a comparator.
//   - [NewFunc], [NewFuncWithCap], and [FromFunc] accept any comparator for any
//     element type.
//
// Elements are added with [Heap.Push] or [Heap.PushN] and removed in priority
// order with [Heap.Pop] or [Heap.PopN]; [Heap.Peek] inspects the least element
// without removing it. [Heap.PushPop] and [Heap.Replace] combine a push and a
// pop in a single sift.
//
// Peek, Len, Cap, and IsEmpty run in O(1). Push runs in amortized O(log n),
// and Pop, PushPop, and Replace run in O(log n). PushN picks the cheaper of k
// sift-ups and a full rebuild, bounded by min(O(k log(n+k)), O(n+k)). The From
// constructors heapify in O(n). Contains and Remove scan linearly in O(n), and
// Sorted runs in O(n log n).
//
// [Heap.All] and [Heap.Slice] expose the internal heap layout: the first
// element is the least, and the order of the rest is unspecified. Use
// [Heap.Sorted] for an ordered snapshot, or [Heap.Pop] and [Heap.PopN] to
// consume elements in priority order. The heap is not stable: elements that
// compare equal come out in unspecified order.
//
// A comparator returns a negative number when a is ordered before b, a
// positive number when a is ordered after b, and zero when they are
// equivalent, the same contract as [slices.SortFunc]. It must be a strict weak
// ordering. An inconsistent comparator breaks the heap order but never memory
// safety.
//
// A nil *Heap[T] behaves as an empty heap for every read-only method (Len,
// Cap, IsEmpty, Peek, All, Slice, Sorted, Contains, Clone) and for removal
// methods (Pop, PopN, PushPop, Remove), so the zero pointer value is safe to
// query without first calling a constructor. [Heap.Clone] of a nil heap
// returns nil, because the comparator cannot be recovered. Methods that store,
// clear, reset, or resize data ([Heap.Push], [Heap.PushN], [Heap.Replace],
// [Heap.Grow], [Heap.Clear], [Heap.Reset], [Heap.Shrink], [Heap.Clip]) require
// a non-nil receiver and panic on a nil one.
//
// The zero value of Heap[T] has no comparator. It behaves as an empty heap for
// reads and removals, but methods that store elements (Push, PushN, Replace)
// panic with a descriptive message. Always create a heap with a constructor.
//
// A Heap is not safe for concurrent use: it performs no internal locking.
// Callers that share a heap across goroutines must provide their own
// synchronization.
package binheap

import (
	"cmp"
	"iter"
	"math/bits"
	"slices"

	"github.com/Nergous/structures/internal/heapcore"
)

// Heap is a generic binary heap backed by a slice in heap order. The top of the
// heap is the least element according to the comparator supplied at
// construction.
//
// A Heap must be created with a constructor such as [New], [NewMax], or
// [NewFunc] before elements are stored. A nil *Heap[T] is treated as a valid
// empty heap by read-only and removal methods only.
//
// Heap is not safe for concurrent use.
type Heap[T any] struct {
	data    []T
	compare func(a, b T) int
}

// New returns an empty min-heap ordered by [cmp.Compare], with no preallocated
// capacity. The smallest value is on top.
func New[T cmp.Ordered]() *Heap[T] {
	return newHeap(0, cmp.Compare[T])
}

// NewWithCap returns an empty min-heap ordered by [cmp.Compare], with a backing
// array preallocated for at least n elements. A negative n is treated as 0. Use
// it when the maximum size is known up front to avoid reallocations while
// pushing.
func NewWithCap[T cmp.Ordered](n int) *Heap[T] {
	return newHeap(n, cmp.Compare[T])
}

// From returns a min-heap ordered by [cmp.Compare] that holds a copy of s. The
// heap never aliases s: later changes to s do not affect the heap, and vice
// versa. The copy is heapified in O(n), which is cheaper than pushing the
// elements one by one. The capacity equals len(s).
func From[T cmp.Ordered](s []T) *Heap[T] {
	return from(s, cmp.Compare[T])
}

// NewMax returns an empty max-heap for [cmp.Ordered] types, with no
// preallocated capacity. The largest value is on top.
func NewMax[T cmp.Ordered]() *Heap[T] {
	return newHeap(0, greater[T])
}

// NewMaxWithCap returns an empty max-heap with a backing array preallocated for
// at least n elements. A negative n is treated as 0.
func NewMaxWithCap[T cmp.Ordered](n int) *Heap[T] {
	return newHeap(n, greater[T])
}

// FromMax returns a max-heap that holds a copy of s, heapified in O(n). The heap
// never aliases s. The capacity equals len(s).
func FromMax[T cmp.Ordered](s []T) *Heap[T] {
	return from(s, greater[T])
}

// NewFunc returns an empty heap ordered by compare, with no preallocated
// capacity. The element for which compare reports the least order is on top.
// See the package documentation for the comparator contract.
//
// NewFunc panics if compare is nil.
func NewFunc[T any](compare func(a, b T) int) *Heap[T] {
	return newHeap(0, compare)
}

// NewFuncWithCap returns an empty heap ordered by compare, with a backing array
// preallocated for at least n elements. A negative n is treated as 0.
//
// NewFuncWithCap panics if compare is nil.
func NewFuncWithCap[T any](n int, compare func(a, b T) int) *Heap[T] {
	return newHeap(n, compare)
}

// FromFunc returns a heap ordered by compare that holds a copy of s, heapified
// in O(n). The heap never aliases s. The capacity equals len(s).
//
// FromFunc panics if compare is nil.
func FromFunc[T any](s []T, compare func(a, b T) int) *Heap[T] {
	return from(s, compare)
}

// Len returns the number of elements currently in the heap. A nil receiver
// reports 0.
func (h *Heap[T]) Len() int {
	if h == nil {
		return 0
	}
	return len(h.data)
}

// Cap returns the capacity of the heap's backing array, i.e. the number of
// elements it can hold before the next reallocation. A nil receiver reports 0.
func (h *Heap[T]) Cap() int {
	if h == nil {
		return 0
	}
	return cap(h.data)
}

// IsEmpty reports whether the heap has no elements. A nil receiver reports
// true.
func (h *Heap[T]) IsEmpty() bool {
	return h == nil || len(h.data) == 0
}

// Peek returns the least element without removing it. The boolean result is
// false when the heap is empty, in which case the returned value is the zero
// value of T. It runs in O(1). A nil receiver is treated as empty and returns
// (zero, false).
func (h *Heap[T]) Peek() (T, bool) {
	if h == nil || len(h.data) == 0 {
		var zero T
		return zero, false
	}
	return h.data[0], true
}

// All returns an iterator over the heap's elements in internal layout order,
// without consuming them; the heap is left unchanged. The first element yielded
// is the least one; the order of the rest is unspecified and is not sorted. Use
// [Heap.Sorted] for an ordered snapshot.
//
// The returned [iter.Seq] is a Go 1.23 range-over-func iterator. Breaking out of
// the loop early stops iteration cleanly. Mutating the heap during iteration is
// undefined behavior. A nil receiver is treated as empty and yields no
// elements.
func (h *Heap[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if h == nil {
			return
		}
		for _, v := range h.data {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the heap that shares the
// comparator. The two heaps share no backing array, so mutating one does not
// affect the other. Elements are copied as-is: if T is a pointer or contains
// references, the pointed-to data is shared between the copies. The clone's
// capacity equals its length. Runs in O(n).
//
// A nil receiver clones to nil, not to a usable heap: the comparator cannot be
// recovered, and a heap without one would panic on the first push, far from
// the actual bug.
func (h *Heap[T]) Clone() *Heap[T] {
	if h == nil {
		return nil
	}
	return &Heap[T]{
		data:    slices.Clone(h.data),
		compare: h.compare,
	}
}

// Slice returns a copy of the heap's elements in internal layout order, the
// same order [Heap.All] yields: the first element is the least one and the rest
// are in unspecified order. The returned slice is independent of the heap, so
// callers may modify it freely. An empty heap returns nil; a nil receiver is
// treated as empty and likewise returns nil. Runs in O(n).
func (h *Heap[T]) Slice() []T {
	if h == nil || len(h.data) == 0 {
		return nil
	}
	return slices.Clone(h.data)
}

// Sorted returns a copy of the heap's elements sorted from least to greatest by
// the comparator, the order in which repeated [Heap.Pop] calls would return
// them. For a max-heap the result is in descending order. The heap is left
// unchanged, and the returned slice is independent of it. Elements that compare
// equal appear in unspecified order. An empty or nil heap returns nil. Runs in
// O(n log n) and allocates once.
func (h *Heap[T]) Sorted() []T {
	if h == nil || len(h.data) == 0 {
		return nil
	}
	out := slices.Clone(h.data)
	slices.SortFunc(out, h.compare)
	return out
}

// Contains reports whether the heap holds an element that compares equal to v,
// i.e. one for which the comparator returns 0. For a comparator that orders by
// a key, such as a priority field, any element with the same key matches. It
// scans the heap in O(n). A nil receiver is treated as empty and reports false.
func (h *Heap[T]) Contains(v T) bool {
	return h.index(v) >= 0
}

// Push adds v to the heap. It runs in amortized O(log n) time; the backing
// array may be reallocated to grow.
//
// Push requires a heap created with a constructor. It panics on a nil receiver
// and on a zero-value Heap, which has no comparator.
func (h *Heap[T]) Push(v T) {
	h.mustStore("Push")
	h.data = append(h.data, v)
	heapcore.Up(h.data, len(h.data)-1, h.compare)
}

// PushN adds vs to the heap. It grows the backing array at most once, making it
// cheaper than repeated [Heap.Push] calls. With no arguments it is a no-op.
//
// PushN picks the cheaper strategy for the batch: it sifts each new element up
// when the batch is small relative to the heap, and rebuilds the whole heap in
// O(n+k) when the batch is large. The cost is bounded by
// min(O(k log(n+k)), O(n+k)) for k new elements.
//
// PushN requires a heap created with a constructor. It panics on a nil receiver
// and on a zero-value Heap, even with no arguments.
func (h *Heap[T]) PushN(vs ...T) {
	h.mustStore("PushN")
	if len(vs) == 0 {
		return
	}

	n := len(h.data)
	h.data = append(h.data, vs...)
	if rebuildCheaper(n, len(vs)) {
		heapcore.Heapify(h.data, h.compare)
		return
	}
	for i := n; i < len(h.data); i++ {
		heapcore.Up(h.data, i, h.compare)
	}
}

// Pop removes and returns the least element. The boolean result is false when
// the heap is empty, in which case the returned value is the zero value of T.
// The vacated slot is zeroed so it no longer retains a reference to the popped
// element. It runs in O(log n). A nil receiver is treated as empty: there is
// nothing to remove, so it returns (zero, false) without panicking.
func (h *Heap[T]) Pop() (T, bool) {
	if h == nil || len(h.data) == 0 {
		var zero T
		return zero, false
	}
	return h.popRoot(), true
}

// PopN removes up to n least elements and returns them in priority order, the
// same order repeated [Heap.Pop] calls would produce. If n exceeds the length,
// PopN drains the heap. It returns nil when n <= 0 or the heap is empty or nil.
// Vacated slots are zeroed. Runs in O(k log n) for k returned elements and
// allocates the result slice once.
func (h *Heap[T]) PopN(n int) []T {
	if h == nil || n <= 0 || len(h.data) == 0 {
		return nil
	}
	n = min(n, len(h.data))

	out := make([]T, n)
	for i := range out {
		out[i] = h.popRoot()
	}
	return out
}

// PushPop pushes v and then pops the least element, in a single sift. It is
// faster than calling [Heap.Push] followed by [Heap.Pop], and never grows the
// backing array.
//
// When the heap is empty, or v is not greater than the least element, v itself
// is the least element: PushPop returns v and leaves the heap untouched.
// Otherwise v replaces the least element, which is returned. Runs in O(log n).
// A nil receiver is treated as empty and returns v.
func (h *Heap[T]) PushPop(v T) T {
	if h == nil || len(h.data) == 0 || h.compare(v, h.data[0]) <= 0 {
		return v
	}

	old := h.data[0]
	h.data[0] = v
	heapcore.Down(h.data, 0, h.compare)
	return old
}

// Replace pops the least element and then pushes v, in a single sift. Unlike
// [Heap.PushPop], the popped element is the heap's old least element even when
// v is smaller, and v is always stored.
//
// The boolean result reports whether an element was popped. On an empty heap
// Replace stores v and returns (zero, false). Runs in O(log n).
//
// Replace requires a heap created with a constructor. It panics on a nil
// receiver and on a zero-value Heap, which has no comparator.
func (h *Heap[T]) Replace(v T) (T, bool) {
	h.mustStore("Replace")
	if len(h.data) == 0 {
		h.data = append(h.data, v)
		var zero T
		return zero, false
	}

	old := h.data[0]
	h.data[0] = v
	heapcore.Down(h.data, 0, h.compare)
	return old, true
}

// Remove deletes one element that compares equal to v, i.e. one for which the
// comparator returns 0, and reports whether such an element was found. When
// several elements compare equal to v, which one is removed is unspecified. The
// vacated slot is zeroed. It scans the heap in O(n) and restores the heap order
// in O(log n). A nil or empty heap has nothing to remove and returns false.
func (h *Heap[T]) Remove(v T) bool {
	i := h.index(v)
	if i < 0 {
		return false
	}
	h.removeAt(i)
	return true
}

// Clear removes all elements and releases the backing array, so its memory
// becomes eligible for garbage collection. The comparator is kept, so the heap
// stays usable. After Clear, Cap reports 0. To empty the heap while keeping its
// capacity for reuse, use [Heap.Reset] instead.
//
// Clear requires a non-nil receiver and panics on a nil one.
func (h *Heap[T]) Clear() {
	h.mustNotNil("Clear")
	h.data = nil
}

// Reset removes all elements but keeps the backing array for reuse, so Cap is
// preserved. The elements are zeroed so the array no longer retains references
// to them. To also release the backing array, use [Heap.Clear] instead.
//
// Reset requires a non-nil receiver and panics on a nil one.
func (h *Heap[T]) Reset() {
	h.mustNotNil("Reset")
	clear(h.data)
	h.data = h.data[:0]
}

// Shrink reduces the backing array to hold exactly Len elements, copying them
// into a new, right-sized array so the previous (larger) array becomes eligible
// for garbage collection immediately. Use it to reclaim memory after a heap
// that grew large has shrunk and will not grow back.
//
// Shrink runs in O(n) and allocates once; it is a no-op when the capacity
// already equals the length. For the cheaper, non-copying variant that only
// reslices and lets memory be reclaimed lazily on the next growth, use
// [Heap.Clip]. Shrink is the counterpart of [Heap.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (h *Heap[T]) Shrink() {
	h.mustNotNil("Shrink")
	if cap(h.data) == len(h.data) {
		return
	}

	trimmed := make([]T, len(h.data))
	copy(trimmed, h.data)
	h.data = trimmed
}

// Clip reduces the reported capacity to the current length without copying, by
// reslicing the backing array (see [slices.Clip]). It runs in O(1) and does not
// allocate. Because the backing array is retained, the unused tail memory is not
// returned to the garbage collector until a later growth reallocates. For an
// immediate, copying reclaim, use [Heap.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (h *Heap[T]) Clip() {
	h.mustNotNil("Clip")
	h.data = slices.Clip(h.data)
}

// Grow ensures the heap has spare capacity for at least n more elements without
// reallocating during subsequent pushes, growing the backing array if needed. It
// is the counterpart of [Heap.Shrink]. Grow runs in O(n) when it reallocates and
// is a no-op when n is non-positive or the capacity is already sufficient. It
// panics if the new capacity overflows, the same condition as [slices.Grow].
//
// Grow requires a non-nil receiver and panics on a nil one.
func (h *Heap[T]) Grow(n int) {
	h.mustNotNil("Grow")
	if n <= 0 {
		return
	}
	h.data = slices.Grow(h.data, n)
}

// index returns the position of the first element that compares equal to v, or
// -1 when there is none. A nil or empty heap reports -1 without touching the
// comparator.
func (h *Heap[T]) index(v T) int {
	if h == nil {
		return -1
	}
	for i, item := range h.data {
		if h.compare(item, v) == 0 {
			return i
		}
	}
	return -1
}

// popRoot removes and returns the least element. The heap must be non-empty.
func (h *Heap[T]) popRoot() T {
	v := h.data[0]
	h.removeAt(0)
	return v
}

// removeAt removes the element at index i, zeroes the vacated tail slot, and
// restores the heap order around i. The moved element may need to go either
// down or up, depending on the subtree it came from.
func (h *Heap[T]) removeAt(i int) {
	h.data = heapcore.RemoveAt(h.data, i, h.compare)
}

// mustStore panics with a descriptive message when h cannot store elements:
// either h is nil or it is a zero-value Heap without a comparator.
func (h *Heap[T]) mustStore(op string) {
	if h == nil {
		panic("binheap: " + op + " on nil receiver")
	}
	if h.compare == nil {
		panic("binheap: " + op + " on zero-value Heap; create it with New, NewMax, or NewFunc")
	}
}

// mustNotNil panics with a descriptive message when h is nil.
func (h *Heap[T]) mustNotNil(op string) {
	if h == nil {
		panic("binheap: " + op + " on nil receiver")
	}
}

// rebuildCheaper reports whether rebuilding the whole heap after appending k
// elements to n existing ones needs fewer comparisons than sifting each new
// element up. A rebuild costs about 2(n+k) comparisons; k sift-ups cost up to
// k*log2(n). A batch at least as large as the heap always rebuilds. This is the
// heuristic used by Rust's BinaryHeap::append.
func rebuildCheaper(n, k int) bool {
	if k >= n {
		return true
	}
	return 2*(n+k) < k*bits.Len(uint(n))
}

// greater orders values in descending order, turning a min-heap into a
// max-heap.
func greater[T cmp.Ordered](a, b T) int {
	return cmp.Compare(b, a)
}

func newHeap[T any](capacity int, compare func(a, b T) int) *Heap[T] {
	if compare == nil {
		panic("binheap: nil comparator")
	}
	return &Heap[T]{
		data:    make([]T, 0, max(capacity, 0)),
		compare: compare,
	}
}

func from[T any](s []T, compare func(a, b T) int) *Heap[T] {
	if compare == nil {
		panic("binheap: nil comparator")
	}
	data := slices.Clone(s)
	heapcore.Heapify(data, compare)
	return &Heap[T]{
		data:    data,
		compare: compare,
	}
}
