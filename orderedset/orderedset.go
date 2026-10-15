// Package orderedset provides a generic set of unique comparable values that
// remembers the order in which values were first added.
//
// Create a set with [New], [NewWithCap], [Of], or [From]. Values are inserted
// with [Set.Add] and [Set.AddN], removed with [Set.Remove], and queried with
// [Set.Contains], [Set.ContainsAll], and [Set.ContainsAny]. The API mirrors the
// unordered set package, plus methods that rely on order: [Set.First],
// [Set.Last], [Set.PopFirst], [Set.PopLast], [Set.MoveToBack], and
// [Set.Backward].
//
// A [Set] iterates in insertion order. Adding a value that is already present
// keeps its place; use [Set.MoveToBack] to make a value the newest one. Set
// algebra returns results in a defined order: [Set.Union] lists the values of
// the receiver first, then new values from each argument in turn;
// [Set.Intersection] and [Set.Difference] keep the order of the receiver;
// [Set.SymmetricDifference] lists the values only in the receiver, then those
// only in the argument. [Set.Equal] compares contents and ignores order.
//
// The set is a thin wrapper over an insertion-ordered map with empty values,
// so it shares that map's costs: Add, Remove, Contains, and the end operations
// run in O(1) on average or amortized. Removing a value leaves a hole that is
// compacted later, which is why [Set.Cap] counts holes and [Set.Shrink]
// reclaims them.
//
// [Set.All] and [Set.Backward] iterate without consuming the set. Mutating the
// set while iterating, including from the loop body, is undefined behavior.
// Use [Set.RemoveFunc] or [Set.RetainAll] to delete values selected during a
// pass.
//
// A nil *Set[T] behaves as an empty set for read-only methods, predicates,
// iteration, and removals. [Set.Clone] of a nil set returns a new, usable empty
// set. Methods that store or reset state ([Set.Add], [Set.AddN], [Set.AddSet],
// [Set.Grow], [Set.Shrink], [Set.Clip], [Set.Reset], [Set.Clear]) require a
// non-nil receiver and panic on a nil one. The zero value of Set[T] is ready to
// use.
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

// Set is a collection of unique comparable values that iterates in insertion
// order.
//
// The zero value is a usable empty set. A nil *Set[T] is treated as an empty
// set by read-only methods, predicates, and removals. Copying a Set value
// shares its storage with the original; use [Set.Clone] for an independent
// copy. Set is not safe for concurrent use.
type Set[T comparable] struct {
	m orderedmap.Map[T, struct{}]
}

// New returns an empty set.
func New[T comparable]() *Set[T] {
	return NewWithCap[T](0)
}

// NewWithCap returns an empty set with room for at least n values preallocated
// in both the entry slice and the index. A negative n is treated as 0.
func NewWithCap[T comparable](n int) *Set[T] {
	return &Set[T]{
		m: *orderedmap.NewWithCap[T, struct{}](n),
	}
}

// Of returns a set containing vs in order of first occurrence. Duplicate values
// are stored once, at the position where they first appear.
func Of[T comparable](vs ...T) *Set[T] {
	s := NewWithCap[T](len(vs))
	for _, v := range vs {
		s.m.Set(v, struct{}{})
	}

	return s
}

// From returns a set containing the values of s in order of first occurrence.
// Duplicate values are stored once. The set does not retain s.
func From[T comparable](s []T) *Set[T] {
	return Of(s...)
}

// Len returns the number of values in the set. A nil receiver reports 0.
func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}

	return s.m.Len()
}

// Cap returns the capacity of the entry slice, which counts live values and
// holes left by removals alike. It does not include the index, whose size Go
// does not expose. A nil receiver reports 0.
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

// Contains reports whether v is present in the set. A nil receiver reports
// false.
func (s *Set[T]) Contains(v T) bool {
	if s == nil {
		return false
	}

	return s.m.Contains(v)
}

// ContainsAll reports whether every value in vs is present in the set. With no
// arguments it reports true. A nil receiver is treated as an empty set.
func (s *Set[T]) ContainsAll(vs ...T) bool {
	for _, v := range vs {
		if !s.Contains(v) {
			return false
		}
	}

	return true
}

// ContainsAny reports whether at least one value in vs is present in the set.
// With no arguments it reports false. A nil receiver is treated as an empty set.
func (s *Set[T]) ContainsAny(vs ...T) bool {
	return slices.ContainsFunc(vs, s.Contains)
}

// All returns an iterator over the values from oldest to newest, without
// consuming them. Breaking out of the loop early stops iteration cleanly.
// Mutating the set during iteration is undefined behavior. A nil receiver is
// treated as empty and yields nothing.
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s == nil {
			return
		}
		for v := range s.m.All() {
			if !yield(v) {
				return
			}
		}
	}
}

// Slice returns a copy of the values from oldest to newest, the same order
// [Set.All] yields. The result shares no memory with the set, so callers may
// modify it freely. An empty set returns nil; a nil receiver is treated as
// empty and likewise returns nil. Runs in O(n).
func (s *Set[T]) Slice() []T {
	if s == nil || s.m.Len() == 0 {
		return nil
	}

	sl := make([]T, 0, s.m.Len())
	for v := range s.m.Keys() {
		sl = append(sl, v)
	}

	return sl
}

// Clone returns an independent shallow copy of the set with the same order.
// The two sets share no storage, so mutating one does not affect the other.
// Values are copied as-is: if they are pointers or contain references, the
// pointed-to data is shared between the copies. The copy has no holes and its
// capacity equals its length. Runs in O(n).
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

// Add inserts v and reports whether it was new. Adding a value that is already
// present returns false and keeps its original position. It runs in O(1)
// amortized.
//
// Add requires a non-nil receiver and panics on a nil one. The zero value is
// usable, so [New] is optional.
func (s *Set[T]) Add(v T) bool {
	shared.MustNotNil(s, "orderedset", "Add")
	return s.m.Set(v, struct{}{})
}

// AddN inserts vs in argument order. Values already present keep their original
// position. It runs in amortized O(k) for k values.
//
// AddN requires a non-nil receiver and panics on a nil one, even when called
// with no arguments. The zero value is usable, so [New] is optional.
func (s *Set[T]) AddN(vs ...T) {
	shared.MustNotNil(s, "orderedset", "AddN")

	for _, v := range vs {
		s.m.Set(v, struct{}{})
	}
}

// AddSet inserts every value of other that s does not already contain,
// appending them in the order of other. Values already in s keep their
// position. It is the in-place form of [Set.Union]. A nil other is treated as
// an empty set. It runs in O(m) for m values in other.
//
// AddSet requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) AddSet(other *Set[T]) {
	shared.MustNotNil(s, "orderedset", "AddSet")

	if other == nil {
		return
	}

	if other == s {
		return
	}

	for v := range other.m.Keys() {
		s.m.Set(v, struct{}{})
	}
}

// Remove deletes v and reports whether it was present. The remaining values
// keep their order. It runs in O(1) amortized. A nil receiver has nothing to
// remove and returns false.
func (s *Set[T]) Remove(v T) bool {
	if s == nil {
		return false
	}

	_, ok := s.m.Remove(v)

	return ok
}

// RemoveSet deletes every value of other from s. The remaining values keep
// their order. It is the in-place form of [Set.Difference]. Passing the set
// itself empties it while keeping its capacity. A nil receiver or a nil other
// makes RemoveSet a no-op. It runs in O(m) for m values in other.
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

// RetainAll keeps only the values that are also present in other. The remaining
// values keep their order. It is the in-place form of [Set.Intersection]. A nil
// other is treated as an empty set and empties s while keeping its capacity. A
// nil receiver makes RetainAll a no-op. It runs in O(n).
func (s *Set[T]) RetainAll(other *Set[T]) {
	if s == nil || s == other {
		return
	}

	if other == nil {
		s.m.Reset()
		return
	}

	s.m.RemoveFunc(func(v T, _ struct{}) bool {
		return !other.Contains(v)
	})
}

// Clear removes all values and releases the backing storage, so its memory
// becomes eligible for garbage collection. The set stays usable. After Clear,
// Cap reports 0. To empty the set while keeping its capacity for reuse, use
// [Set.Reset] instead.
//
// Clear requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Clear() {
	shared.MustNotNil(s, "orderedset", "Clear")
	s.m.Clear()
}

// Reset removes all values but keeps the backing storage for reuse, so Cap is
// preserved. The vacated slots are zeroed so they no longer retain references.
// To also release the storage, use [Set.Clear] instead.
//
// Reset requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Reset() {
	shared.MustNotNil(s, "orderedset", "Reset")
	s.m.Reset()
}

// Union returns a new set with the values of s, followed by the values of each
// set in others that are not already present, in argument order. Nil operands
// are treated as empty sets.
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
// sets. With no others it returns a copy of s.
func (s *Set[T]) Intersection(others ...*Set[T]) *Set[T] {
	out := NewWithCap[T](s.Len())
	for v := range s.All() {
		if containedInAll(v, others) {
			out.m.Set(v, struct{}{})
		}
	}

	return out
}

// containedInAll reports whether v is present in every set in sets.
func containedInAll[T comparable](v T, sets []*Set[T]) bool {
	for _, o := range sets {
		if !o.Contains(v) {
			return false
		}
	}
	return true
}

// Difference returns a new set with the values of s that are absent from every
// set in others, in the order of s. Nil operands are treated as empty sets.
// With no others it returns a copy of s.
func (s *Set[T]) Difference(others ...*Set[T]) *Set[T] {
	out := NewWithCap[T](s.Len())
	for v := range s.All() {
		if !containedInAny(v, others) {
			out.m.Set(v, struct{}{})
		}
	}
	return out
}

// containedInAny reports whether v is present in at least one set in sets.
func containedInAny[T comparable](v T, sets []*Set[T]) bool {
	for _, o := range sets {
		if o.Contains(v) {
			return true
		}
	}
	return false
}

// SymmetricDifference returns a new set with the values that are in exactly one
// of s and other: first those only in s, in the order of s, then those only in
// other, in the order of other. Nil operands are treated as empty sets.
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
// order. To also compare order, use slices.Equal(s.Slice(), other.Slice()).
// Nil operands are treated as empty sets.
func (s *Set[T]) Equal(other *Set[T]) bool {
	return s.Len() == other.Len() && s.IsSubset(other)
}

// IsSubset reports whether every value of s is also present in other. Nil
// operands are treated as empty sets.
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
// operands are treated as empty sets.
func (s *Set[T]) IsSuperset(other *Set[T]) bool {
	return other.IsSubset(s)
}

// IsDisjoint reports whether s and other have no values in common. It iterates
// over the smaller set. Nil operands are treated as empty sets.
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

// First returns the oldest value without removing it. The boolean result is
// false when the set is empty, in which case the value is the zero value of T.
// It runs in O(1). A nil receiver is treated as empty.
func (s *Set[T]) First() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.First()
	return v, ok
}

// Last returns the newest value without removing it. The boolean result is
// false when the set is empty, in which case the value is the zero value of T.
// It runs in O(1). A nil receiver is treated as empty.
func (s *Set[T]) Last() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.Last()

	return v, ok
}

// PopFirst removes and returns the oldest value. The boolean result is false
// when the set is empty, in which case the value is the zero value of T. It
// runs in O(1) amortized. A nil receiver has nothing to remove and returns
// false.
func (s *Set[T]) PopFirst() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.PopFirst()

	return v, ok
}

// PopLast removes and returns the newest value. The boolean result is false
// when the set is empty, in which case the value is the zero value of T. It
// runs in O(1) amortized. A nil receiver has nothing to remove and returns
// false.
func (s *Set[T]) PopLast() (T, bool) {
	if s == nil {
		var zero T
		return zero, false
	}

	v, _, ok := s.m.PopLast()

	return v, ok
}

// MoveToBack makes v the newest value and reports whether it was present. An
// absent value is not added. It runs in O(1) amortized. A nil receiver reports
// false.
func (s *Set[T]) MoveToBack(v T) bool {
	if s == nil {
		return false
	}

	return s.m.MoveToBack(v)
}

// Backward returns an iterator over the values from newest to oldest, without
// consuming them. Breaking out of the loop early stops iteration cleanly.
// Mutating the set during iteration is undefined behavior. A nil receiver is
// treated as empty and yields nothing.
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

// RemoveFunc deletes every value for which del returns true and returns the
// number of deleted values. del is called exactly once per value, from oldest
// to newest, and must not modify the set or panic. The remaining values keep
// their order, and the entry slice is left without holes. It runs in O(n).
//
// RemoveFunc panics if del is nil. A nil receiver has nothing to remove and
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

// Grow ensures the entry slice has spare capacity for at least n more values
// without reallocating during subsequent adds. The index grows on demand and is
// not preallocated by Grow; [NewWithCap] sizes both. Grow is a no-op when n is
// non-positive or the capacity is already sufficient.
//
// Grow requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Grow(n int) {
	shared.MustNotNil(s, "orderedset", "Grow")

	s.m.Grow(n)
}

// Clip reduces the capacity of the entry slice to its current length without
// copying. It runs in O(1) and does not allocate. Holes, the unused tail of the
// array, and the index are left as they are; use [Set.Shrink] to reclaim them.
//
// Clip requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Clip() {
	shared.MustNotNil(s, "orderedset", "Clip")
	s.m.Clip()
}

// Shrink removes all holes, copies the values into a right-sized slice, and
// rebuilds the index, so memory held by a set that grew large and has since
// shrunk becomes eligible for garbage collection immediately. Go maps never
// shrink in place, so Shrink is the only way to reclaim the index's memory
// short of [Set.Clear]. It runs in O(n) and allocates.
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Shrink() {
	shared.MustNotNil(s, "orderedset", "Shrink")

	s.m.Shrink()
}
