package orderedset_test

import (
	"fmt"
	"slices"

	"github.com/Nergous/structures/orderedset"
)

// An ordered set is a set that remembers the order in which values were first
// added: unlike a plain set, it always yields its values from the oldest to the
// newest.
func Example() {
	s := orderedset.New[string]()
	s.Add("b")
	s.Add("a")
	s.Add("b") // existing: place stays

	for !s.IsEmpty() {
		v, _ := s.PopFirst()
		fmt.Println(v)
	}
	// Output:
	// b
	// a
}

// NewWithCap preallocates room for a known number of values. Use it when the
// final size is known up front to avoid reallocations during Add.
func ExampleNewWithCap() {
	s := orderedset.NewWithCap[string](3)
	s.Add("a")
	s.Add("b")
	s.Add("c")

	fmt.Println(s.Len(), s.Cap())
	// Output: 3 3
}

// Of builds a set from explicit values. A duplicate keeps the position of its
// first occurrence.
func ExampleOf() {
	s := orderedset.Of("b", "a", "b", "c", "a")

	fmt.Println(s.Slice())
	// Output: [b a c]
}

// From builds a set from a slice. A duplicate keeps the position of its first
// occurrence, and the set never aliases the slice.
func ExampleFrom() {
	vs := []int{3, 1, 3, 2, 1}
	s := orderedset.From(vs)
	vs[0] = 99 // does not affect s

	fmt.Println(s.Slice())
	// Output: [3 1 2]
}

// ContainsAll reports whether every given value is in the set. With no values
// it reports true.
func ExampleSet_ContainsAll() {
	s := orderedset.Of("read", "write")

	fmt.Println(s.ContainsAll("read", "write"))
	fmt.Println(s.ContainsAll("read", "exec"))
	fmt.Println(s.ContainsAll())
	// Output:
	// true
	// false
	// true
}

// ContainsAny reports whether at least one given value is in the set. With no
// values it reports false.
func ExampleSet_ContainsAny() {
	s := orderedset.Of("read", "write")

	fmt.Println(s.ContainsAny("exec", "write"))
	fmt.Println(s.ContainsAny("exec"))
	fmt.Println(s.ContainsAny())
	// Output:
	// true
	// false
	// false
}

// First and Last return the oldest and the newest value without removing them.
// The boolean result is false for an empty set.
func ExampleSet_First() {
	s := orderedset.Of("a", "b", "c")

	v, _ := s.First()
	fmt.Println("oldest:", v)
	v, _ = s.Last()
	fmt.Println("newest:", v)
	fmt.Println("len unchanged:", s.Len())

	empty := orderedset.New[string]()
	_, ok := empty.First()
	fmt.Println(ok)
	// Output:
	// oldest: a
	// newest: c
	// len unchanged: 3
	// false
}

// All returns a range-over-func iterator over the values from oldest to newest
// without consuming them. The set is left intact after iteration. Unlike a
// plain set, the order is insertion order, every time.
func ExampleSet_All() {
	s := orderedset.Of("banana", "apple", "cherry")

	for v := range s.All() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", s.Len())
	// Output:
	// banana
	// apple
	// cherry
	// len unchanged: 3
}

// Backward returns a range-over-func iterator over the values from newest to
// oldest without consuming them. The set is left intact after iteration.
func ExampleSet_Backward() {
	s := orderedset.Of("banana", "apple", "cherry")

	for v := range s.Backward() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", s.Len())
	// Output:
	// cherry
	// apple
	// banana
	// len unchanged: 3
}

// Clone returns an independent shallow copy with the same order. Mutating one
// set does not affect the other, since they share no backing storage.
func ExampleSet_Clone() {
	a := orderedset.Of("a", "b", "c")

	b := a.Clone()
	b.Remove("a") // removes "a" from b only

	fmt.Println(a.Slice())
	fmt.Println(b.Slice())
	// Output:
	// [a b c]
	// [b c]
}

// Slice returns a copy of the values from oldest to newest, the same order All
// yields. The returned slice is independent of the set.
func ExampleSet_Slice() {
	s := orderedset.Of(3, 1, 2)

	out := s.Slice()
	fmt.Println(out)

	// The slice is detached: modifying it leaves the set untouched.
	out[0] = 99
	first, _ := s.First()
	fmt.Println("set first:", first)

	// An empty set returns nil.
	empty := orderedset.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// [3 1 2]
	// set first: 3
	// true
}

// Add reports whether the value was new. Adding an existing value is a no-op:
// it keeps its original place.
func ExampleSet_Add() {
	s := orderedset.New[string]()

	fmt.Println(s.Add("a"))
	fmt.Println(s.Add("b"))
	fmt.Println(s.Add("a")) // existing: place stays
	fmt.Println(s.Slice())
	// Output:
	// true
	// true
	// false
	// [a b]
}

// AddN adds several values in order, skipping the ones already present.
func ExampleSet_AddN() {
	s := orderedset.Of("a")
	s.AddN("b", "a", "c", "b")

	fmt.Println(s.Slice())
	// Output: [a b c]
}

// AddSet adds the other set's values to the receiver in place, in its order,
// skipping the ones already present.
func ExampleSet_AddSet() {
	s := orderedset.Of(1, 2)
	s.AddSet(orderedset.Of(2, 3, 1, 4))

	fmt.Println(s.Slice())
	// Output: [1 2 3 4]
}

// Remove deletes a value and reports whether it was present. The remaining
// values keep their order.
func ExampleSet_Remove() {
	s := orderedset.Of("a", "b", "c")

	fmt.Println(s.Remove("b"))
	fmt.Println(s.Remove("zzz")) // not present
	fmt.Println(s.Slice())
	// Output:
	// true
	// false
	// [a c]
}

// RemoveSet deletes every value of the other set from the receiver in place;
// the rest keep their order.
func ExampleSet_RemoveSet() {
	s := orderedset.Of(1, 2, 3, 4, 5)
	s.RemoveSet(orderedset.Of(4, 2))

	fmt.Println(s.Slice())
	// Output: [1 3 5]
}

// RetainSet keeps only the values also present in the other set, in the
// receiver's order.
func ExampleSet_RetainSet() {
	s := orderedset.Of("x", "b", "y", "a")
	s.RetainSet(orderedset.Of("a", "b"))

	fmt.Println(s.Slice())
	// Output: [b a]
}

// RemoveFunc deletes every value the predicate selects in a single pass and
// reports how many were removed. Deleting inside a range loop over the set is
// not supported; this is the way to do it.
func ExampleSet_RemoveFunc() {
	s := orderedset.Of(1, 2, 3, 4, 5, 6, 7, 8)

	n := s.RemoveFunc(func(v int) bool {
		return v%2 == 0
	})

	fmt.Println(n, s.Slice())
	// Output: 4 [1 3 5 7]
}

// PopFirst removes and returns the oldest value. The boolean result reports
// whether the set was non-empty; on an empty set it is false and the returned
// value is the zero value.
func ExampleSet_PopFirst() {
	s := orderedset.Of("a", "b")

	v, ok := s.PopFirst()
	fmt.Println(v, ok)
	s.PopFirst()         // removes "b"
	_, ok = s.PopFirst() // set is now empty
	fmt.Println(ok)
	// Output:
	// a true
	// false
}

// PopLast removes and returns the newest value. The boolean result reports
// whether the set was non-empty; on an empty set it is false and the returned
// value is the zero value.
func ExampleSet_PopLast() {
	s := orderedset.Of("a", "b")

	v, ok := s.PopLast()
	fmt.Println(v, ok)
	s.PopLast()         // removes "a"
	_, ok = s.PopLast() // set is now empty
	fmt.Println(ok)
	// Output:
	// b true
	// false
}

// MoveToBack makes a value the newest one and reports whether it was present.
// An absent value is not added.
func ExampleSet_MoveToBack() {
	s := orderedset.Of("a", "b", "c")

	fmt.Println(s.MoveToBack("a"))
	fmt.Println(s.MoveToBack("zzz")) // not present
	fmt.Println(s.Slice())
	// Output:
	// true
	// false
	// [b c a]
}

// Union returns a new set: the receiver's values in their order, followed by
// values new to it in the order the other sets list them.
func ExampleSet_Union() {
	a := orderedset.Of("b", "a")
	b := orderedset.Of("c", "a", "d")

	fmt.Println(a.Union(b).Slice())
	// Output: [b a c d]
}

// Intersection returns a new set with the values present in every input set,
// in the receiver's order.
func ExampleSet_Intersection() {
	a := orderedset.Of("d", "c", "b", "a")
	b := orderedset.Of("a", "b", "x")

	fmt.Println(a.Intersection(b).Slice())
	// Output: [b a]
}

// Difference returns a new set with the receiver's values that are in none of
// the other sets, in the receiver's order.
func ExampleSet_Difference() {
	a := orderedset.Of(1, 2, 3, 4, 5)

	fmt.Println(a.Difference(orderedset.Of(4, 1), orderedset.Of(5)).Slice())
	// Output: [2 3]
}

// SymmetricDifference returns a new set with the values in exactly one of the
// two sets: the receiver's own values in their order, followed by the other
// set's own values in its order.
func ExampleSet_SymmetricDifference() {
	a := orderedset.Of(1, 2, 3, 4)
	b := orderedset.Of(5, 3, 6, 1)

	fmt.Println(a.SymmetricDifference(b).Slice())
	// Output: [2 4 5 6]
}

// Equal reports whether two sets hold the same values. It compares membership
// only and ignores order; to compare order as well, compare the slices.
func ExampleSet_Equal() {
	a := orderedset.Of(1, 2, 3)
	b := orderedset.Of(3, 2, 1)

	fmt.Println(a.Equal(b))
	fmt.Println(slices.Equal(a.Slice(), b.Slice()))
	// Output:
	// true
	// false
}

// IsSubset reports whether every value of the receiver is in the other set;
// IsSuperset is the reverse. Order does not matter.
func ExampleSet_IsSubset() {
	a := orderedset.Of(1, 2)
	b := orderedset.Of(3, 2, 1)

	fmt.Println(a.IsSubset(b))
	fmt.Println(b.IsSubset(a))
	fmt.Println(b.IsSuperset(a))
	// Output:
	// true
	// false
	// true
}

// IsDisjoint reports whether the two sets share no values.
func ExampleSet_IsDisjoint() {
	a := orderedset.Of(1, 2)

	fmt.Println(a.IsDisjoint(orderedset.Of(3, 4)))
	fmt.Println(a.IsDisjoint(orderedset.Of(2, 3)))
	// Output:
	// true
	// false
}

// Clear empties the set and releases its backing storage: capacity drops back
// to zero, and the set stays usable.
func ExampleSet_Clear() {
	s := orderedset.NewWithCap[int](8)
	s.AddN(1, 2)

	s.Clear()
	fmt.Println(s.Len(), s.Cap())

	s.Add(3)
	fmt.Println(s.Slice())
	// Output:
	// 0 0
	// [3]
}

// Reset, unlike Clear, empties the set but keeps the allocated backing
// storage, so it can be reused without a new allocation.
func ExampleSet_Reset() {
	s := orderedset.NewWithCap[int](8)
	s.AddN(1, 2)

	s.Reset()
	fmt.Println(s.Len(), s.Cap())
	// Output: 0 8
}

// Shrink copies the values into a right-sized array, rebuilds the index, and
// frees the old storage immediately. Grow reserves capacity up front, and Clip
// trims spare capacity without copying.
func ExampleSet_Shrink() {
	s := orderedset.NewWithCap[int](1024)
	s.AddN(1, 2, 3)

	s.Grow(2000) // reserve room for 2000 more elements
	fmt.Println(s.Cap() >= 2003)

	s.Shrink() // copy into a right-sized array, free the old one now
	fmt.Println(s.Cap())

	s.Clip() // already tight: no-op
	fmt.Println(s.Cap())
	// Output:
	// true
	// 3
	// 3
}

// A nil *Set[T] behaves as an empty set for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a set
// created with New (or a zero value).
func Example_nilReceiver() {
	var s *orderedset.Set[int] // nil, never initialized

	fmt.Println(s.Len(), s.IsEmpty(), s.Cap())

	v, ok := s.PopFirst()
	fmt.Println(v, ok)
	// Output:
	// 0 true 0
	// 0 false
}

// The zero value of Set[T] is usable as an empty set, so New is optional.
func Example_zeroValue() {
	var s orderedset.Set[string]
	s.Add("b")
	s.Add("a")

	fmt.Println(s.Len(), s.Slice())
	// Output: 2 [b a]
}

// A practical example: de-duplicate a slice while keeping the first occurrence
// of every value in its original place, so the output is stable without
// sorting.
func Example_deduplicate() {
	seen := orderedset.New[string]()
	for _, name := range []string{"alice", "bob", "alice", "cora"} {
		if !seen.Add(name) {
			fmt.Println("duplicate:", name)
		}
	}

	fmt.Println(seen.Slice())
	// Output:
	// duplicate: alice
	// [alice bob cora]
}
