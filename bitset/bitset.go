// Package bitset provides a set of non-negative ints backed by a bit vector.
//
// Create a set with [New], [NewWithCap], [Of], or [From]. Values are added
// with [Set.Add] or [Set.AddN], removed with [Set.Remove] or
// [Set.RemoveFunc], and queried with [Set.Contains], [Set.ContainsAll],
// [Set.ContainsAny], [Set.Min], and [Set.Max]. [Set.Union],
// [Set.Intersection], [Set.Difference], and [Set.SymmetricDifference] return
// new sets; [Set.AddSet], [Set.RemoveSet], and [Set.RetainSet] are their
// in-place forms.
//
// Value v is bit v%64 of word v/64, so memory grows with the largest stored
// value, not with the number of values: a set holding only 1<<20 still uses
// 128 KiB. Prefer the map-backed set package of this module for sparse or
// very large values. Negative values are never present: lookups and removals
// report false for them, and [Set.Add] and [Set.AddN] panic on them.
//
// Contains and Remove run in O(1). Add runs in O(1) once the word holding the
// value is allocated. Len, IsEmpty, and Max run in O(1). Set algebra works on
// whole words, so it runs in O(m/64), where m is the largest value involved.
// Iteration and copying run in O(m/64 + n), where n is the number of values.
//
// Iteration order is ascending: [Set.All] and [Set.Slice] yield the smallest
// value first, and [Set.Backward] yields the values in descending order. The
// order is stable between calls.
//
// [Set.All] iterates without consuming the set. Mutating the set while
// iterating, including from the loop body, is undefined behavior. Use
// [Set.RetainSet] or [Set.RemoveFunc] to delete values selected by a
// predicate.
//
// The zero value of Set is usable as an empty set. [Set.Add] allocates the
// backing array on first insertion. A nil *Set behaves as an empty set for
// every read-only method and for removal methods such as [Set.Remove], so the
// zero pointer value is safe to query without first calling [New]. Methods
// that store, clear, reset, or resize data ([Set.Add], [Set.AddN],
// [Set.AddSet], [Set.Clear], [Set.Reset], [Set.Shrink], [Set.Clip],
// [Set.Grow]) require a non-nil receiver and panic on a nil one; see each
// method's documentation.
//
// A Set is not safe for concurrent use: it performs no internal locking.
// Callers that share a set across goroutines must provide their own
// synchronization.
package bitset

import (
	"iter"
	"math/bits"
	"slices"

	"github.com/Nergous/structures/internal/shared"
)

const (
	wordBits  = 64           // bits per word in data
	wordShift = 6            // log2(wordBits): v >> wordShift is the word
	wordMask  = wordBits - 1 // v & wordMask is the bit within the word
)

// Set is a set of non-negative ints backed by a bit vector of 64-bit words.
// Value v is present when bit v%64 of word v/64 is set, so memory grows with
// the largest stored value, not with the number of values. Iteration runs in
// ascending order.
//
// The zero value is a valid empty set. A nil *Set is treated as a valid empty
// set by read-only and removal methods, but methods that store, clear, reset,
// or resize data require a non-nil receiver. Copying a Set value shares its
// storage with the original; use [Set.Clone] for an independent copy.
//
// Set is not safe for concurrent use.
type Set struct {
	// Invariants, which every method restores before returning:
	//
	//   - n equals the total number of set bits in data;
	//   - data is empty or its last word is non-zero, so len(data) is
	//     word(m)+1 for the largest value m;
	//   - words in data[len(data):cap(data)] may hold stale bits; grow
	//     clears them before they are reused.
	data []uint64 // bit vector; bit v&wordMask of data[v>>wordShift] is v
	n    int      // number of values in the set
}

// New returns an empty set with no preallocated capacity.
func New() *Set {
	return NewWithCap(0)
}

// NewWithCap returns an empty set with a backing array preallocated for the
// values 0 through n-1. Use it when the largest value is known up front to
// avoid reallocations while adding. The capacity is rounded up to a multiple
// of 64. A negative n is treated as 0.
func NewWithCap(n int) *Set {
	return &Set{
		data: make([]uint64, 0, wordsFor(n)),
	}
}

// Of returns a set with the values of vs. Duplicate values are stored once.
// Memory grows with the largest value in vs, not with len(vs). It runs in
// O(len(vs) + m/64), where m is the largest value in vs.
//
// Of panics if any value in vs is negative.
func Of(vs ...int) *Set {
	s := New()
	s.AddN(vs...)

	return s
}

// From returns a set with the values of s. Duplicate values are stored once.
// The set never aliases s: later changes to s do not affect the set, and vice
// versa. Memory grows with the largest value in s, not with len(s). It runs
// in O(len(s) + m/64), where m is the largest value in s.
//
// From panics if any value in s is negative.
func From(s []int) *Set {
	return Of(s...)
}

// Len returns the number of values currently in the set. A nil receiver
// reports 0. It runs in O(1).
func (s *Set) Len() int {
	if s == nil {
		return 0
	}

	return s.n
}

// Cap returns the number of values the backing array covers, i.e. values 0
// through Cap()-1 can be added before the next reallocation. It bounds the
// largest value, not the number of values, and is always a multiple of 64. A
// nil receiver reports 0. It runs in O(1).
func (s *Set) Cap() int {
	if s == nil {
		return 0
	}

	return cap(s.data) << wordShift
}

// IsEmpty reports whether the set has no values. A nil receiver reports true.
func (s *Set) IsEmpty() bool {
	return s == nil || s.n == 0
}

// Contains reports whether v is present in the set. Negative values are
// never present. It runs in O(1). A nil receiver is treated as empty and
// reports false.
func (s *Set) Contains(v int) bool {
	if s == nil || v < 0 {
		return false
	}

	w := word(v)

	return w < len(s.data) && s.data[w]&bit(v) != 0
}

// ContainsAll reports whether every value in vs is present in the set. With no
// arguments it reports true. It runs in O(k), where k is the number of values
// in vs. A nil receiver is treated as empty.
func (s *Set) ContainsAll(vs ...int) bool {
	for _, v := range vs {
		if !s.Contains(v) {
			return false
		}
	}

	return true
}

// ContainsAny reports whether at least one value in vs is present in the set.
// With no arguments it reports false. It runs in O(k), where k is the number
// of values in vs. A nil receiver is treated as empty and reports false.
func (s *Set) ContainsAny(vs ...int) bool {
	return slices.ContainsFunc(vs, s.Contains)
}

// Min returns the smallest value without removing it. The boolean result is
// false when the set is empty, in which case the returned value is 0. It runs
// in O(m/64), where m is the smallest value. A nil receiver is treated as
// empty and returns (0, false).
func (s *Set) Min() (int, bool) {
	if s == nil || s.n == 0 {
		return 0, false
	}

	for i, w := range s.data {
		if w != 0 {
			return number(i, bits.TrailingZeros64(w)), true
		}
	}

	return 0, false
}

// Max returns the largest value without removing it. The boolean result is
// false when the set is empty, in which case the returned value is 0. The
// last word of the backing array is never empty, so Max runs in O(1). A nil
// receiver is treated as empty and returns (0, false).
func (s *Set) Max() (int, bool) {
	if s == nil || s.n == 0 {
		return 0, false
	}

	i := len(s.data) - 1

	return number(i, wordMask-bits.LeadingZeros64(s.data[i])), true
}

// All returns an iterator over the set's values in ascending order, without
// consuming them; the set is left unchanged. The returned [iter.Seq] is a Go
// 1.23 range-over-func iterator, so it can be used directly in a range loop.
// Breaking out of the loop early stops iteration cleanly. Mutating the set
// during iteration is undefined behavior. A full pass runs in O(m/64 + n),
// where m is the largest value and n is the number of values. A nil receiver
// is treated as empty and yields nothing.
func (s *Set) All() iter.Seq[int] {
	return func(yield func(int) bool) {
		if s == nil {
			return
		}

		for i, w := range s.data {
			for w != 0 {
				if !yield(number(i, bits.TrailingZeros64(w))) {
					return
				}

				w &= w - 1
			}
		}
	}
}

// Backward returns an iterator over the set's values in descending order,
// without consuming them; the set is left unchanged. The returned [iter.Seq]
// is a Go 1.23 range-over-func iterator, so it can be used directly in a range
// loop. Breaking out of the loop early stops iteration cleanly. Mutating the
// set during iteration is undefined behavior. A full pass runs in
// O(m/64 + n), where m is the largest value and n is the number of values. A
// nil receiver is treated as empty and yields nothing.
func (s *Set) Backward() iter.Seq[int] {
	return func(yield func(int) bool) {
		if s == nil {
			return
		}

		for i, w := range slices.Backward(s.data) {
			for w != 0 {
				p := wordMask - bits.LeadingZeros64(w)
				if !yield(number(i, p)) {
					return
				}

				w &^= 1 << p
			}
		}
	}
}

// Clone returns an independent copy of the set. The two sets share no backing
// array, so mutating one does not affect the other. The clone's backing array
// covers exactly the words in use, without the original's spare capacity. It
// runs in O(m/64), where m is the largest value.
//
// A nil receiver clones to a new, usable empty set.
func (s *Set) Clone() *Set {
	if s == nil {
		return New()
	}

	c := &Set{
		data: make([]uint64, len(s.data)),
		n:    s.n,
	}

	copy(c.data, s.data)

	return c
}

// Slice returns a copy of the set's values in ascending order, the same order
// [Set.All] yields. The returned slice is independent of the set, so callers
// may modify it freely. An empty set returns nil; a nil receiver is treated as
// empty and likewise returns nil. It runs in O(m/64 + n), where m is the
// largest value and n is the number of values.
func (s *Set) Slice() []int {
	if s == nil || s.n == 0 {
		return nil
	}

	out := make([]int, 0, s.n)
	for v := range s.All() {
		out = append(out, v)
	}

	return out
}

// Add inserts v into the set and reports whether v was new. If v was already
// present, Add leaves the set unchanged and returns false. It runs in O(1)
// when the word holding v is already allocated; otherwise it first extends the
// backing array to cover v, which costs O(v/64) time and about v/8 bytes.
//
// Add requires a non-nil receiver and panics on a nil one: it must store the
// value somewhere. The zero value is usable, so [New] is optional. Add panics
// if v is negative, leaving the set unchanged.
func (s *Set) Add(v int) bool {
	shared.MustNotNil(s, "bitset", "Add")

	if v < 0 {
		panic("bitset: Add of negative value")
	}

	s.data = grow(s.data, word(v)+1)

	return s.insert(v)
}

// AddN inserts every value in vs into the set. Duplicate values are stored
// once. With no arguments it is a no-op. It extends the backing array at most
// once, to cover the largest value, and runs in O(k + m/64), where k is the
// number of values in vs and m is the largest of them.
//
// AddN requires a non-nil receiver and panics on a nil one, even when called
// with no arguments: it must store the values somewhere. The zero value is
// usable, so [New] is optional. AddN panics if any value in vs is negative;
// the values are checked first, so the set is left unchanged.
func (s *Set) AddN(vs ...int) {
	shared.MustNotNil(s, "bitset", "AddN")

	if len(vs) == 0 {
		return
	}

	m := 0
	for _, v := range vs {
		if v < 0 {
			panic("bitset: AddN of negative value")
		}

		m = max(m, v)
	}

	s.data = grow(s.data, word(m)+1)

	for _, v := range vs {
		s.insert(v)
	}
}

// AddSet inserts every value of other into s. It is the in-place form of
// [Set.Union]. A nil other is treated as an empty set and makes AddSet a
// no-op. It extends the backing array at most once and merges whole words,
// so it runs in O(m/64), where m is the largest value in other.
//
// AddSet requires a non-nil receiver and panics on a nil one: it must store the
// values somewhere. The zero value is usable, so [New] is optional.
func (s *Set) AddSet(other *Set) {
	shared.MustNotNil(s, "bitset", "AddSet")

	if other == nil || s == other || other.n == 0 {
		return
	}

	s.data = grow(s.data, len(other.data))

	for i, w := range other.data {
		s.n += bits.OnesCount64(w &^ s.data[i])
		s.data[i] |= w
	}
}

// Remove deletes v from the set and reports whether v was present. Negative
// values are never present, so Remove reports false for them instead of
// panicking. Removing the largest value drops the empty words at the end of
// the backing array but keeps its capacity. It runs in amortized O(1). A nil
// receiver is treated as empty and returns false.
func (s *Set) Remove(v int) bool {
	if !s.Contains(v) {
		return false
	}

	w := word(v)
	s.data[w] &^= bit(v)
	s.n--

	if w == len(s.data)-1 && s.data[w] == 0 {
		s.trim()
	}

	return true
}

// RemoveSet deletes every value of other from s. It is the in-place form of
// [Set.Difference]. Passing the set itself empties it while keeping the
// backing array. Nil operands are treated as empty sets, so a nil receiver or
// a nil other makes RemoveSet a no-op. It clears whole words, so it runs in
// O(m/64), where m is the smaller of the two largest values.
func (s *Set) RemoveSet(other *Set) {
	if s == nil || other == nil || s.n == 0 || other.n == 0 {
		return
	}

	if s == other {
		s.data = s.data[:0]
		s.n = 0

		return
	}

	k := min(len(s.data), len(other.data))
	for i, w := range other.data[:k] {
		s.n -= bits.OnesCount64(s.data[i] & w)
		s.data[i] &^= w
	}

	s.trim()
}

// RetainSet keeps only the values of s that are also present in other. It is
// the in-place form of [Set.Intersection]. Nil operands are treated as empty
// sets: a nil other empties s while keeping the backing array, and a nil
// receiver makes RetainSet a no-op. It intersects whole words, so it runs in
// O(m/64), where m is the largest value in s.
func (s *Set) RetainSet(other *Set) {
	if s == nil || s == other || s.n == 0 {
		return
	}

	if other == nil || other.n == 0 {
		s.data = s.data[:0]
		s.n = 0

		return
	}

	s.data = s.data[:min(len(s.data), len(other.data))]
	s.n = 0
	for i, w := range s.data {
		s.data[i] = w & other.data[i]
		s.n += bits.OnesCount64(s.data[i])
	}

	s.trim()
}

// RemoveFunc deletes every value for which del returns true and returns the
// number of deleted values. del is called exactly once per value, in
// ascending order, and must not modify the set. Removing the largest values
// drops the empty words at the end of the backing array but keeps its
// capacity. It runs in O(m/64 + n), where m is the largest value and n is the
// number of values.
//
// RemoveFunc panics if del is nil. A nil receiver is treated as empty and
// returns 0. If del panics, the values it already selected stay deleted and
// the set remains consistent.
func (s *Set) RemoveFunc(del func(int) bool) int {
	if del == nil {
		panic("bitset: nil predicate")
	}

	if s == nil {
		return 0
	}

	n := 0
	for i, w := range s.data {
		var drop uint64
		for rest := w; rest != 0; rest &= rest - 1 {
			b := bits.TrailingZeros64(rest)
			if del(number(i, b)) {
				drop |= 1 << b
			}
		}

		if drop != 0 {
			c := bits.OnesCount64(drop)
			s.data[i] = w &^ drop
			s.n -= c
			n += c
		}
	}

	if n > 0 {
		s.trim()
	}

	return n
}

// Union returns a new set with the values of s and of every set in others.
// Nil operands are treated as empty sets. It allocates the result once and
// merges whole words, so it runs in O(k*m/64), where k is the number of
// operands and m is the largest value among them.
func (s *Set) Union(others ...*Set) *Set {
	k := len(s.words())
	for _, other := range others {
		k = max(k, len(other.words()))
	}

	out := &Set{data: make([]uint64, k)}
	copy(out.data, s.words())

	for _, other := range others {
		for i, w := range other.words() {
			out.data[i] |= w
		}
	}

	out.n = count(out.data)

	return out
}

// Intersection returns a new set with the values of s that are present in
// every set in others. Nil operands are treated as empty sets. With no others
// it returns a copy of s. It intersects whole words, so it runs in
// O(k*m/64), where k is the number of sets in others and m is the largest
// value in s.
func (s *Set) Intersection(others ...*Set) *Set {
	out := s.Clone()
	for _, other := range others {
		out.RetainSet(other)
	}

	return out
}

// Difference returns a new set with the values of s that are absent from
// every set in others. Nil operands are treated as empty sets. With no others
// it returns a copy of s. It clears whole words, so it runs in O(k*m/64),
// where k is the number of sets in others and m is the largest value in s.
func (s *Set) Difference(others ...*Set) *Set {
	out := s.Clone()
	for _, other := range others {
		out.RemoveSet(other)
	}

	return out
}

// SymmetricDifference returns a new set with the values that are in exactly
// one of s and other. Nil operands are treated as empty sets. It combines
// whole words, so it runs in O(m/64), where m is the largest value of either
// operand.
func (s *Set) SymmetricDifference(other *Set) *Set {
	long, short := s.words(), other.words()
	if len(long) < len(short) {
		long, short = short, long
	}

	out := &Set{data: make([]uint64, len(long))}
	copy(out.data, long)

	for i, w := range short {
		out.data[i] ^= w
	}

	out.n = count(out.data)
	out.trim()

	return out
}

// Equal reports whether s and other contain the same values. Nil operands are
// treated as empty sets. It compares whole words, so it runs in O(m/64),
// where m is the largest value.
func (s *Set) Equal(other *Set) bool {
	return s.Len() == other.Len() && slices.Equal(s.words(), other.words())
}

// IsSubset reports whether every value of s is also present in other. Nil
// operands are treated as empty sets. It compares whole words, so it runs in
// O(m/64), where m is the largest value in s.
func (s *Set) IsSubset(other *Set) bool {
	a, b := s.words(), other.words()
	if s.Len() > other.Len() || len(a) > len(b) {
		return false
	}

	for i, w := range a {
		if w&^b[i] != 0 {
			return false
		}
	}

	return true
}

// IsSuperset reports whether every value of other is also present in s. Nil
// operands are treated as empty sets. It runs in O(m/64), where m is the
// largest value in other.
func (s *Set) IsSuperset(other *Set) bool {
	return other.IsSubset(s)
}

// IsDisjoint reports whether s and other have no values in common. Nil
// operands are treated as empty sets. It compares only the words both sets
// use, so it runs in O(m/64), where m is the smaller of the two largest
// values.
func (s *Set) IsDisjoint(other *Set) bool {
	a, b := s.words(), other.words()
	for i := range min(len(a), len(b)) {
		if a[i]&b[i] != 0 {
			return false
		}
	}

	return true
}

// Clear removes all values and releases the backing array, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To
// empty the set while keeping its capacity for reuse, use [Set.Reset]
// instead. It runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (s *Set) Clear() {
	shared.MustNotNil(s, "bitset", "Clear")
	s.data = nil
	s.n = 0
}

// Reset removes all values but keeps the backing array for reuse, so Cap is
// preserved. The words are not zeroed here: the array holds no references,
// and words are cleared when a later Add reuses them. To also release the
// backing array, use [Set.Clear] instead. It runs in O(1).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (s *Set) Reset() {
	shared.MustNotNil(s, "bitset", "Reset")
	s.data = s.data[:0]
	s.n = 0
}

// Shrink reduces the backing array to cover exactly the values up to the
// largest one, rounded up to a multiple of 64, copying the words in use into
// a new, right-sized array so the previous (larger) array becomes eligible
// for garbage collection immediately. Use it to reclaim memory after the
// largest values were removed from a set that will not grow back. An empty
// set releases its backing array, as after [Set.Clear].
//
// Shrink runs in O(m/64), where m is the largest value, and allocates once;
// it is a no-op when the capacity already equals the words in use. For the
// cheaper, non-copying variant that only reslices and lets memory be
// reclaimed lazily on the next growth, use [Set.Clip]. Shrink is the
// counterpart of [Set.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (s *Set) Shrink() {
	shared.MustNotNil(s, "bitset", "Shrink")

	if len(s.data) == 0 {
		s.data = nil
		return
	}

	s.data = shared.ShrinkSlice(s.data)
}

// Clip reduces the reported capacity to the words in use without copying, by
// reslicing the backing array (see [slices.Clip]). It runs in O(1) and does
// not allocate. Because the backing array is retained, the unused tail memory
// is not returned to the garbage collector until a later growth reallocates.
// For an immediate, copying reclaim, use [Set.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (s *Set) Clip() {
	shared.MustNotNil(s, "bitset", "Clip")
	s.data = slices.Clip(s.data)
}

// Grow ensures the backing array covers the values 0 through n-1, so they can
// be added without reallocating, the same way [NewWithCap] sizes a new set.
// Capacity in a Set bounds the largest value, not the number of values, so n
// is a value bound rather than a count of additional values. It is the
// counterpart of [Set.Shrink]. Grow runs in O(n/64) when it reallocates and is
// a no-op when n is non-positive or Cap already reaches n. It panics if the
// new capacity overflows, the same condition as [slices.Grow].
//
// Grow requires a non-nil receiver and panics on a nil one.
func (s *Set) Grow(n int) {
	shared.MustNotNil(s, "bitset", "Grow")
	s.data = shared.GrowSlice(s.data, wordsFor(n)-len(s.data))
}

// insert sets the bit for v, which must be non-negative and covered by
// s.data, and reports whether it was clear before.
func (s *Set) insert(v int) bool {
	w, b := word(v), bit(v)
	if s.data[w]&b != 0 {
		return false
	}

	s.data[w] |= b
	s.n++

	return true
}

// trim drops the empty words at the end of s.data, so the last word is never
// empty. The capacity is kept; grow clears the dropped words if they are
// reused.
func (s *Set) trim() {
	i := len(s.data)
	for i > 0 && s.data[i-1] == 0 {
		i--
	}

	s.data = s.data[:i]
}

// words returns the words in use, treating a nil receiver as an empty set.
func (s *Set) words() []uint64 {
	if s == nil {
		return nil
	}

	return s.data
}

// grow returns data extended to length k with the new words zeroed. Words in
// data[len(data):cap(data)] may hold stale bits from an earlier truncation, so
// they are cleared rather than trusted. data is returned unchanged when it is
// already long enough; it is reallocated, with amortized growth, only when k
// exceeds its capacity.
func grow(data []uint64, k int) []uint64 {
	n := len(data)
	if n >= k {
		return data
	}

	data = shared.GrowSlice(data, k-n)
	data = data[:k]
	clear(data[n:])

	return data
}

// count returns the number of set bits in data.
func count(data []uint64) int {
	n := 0
	for _, w := range data {
		n += bits.OnesCount64(w)
	}

	return n
}

// wordsFor returns the number of words needed to cover the values 0 through
// n-1. A non-positive n needs no words.
func wordsFor(n int) int {
	if n <= 0 {
		return 0
	}

	k := n >> wordShift
	if n&wordMask != 0 {
		k++
	}

	return k
}

// word returns the index of the word holding v.
func word(v int) int {
	return v >> wordShift
}

// bit returns the mask of v within its word.
func bit(v int) uint64 {
	return 1 << (v & wordMask)
}

// number returns the value stored at bit b of word w.
func number(w, b int) int {
	return w<<wordShift + b
}
