// Package pqueue provides a generic, heap-backed keyed priority queue.
//
// Create a priority queue with one of three constructor families:
//
//   - [New], [NewWithCap], and [From] order [cmp.Ordered] priorities with
//     [cmp.Compare], smallest first.
//   - [NewMax], [NewMaxWithCap], and [FromMax] order [cmp.Ordered] priorities
//     largest first.
//   - [NewFunc], [NewFuncWithCap], and [FromFunc] accept any comparator for
//     any priority type.
//
// Keys are added with [Queue.Push] or [Queue.PushN] and removed in priority
// order with [Queue.Pop] or [Queue.PopN]; [Queue.Peek] inspects the top key
// without removing it. Keys only need to be comparable; they are never
// ordered.
//
// A [Queue] stores unique keys, each with a priority. Unlike a plain heap, it
// knows where every key lives. Pushing a key that is already queued changes
// its priority in place, and [Queue.Contains], [Queue.Priority], and
// [Queue.Remove] find a key without a scan. This is the shape Dijkstra's
// algorithm, A*, and schedulers need: lower a vertex's distance, bump a task's
// urgency, or cancel a job by its id.
//
// Peek, Len, Cap, and IsEmpty run in O(1), and Contains and Priority run in
// O(1) on average. Push runs in amortized O(log n), and Pop and Remove run in
// O(log n). PushN runs in amortized O(k log(n+k)), where k is the number of
// items pushed. The From constructors heapify in O(n). Sorted runs in
// O(n log n). Methods that copy or iterate over every key, such as All,
// Clone, and Slice, run in O(n).
//
// A priority queue keeps the key with the least priority, as defined by its
// comparator, on top. Throughout this documentation "least" always means least
// according to the queue's priority comparator: for a queue built with
// [NewMax] it is the largest priority. [Queue.All] and [Queue.Slice] expose
// the internal heap layout: the first key is the least, and the order of the
// rest is unspecified. Use [Queue.Sorted] for an ordered snapshot, or
// [Queue.Pop] and [Queue.PopN] to consume keys in priority order. The queue is
// stable: keys with equal priorities come out in the order they were first
// pushed, and changing a key's priority with [Queue.Push] keeps its original
// place among them.
//
// A comparator returns a negative number when a is ordered before b, a
// positive number when a is ordered after b, and zero when they are
// equivalent, the same contract as [slices.SortFunc]. It must be a strict weak
// ordering. An inconsistent comparator breaks the queue order but never memory
// safety.
//
// The zero value of Queue[K, P] has no comparator. It behaves as an empty
// priority queue for reads and removals, but methods that store keys
// ([Queue.Push], [Queue.PushN]) panic with a descriptive message. Always
// create a priority queue with a constructor.
//
// A nil *Queue[K, P] behaves as an empty priority queue for every read-only
// method and for removal methods such as [Queue.Pop], so the zero pointer
// value is safe to query without first calling [New]. [Queue.Clone] of a nil
// priority queue returns nil, because the comparator cannot be recovered.
// Methods that store, clear, reset, or resize data ([Queue.Push],
// [Queue.PushN], [Queue.Clear], [Queue.Reset], [Queue.Shrink], [Queue.Clip],
// [Queue.Grow]) require a non-nil receiver and panic on a nil one; see each
// method's documentation.
//
// A Queue is not safe for concurrent use: it performs no internal locking.
// Callers that share a priority queue across goroutines must provide their
// own synchronization.
package pqueue

import (
	"cmp"
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/heapcore"
	"github.com/Nergous/structures/internal/shared"
)

// entry is one element of the heap. seq is the push order of the key and
// breaks ties between equal priorities, which makes the queue stable.
type entry[K comparable, P any] struct {
	key      K
	priority P
	seq      uint64
}

// Queue is a generic keyed priority queue backed by a binary heap in a slice
// and a map from each key to its heap position. The key with the least
// priority according to the comparator supplied at construction is on top.
//
// The zero value is not usable: it has no comparator, so create a Queue with a
// constructor such as [New], [NewMax], or [NewFunc] before storing keys. A nil
// *Queue[K, P] is treated as a valid empty priority queue by read-only and
// removal methods, but methods that store, clear, reset, or resize data
// require a non-nil receiver. Copying a Queue value shares its storage with
// the original; use [Queue.Clone] for an independent copy.
//
// Queue is not safe for concurrent use.
type Queue[K comparable, P any] struct {
	data    []entry[K, P]                // heap order; data[0] is the least
	index   map[K]int                    // key -> position in data
	compare func(a, b entry[K, P]) int   // ordering; nil only in the zero value
	moved   func(s []entry[K, P], i int) // records relocated entries in index
	nextSeq uint64                       // seq of the next new key
}

// Item is a key with its priority, as accepted by [From], [FromMax],
// [FromFunc], and [Queue.PushN] and returned by [Queue.Slice],
// [Queue.Sorted], and [Queue.PopN].
type Item[K comparable, P any] struct {
	Key      K
	Priority P
}

// New returns an empty priority queue with no preallocated capacity, ordered
// by [cmp.Compare] on priorities. The key with the smallest priority is on
// top.
func New[K comparable, P cmp.Ordered]() *Queue[K, P] {
	return newQueue[K](0, cmp.Compare[P])
}

// NewWithCap returns an empty priority queue ordered by [cmp.Compare] on
// priorities, with a backing array and an index preallocated for at least n
// keys. Use it when the maximum size is known up front to avoid reallocations
// while pushing. A negative n is treated as 0.
func NewWithCap[K comparable, P cmp.Ordered](n int) *Queue[K, P] {
	return newQueue[K](n, cmp.Compare[P])
}

// From returns a priority queue ordered by [cmp.Compare] on priorities that
// holds the keys and priorities of items, heapified in O(n), which is cheaper
// than pushing them one by one. A key that appears more than once keeps the
// priority of its last occurrence and the place among equal priorities of its
// first, exactly as if the items were pushed one by one with [Queue.Push]. The
// capacity equals len(items). The priority queue never aliases items: later
// changes to items do not affect the priority queue, and vice versa. It runs
// in O(n).
func From[K comparable, P cmp.Ordered](items []Item[K, P]) *Queue[K, P] {
	return from(items, cmp.Compare[P])
}

// NewMax returns an empty priority queue with no preallocated capacity, for
// [cmp.Ordered] priorities. The key with the largest priority is on top.
func NewMax[K comparable, P cmp.Ordered]() *Queue[K, P] {
	return newQueue[K](0, shared.Greater[P])
}

// NewMaxWithCap returns an empty priority queue for [cmp.Ordered] priorities,
// largest first, with a backing array and an index preallocated for at least n
// keys. Use it when the maximum size is known up front to avoid reallocations
// while pushing. A negative n is treated as 0.
func NewMaxWithCap[K comparable, P cmp.Ordered](n int) *Queue[K, P] {
	return newQueue[K](n, shared.Greater[P])
}

// FromMax returns a priority queue for [cmp.Ordered] priorities, largest
// first, that holds the keys and priorities of items, heapified in O(n).
// Duplicate keys are resolved as in [From]. The capacity equals len(items).
// The priority queue never aliases items: later changes to items do not affect
// the priority queue, and vice versa. It runs in O(n).
func FromMax[K comparable, P cmp.Ordered](items []Item[K, P]) *Queue[K, P] {
	return from(items, shared.Greater[P])
}

// NewFunc returns an empty priority queue with no preallocated capacity,
// ordered by compare on priorities. The key whose priority compare reports as
// least is on top. See the package documentation for the comparator contract.
//
// NewFunc panics if compare is nil.
func NewFunc[K comparable, P any](compare func(a, b P) int) *Queue[K, P] {
	return newQueue[K](0, compare)
}

// NewFuncWithCap returns an empty priority queue ordered by compare on
// priorities, with a backing array and an index preallocated for at least n
// keys. Use it when the maximum size is known up front to avoid reallocations
// while pushing. A negative n is treated as 0.
//
// NewFuncWithCap panics if compare is nil.
func NewFuncWithCap[K comparable, P any](
	n int,
	compare func(a, b P) int,
) *Queue[K, P] {
	return newQueue[K](n, compare)
}

// FromFunc returns a priority queue ordered by compare on priorities that
// holds the keys and priorities of items, heapified in O(n). Duplicate keys
// are resolved as in [From]. The capacity equals len(items). The priority
// queue never aliases items: later changes to items do not affect the
// priority queue, and vice versa. It runs in O(n).
//
// FromFunc panics if compare is nil.
func FromFunc[K comparable, P any](
	items []Item[K, P],
	compare func(a, b P) int,
) *Queue[K, P] {
	return from(items, compare)
}

// Len returns the number of keys currently in the priority queue. A nil
// receiver reports 0.
func (q *Queue[K, P]) Len() int {
	if q == nil {
		return 0
	}

	return len(q.data)
}

// Cap returns the capacity of the priority queue's backing array, i.e. the
// number of keys it can hold before the next reallocation. A nil receiver
// reports 0.
func (q *Queue[K, P]) Cap() int {
	if q == nil {
		return 0
	}

	return cap(q.data)
}

// IsEmpty reports whether the priority queue has no keys. A nil receiver
// reports true.
func (q *Queue[K, P]) IsEmpty() bool {
	return q == nil || len(q.data) == 0
}

// Peek returns the key with the least priority, and that priority, without
// removing it. The boolean result is false when the priority queue is empty,
// in which case the returned key and priority are zero values. It runs in
// O(1). A nil receiver is treated as empty and returns (zero, zero, false).
func (q *Queue[K, P]) Peek() (K, P, bool) {
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

// Contains reports whether key is present in the priority queue. It runs in
// O(1) on average. A nil receiver is treated as empty and reports false.
func (q *Queue[K, P]) Contains(key K) bool {
	if q == nil {
		return false
	}

	_, ok := q.index[key]

	return ok
}

// Priority returns the current priority of key. The boolean result is false
// when key is not in the priority queue, in which case the returned priority
// is the zero value of P. It runs in O(1) on average. A nil receiver is
// treated as empty and returns (zero, false).
func (q *Queue[K, P]) Priority(key K) (P, bool) {
	if q == nil {
		var zero P
		return zero, false
	}

	i, ok := q.index[key]
	if !ok {
		var zero P
		return zero, false
	}

	return q.data[i].priority, true
}

// All returns an iterator over the priority queue's keys and priorities in
// internal layout order, without consuming them; the priority queue is left
// unchanged. The first pair yielded has the least priority; the order of the
// rest is unspecified and is not sorted. Use [Queue.Sorted] for an ordered
// snapshot. The returned [iter.Seq2] is a Go 1.23 range-over-func iterator, so
// it can be used directly in a range loop. Breaking out of the loop early
// stops iteration cleanly. Mutating the priority queue during iteration is
// undefined behavior. A nil receiver is treated as empty and yields nothing.
func (q *Queue[K, P]) All() iter.Seq2[K, P] {
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

// Clone returns an independent shallow copy of the priority queue that shares
// the comparator. The two priority queues share no backing storage, so
// mutating one does not affect the other. Keys and priorities are copied
// as-is: if K or P is a pointer or contains references, the pointed-to data
// is shared between the copies. The clone's capacity equals its length. It
// runs in O(n).
//
// A nil receiver clones to nil: a nil priority queue has no comparator to
// copy.
func (q *Queue[K, P]) Clone() *Queue[K, P] {
	if q == nil {
		return nil
	}

	c := &Queue[K, P]{
		data:    make([]entry[K, P], len(q.data)),
		compare: q.compare,
		nextSeq: q.nextSeq,
	}
	copy(c.data, q.data)

	if q.compare == nil {
		return c
	}

	c.index = rebuildIndex(c.data)
	c.bindMoved()

	return c
}

// Slice returns a copy of the priority queue's keys and priorities in internal
// layout order, the same order [Queue.All] yields: the first item has the
// least priority and the rest are in unspecified order. The returned slice is
// independent of the priority queue, so callers may modify it freely. An
// empty priority queue returns nil; a nil receiver is treated as empty and
// likewise returns nil. It runs in O(n).
func (q *Queue[K, P]) Slice() []Item[K, P] {
	if q == nil || len(q.data) == 0 {
		return nil
	}

	return items(q.data)
}

// Sorted returns a copy of the priority queue's keys and priorities in
// priority order, the order in which repeated [Queue.Pop] calls would return
// them, including the first-pushed-first order of equal priorities. The
// priority queue is left unchanged. The returned slice is independent of the
// priority queue, so callers may modify it freely. An empty priority queue
// returns nil; a nil receiver is treated as empty and likewise returns nil.
// It runs in O(n log n) and allocates twice.
func (q *Queue[K, P]) Sorted() []Item[K, P] {
	if q == nil || len(q.data) == 0 {
		return nil
	}

	sorted := slices.Clone(q.data)
	slices.SortFunc(sorted, q.compare)

	return items(sorted)
}

// Push adds key with the given priority and reports whether key was new. If
// key is already in the priority queue, Push changes its priority in place
// and returns false; the key keeps its original place among equal priorities.
// It runs in amortized O(log n); the backing array may be reallocated to grow.
//
// Push requires a priority queue created with a constructor. It panics on a
// nil receiver and on a zero-value Queue, which has no comparator.
func (q *Queue[K, P]) Push(key K, priority P) bool {
	q.mustStore("Push")

	return q.push(key, priority)
}

// PushN pushes items in argument order with the semantics of [Queue.Push]: a
// new key is added, and a key already in the priority queue, or repeated
// within items, has its priority changed in place, so its last occurrence
// wins. It grows the backing array at most once, making it cheaper than
// repeated [Queue.Push] calls. With no arguments it is a no-op. It runs in
// amortized O(k log(n+k)), where k is the number of items pushed.
//
// PushN requires a priority queue created with a constructor, even when called
// with no arguments. It panics on a nil receiver and on a zero-value Queue,
// which has no comparator.
func (q *Queue[K, P]) PushN(items ...Item[K, P]) {
	q.mustStore("PushN")

	if len(items) == 0 {
		return
	}

	q.data = shared.GrowSlice(q.data, len(items))
	for _, it := range items {
		q.push(it.Key, it.Priority)
	}
}

// Pop removes the key with the least priority and returns it with its
// priority. The boolean result is false when the priority queue is empty, in
// which case the returned key and priority are zero values. The vacated slot
// is zeroed so it no longer retains a reference to the removed key. It runs in
// O(log n). A nil receiver is treated as empty and returns (zero, zero,
// false).
func (q *Queue[K, P]) Pop() (K, P, bool) {
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

// PopN removes and returns up to n keys from the top of the priority queue in
// priority order, the same order repeated [Queue.Pop] calls would produce. If
// n is greater than Len, it drains the priority queue. If n is non-positive,
// or the priority queue is empty or nil, PopN returns nil. The vacated slots
// are zeroed so they no longer retain references, and the capacity is
// preserved. It runs in O(k log n), where k is the number of keys returned,
// and allocates the result slice once.
func (q *Queue[K, P]) PopN(n int) []Item[K, P] {
	if q == nil || n <= 0 || len(q.data) == 0 {
		return nil
	}

	n = min(n, len(q.data))
	out := make([]Item[K, P], n)
	for i := range out {
		e := q.removeAt(0)
		out[i] = Item[K, P]{
			Key:      e.key,
			Priority: e.priority,
		}
	}

	return out
}

// Remove deletes key from the priority queue and returns its priority. The
// boolean result reports whether key was present; when it is false, the
// returned priority is the zero value of P. The vacated slot is zeroed so it
// no longer retains a reference to the removed key. It runs in O(log n). A nil
// receiver is treated as empty and returns (zero, false).
func (q *Queue[K, P]) Remove(key K) (P, bool) {
	if q == nil {
		var zero P
		return zero, false
	}

	i, ok := q.index[key]
	if !ok {
		var zero P
		return zero, false
	}

	return q.removeAt(i).priority, true
}

// Clear removes all keys and releases the backing storage, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To
// empty the priority queue while keeping its capacity for reuse, use
// [Queue.Reset] instead. The comparator is kept, so the priority queue stays
// usable. It runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (q *Queue[K, P]) Clear() {
	shared.MustNotNil(q, "pqueue", "Clear")
	q.data = nil
	q.index = nil
	q.nextSeq = 0
}

// Reset removes all keys but keeps the backing storage for reuse, so Cap is
// preserved. The keys are zeroed so the storage no longer retains references
// to them. To also release the backing storage, use [Queue.Clear] instead. It
// runs in O(n).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (q *Queue[K, P]) Reset() {
	shared.MustNotNil(q, "pqueue", "Reset")
	clear(q.data)
	q.data = q.data[:0]
	clear(q.index)
	q.nextSeq = 0
}

// Shrink reduces the backing array to hold exactly Len keys, copying them into
// a new, right-sized array, and rebuilds the index, so memory held by a
// priority queue that grew large and has since shrunk becomes eligible for
// garbage collection immediately. Go maps never shrink in place, so Shrink is
// the only way to reclaim the index's memory short of [Queue.Clear].
//
// Shrink runs in O(n) and allocates; the backing array is copied only when it
// has spare capacity, but the index is always rebuilt. For the cheaper,
// non-copying variant that only reslices the backing array and lets memory be
// reclaimed lazily on the next growth, use [Queue.Clip]. Shrink is the
// counterpart of [Queue.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (q *Queue[K, P]) Shrink() {
	shared.MustNotNil(q, "pqueue", "Shrink")
	q.data = shared.ShrinkSlice(q.data)
	q.index = rebuildIndex(q.data)
}

// Clip reduces the reported capacity to the current length without copying,
// by reslicing the backing array (see [slices.Clip]). It runs in O(1) and does
// not allocate. The index is left as it is, and because the backing array is
// retained, the unused tail memory is not returned to the garbage collector
// until a later growth reallocates. For an immediate, copying reclaim, use
// [Queue.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (q *Queue[K, P]) Clip() {
	shared.MustNotNil(q, "pqueue", "Clip")
	q.data = slices.Clip(q.data)
}

// Grow ensures the priority queue has spare capacity for at least n more keys
// without reallocating during subsequent pushes, growing the backing array if
// needed. It is the counterpart of [Queue.Shrink]. Grow runs in O(n) when it
// reallocates and is a no-op when n is non-positive or the capacity is
// already sufficient. It panics if the new capacity overflows, the same
// condition as [slices.Grow]. The index map is not preallocated by Grow;
// [NewWithCap] sizes both.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (q *Queue[K, P]) Grow(n int) {
	shared.MustNotNil(q, "pqueue", "Grow")
	q.data = shared.GrowSlice(q.data, n)
}

// push stores key with priority, or changes the priority of key in place when
// it is already queued, and reports whether key was new. The index is
// allocated lazily, so push works after [Queue.Clear]. q must have a
// comparator.
func (q *Queue[K, P]) push(key K, priority P) bool {
	if i, ok := q.index[key]; ok {
		q.data[i].priority = priority
		heapcore.FixMoved(q.data, i, q.compare, q.moved)
		return false
	}

	if q.index == nil {
		q.index = make(map[K]int)
	}

	i := len(q.data)
	q.index[key] = i
	q.data = append(q.data, entry[K, P]{
		key:      key,
		priority: priority,
		seq:      q.nextSeq,
	})
	q.nextSeq++
	heapcore.UpMoved(q.data, i, q.compare, q.moved)

	return true
}

// removeAt removes the entry at heap position i, drops its key from the index,
// and returns it. i must be a valid index.
func (q *Queue[K, P]) removeAt(i int) entry[K, P] {
	e := q.data[i]
	delete(q.index, e.key)
	q.data = heapcore.RemoveAtMoved(q.data, i, q.compare, q.moved)

	return e
}

// bindMoved installs the position hook for q. The hook captures q itself, so
// every queue, including a clone, needs its own.
func (q *Queue[K, P]) bindMoved() {
	q.moved = func(s []entry[K, P], i int) {
		q.index[s[i].key] = i
	}
}

// mustStore panics with a descriptive message when q cannot store keys:
// either q is nil or it is a zero-value Queue without a comparator.
func (q *Queue[K, P]) mustStore(op string) {
	shared.MustNotNil(q, "pqueue", op)

	if q.compare == nil {
		panic("pqueue: " + op + " on zero-value Queue; " +
			"create it with New, NewMax, or NewFunc")
	}
}

// newQueue returns an empty queue whose priorities are ordered by compare,
// with a backing array and an index preallocated for at least n keys. Equal
// priorities are ordered by push sequence, which makes the queue stable. A
// negative n is treated as 0. It panics if compare is nil.
func newQueue[K comparable, P any](
	n int,
	compare func(a, b P) int,
) *Queue[K, P] {
	if compare == nil {
		panic("pqueue: nil comparator")
	}

	n = max(n, 0)

	q := &Queue[K, P]{
		data:  make([]entry[K, P], 0, n),
		index: make(map[K]int, n),
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

// from returns a queue whose priorities are ordered by compare and that holds
// the keys and priorities of items, heapified in O(n). Duplicate keys keep the
// priority of their last occurrence and the push sequence of their first, the
// same result as pushing the items one by one. It panics if compare is nil.
func from[K comparable, P any](
	items []Item[K, P],
	compare func(a, b P) int,
) *Queue[K, P] {
	q := newQueue[K](len(items), compare)
	for _, it := range items {
		if i, ok := q.index[it.Key]; ok {
			q.data[i].priority = it.Priority
			continue
		}

		q.index[it.Key] = len(q.data)
		q.data = append(q.data, entry[K, P]{
			key:      it.Key,
			priority: it.Priority,
			seq:      q.nextSeq,
		})
		q.nextSeq++
	}

	heapcore.HeapifyMoved(q.data, q.compare, q.moved)

	return q
}

// rebuildIndex returns a new index for entries.
func rebuildIndex[K comparable, P any](entries []entry[K, P]) map[K]int {
	index := make(map[K]int, len(entries))
	for i, e := range entries {
		index[e.key] = i
	}

	return index
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
