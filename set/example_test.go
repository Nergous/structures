package set_test

import (
	"fmt"
	"slices"

	"github.com/Nergous/structures/set"
)

// A set is a collection that stores each value at most once. Iteration order
// is unspecified, so the values are sorted before printing.
func Example() {
	s := set.New[int]()
	s.Add(1)
	s.Add(2)
	s.Add(1)

	fmt.Println(s.Len())
	fmt.Println(s.Contains(2))
	fmt.Println(s.Contains(3))

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output:
	// 2
	// true
	// false
	// [1 2]
}

// NewWithCap preallocates room for a known number of values. Use it when the
// final size is known up front to avoid reallocations during Add.
func ExampleNewWithCap() {
	s := set.NewWithCap[string](3)
	s.Add("a")
	s.Add("b")
	s.Add("c")

	fmt.Println(s.Len())
	// Output: 3
}

// Of builds a set from explicit values. A duplicate is stored once.
func ExampleOf() {
	s := set.Of("b", "a", "b", "c", "a")

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [a b c]
}

// From builds a set from a slice. A duplicate is stored once, and the set
// never aliases the slice.
func ExampleFrom() {
	vs := []int{3, 1, 3, 2, 1}
	s := set.From(vs)
	vs[0] = 99 // does not affect s

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [1 2 3]
}

// ContainsAll reports whether every given value is in the set. With no values
// it reports true.
func ExampleSet_ContainsAll() {
	s := set.Of("read", "write")

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
	s := set.Of("read", "write")

	fmt.Println(s.ContainsAny("exec", "write"))
	fmt.Println(s.ContainsAny("exec"))
	fmt.Println(s.ContainsAny())
	// Output:
	// true
	// false
	// false
}

// All returns a range-over-func iterator over the values in unspecified order
// without consuming them. The set is left intact after iteration. Sort the
// collected values when stable output matters.
func ExampleSet_All() {
	s := set.Of(3, 1, 2)

	var out []int
	for v := range s.All() {
		out = append(out, v)
	}

	slices.Sort(out)
	fmt.Println(out)
	fmt.Println("len unchanged:", s.Len())
	// Output:
	// [1 2 3]
	// len unchanged: 3
}

// Clone returns an independent shallow copy. Mutating one set does not affect
// the other, since they share no backing map.
func ExampleSet_Clone() {
	a := set.Of("a", "b")

	b := a.Clone()
	b.Remove("a") // removes "a" from b only

	av, bv := a.Slice(), b.Slice()
	slices.Sort(av)
	slices.Sort(bv)
	fmt.Println(av)
	fmt.Println(bv)
	// Output:
	// [a b]
	// [b]
}

// Slice returns a copy of the values in unspecified order. The returned slice
// is independent of the set.
func ExampleSet_Slice() {
	s := set.Of(3, 1, 2)

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)

	// The slice is detached: modifying it leaves the set untouched.
	out[0] = 99
	fmt.Println("set contains 1:", s.Contains(1))

	// An empty set returns nil.
	empty := set.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// [1 2 3]
	// set contains 1: true
	// true
}

// Add reports whether the value was new. Adding an existing value is a no-op.
func ExampleSet_Add() {
	s := set.New[string]()

	fmt.Println(s.Add("a"))
	fmt.Println(s.Add("b"))
	fmt.Println(s.Add("a")) // existing: no-op

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output:
	// true
	// true
	// false
	// [a b]
}

// AddN adds several values, skipping the ones already present.
func ExampleSet_AddN() {
	s := set.Of("a")
	s.AddN("b", "a", "c", "b")

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [a b c]
}

// AddSet adds the other set's values to the receiver in place, skipping the
// ones already present.
func ExampleSet_AddSet() {
	s := set.Of(1, 2)
	s.AddSet(set.Of(2, 3, 1, 4))

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [1 2 3 4]
}

// Remove deletes a value and reports whether it was present.
func ExampleSet_Remove() {
	s := set.Of("a", "b", "c")

	fmt.Println(s.Remove("b"))
	fmt.Println(s.Remove("zzz")) // not present

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output:
	// true
	// false
	// [a c]
}

// RemoveSet deletes every value of the other set from the receiver in place.
func ExampleSet_RemoveSet() {
	s := set.Of(1, 2, 3, 4, 5)
	s.RemoveSet(set.Of(4, 2))

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [1 3 5]
}

// RetainSet keeps only the values also present in the other set.
func ExampleSet_RetainSet() {
	s := set.Of("x", "b", "y", "a")
	s.RetainSet(set.Of("a", "b"))

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [a b]
}

// RemoveFunc deletes every value the predicate selects in a single pass and
// reports how many were removed. Deleting inside a range loop over the set is
// not supported; this is the way to do it.
func ExampleSet_RemoveFunc() {
	s := set.Of(1, 2, 3, 4, 5, 6, 7, 8)

	n := s.RemoveFunc(func(v int) bool {
		return v%2 == 0
	})

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(n, out)
	// Output: 4 [1 3 5 7]
}

// Union returns a new set with the values of every input set.
func ExampleSet_Union() {
	a := set.Of("b", "a")
	b := set.Of("c", "a", "d")

	out := a.Union(b).Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [a b c d]
}

// Intersection returns a new set with the values present in every input set.
func ExampleSet_Intersection() {
	a := set.Of("d", "c", "b", "a")
	b := set.Of("a", "b", "x")

	out := a.Intersection(b).Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [a b]
}

// Difference returns a new set with the receiver's values that are in none of
// the other sets.
func ExampleSet_Difference() {
	a := set.Of(1, 2, 3, 4, 5)

	out := a.Difference(set.Of(4, 1), set.Of(5)).Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [2 3]
}

// SymmetricDifference returns a new set with the values in exactly one of the
// two sets.
func ExampleSet_SymmetricDifference() {
	a := set.Of(1, 2, 3, 4)
	b := set.Of(5, 3, 6, 1)

	out := a.SymmetricDifference(b).Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output: [2 4 5 6]
}

// Equal reports whether two sets hold the same values.
func ExampleSet_Equal() {
	a := set.Of(1, 2, 3)
	b := set.Of(3, 2, 1)

	fmt.Println(a.Equal(b))
	fmt.Println(a.Equal(set.Of(1, 2)))
	// Output:
	// true
	// false
}

// IsSubset reports whether every value of the receiver is in the other set;
// IsSuperset is the reverse.
func ExampleSet_IsSubset() {
	a := set.Of(1, 2)
	b := set.Of(3, 2, 1)

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
	a := set.Of(1, 2)

	fmt.Println(a.IsDisjoint(set.Of(3, 4)))
	fmt.Println(a.IsDisjoint(set.Of(2, 3)))
	// Output:
	// true
	// false
}

// Clear empties the set and releases its backing map, and the set stays
// usable.
func ExampleSet_Clear() {
	s := set.NewWithCap[int](8)
	s.AddN(1, 2)

	s.Clear()
	fmt.Println(s.Len())

	s.Add(3)
	fmt.Println(s.Slice())
	// Output:
	// 0
	// [3]
}

// Reset, unlike Clear, empties the set but keeps the allocated backing map, so
// it can be reused without a new allocation.
func ExampleSet_Reset() {
	s := set.NewWithCap[int](8)
	s.AddN(1, 2)

	s.Reset()
	fmt.Println(s.Len())
	// Output: 0
}

// Shrink copies the values into a right-sized map and frees the old one
// immediately. Go maps never shrink in place, so a set that grew large and then
// lost values keeps its memory until Shrink or Clear.
func ExampleSet_Shrink() {
	s := set.NewWithCap[int](1024)
	for i := range 1000 {
		s.Add(i)
	}

	s.RemoveFunc(func(v int) bool {
		return v >= 3
	})
	s.Shrink() // copy into a right-sized map, free the old one now

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(s.Len(), out)
	// Output: 3 [0 1 2]
}

// A nil *Set[T] behaves as an empty set for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a set
// created with New (or a zero value).
func Example_nilReceiver() {
	var s *set.Set[int] // nil, never initialized

	fmt.Println(s.Len(), s.IsEmpty())

	ok := s.Remove(1)
	fmt.Println(ok)
	// Output:
	// 0 true
	// false
}

// The zero value of Set[T] is usable as an empty set, so New is optional.
func Example_zeroValue() {
	var s set.Set[string]
	s.Add("b")
	s.Add("a")

	out := s.Slice()
	slices.Sort(out)
	fmt.Println(s.Len(), out)
	// Output: 2 [a b]
}

// A practical example: de-duplicate a slice while keeping membership checks
// explicit.
func Example_deduplicate() {
	seen := set.New[string]()
	for _, name := range []string{"alice", "bob", "alice", "cora"} {
		if !seen.Add(name) {
			fmt.Println("duplicate:", name)
		}
	}

	out := seen.Slice()
	slices.Sort(out)
	fmt.Println(out)
	// Output:
	// duplicate: alice
	// [alice bob cora]
}
