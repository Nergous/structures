// Package set provides a generic, map-backed set of unique comparable values.
//
// Create a set with [New], [NewWithCap], [Of], or [From]. Values are added
// with [Set.Add] or [Set.AddN], removed with [Set.Remove] or [Set.RemoveFunc],
// and queried with [Set.Contains], [Set.ContainsAll], and [Set.ContainsAny].
// [Set.Union], [Set.Intersection], [Set.Difference], and
// [Set.SymmetricDifference] return new sets; [Set.AddSet], [Set.RemoveSet],
// and [Set.RetainSet] are their in-place forms. Go maps do not expose their
// capacity, so a Set has no Cap, Clip, or Grow: [NewWithCap] sizes the
// backing map up front and [Set.Shrink] rebuilds it right-sized.
//
// Add, Remove, and Contains run in O(1) on average. Len and IsEmpty run in
// O(1). The cost of set algebra depends on the sizes of its operands; see
// each method's documentation. Methods that copy or iterate over every value,
// such as All, Clone, and Slice, run in O(n).
//
// Iteration order is unspecified. The set is backed by a Go map, so [Set.All]
// and [Set.Slice] may return values in a different order between calls. Sort
// the result outside the package when stable output is needed.
//
// [Set.All] iterates without consuming the set. Mutating the set while
// iterating, including from the loop body, is undefined behavior. Use
// [Set.RetainSet] or [Set.RemoveFunc] to delete values selected by a
// predicate.
//
// The zero value of Set[T] is usable as an empty set. [Set.Add] allocates the
// backing map on first insertion. A nil *Set[T] behaves as an empty set for
// every read-only method and for removal methods such as [Set.Remove], so the
// zero pointer value is safe to query without first calling [New]. Methods
// that store, clear, reset, or resize data ([Set.Add], [Set.AddN],
// [Set.AddSet], [Set.Clear], [Set.Reset], [Set.Shrink]) require a non-nil
// receiver and panic on a nil one; see each method's documentation.
//
// A Set is not safe for concurrent use: it performs no internal locking.
// Callers that share a set across goroutines must provide their own
// synchronization.
package set

import (
	"iter"
	"slices"

	"github.com/Nergous/structures/internal/shared"
)

// Set is a generic collection of unique comparable values backed by a Go map.
// Iteration order is unspecified.
//
// The zero value is a valid empty set. A nil *Set[T] is treated as a valid
// empty set by read-only and removal methods, but methods that store, clear,
// reset, or resize data require a non-nil receiver. Copying a Set value shares
// its storage with the original; use [Set.Clone] for an independent copy.
//
// Set is not safe for concurrent use.
type Set[T comparable] struct {
	m map[T]struct{}
}

// New returns an empty set with no preallocated capacity.
func New[T comparable]() *Set[T] {
	return NewWithCap[T](0)
}

// NewWithCap returns an empty set with a backing map preallocated for at least
// n values. Use it when the maximum size is known up front to avoid
// reallocations while adding. A negative n is treated as 0.
func NewWithCap[T comparable](n int) *Set[T] {
	n = max(n, 0)

	return &Set[T]{
		m: make(map[T]struct{}, n),
	}
}

// Of returns a set with the values of vs. Duplicate values are stored once.
// It runs in O(n).
func Of[T comparable](vs ...T) *Set[T] {
	out := NewWithCap[T](len(vs))
	for _, v := range vs {
		out.m[v] = struct{}{}
	}

	return out
}

// From returns a set with the values of s. Duplicate values are stored once.
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

	return len(s.m)
}

// IsEmpty reports whether the set has no values. A nil receiver reports true.
func (s *Set[T]) IsEmpty() bool {
	return s == nil || len(s.m) == 0
}

// Contains reports whether v is present in the set. It runs in O(1) on
// average. A nil receiver is treated as empty and reports false.
func (s *Set[T]) Contains(v T) bool {
	if s == nil {
		return false
	}

	_, ok := s.m[v]

	return ok
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

// All returns an iterator over the set's values in unspecified order, without
// consuming them; the set is left unchanged. The order may differ between
// calls. The returned [iter.Seq] is a Go 1.23 range-over-func iterator, so it
// can be used directly in a range loop. Breaking out of the loop early stops
// iteration cleanly. Mutating the set during iteration is undefined behavior.
// A nil receiver is treated as empty and yields nothing.
func (s *Set[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s == nil {
			return
		}

		for v := range s.m {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns an independent shallow copy of the set. The two sets share no
// backing map, so mutating one does not affect the other. Values are copied
// as-is: if T is a pointer or contains references, the pointed-to data is
// shared between the copies. The clone's backing map is sized for its length.
// It runs in O(n).
//
// A nil receiver clones to a new, usable empty set.
func (s *Set[T]) Clone() *Set[T] {
	if s == nil {
		return New[T]()
	}

	c := NewWithCap[T](len(s.m))
	for v := range s.m {
		c.m[v] = struct{}{}
	}

	return c
}

// Slice returns a copy of the set's values in unspecified order. The returned
// slice is independent of the set, so callers may modify it freely. An empty
// set returns nil; a nil receiver is treated as empty and likewise returns
// nil. It runs in O(n).
func (s *Set[T]) Slice() []T {
	if s == nil || len(s.m) == 0 {
		return nil
	}

	out := make([]T, 0, len(s.m))
	for v := range s.m {
		out = append(out, v)
	}

	return out
}

// Add inserts v into the set and reports whether v was new. If v was already
// present, Add leaves the set unchanged and returns false. It runs in O(1) on
// average.
//
// Add requires a non-nil receiver and panics on a nil one: it must store the
// value somewhere. The zero value is usable, so [New] is optional.
func (s *Set[T]) Add(v T) bool {
	shared.MustNotNil(s, "set", "Add")

	if _, ok := s.m[v]; ok {
		return false
	}

	if s.m == nil {
		s.m = make(map[T]struct{})
	}

	s.m[v] = struct{}{}

	return true
}

// AddN inserts every value in vs into the set. Duplicate values are stored
// once. With no arguments it is a no-op. It runs in O(k) on average, where k
// is the number of values in vs.
//
// AddN requires a non-nil receiver and panics on a nil one, even when called
// with no arguments: it must store the values somewhere. The zero value is
// usable, so [New] is optional.
func (s *Set[T]) AddN(vs ...T) {
	shared.MustNotNil(s, "set", "AddN")

	if len(vs) == 0 {
		return
	}

	if s.m == nil {
		s.m = make(map[T]struct{}, len(vs))
	}

	for _, v := range vs {
		s.m[v] = struct{}{}
	}
}

// AddSet inserts every value of other into s. It is the in-place form of
// [Set.Union]. A nil other is treated as an empty set and makes AddSet a
// no-op. It runs in O(m) on average, where m is the number of values in other.
//
// AddSet requires a non-nil receiver and panics on a nil one: it must store the
// values somewhere. The zero value is usable, so [New] is optional.
func (s *Set[T]) AddSet(other *Set[T]) {
	shared.MustNotNil(s, "set", "AddSet")

	if other == nil || s == other {
		return
	}

	if s.m == nil {
		s.m = make(map[T]struct{}, len(other.m))
	}

	for v := range other.m {
		s.m[v] = struct{}{}
	}
}

// Remove deletes v from the set and reports whether v was present. It runs in
// O(1) on average. A nil receiver is treated as empty and returns false.
func (s *Set[T]) Remove(v T) bool {
	if s == nil {
		return false
	}

	_, ok := s.m[v]
	delete(s.m, v)

	return ok
}

// RemoveSet deletes every value of other from s. It is the in-place form of
// [Set.Difference]. Passing the set itself empties it while keeping the
// backing map. Nil operands are treated as empty sets, so a nil receiver or a
// nil other makes RemoveSet a no-op. It runs in O(m) on average, where m is
// the number of values in other.
func (s *Set[T]) RemoveSet(other *Set[T]) {
	if s == nil || other == nil {
		return
	}

	if s == other {
		clear(s.m)
		return
	}

	for v := range other.m {
		delete(s.m, v)
	}
}

// RetainSet keeps only the values of s that are also present in other. It is
// the in-place form of [Set.Intersection]. Nil operands are treated as empty
// sets: a nil other empties s while keeping the backing map, and a nil
// receiver makes RetainSet a no-op. It runs in O(n) on average.
func (s *Set[T]) RetainSet(other *Set[T]) {
	if s == nil || s == other {
		return
	}

	if other == nil {
		clear(s.m)
		return
	}

	for v := range s.m {
		if _, ok := other.m[v]; !ok {
			delete(s.m, v)
		}
	}
}

// RemoveFunc deletes every value for which del returns true and returns the
// number of deleted values. del is called exactly once per value, in
// unspecified order, and must not modify the set. It runs in O(n).
//
// RemoveFunc panics if del is nil. A nil receiver is treated as empty and
// returns 0.
func (s *Set[T]) RemoveFunc(del func(T) bool) int {
	if del == nil {
		panic("set: nil predicate")
	}

	if s == nil {
		return 0
	}

	n := 0
	for v := range s.m {
		if del(v) {
			delete(s.m, v)
			n++
		}
	}

	return n
}

// Union returns a new set with the values of s and of every set in others.
// Nil operands are treated as empty sets. It runs in O(n+m) on average, where
// m is the total number of values in others.
func (s *Set[T]) Union(others ...*Set[T]) *Set[T] {
	out := s.Clone()
	for _, other := range others {
		if other == nil {
			continue
		}

		for v := range other.m {
			out.m[v] = struct{}{}
		}
	}

	return out
}

// Intersection returns a new set with the values of s that are present in
// every set in others. Nil operands are treated as empty sets. With no others
// it returns a copy of s. It runs in O(n*k) on average, where k is the number
// of sets in others.
func (s *Set[T]) Intersection(others ...*Set[T]) *Set[T] {
	out := s.Clone()
	for _, other := range others {
		for v := range out.m {
			if !other.Contains(v) {
				delete(out.m, v)
			}
		}
	}

	return out
}

// Difference returns a new set with the values of s that are absent from
// every set in others. Nil operands are treated as empty sets. With no others
// it returns a copy of s. It runs in O(n*k) on average, where k is the number
// of sets in others.
func (s *Set[T]) Difference(others ...*Set[T]) *Set[T] {
	out := s.Clone()
	for _, other := range others {
		for v := range out.m {
			if other.Contains(v) {
				delete(out.m, v)
			}
		}
	}

	return out
}

// SymmetricDifference returns a new set with the values that are in exactly
// one of s and other. Nil operands are treated as empty sets. It runs in
// O(n+m) on average, where m is the number of values in other.
func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T] {
	out := s.Clone()
	if other == nil {
		return out
	}

	for v := range other.m {
		if _, ok := out.m[v]; ok {
			delete(out.m, v)
		} else {
			out.m[v] = struct{}{}
		}
	}

	return out
}

// Equal reports whether s and other contain the same values. Nil operands are
// treated as empty sets. It runs in O(n) on average.
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

	for v := range s.m {
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

// Clear removes all values and releases the backing map, so its memory
// becomes eligible for garbage collection. To empty the set while keeping the
// backing map for reuse, use [Set.Reset] instead. It runs in O(1).
//
// Clear requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Clear() {
	shared.MustNotNil(s, "set", "Clear")
	s.m = nil
}

// Reset removes all values but keeps the backing map for reuse. The values are
// cleared so the map no longer retains references to them. To also release
// the backing map, use [Set.Clear] instead. It runs in O(n).
//
// Reset requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Reset() {
	shared.MustNotNil(s, "set", "Reset")
	clear(s.m)
}

// Shrink copies the values into a new backing map sized for the current
// length, so memory held by a set that grew large and has since shrunk becomes
// eligible for garbage collection immediately. Go maps never shrink in place,
// so Shrink is the only way to reclaim that memory short of [Set.Clear]. An
// empty set releases its backing map, as after Clear.
//
// Shrink runs in O(n) and allocates a new map for a non-empty set. Go maps do
// not expose their capacity, so a Set has no Cap, Clip, or Grow, and Shrink
// rebuilds the map even when it is already right-sized.
//
// Shrink requires a non-nil receiver and panics on a nil one.
func (s *Set[T]) Shrink() {
	shared.MustNotNil(s, "set", "Shrink")

	if len(s.m) == 0 {
		s.m = nil
		return
	}

	trimmed := make(map[T]struct{}, len(s.m))
	for v := range s.m {
		trimmed[v] = struct{}{}
	}

	s.m = trimmed
}
