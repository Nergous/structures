// Package orderedset provides a generic, insertion-ordered set of unique
// comparable values.
//
// Create a set with [New], [NewWithCap], [Of], or [From]. Values are added
// with [Set.Add] or [Set.AddN], removed with [Set.Remove] or [Set.RemoveFunc],
// and queried with [Set.Contains], [Set.ContainsAll], and [Set.ContainsAny].
// [Set.Union], [Set.Intersection], [Set.Difference], and
// [Set.SymmetricDifference] return new sets; [Set.AddSet], [Set.RemoveSet],
// and [Set.RetainSet] are their in-place forms. The API mirrors
// [github.com/Nergous/structures/set], plus methods that rely on order
// ([Set.First], [Set.Last], [Set.Backward], [Set.PopFirst], [Set.PopLast],
// and [Set.MoveToBack]) and entry-slice capacity controls ([Set.Cap],
// [Set.Clip], and [Set.Grow]).
//
// Contains runs in O(1) on average. Add, Remove, PopFirst, PopLast, and
// MoveToBack run in amortized O(1). First, Last, Len, Cap, and IsEmpty run in
// O(1). The cost of set algebra depends on the sizes of its operands; see
// each method's documentation. Methods that copy or iterate over every value,
// such as All, Clone, and Slice, run in O(n).
//
// The set is a thin wrapper over an insertion-ordered map with empty values:
// removing a value leaves a hole that is compacted later, which is why
// [Set.Cap] counts holes and [Set.Shrink] reclaims them.
//
// A [Set] iterates in insertion order. Adding a value that is already present
// keeps its place; use [Set.MoveToBack] to make a value the newest one. Set
// algebra returns results in a defined order: [Set.Union] lists the values of
// the receiver first, then new values from each argument in turn;
// [Set.Intersection] and [Set.Difference] keep the order of the receiver;
// [Set.SymmetricDifference] lists the values only in the receiver, then those
// only in the argument. [Set.Equal] compares contents and ignores order.
//
// [Set.All] and [Set.Backward] iterate without consuming the set. Mutating the
// set while iterating, including from the loop body, is undefined behavior
// because compaction can move entries under the iterator. Use [Set.RetainSet]
// or [Set.RemoveFunc] to delete values selected by a predicate.
//
// The zero value of Set[T] is usable as an empty set. [Set.Add] allocates the
// index on first insertion. A nil *Set[T] behaves as an empty set for every
// read-only method and for removal methods such as [Set.Remove], so the zero
// pointer value is safe to query without first calling [New]. Methods that
// store, clear, reset, or resize data ([Set.Add], [Set.AddN], [Set.AddSet],
// [Set.Clear], [Set.Reset], [Set.Shrink], [Set.Clip], [Set.Grow]) require a
// non-nil receiver and panic on a nil one; see each method's documentation.
//
// A Set is not safe for concurrent use: it performs no internal locking.
// Callers that share a set across goroutines must provide their own
// synchronization.
package orderedset

import (
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/shared"
	"github.com/Nergous/structures/orderedmap"
)

// Set is a generic collection of unique comparable values backed by an
// insertion-ordered map. It iterates from the oldest value to the newest.
//
// The zero value is a valid empty set. A nil *Set[T] is treated as a valid
// empty set by read-only and removal methods, but methods that store, clear,
// reset, or resize data require a non-nil receiver. Copying a Set value shares
// its storage with the original; use [Set.Clone] for an independent copy.
//
// Set is not safe for concurrent use.
type Set[T comparable] struct {
	m orderedmap.Map[T, struct{}]
}

// New returns an empty set with no preallocated capacity.
func New[T comparable]() *Set[T] {
	return NewWithCap[T](0)
}

// NewWithCap returns an empty set with an entry slice and an index
// preallocated for at least n values. Use it when the maximum size is known up
// front to avoid reallocations while adding. A negative n is treated as 0.
func NewWithCap[T comparable](n int) *Set[T] {
	n = max(n, 0)

	return &Set[T]{
		m: *orderedmap.NewWithCap[T, struct{}](n),
	}
}

// Of returns a set with the values of vs in order of first occurrence.
// Duplicate values are stored once, at the position where they first appear.
// It runs in O(n).
func Of[T comparable](vs ...T) *Set[T] {
	out := NewWithCap[T](len(vs))
	for _, v := range vs {
		out.m.Set(v, struct{}{})
	}

	return out
}

// From returns a set with the values of s in order of first occurrence.
// Duplicate values are stored once, at the position where they first appear.
// The set never aliases s: later changes to s do not affect the set, and vice
// versa. It runs in O(n).
func From[T comparable](s []T) *Set[T] {
	return Of(s...)
}

// Len returns the number of values currently in the set. A nil receiver
// reports 0.
func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}

	return s.m.Len()
}

// Cap returns the capacity of the set's entry slice, which counts live values
// and holes alike. It is not the number of values that can be added without
// growing when holes are present, and it does not include the index, whose
// size Go does not expose. A nil receiver reports 0.
func (s *Set[T]) Cap() int {
	if s == nil {
		return 0
	}

	return s.m.Cap()
}

// IsEmpty reports whether the set has no values. A nil receiver reports true.
func (s *Set[T]) IsEmpty() bool {
	return s == nil || s.m.Len() == 0
}

// Contains reports whether v is present in the set. It runs in O(1) on
// average. A nil receiver is treated as empty and reports false.
func (s *Set[T]) Contains(v T) bool {
	if s == nil {
		return false
	}

	return s.m.Contains(v)
}

// ContainsAll reports whether every value in vs is present in the set. With no
// arguments it reports true. It runs in O(k) on average, where k is the number
// of values in vs. A nil receiver is treated as empty.
func (s *Set[T]) ContainsAll(vs ...T) bool {
	for _, v := range vs {
		if !s.Contains(v) {
			return false
		}
	}

	return true
}

// ContainsAny reports whether at least one value in vs is present in the set.
// With no arguments it reports false. It runs in O(k) on average, where k is
// the number of values in vs. A nil receiver is treated as empty and reports
// false.
func (s *Set[T]) ContainsAny(vs ...T) bool {
	return slices.ContainsFunc(vs, s.Contains)
}

// First returns the oldest value without removing it. The boolean result is
// false when the set is empty, in which case the returned value is the zero
// value of T. It runs in O(1). A nil receiver is treated as empty and returns
// (zero, false).
func (s *Set[T]) First() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.First()

	return v, ok
}

// Last returns the newest value without removing it. The boolean result is
// false when the set is empty, in which case the returned value is the zero
// value of T. It runs in O(1). A nil receiver is treated as empty and returns
// (zero, false).
func (s *Set[T]) Last() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.Last()

	return v, ok
}

// All returns an iterator over the set's values from oldest to newest, without
// consuming them; the set is left unchanged. The returned [iter.Seq] is a
// Go 1.23 range-over-func iterator, so it can be used directly in a range
// loop. Breaking out of the loop early stops iteration cleanly. Mutating the
// set during iteration is undefined behavior. A nil receiver is treated as
// empty and yields nothing.
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s == nil {
			return
		}

		for v := range s.m.Keys() {
			if !yield(v) {
				return
			}
		}
	}
}

// Backward returns an iterator over the set's values from newest to oldest,
// without consuming them; the set is left unchanged. The returned [iter.Seq]
// is a Go 1.23 range-over-func iterator, so it can be used directly in a range
// loop. Breaking out of the loop early stops iteration cleanly. Mutating the
// set during iteration is undefined behavior. A nil receiver is treated as
// empty and yields nothing.
func (s *Set[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s == nil {
			return
		}

		for v := range s.m.Backward() {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the set with the same order.
// The two sets share no backing storage, so mutating one does not affect the
// other. Values are copied as-is: if T is a pointer or contains references,
// the pointed-to data is shared between the copies. The clone has no holes,
// and its capacity equals its length. It runs in O(n).
//
// A nil receiver clones to a new, usable empty set.
func (s *Set[T]) Clone() *Set[T] {
	if s == nil {
		return New[T]()
	}

	return &Set[T]{
		m: *s.m.Clone(),
	}
}

// Slice returns a copy of the set's values from oldest to newest, the same
// order [Set.All] yields. The returned slice is independent of the set, so
// callers may modify it freely. An empty set returns nil; a nil receiver is
// treated as empty and likewise returns nil. It runs in O(n).
func (s *Set[T]) Slice() []T {
	if s == nil || s.m.Len() == 0 {
		return nil
	}

	out := make([]T, 0, s.m.Len())
	for v := range s.m.Keys() {
		out = append(out, v)
	}

	return out
}

// Add inserts v into the set and reports whether v was new. If v was already
// present, Add leaves the set unchanged and returns false; v keeps its
// original position. It runs in amortized O(1).
//
// Add requires a non-nil receiver and panics on a nil one: it must store the
// value somewhere. The zero value is usable, so [New] is optional.
func (s *Set[T]) Add(v T) bool {
	shared.MustNotNil(s, "orderedset", "Add")

	return s.m.Set(v, struct{}{})
}

// AddN inserts every value in vs into the set, in argument order. Values
// already present keep their original position. With no arguments it is a
// no-op. It runs in amortized O(k), where k is the number of values in vs.
//
// AddN requires a non-nil receiver and panics on a nil one, even when called
// with no arguments: it must store the values somewhere. The zero value is
// usable, so [New] is optional.
func (s *Set[T]) AddN(vs ...T) {
	shared.MustNotNil(s, "orderedset", "AddN")

	for _, v := range vs {
		s.m.Set(v, struct{}{})
	}
}

// AddSet inserts every value of other into s, appending new values in the
// order of other. Values already in s keep their position. It is the in-place
// form of [Set.Union]. A nil other is treated as an empty set and makes AddSet
// a no-op. It runs in amortized O(m), where m is the number of values in
// other.
//
// AddSet requires a non-nil receiver and panics on a nil one: it must store the
// values somewhere. The zero value is usable, so [New] is optional.
func (s *Set[T]) AddSet(other *Set[T]) {
	shared.MustNotNil(s, "orderedset", "AddSet")

	if other == nil || s == other {
		return
	}

	for v := range other.m.Keys() {
		s.m.Set(v, struct{}{})
	}
}

// Remove deletes v from the set and reports whether v was present. The
// remaining values keep their order. It runs in amortized O(1). A nil receiver
// is treated as empty and returns false.
func (s *Set[T]) Remove(v T) bool {
	if s == nil {
		return false
	}

	_, ok := s.m.Remove(v)

	return ok
}

// RemoveSet deletes every value of other from s. The remaining values keep
// their order. It is the in-place form of [Set.Difference]. Passing the set
// itself empties it while keeping its capacity. Nil operands are treated as
// empty sets, so a nil receiver or a nil other makes RemoveSet a no-op. It
// runs in amortized O(m), where m is the number of values in other.
func (s *Set[T]) RemoveSet(other *Set[T]) {
	if s == nil || other == nil {
		return
	}

	if s == other {
		s.m.Reset()
		return
	}

	for v := range other.m.Keys() {
		s.m.Remove(v)
	}
}

// RetainSet keeps only the values of s that are also present in other. The
// remaining values keep their order. It is the in-place form of
// [Set.Intersection]. Nil operands are treated as empty sets: a nil other
// empties s while keeping its capacity, and a nil receiver makes RetainSet a
// no-op. It runs in O(n) on average.
func (s *Set[T]) RetainSet(other *Set[T]) {
	if s == nil || s == other {
		return
	}

	if other == nil {
		s.m.Reset()
		return
	}

	s.m.RemoveFunc(func(v T, _ struct{}) bool {
		return !other.m.Contains(v)
	})
}

// RemoveFunc deletes every value for which del returns true and returns the
// number of deleted values. del is called exactly once per value, from oldest
// to newest, and must not modify the set or panic. The remaining values keep
// their order, and the entry slice is left without holes. It runs in O(n).
//
// RemoveFunc panics if del is nil. A nil receiver is treated as empty and
// returns 0.
func (s *Set[T]) RemoveFunc(del func(T) bool) int {
	if del == nil {
		panic("orderedset: nil predicate")
	}

	if s == nil {
		return 0
	}

	return s.m.RemoveFunc(func(v T, _ struct{}) bool {
		return del(v)
	})
}

// PopFirst removes and returns the oldest value. The boolean result is false
// when the set is empty, in which case the returned value is the zero value of
// T. The vacated slot is zeroed so it no longer retains a reference to the
// removed value. It runs in amortized O(1). A nil receiver is treated as empty
// and returns (zero, false).
func (s *Set[T]) PopFirst() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.PopFirst()

	return v, ok
}

// PopLast removes and returns the newest value. The boolean result is false
// when the set is empty, in which case the returned value is the zero value of
// T. The vacated slot is zeroed so it no longer retains a reference to the
// removed value. It runs in amortized O(1). A nil receiver is treated as empty
// and returns (zero, false).
func (s *Set[T]) PopLast() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.PopLast()

	return v, ok
}

// MoveToBack makes v the newest value and reports whether v was present. An
// absent value is not added. Moving the newest value is a no-op. It runs in
// amortized O(1). A nil receiver is treated as empty and reports false.
func (s *Set[T]) MoveToBack(v T) bool {
	if s == nil {
		return false
	}

	return s.m.MoveToBack(v)
}

// Union returns a new set with the values of s, followed by the values of each
// set in others that are not already present, in argument order. Nil operands
// are treated as empty sets. It runs in O(n+m) on average, where m is the
// total number of values in others.
func (s *Set[T]) Union(others ...*Set[T]) *Set[T] {
	out := s.Clone()
	for _, other := range others {
		if other == nil {
			continue
		}

		for v := range other.m.Keys() {
			out.m.Set(v, struct{}{})
		}
	}

	return out
}

// Intersection returns a new set with the values of s that are present in
// every set in others, in the order of s. Nil operands are treated as empty
// sets. With no others it returns a copy of s. It runs in O(n*k) on average,
// where k is the number of sets in others.
func (s *Set[T]) Intersection(others ...*Set[T]) *Set[T] {
	out := NewWithCap[T](s.Len())
	for v := range s.All() {
		if containedInAll(v, others) {
			out.m.Set(v, struct{}{})
		}
	}

	return out
}

// Difference returns a new set with the values of s that are absent from every
// set in others, in the order of s. Nil operands are treated as empty sets.
// With no others it returns a copy of s. It runs in O(n*k) on average, where k
// is the number of sets in others.
func (s *Set[T]) Difference(others ...*Set[T]) *Set[T] {
	out := NewWithCap[T](s.Len())
	for v := range s.All() {
		if !containedInAny(v, others) {
			out.m.Set(v, struct{}{})
		}
	}

	return out
}

// SymmetricDifference returns a new set with the values that are in exactly one
// of s and other: first those only in s, in the order of s, then those only in
// other, in the order of other. Nil operands are treated as empty sets. It runs
// in O(n+m) on average, where m is the number of values in other.
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	out := s.Clone()
	if other == nil {
		return out
	}

	for v := range other.m.Keys() {
		if _, ok := out.m.Remove(v); !ok {
			out.m.Set(v, struct{}{})
		}
	}

	return out
}

// Equal reports whether s and other contain the same values, regardless of
// order. To also compare order, use [slices.Equal] on the results of
// [Set.Slice]. Nil operands are treated as empty sets. It runs in O(n) on
// average.
func (s *Set[T]) Equal(other *Set[T]) bool {
	return s.Len() == other.Len() && s.IsSubset(other)
}

// IsSubset reports whether every value of s is also present in other. Nil
// operands are treated as empty sets. It runs in O(n) on average.
func (s *Set[T]) IsSubset(other *Set[T]) bool {
	if s.Len() > other.Len() {
		return false
	}

	if s == nil {
		return true
	}

	for v := range s.m.Keys() {
		if !other.Contains(v) {
			return false
		}
	}

	return true
}

// IsSuperset reports whether every value of other is also present in s. Nil
// operands are treated as empty sets. It runs in O(m) on average, where m is
// the number of values in other.
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// IsDisjoint reports whether s and other have no values in common. Nil
// operands are treated as empty sets. It iterates over the smaller set, so it
// runs in O(min(n, m)) on average, where m is the number of values in other.
func (s *Set[T]) IsDisjoint(other *Set[T]) bool {
	small, large := s, other
	if small.Len() > large.Len() {
		small, large = large, small
	}

	for v := range small.All() {
		if large.Contains(v) {
			return false
		}
	}

	return true
}

// Clear removes all values and releases the backing storage, so its memory
// becomes eligible for garbage collection. After Clear, Cap reports 0. To empty
// the set while keeping its capacity for reuse, use [Set.Reset] instead. It
// runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Clear() {
	shared.MustNotNil(s, "orderedset", "Clear")
	s.m.Clear()
}

// Reset removes all values but keeps the backing storage for reuse, so Cap is
// preserved. The values are zeroed so the storage no longer retains references
// to them. To also release the backing storage, use [Set.Clear] instead. It
// runs in O(n).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Reset() {
	shared.MustNotNil(s, "orderedset", "Reset")
	s.m.Reset()
}

// Shrink removes all holes, copies the live values into a right-sized entry
// slice, and rebuilds the index, so memory held by a set that grew large and
// has since shrunk becomes eligible for garbage collection immediately. Go
// maps never shrink in place, so Shrink is the only way to reclaim the index's
// memory short of [Set.Clear].
//
// Shrink runs in O(n) and allocates; the entry slice is copied only when it
// has holes or spare capacity, but the index is always rebuilt. For the
// cheaper, non-copying variant that only reslices the entry slice and lets
// memory be reclaimed lazily on the next growth, use [Set.Clip]. Shrink is the
// counterpart of [Set.Grow].
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Shrink() {
	shared.MustNotNil(s, "orderedset", "Shrink")
	s.m.Shrink()
}

// Clip reduces the capacity of the entry slice to its current length without
// copying, by reslicing it (see [slices.Clip]). It runs in O(1) and does not
// allocate. Holes and the index are left as they are, and because the backing
// array is retained, the unused tail memory is not returned to the garbage
// collector until a later growth reallocates. For an immediate, copying
// reclaim, use [Set.Shrink].
//
// Clip requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Clip() {
	shared.MustNotNil(s, "orderedset", "Clip")
	s.m.Clip()
}

// Grow ensures the entry slice has spare capacity for at least n more values
// without reallocating during subsequent adds, growing it if needed. It is the
// counterpart of [Set.Shrink]. Grow runs in O(n) when it reallocates and is a
// no-op when n is non-positive or the capacity is already sufficient. It panics
// if the new capacity overflows, the same condition as [slices.Grow]. The index
// map is not preallocated by Grow; [NewWithCap] sizes both.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Grow(n int) {
	shared.MustNotNil(s, "orderedset", "Grow")
	s.m.Grow(n)
}

// containedInAll reports whether v is present in every set in others. A nil
// set in others is treated as empty.
func containedInAll[T comparable](v T, others []*Set[T]) bool {
	for _, other := range others {
		if !other.Contains(v) {
			return false
		}
	}

	return true
}

// containedInAny reports whether v is present in at least one set in others. A
// nil set in others is treated as empty.
func containedInAny[T comparable](v T, others []*Set[T]) bool {
	for _, other := range others {
		if other.Contains(v) {
			return true
		}
	}

	return false
}
