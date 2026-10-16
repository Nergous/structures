// Package orderedmap provides a generic, insertion-ordered hash map.
//
// Create a map with [New] or [NewWithCap]. Pairs are stored with [Map.Set],
// read with [Map.Get] and [Map.Contains], and removed with [Map.Remove] or
// [Map.RemoveFunc]; [Map.First], [Map.Last], [Map.PopFirst], and
// [Map.PopLast] work at the ends. Setting an existing key updates its value
// and keeps its place; use [Map.MoveToBack] to make a key the newest one.
//
// Get and Contains run in O(1) on average. Set, Remove, PopFirst, PopLast, and
// MoveToBack run in amortized O(1). First, Last, Len, Cap, and IsEmpty run in
// O(1). Methods that copy or iterate over every pair, such as All, Clone, and
// RemoveFunc, run in O(n).
//
// Entries live in a slice in insertion order, next to a map from each key to
// its position in that slice. Deleting an entry leaves a hole in the slice
// instead of shifting its neighbors; once holes make up more than half of a
// slice of 16 or more entries, the slice is compacted in place. Compaction is
// O(n) but happens only after O(n) deletions, so deletes stay amortized O(1).
// Holes are zeroed immediately, so deleted keys and values are not retained.
// There is deliberately no MoveToFront: inserting at the front of a slice
// costs O(n), and a least-recently-used cache needs only [Map.MoveToBack] and
// [Map.PopFirst].
//
// A [Map] iterates in insertion order, from the oldest key to the newest.
// [Map.All], [Map.Backward], [Map.Keys], and [Map.Values] iterate without
// consuming the map. Mutating the map while iterating, including from the loop
// body, is undefined behavior because compaction can move entries under the
// iterator. Use [Map.RemoveFunc] to delete pairs selected by a predicate.
//
// The zero value of Map[K, V] is usable as an empty map. [Map.Set] allocates
// the index on first insertion. A nil *Map[K, V] behaves as an empty map for
// every read-only method and for removal methods such as [Map.Remove], so the
// zero pointer value is safe to query without first calling [New]. Methods
// that store, clear, reset, or resize data ([Map.Set], [Map.Clear],
// [Map.Reset], [Map.Shrink], [Map.Clip], [Map.Grow]) require a non-nil
// receiver and panic on a nil one; see each method's documentation.
//
// A Map is not safe for concurrent use: it performs no internal locking.
// Callers that share a map across goroutines must provide their own
// synchronization.
package orderedmap

import (
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/shared"
)

// minCompact is the smallest entries length at which compaction is considered.
const minCompact = 16

// entry is one slot of the insertion-ordered slice. alive is false for a hole
// left behind by a deletion; holes are always zeroed.
type entry[K comparable, V any] struct {
	key   K
	value V
	alive bool
}

// Map is a generic hash map backed by a slice of entries in insertion order
// and a Go map from each key to its position. It iterates from the oldest key
// to the newest.
//
// The zero value is a valid empty map. A nil *Map[K, V] is treated as a valid
// empty map by read-only and removal methods, but methods that store, clear,
// reset, or resize data require a non-nil receiver. Copying a Map value shares
// its storage with the original; use [Map.Clone] for an independent copy.
//
// Map is not safe for concurrent use.
type Map[K comparable, V any] struct {
	// Invariants, which every method restores before returning:
	//
	//   - index holds exactly the live keys, and entries[index[k]].key == k;
	//   - len(index) == len(entries) - holes;
	//   - the last entry is live, so there are no trailing holes;
	//   - entries[head] is the first live entry; for an empty map entries is
	//     empty and head and holes are 0;
	//   - after a removal, holes is at most half of len(entries) once
	//     len(entries) >= minCompact (insertions never compact, so up to
	//     minCompact-1 holes may remain while the slice grows past it).
	entries []entry[K, V] // entries in insertion order, with holes
	index   map[K]int     // key -> position in entries
	head    int           // position of the first live entry
	holes   int           // number of holes in entries
}

// New returns an empty map with no preallocated capacity.
func New[K comparable, V any]() *Map[K, V] {
	return NewWithCap[K, V](0)
}

// NewWithCap returns an empty map with an entry slice and an index
// preallocated for at least n keys. Use it when the maximum size is known up
// front to avoid reallocations while setting. A negative n is treated as 0.
func NewWithCap[K comparable, V any](n int) *Map[K, V] {
	n = max(n, 0)

	return &Map[K, V]{
		entries: make([]entry[K, V], 0, n),
		index:   make(map[K]int, n),
	}
}

// Len returns the number of keys currently in the map. A nil receiver reports
// 0.
func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}

	return len(m.index)
}

// Cap returns the capacity of the map's entry slice, which counts live entries
// and holes alike. It is not the number of keys that can be added without
// growing when holes are present, and it does not include the index, whose
// size Go does not expose. A nil receiver reports 0.
func (m *Map[K, V]) Cap() int {
	if m == nil {
		return 0
	}

	return cap(m.entries)
}

// IsEmpty reports whether the map has no keys. A nil receiver reports true.
func (m *Map[K, V]) IsEmpty() bool {
	return m == nil || len(m.index) == 0
}

// Get returns the value stored under key. The boolean result is false when key
// is absent, in which case the returned value is the zero value of V. It runs
// in O(1) on average. A nil receiver is treated as empty and returns (zero,
// false).
func (m *Map[K, V]) Get(key K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}

	i, ok := m.index[key]
	if !ok {
		var zero V
		return zero, false
	}

	return m.entries[i].value, true
}

// Contains reports whether key is present in the map. It runs in O(1) on
// average. A nil receiver is treated as empty and reports false.
func (m *Map[K, V]) Contains(key K) bool {
	if m == nil {
		return false
	}

	_, ok := m.index[key]

	return ok
}

// First returns the oldest key and its value without removing them. The
// boolean result is false when the map is empty, in which case the returned
// key and value are zero values. It runs in O(1). A nil receiver is treated as
// empty and returns (zero, zero, false).
func (m *Map[K, V]) First() (K, V, bool) {
	if m == nil || len(m.index) == 0 {
		var (
			zeroK K
			zeroV V
		)
		return zeroK, zeroV, false
	}

	e := m.entries[m.head]

	return e.key, e.value, true
}

// Last returns the newest key and its value without removing them. The boolean
// result is false when the map is empty, in which case the returned key and
// value are zero values. It runs in O(1). A nil receiver is treated as empty
// and returns (zero, zero, false).
func (m *Map[K, V]) Last() (K, V, bool) {
	if m == nil || len(m.index) == 0 {
		var (
			zeroK K
			zeroV V
		)
		return zeroK, zeroV, false
	}

	e := m.entries[len(m.entries)-1]

	return e.key, e.value, true
}

// All returns an iterator over the map's pairs from oldest to newest, without
// consuming them; the map is left unchanged. The returned [iter.Seq2] is a Go
// 1.23 range-over-func iterator, so it can be used directly in a range loop.
// Breaking out of the loop early stops iteration cleanly. Mutating the map
// during iteration is undefined behavior. A nil receiver is treated as empty
// and yields nothing.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil {
			return
		}

		for _, e := range m.entries {
			if e.alive && !yield(e.key, e.value) {
				return
			}
		}
	}
}

// Backward returns an iterator over the map's pairs from newest to oldest,
// without consuming them; the map is left unchanged. The returned [iter.Seq2]
// is a Go 1.23 range-over-func iterator, so it can be used directly in a range
// loop. Breaking out of the loop early stops iteration cleanly. Mutating the
// map during iteration is undefined behavior. A nil receiver is treated as
// empty and yields nothing.
func (m *Map[K, V]) Backward() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil {
			return
		}

		for _, e := range slices.Backward(m.entries) {
			if e.alive && !yield(e.key, e.value) {
				return
			}
		}
	}
}

// Keys returns an iterator over the map's keys from oldest to newest, without
// consuming them; the map is left unchanged. The returned [iter.Seq] is a Go
// 1.23 range-over-func iterator, so it can be used directly in a range loop;
// use [slices.Collect] to get a slice. Breaking out of the loop early stops
// iteration cleanly. Mutating the map during iteration is undefined behavior.
// A nil receiver is treated as empty and yields nothing.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if m == nil {
			return
		}

		for _, e := range m.entries {
			if e.alive && !yield(e.key) {
				return
			}
		}
	}
}

// Values returns an iterator over the map's values from the oldest key to the
// newest, without consuming them; the map is left unchanged. The returned
// [iter.Seq] is a Go 1.23 range-over-func iterator, so it can be used directly
// in a range loop; use [slices.Collect] to get a slice. Breaking out of the
// loop early stops iteration cleanly. Mutating the map during iteration is
// undefined behavior. A nil receiver is treated as empty and yields nothing.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		if m == nil {
			return
		}

		for _, e := range m.entries {
			if e.alive && !yield(e.value) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the map with the same order.
// The two maps share no backing storage, so mutating one does not affect the
// other. Keys and values are copied as-is: if K or V is a pointer or contains
// references, the pointed-to data is shared between the copies. The clone has
// no holes, and its capacity equals its length. It runs in O(n).
//
// A nil receiver clones to a new, usable empty map.
func (m *Map[K, V]) Clone() *Map[K, V] {
	if m == nil {
		return New[K, V]()
	}

	entries := packArray(m.entries, m.holes)

	return &Map[K, V]{
		entries: entries,
		index:   rebuildIndex(entries),
	}
}

// Set stores value under key and reports whether key was new. If key is
// already in the map, Set replaces its value and returns false; the key keeps
// its original place in the order. Otherwise the key becomes the newest and
// Set returns true. It runs in amortized O(1).
//
// Set requires a non-nil receiver and panics on a nil one: it must store the
// pair somewhere. The zero value is usable, so [New] is optional.
func (m *Map[K, V]) Set(key K, value V) bool {
	shared.MustNotNil(m, "orderedmap", "Set")

	if i, ok := m.index[key]; ok {
		m.entries[i].value = value
		return false
	}

	if m.index == nil {
		m.index = make(map[K]int)
	}

	m.index[key] = len(m.entries)
	m.entries = append(m.entries, entry[K, V]{
		key:   key,
		value: value,
		alive: true,
	})

	return true
}

// Remove deletes key and returns its value. The boolean result reports whether
// key was present; when it is false, the returned value is the zero value of
// V. The vacated slot is zeroed so it no longer retains a reference to the
// removed pair. It runs in amortized O(1). A nil receiver is treated as empty
// and returns (zero, false).
func (m *Map[K, V]) Remove(key K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}

	i, ok := m.index[key]
	if !ok {
		var zero V
		return zero, false
	}

	return m.removeAt(i).value, true
}

// RemoveFunc deletes every pair for which del returns true and returns the
// number of deleted pairs. del is called exactly once per pair, from oldest to
// newest, and must not modify the map or panic. The remaining pairs keep their
// order, and the entry slice is left without holes. It runs in O(n).
//
// RemoveFunc panics if del is nil. A nil receiver is treated as empty and
// returns 0.
func (m *Map[K, V]) RemoveFunc(del func(K, V) bool) int {
	if del == nil {
		panic("orderedmap: nil predicate")
	}

	if m == nil || len(m.index) == 0 {
		return 0
	}

	lenBefore := len(m.index)
	w := 0
	for _, e := range m.entries {
		if !e.alive {
			continue
		}

		if del(e.key, e.value) {
			delete(m.index, e.key)
			continue
		}

		m.entries[w] = e
		m.index[e.key] = w
		w++
	}

	clear(m.entries[w:])
	m.entries = m.entries[:w]
	m.head = 0
	m.holes = 0

	return lenBefore - len(m.index)
}

// PopFirst removes and returns the oldest key and its value. The boolean
// result is false when the map is empty, in which case the returned key and
// value are zero values. The vacated slot is zeroed so it no longer retains a
// reference to the removed pair. It runs in amortized O(1). A nil receiver is
// treated as empty and returns (zero, zero, false).
func (m *Map[K, V]) PopFirst() (K, V, bool) {
	if m == nil || len(m.index) == 0 {
		var (
			zeroK K
			zeroV V
		)
		return zeroK, zeroV, false
	}

	e := m.removeAt(m.head)

	return e.key, e.value, true
}

// PopLast removes and returns the newest key and its value. The boolean result
// is false when the map is empty, in which case the returned key and value are
// zero values. The vacated slot is zeroed so it no longer retains a reference
// to the removed pair. It runs in amortized O(1). A nil receiver is treated as
// empty and returns (zero, zero, false).
func (m *Map[K, V]) PopLast() (K, V, bool) {
	if m == nil || len(m.index) == 0 {
		var (
			zeroK K
			zeroV V
		)
		return zeroK, zeroV, false
	}

	e := m.removeAt(len(m.entries) - 1)

	return e.key, e.value, true
}

// MoveToBack makes key the newest key and reports whether key was present. An
// absent key is not added, and the value is unchanged. Moving the newest key
// is a no-op. It runs in amortized O(1). A nil receiver is treated as empty
// and reports false.
func (m *Map[K, V]) MoveToBack(key K) bool {
	if m == nil {
		return false
	}

	i, ok := m.index[key]
	if !ok {
		return false
	}

	if i == len(m.entries)-1 {
		return true
	}

	e := m.removeAt(i)
	m.index[key] = len(m.entries)
	m.entries = append(m.entries, e)

	return true
}

// Clear removes all pairs and releases the backing storage, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To
// empty the map while keeping its capacity for reuse, use [Map.Reset] instead.
// It runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (m *Map[K, V]) Clear() {
	shared.MustNotNil(m, "orderedmap", "Clear")
	m.entries = nil
	m.index = nil
	m.head = 0
	m.holes = 0
}

// Reset removes all pairs but keeps the backing storage for reuse, so Cap is
// preserved. The pairs are zeroed so the storage no longer retains references
// to them. To also release the backing storage, use [Map.Clear] instead. It
// runs in O(n).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (m *Map[K, V]) Reset() {
	shared.MustNotNil(m, "orderedmap", "Reset")
	clear(m.entries)
	m.entries = m.entries[:0]
	clear(m.index)
	m.head = 0
	m.holes = 0
}

// Shrink removes all holes, copies the live pairs into a right-sized entry
// slice, and rebuilds the index, so memory held by a map that grew large
// and has since shrunk becomes eligible for garbage collection immediately. Go
// maps never shrink in place, so Shrink is the only way to reclaim the index's
// memory short of [Map.Clear].
//
// Shrink runs in O(n) and allocates; the entry slice is copied only when it
// has holes or spare capacity, but the index is always rebuilt. For the
// cheaper, non-copying variant that only reslices the entry slice and lets
// memory be reclaimed lazily on the next growth, use [Map.Clip]. Shrink is the
// counterpart of [Map.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (m *Map[K, V]) Shrink() {
	shared.MustNotNil(m, "orderedmap", "Shrink")

	if m.holes > 0 || cap(m.entries) != len(m.entries) {
		m.entries = packArray(m.entries, m.holes)
		m.head = 0
		m.holes = 0
	}

	m.index = rebuildIndex(m.entries)
}

// Clip reduces the capacity of the entry slice to its current length without
// copying, by reslicing it (see [slices.Clip]). It runs in O(1) and does not
// allocate. Holes and the index are left as they are, and because the backing
// array is retained, the unused tail memory is not returned to the garbage
// collector until a later growth reallocates. For an immediate, copying
// reclaim, use [Map.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (m *Map[K, V]) Clip() {
	shared.MustNotNil(m, "orderedmap", "Clip")
	m.entries = slices.Clip(m.entries)
}

// Grow ensures the entry slice has spare capacity for at least n more keys
// without reallocating during subsequent sets, growing it if needed. It is the
// counterpart of [Map.Shrink]. Grow runs in O(n) when it reallocates and is a
// no-op when n is non-positive or the capacity is already sufficient. It
// panics if the new capacity overflows, the same condition as [slices.Grow].
// The index map is not preallocated by Grow; [NewWithCap] sizes both.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (m *Map[K, V]) Grow(n int) {
	shared.MustNotNil(m, "orderedmap", "Grow")
	m.entries = shared.GrowSlice(m.entries, n)
}

// removeAt deletes the live entry at position i and returns it. It zeroes the
// slot, then restores the invariants: trailing holes are trimmed, head moves
// to the next live entry, and the slice is compacted when holes dominate. i
// must be the position of a live entry.
func (m *Map[K, V]) removeAt(i int) entry[K, V] {
	e := m.entries[i]
	delete(m.index, e.key)
	m.entries[i] = entry[K, V]{}
	m.holes++

	m.trimTail()
	if len(m.entries) > 0 && i == m.head {
		m.advanceHead()
	}

	m.maybeCompact()

	return e
}

// trimTail cuts trailing holes off the entry slice. When the slice becomes
// empty, head is reset.
func (m *Map[K, V]) trimTail() {
	n := len(m.entries)
	for n > 0 && !m.entries[n-1].alive {
		n--
		m.holes--
	}

	m.entries = m.entries[:n]
	if n == 0 {
		m.head = 0
	}
}

// advanceHead moves head forward to the first live entry. The entry slice must
// be non-empty; its last entry is live, so the scan always stops.
func (m *Map[K, V]) advanceHead() {
	for !m.entries[m.head].alive {
		m.head++
	}
}

// maybeCompact compacts the entry slice when holes make up more than half of a
// slice of at least minCompact entries.
func (m *Map[K, V]) maybeCompact() {
	if len(m.entries) >= minCompact && m.holes > len(m.entries)/2 {
		m.compact()
	}
}

// compact moves live entries to the front of the slice in place, rewriting
// their index positions, and zeroes the vacated tail so it retains no
// references. Capacity is preserved.
func (m *Map[K, V]) compact() {
	w := 0
	for _, e := range m.entries {
		if e.alive {
			m.entries[w] = e
			m.index[e.key] = w
			w++
		}
	}

	clear(m.entries[w:])
	m.entries = m.entries[:w]
	m.head = 0
	m.holes = 0
}

// packArray returns a new slice with exactly the live entries of entries, in
// order. holes must be the number of holes in entries. The caller must
// rebuild the index and reset head and holes.
func packArray[K comparable, V any](
	entries []entry[K, V],
	holes int,
) []entry[K, V] {
	packed := make([]entry[K, V], 0, len(entries)-holes)
	for _, e := range entries {
		if e.alive {
			packed = append(packed, e)
		}
	}

	return packed
}

// rebuildIndex returns a new index for entries.
func rebuildIndex[K comparable, V any](entries []entry[K, V]) map[K]int {
	index := make(map[K]int, len(entries))
	for i, e := range entries {
		if e.alive {
			index[e.key] = i
		}
	}

	return index
}
