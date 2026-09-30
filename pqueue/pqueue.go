// Package pqueue provides a generic, keyed priority queue.
//
// A [PriorityQueue] stores unique keys, each with a priority, and always hands
// out the key with the least priority first. Throughout this documentation
// "least" means least according to the queue's priority comparator: for a
// queue built with [NewMax] it is the largest priority.
//
// Unlike a plain heap, the queue knows where every key lives. Pushing a key
// that is already queued changes its priority in place, and [PriorityQueue.Remove],
// [PriorityQueue.Priority], and [PriorityQueue.Contains] find a key without a
// scan. This is the shape Dijkstra's algorithm, A*, and schedulers need: lower
// a vertex's distance, bump a task's urgency, or cancel a job by its id.
//
// Queues are created with one of three constructor families:
//
//   - [New] and [NewWithCap] order cmp.Ordered priorities with [cmp.Compare],
//     smallest first.
//   - [NewMax] and [NewMaxWithCap] order cmp.Ordered priorities largest first.
//   - [NewFunc] and [NewFuncWithCap] accept any comparator for any priority
//     type.
//
// Keys only need to be comparable; they are never ordered.
//
// The queue is stable: keys with equal priorities come out in the order they
// were first pushed. Changing a key's priority with [PriorityQueue.Push] keeps
// its original place among equal priorities.
//
// Peek, Len, Cap, IsEmpty, Priority, and Contains run in O(1); Priority and
// Contains are average-case map lookups. Push, Pop, and Remove run in O(log n),
// Push amortized over growth. Sorted runs in O(n log n).
//
// [PriorityQueue.All] and [PriorityQueue.Slice] expose the internal heap
// layout: the first entry is the least, and the order of the rest is
// unspecified. Use [PriorityQueue.Sorted] for an ordered snapshot, or
// [PriorityQueue.Pop] and [PriorityQueue.PopN] to consume entries in priority
// order.
//
// A comparator returns a negative number when a is ordered before b, a positive
// number when a is ordered after b, and zero when they are equivalent, the same
// contract as [slices.SortFunc]. It must be a strict weak ordering. An
// inconsistent comparator breaks the queue order but never memory safety.
//
// A nil *PriorityQueue[K, P] behaves as an empty queue for every read-only
// method (Len, Cap, IsEmpty, Peek, Priority, Contains, All, Slice, Sorted,
// Clone) and for removal methods (Pop, PopN, Remove), so the zero pointer value
// is safe to query without first calling a constructor.
// [PriorityQueue.Clone] of a nil queue returns nil, because the comparator
// cannot be recovered. Methods that store, clear, reset, or resize data
// ([PriorityQueue.Push], [PriorityQueue.Grow], [PriorityQueue.Clear],
// [PriorityQueue.Reset], [PriorityQueue.Shrink], [PriorityQueue.Clip]) require
// a non-nil receiver and panic on a nil one.
//
// The zero value of PriorityQueue[K, P] has no comparator. It behaves as an
// empty queue for reads and removals, but [PriorityQueue.Push] panics with a
// descriptive message. Always create a queue with a constructor.
//
// A PriorityQueue is not safe for concurrent use: it performs no internal
// locking. Callers that share a queue across goroutines must provide their own
// synchronization.
package pqueue

import (
	"cmp"
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/heapcore"
)

// Item is a key with its priority, as returned by [PriorityQueue.PopN],
// [PriorityQueue.Slice], and [PriorityQueue.Sorted].
type Item[K comparable, P any] struct {
	Key      K
	Priority P
}

// entry is one element of the heap. seq is the push order of the key and
// breaks ties between equal priorities, which makes the queue stable.
type entry[K comparable, P any] struct {
	key      K
	priority P
	seq      uint64
}

// PriorityQueue is a generic priority queue of unique keys, backed by a binary
// heap in a slice and a map from each key to its heap position. The key with
// the least priority according to the comparator supplied at construction is
// on top.
//
// A PriorityQueue must be created with a constructor such as [New], [NewMax],
// or [NewFunc] before keys are stored. A nil *PriorityQueue[K, P] is treated as
// a valid empty queue by read-only and removal methods only.
//
// PriorityQueue is not safe for concurrent use.
type PriorityQueue[K comparable, P any] struct {
	data    []entry[K, P]
	pos     map[K]int
	compare func(a, b entry[K, P]) int
	moved   func(s []entry[K, P], i int)
	nextSeq uint64
}

// New returns an empty queue ordered by [cmp.Compare] on priorities, with no
// preallocated capacity. The key with the smallest priority is on top.
func New[K comparable, P cmp.Ordered]() *PriorityQueue[K, P] {
	return newQueue[K, P](0, cmp.Compare)
}

// NewWithCap returns an empty queue ordered by [cmp.Compare] on priorities,
// with room for at least n keys preallocated. A negative n is treated as 0.
// Use it when the maximum size is known up front to avoid reallocations while
// pushing.
func NewWithCap[K comparable, P cmp.Ordered](n int) *PriorityQueue[K, P] {
	return newQueue[K, P](n, cmp.Compare)
}

// NewMax returns an empty queue for cmp.Ordered priorities, with no
// preallocated capacity. The key with the largest priority is on top.
func NewMax[K comparable, P cmp.Ordered]() *PriorityQueue[K, P] {
	return newQueue[K, P](0, greater)
}

// NewMaxWithCap returns an empty queue that puts the largest priority on top,
// with room for at least n keys preallocated. A negative n is treated as 0.
func NewMaxWithCap[K comparable, P cmp.Ordered](n int) *PriorityQueue[K, P] {
	return newQueue[K, P](n, greater)
}

// NewFunc returns an empty queue whose priorities are ordered by compare, with
// no preallocated capacity. The key whose priority compare reports as least is
// on top. See the package documentation for the comparator contract.
//
// NewFunc panics if compare is nil.
func NewFunc[K comparable, P any](compare func(a, b P) int) *PriorityQueue[K, P] {
	return newQueue[K](0, compare)
}

// NewFuncWithCap returns an empty queue whose priorities are ordered by
// compare, with room for at least n keys preallocated. A negative n is treated
// as 0.
//
// NewFuncWithCap panics if compare is nil.
func NewFuncWithCap[K comparable, P any](n int, compare func(a, b P) int) *PriorityQueue[K, P] {
	return newQueue[K](n, compare)
}

// Len returns the number of keys currently in the queue. A nil receiver
// reports 0.
func (q *PriorityQueue[K, P]) Len() int {
	if q == nil {
		return 0
	}
	return len(q.data)
}

// Cap returns the capacity of the queue's backing array, i.e. the number of
// keys it can hold before the next reallocation. A nil receiver reports 0.
func (q *PriorityQueue[K, P]) Cap() int {
	if q == nil {
		return 0
	}
	return cap(q.data)
}

// IsEmpty reports whether the queue has no keys. A nil receiver reports true.
func (q *PriorityQueue[K, P]) IsEmpty() bool {
	return q == nil || len(q.data) == 0
}

// Peek returns the key with the least priority, and that priority, without
// removing it. The boolean result is false when the queue is empty, in which
// case the key and priority are zero values. It runs in O(1). A nil receiver
// is treated as empty.
func (q *PriorityQueue[K, P]) Peek() (K, P, bool) {
	if q == nil || len(q.data) == 0 {
		var (
			zeroK K
			zeroP P
		)
		return zeroK, zeroP, false
	}
	e := q.data[0]
	return e.key, e.priority, true
}

// Priority returns the current priority of key. The boolean result is false
// when key is not in the queue, in which case the priority is the zero value.
// It runs in O(1) on average. A nil receiver is treated as empty.
func (q *PriorityQueue[K, P]) Priority(key K) (P, bool) {
	if q == nil {
		var zero P
		return zero, false
	}
	i, ok := q.pos[key]
	if !ok {
		var zero P
		return zero, false
	}
	return q.data[i].priority, true
}

// Contains reports whether key is in the queue. It runs in O(1) on average.
// A nil receiver is treated as empty and reports false.
func (q *PriorityQueue[K, P]) Contains(key K) bool {
	if q == nil {
		return false
	}
	_, ok := q.pos[key]
	return ok
}

// All returns an iterator over the queue's keys and priorities in internal
// layout order, without consuming them; the queue is left unchanged. The first
// pair yielded has the least priority; the order of the rest is unspecified
// and is not sorted. Use [PriorityQueue.Sorted] for an ordered snapshot.
//
// The returned [iter.Seq2] is a Go 1.23 range-over-func iterator. Breaking out
// of the loop early stops iteration cleanly. Mutating the queue during
// iteration is undefined behavior. A nil receiver is treated as empty and
// yields nothing.
func (q *PriorityQueue[K, P]) All() iter.Seq2[K, P] {
	return func(yield func(K, P) bool) {
		if q == nil {
			return
		}
		for _, e := range q.data {
			if !yield(e.key, e.priority) {
				return
			}
		}
	}
}

// Slice returns a copy of the queue's keys and priorities in internal layout
// order, the same order [PriorityQueue.All] yields: the first item has the
// least priority and the rest are in unspecified order. The returned slice is
// independent of the queue. An empty or nil queue returns nil. Runs in O(n).
func (q *PriorityQueue[K, P]) Slice() []Item[K, P] {
	if q == nil || len(q.data) == 0 {
		return nil
	}
	return items(q.data)
}

// Sorted returns a copy of the queue's keys and priorities in priority order,
// the order in which repeated [PriorityQueue.Pop] calls would return them,
// including the first-pushed-first order of equal priorities. The queue is
// left unchanged. An empty or nil queue returns nil. Runs in O(n log n) and
// allocates twice.
func (q *PriorityQueue[K, P]) Sorted() []Item[K, P] {
	if q == nil || len(q.data) == 0 {
		return nil
	}
	sorted := slices.Clone(q.data)
	slices.SortFunc(sorted, q.compare)
	return items(sorted)
}

// Clone returns an independent copy of the queue that shares the comparator.
// The two queues share no backing storage, so mutating one does not affect the
// other. Keys and priorities are copied as-is: if they are pointers or contain
// references, the pointed-to data is shared between the copies. The clone's
// capacity equals its length. Runs in O(n).
//
// A nil receiver clones to nil, not to a usable queue: the comparator cannot be
// recovered.
func (q *PriorityQueue[K, P]) Clone() *PriorityQueue[K, P] {
	if q == nil {
		return nil
	}
	c := &PriorityQueue[K, P]{
		data:    slices.Clone(q.data),
		compare: q.compare,
		nextSeq: q.nextSeq,
	}
	if q.compare == nil {
		return c
	}
	c.pos = positions(c.data)
	c.bindMoved()
	return c
}

// Push adds key with the given priority and reports whether key was new.
//
// If key is already in the queue, Push changes its priority in place and
// returns false; the key keeps its original place among equal priorities.
// Otherwise it stores key and returns true. It runs in O(log n), amortized
// over growth of the backing array.
//
// Push requires a queue created with a constructor. It panics on a nil
// receiver and on a zero-value PriorityQueue, which has no comparator.
func (q *PriorityQueue[K, P]) Push(key K, priority P) bool {
	q.mustStore("Push")
	if i, ok := q.pos[key]; ok {
		q.data[i].priority = priority
		heapcore.FixMoved(q.data, i, q.compare, q.moved)
		return false
	}

	i := len(q.data)
	q.data = append(q.data, entry[K, P]{
		key:      key,
		priority: priority,
		seq:      q.nextSeq,
	})
	q.nextSeq++
	q.pos[key] = i
	heapcore.UpMoved(q.data, i, q.compare, q.moved)
	return true
}

// Pop removes the key with the least priority and returns it with its
// priority. The boolean result is false when the queue is empty, in which case
// the key and priority are zero values. The vacated slot is zeroed so it no
// longer retains references. It runs in O(log n). A nil receiver is treated as
// empty.
func (q *PriorityQueue[K, P]) Pop() (K, P, bool) {
	if q == nil || len(q.data) == 0 {
		var (
			zeroK K
			zeroP P
		)
		return zeroK, zeroP, false
	}
	e := q.removeAt(0)
	return e.key, e.priority, true
}

// PopN removes up to n keys with the least priorities and returns them in
// priority order, the same order repeated [PriorityQueue.Pop] calls would
// produce. If n exceeds the length, PopN drains the queue. It returns nil when
// n <= 0 or the queue is empty or nil. Runs in O(k log n) for k returned items
// and allocates the result slice once.
func (q *PriorityQueue[K, P]) PopN(n int) []Item[K, P] {
	if q == nil || n <= 0 || len(q.data) == 0 {
		return nil
	}
	out := make([]Item[K, P], min(n, len(q.data)))
	for i := range out {
		e := q.removeAt(0)
		out[i] = Item[K, P]{
			Key:      e.key,
			Priority: e.priority,
		}
	}
	return out
}

// Remove deletes key from the queue and returns its priority. The boolean
// result reports whether key was present; when it is false, the priority is
// the zero value. It runs in O(log n). A nil or empty queue has nothing to
// remove and returns false.
func (q *PriorityQueue[K, P]) Remove(key K) (P, bool) {
	if q == nil {
		var zero P
		return zero, false
	}
	i, ok := q.pos[key]
	if !ok {
		var zero P
		return zero, false
	}
	return q.removeAt(i).priority, true
}

// Clear removes all keys and releases the backing storage, so its memory
// becomes eligible for garbage collection. The comparator is kept, so the
// queue stays usable. After Clear, Cap reports 0. To empty the queue while
// keeping its capacity for reuse, use [PriorityQueue.Reset] instead.
//
// Clear requires a non-nil receiver and panics on a nil one.
func (q *PriorityQueue[K, P]) Clear() {
	q.mustNotNil("Clear")
	q.data = nil
	q.pos = make(map[K]int)
	q.nextSeq = 0
}

// Reset removes all keys but keeps the backing storage for reuse, so Cap is
// preserved. The vacated slots are zeroed so they no longer retain references.
// To also release the storage, use [PriorityQueue.Clear] instead.
//
// Reset requires a non-nil receiver and panics on a nil one.
func (q *PriorityQueue[K, P]) Reset() {
	q.mustNotNil("Reset")
	clear(q.data)
	q.data = q.data[:0]
	clear(q.pos)
	q.nextSeq = 0
}

// Grow ensures the backing array has spare capacity for at least n more keys
// without reallocating during subsequent pushes. The key index map grows on
// demand and is not preallocated by Grow; the WithCap constructors size both.
// Grow is a no-op when n is non-positive or the capacity is already
// sufficient. It panics if the new capacity overflows, the same condition as
// [slices.Grow].
//
// Grow requires a non-nil receiver and panics on a nil one.
func (q *PriorityQueue[K, P]) Grow(n int) {
	q.mustNotNil("Grow")
	if n <= 0 {
		return
	}
	q.data = slices.Grow(q.data, n)
}

// Shrink reduces the backing storage to hold exactly Len keys. It copies the
// heap into a right-sized array and rebuilds the key index map, so memory held
// by a queue that grew large and has since shrunk becomes eligible for garbage
// collection immediately. Go maps never shrink in place, so Shrink is the only
// way to reclaim the index map's memory short of [PriorityQueue.Clear].
//
// Shrink runs in O(n) and allocates. The array is copied only when its
// capacity exceeds the length; the index map is always rebuilt, because its
// size cannot be observed. For the cheaper variant that only reslices the
// backing array, use [PriorityQueue.Clip].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (q *PriorityQueue[K, P]) Shrink() {
	q.mustNotNil("Shrink")
	if cap(q.data) != len(q.data) {
		trimmed := make([]entry[K, P], len(q.data))
		copy(trimmed, q.data)
		q.data = trimmed
	}
	q.pos = positions(q.data)
}

// Clip reduces the reported capacity of the backing array to the current
// length without copying (see [slices.Clip]). It runs in O(1) and does not
// allocate. The unused tail of the array, and the key index map, are not
// returned to the garbage collector; use [PriorityQueue.Shrink] for that.
//
// Clip requires a non-nil receiver and panics on a nil one.
func (q *PriorityQueue[K, P]) Clip() {
	q.mustNotNil("Clip")
	q.data = slices.Clip(q.data)
}

// removeAt removes the entry at heap index i, drops its key from the index,
// and returns it. i must be a valid index.
func (q *PriorityQueue[K, P]) removeAt(i int) entry[K, P] {
	e := q.data[i]
	delete(q.pos, e.key)
	q.data = heapcore.RemoveAtMoved(q.data, i, q.compare, q.moved)
	return e
}

// bindMoved installs the position hook for q. The hook captures q itself, so
// every queue, including a clone, needs its own.
func (q *PriorityQueue[K, P]) bindMoved() {
	q.moved = func(s []entry[K, P], i int) {
		q.pos[s[i].key] = i
	}
}

// mustNotNil panics with a descriptive message when q is nil.
func (q *PriorityQueue[K, P]) mustNotNil(op string) {
	if q == nil {
		panic("pqueue: " + op + " on nil receiver")
	}
}

// mustStore panics with a descriptive message when q cannot store keys:
// either q is nil or it is a zero-value PriorityQueue without a comparator.
func (q *PriorityQueue[K, P]) mustStore(op string) {
	q.mustNotNil(op)
	if q.compare == nil {
		panic("pqueue: " + op + " on zero-value PriorityQueue; create it with New, NewMax, or NewFunc")
	}
}

func newQueue[K comparable, P any](capacity int, compare func(a, b P) int) *PriorityQueue[K, P] {
	if compare == nil {
		panic("pqueue: nil comparator")
	}
	capacity = max(capacity, 0)
	q := &PriorityQueue[K, P]{
		data: make([]entry[K, P], 0, capacity),
		pos:  make(map[K]int, capacity),
		compare: func(a, b entry[K, P]) int {
			if c := compare(a.priority, b.priority); c != 0 {
				return c
			}
			return cmp.Compare(a.seq, b.seq)
		},
	}
	q.bindMoved()
	return q
}

// positions builds a key index for s.
func positions[K comparable, P any](s []entry[K, P]) map[K]int {
	pos := make(map[K]int, len(s))
	for i, e := range s {
		pos[e.key] = i
	}
	return pos
}

// items copies s into a slice of Items in the same order.
func items[K comparable, P any](s []entry[K, P]) []Item[K, P] {
	out := make([]Item[K, P], len(s))
	for i, e := range s {
		out[i] = Item[K, P]{
			Key:      e.key,
			Priority: e.priority,
		}
	}
	return out
}

// greater orders priorities in descending order, putting the largest on top.
func greater[P cmp.Ordered](a, b P) int {
	return cmp.Compare(b, a)
}
