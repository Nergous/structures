// Package orderedmap provides a generic hash map that remembers the order in
// which keys were first inserted.
//
// A [Map] behaves like a built-in map for lookups, inserts, and deletes, which
// run in O(1) on average, but iterates in insertion order instead of random
// order. Setting an existing key updates its value and keeps its place; use
// [Map.MoveToBack] to make a key the newest one.
//
// Entries live in a slice in insertion order, next to a map from each key to
// its position in that slice. Deleting an entry leaves a hole in the slice
// instead of shifting its neighbors; once holes make up more than half of a
// slice of 16 or more entries, the slice is compacted in place. Compaction is
// O(n) but happens only after O(n) deletions, so deletes stay O(1) amortized.
// Holes are zeroed immediately, so deleted keys and values are not retained.
//
// [Map.First], [Map.Last], [Map.PopFirst], and [Map.PopLast] work at the ends
// in O(1). There is deliberately no MoveToFront: inserting at the front of a
// slice costs O(n). A least-recently-used cache needs only [Map.MoveToBack] and
// [Map.PopFirst].
//
// [Map.All], [Map.Backward], [Map.Keys], and [Map.Values] iterate without
// consuming the map. Mutating the map while iterating, including from the loop
// body, is undefined behavior because compaction can move entries under the
// iterator. Use [Map.RemoveFunc] to delete entries selected by a predicate.
//
// A nil *Map[K, V] behaves as an empty map for every read-only method, for
// iteration, and for removals ([Map.Remove], [Map.RemoveFunc], [Map.PopFirst],
// [Map.PopLast], [Map.MoveToBack]). [Map.Clone] of a nil map returns a new,
// usable empty map. Methods that store or reset state ([Map.Set], [Map.Grow],
// [Map.Shrink], [Map.Clip], [Map.Reset], [Map.Clear]) require a non-nil
// receiver and panic on a nil one. The zero value of Map[K, V] is ready to
// use: [Map.Set] allocates the index on first insertion.
//
// A Map is not safe for concurrent use: it performs no internal locking.
// Callers that share a map across goroutines must provide their own
// synchronization.
package orderedmap

import (
	"iter"
	"slices"
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

// Map is a generic hash map that iterates in insertion order.
//
// Invariants, which every method restores before returning:
//
//   - index holds exactly the live keys, and entries[index[k]].key == k;
//   - len(index) == len(entries) - holes;
//   - the last entry is live, so there are no trailing holes;
//   - entries[head] is the first live entry; for an empty map entries is empty
//     and head and holes are 0;
//   - holes is at most half of len(entries) once len(entries) >= minCompact.
//
// A Map should be created with [New] or [NewWithCap], but the zero value is
// also usable. Map is not safe for concurrent use.
type Map[K comparable, V any] struct {
	entries []entry[K, V] // entries in insertion order, with holes
	index   map[K]int     // key -> position in entries
	head    int           // position of the first live entry
	holes   int           // number of holes in entries
}

// New returns an empty map.
func New[K comparable, V any]() *Map[K, V] {
	return NewWithCap[K, V](0)
}

// NewWithCap returns an empty map with room for at least n keys preallocated in
// both the entry slice and the index. A negative n is treated as 0.
func NewWithCap[K comparable, V any](n int) *Map[K, V] {
	n = max(n, 0)
	return &Map[K, V]{
		entries: make([]entry[K, V], 0, n),
		index:   make(map[K]int, n),
	}
}

// Len returns the number of key-value pairs in the map. A nil receiver reports
// 0.
func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.index)
}

// Cap returns the capacity of the entry slice, which counts live entries and
// holes alike. It is not the number of keys that can be added without growing
// when holes are present, and it does not include the index map, whose size
// Go does not expose. A nil receiver reports 0.
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

// Get returns the value for k. The boolean result is false when k is absent, in
// which case the value is the zero value of V. A nil receiver is treated as
// empty.
func (m *Map[K, V]) Get(k K) (V, bool) {
	if m != nil {
		if i, ok := m.index[k]; ok {
			return m.entries[i].value, true
		}
	}

	var zero V
	return zero, false
}

// Contains reports whether k is in the map. A nil receiver is treated as empty
// and reports false.
func (m *Map[K, V]) Contains(k K) bool {
	if m == nil {
		return false
	}
	_, ok := m.index[k]
	return ok
}

// First returns the oldest key and its value without removing them. The boolean
// result is false when the map is empty. It runs in O(1). A nil receiver is
// treated as empty.
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
// result is false when the map is empty. It runs in O(1). A nil receiver is
// treated as empty.
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

// Set stores v under k and reports whether k was new.
//
// If k is already in the map, Set replaces its value and returns false; the key
// keeps its original place in the order. Otherwise the key becomes the newest
// and Set returns true. It runs in O(1) amortized.
//
// Set panics on a nil receiver.
func (m *Map[K, V]) Set(k K, v V) bool {
	m.mustNotNil("Set")

	if i, ok := m.index[k]; ok {
		m.entries[i].value = v
		return false
	}

	if m.index == nil {
		m.index = make(map[K]int)
	}
	m.index[k] = len(m.entries)
	m.entries = append(m.entries, entry[K, V]{key: k, value: v, alive: true})
	return true
}

// Remove deletes k and returns its value. The boolean result reports whether k
// was present; when it is false, the value is the zero value of V. It runs in
// O(1) amortized. A nil receiver has nothing to remove and returns false.
func (m *Map[K, V]) Remove(k K) (V, bool) {
	if m == nil {
		var zero V
		return zero, false
	}

	i, ok := m.index[k]
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
// RemoveFunc panics if del is nil. A nil receiver has nothing to remove and
// returns 0.
func (m *Map[K, V]) RemoveFunc(del func(K, V) bool) int {
	if del == nil {
		panic("orderedmap: nil predicate")
	}
	if m == nil || len(m.index) == 0 {
		return 0
	}

	before := len(m.index)
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
	m.head, m.holes = 0, 0
	return before - len(m.index)
}

// PopFirst removes and returns the oldest key and its value. The boolean result
// is false when the map is empty. It runs in O(1) amortized. A nil receiver is
// treated as empty.
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
// is false when the map is empty. It runs in O(1) amortized. A nil receiver is
// treated as empty.
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

// MoveToBack makes k the newest key and reports whether k was present. The
// value is unchanged. Moving the newest key is a no-op. It runs in O(1)
// amortized. A nil receiver is treated as empty and reports false.
func (m *Map[K, V]) MoveToBack(k K) bool {
	if m == nil {
		return false
	}

	i, ok := m.index[k]
	if !ok {
		return false
	}
	if i == len(m.entries)-1 {
		return true
	}

	e := m.removeAt(i)
	m.index[k] = len(m.entries)
	m.entries = append(m.entries, e)
	return true
}

// Clone returns an independent shallow copy of the map with the same order. The
// two maps share no storage, so mutating one does not affect the other. Keys
// and values are copied as-is: if they are pointers or contain references, the
// pointed-to data is shared between the copies. The copy has no holes and its
// capacity equals its length. Runs in O(n).
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

// All returns an iterator over the pairs from oldest to newest, without
// consuming them. Breaking out of the loop early stops iteration cleanly.
// Mutating the map during iteration is undefined behavior. A nil receiver is
// treated as empty and yields nothing.
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

// Backward returns an iterator over the pairs from newest to oldest, without
// consuming them. Breaking out of the loop early stops iteration cleanly.
// Mutating the map during iteration is undefined behavior. A nil receiver is
// treated as empty and yields nothing.
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

// Keys returns an iterator over the keys from oldest to newest. Use
// slices.Collect to get a slice. Mutating the map during iteration is
// undefined behavior. A nil receiver yields nothing.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range m.All() {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns an iterator over the values from the oldest key to the newest.
// Use slices.Collect to get a slice. Mutating the map during iteration is
// undefined behavior. A nil receiver yields nothing.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// Clear removes all pairs and releases the backing storage, so its memory
// becomes eligible for garbage collection. The map stays usable. After Clear,
// Cap reports 0. To empty the map while keeping its capacity for reuse, use
// [Map.Reset] instead.
//
// Clear panics on a nil receiver.
func (m *Map[K, V]) Clear() {
	m.mustNotNil("Clear")
	m.entries = nil
	m.index = make(map[K]int)
	m.head = 0
	m.holes = 0
}

// Reset removes all pairs but keeps the backing storage for reuse, so Cap is
// preserved. The vacated slots are zeroed so they no longer retain references.
// To also release the storage, use [Map.Clear] instead.
//
// Reset panics on a nil receiver.
func (m *Map[K, V]) Reset() {
	m.mustNotNil("Reset")
	clear(m.entries)
	m.entries = m.entries[:0]
	clear(m.index)
	m.head = 0
	m.holes = 0
}

// Grow ensures the entry slice has spare capacity for at least n more keys
// without reallocating during subsequent sets. The index map grows on demand
// and is not preallocated by Grow; the WithCap constructor sizes both. Grow is
// a no-op when n is non-positive or the capacity is already sufficient.
//
// Grow panics on a nil receiver.
func (m *Map[K, V]) Grow(n int) {
	m.mustNotNil("Grow")
	if n <= 0 {
		return
	}
	m.entries = slices.Grow(m.entries, n)
}

// Clip reduces the capacity of the entry slice to its current length without
// copying (see [slices.Clip]). It runs in O(1) and does not allocate. Holes, the
// unused tail of the array, and the index map are left as they are; use
// [Map.Shrink] to reclaim them.
//
// Clip panics on a nil receiver.
func (m *Map[K, V]) Clip() {
	m.mustNotNil("Clip")
	m.entries = slices.Clip(m.entries)
}

// Shrink removes all holes and copies the entries into a right-sized slice, and
// rebuilds the index map, so memory held by a map that grew large and has since
// shrunk becomes eligible for garbage collection immediately. Go maps never
// shrink in place, so Shrink is the only way to reclaim the index's memory
// short of [Map.Clear]. It runs in O(n) and allocates.
//
// Shrink panics on a nil receiver.
func (m *Map[K, V]) Shrink() {
	m.mustNotNil("Shrink")
	if m.holes > 0 || cap(m.entries) != len(m.entries) {
		m.entries = packArray(m.entries, m.holes)
		m.head, m.holes = 0, 0
	}
	m.index = rebuildIndex(m.entries)
}

// removeAt deletes the live entry at position i and returns it. It zeroes the
// slot, then restores the invariants: trailing holes are trimmed, head moves to
// the next live entry, and the slice is compacted when holes dominate.
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
	m.head, m.holes = 0, 0
}

// packArray returns a new slice with exactly the live entries of entries, in
// order. holes is the number of holes in entries. The caller must rebuild the
// index and reset head and holes.
func packArray[K comparable, V any](entries []entry[K, V], holes int) []entry[K, V] {
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

// mustNotNil panics with a descriptive message when m is nil.
func (m *Map[K, V]) mustNotNil(op string) {
	if m == nil {
		panic("orderedmap: " + op + " on nil receiver")
	}
}
