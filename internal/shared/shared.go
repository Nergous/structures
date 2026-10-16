// Package shared provides small helpers shared by the stack, queue, deque,
// set, binheap, pqueue, orderedmap, and orderedset packages.
//
// It is internal to this module: the helpers exist to keep behavior and
// messages identical across packages, not as public API.
package shared

import (
	"cmp"
	"slices"
)

// MustNotNil panics with a descriptive message when recv is nil. pkg is the
// name of the calling package and op the name of the method; together they
// produce messages of the form "pkg: Op on nil receiver".
func MustNotNil[T any](recv *T, pkg, op string) {
	if recv == nil {
		panic(pkg + ": " + op + " on nil receiver")
	}
}

// Greater orders values in descending order. Passing it where a comparison
// function is expected turns a min-ordered structure into a max-ordered one.
func Greater[T cmp.Ordered](a, b T) int {
	return cmp.Compare(b, a)
}

// ShrinkSlice returns a slice with the same elements as s and capacity equal
// to its length. When s is already right-sized it is returned unchanged;
// otherwise the elements are copied into a new array, so the old one becomes
// eligible for garbage collection. It runs in O(n) when it copies.
func ShrinkSlice[T any](s []T) []T {
	if cap(s) == len(s) {
		return s
	}

	trimmed := make([]T, len(s))
	copy(trimmed, s)

	return trimmed
}

// GrowSlice returns a slice with the same elements as s and spare capacity for
// at least n more elements. When n is non-positive or the capacity is already
// sufficient, s is returned unchanged; otherwise the elements are copied into a
// larger array, as [slices.Grow] does. It runs in O(n) when it copies and
// panics if the new capacity overflows, the same condition as [slices.Grow].
func GrowSlice[T any](s []T, n int) []T {
	if n <= 0 {
		return s
	}

	return slices.Grow(s, n)
}
