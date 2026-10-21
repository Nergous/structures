package bitset_test

import (
	"fmt"

	"github.com/Nergous/structures/bitset"
)

// A bitset stores each non-negative int at most once and iterates in
// ascending order, so no sorting is needed before printing.
func Example() {
	s := bitset.New()
	s.Add(5)
	s.Add(1)
	s.Add(5)

	fmt.Println(s.Len())
	fmt.Println(s.Contains(1))
	fmt.Println(s.Contains(3))
	fmt.Println(s.Slice())
	// Output:
	// 2
	// true
	// false
	// [1 5]
}

// NewWithCap preallocates room for the values 0 through n-1. Capacity bounds
// the largest value, not the number of values, and is rounded up to a
// multiple of 64.
func ExampleNewWithCap() {
	s := bitset.NewWithCap(100)
	fmt.Println(s.Cap())

	s.Add(99) // covered: no reallocation
	fmt.Println(s.Cap())
	// Output:
	// 128
	// 128
}

// Of builds a set from explicit values. A duplicate is stored once.
func ExampleOf() {
	s := bitset.Of(64, 3, 64, 0, 3)

	fmt.Println(s.Slice())
	// Output: [0 3 64]
}

// From builds a set from a slice. A duplicate is stored once, and the set
// never aliases the slice.
func ExampleFrom() {
	vs := []int{3, 1, 3, 2, 1}
	s := bitset.From(vs)
	vs[0] = 99 // does not affect s

	fmt.Println(s.Slice())
	// Output: [1 2 3]
}

// ContainsAll reports whether every given value is in the set. With no values
// it reports true.
func ExampleSet_ContainsAll() {
	s := bitset.Of(1, 2, 64)

	fmt.Println(s.ContainsAll(1, 64))
	fmt.Println(s.ContainsAll(1, 3))
	fmt.Println(s.ContainsAll())
	// Output:
	// true
	// false
	// true
}

// ContainsAny reports whether at least one given value is in the set. With no
// values it reports false. Negative values are never present.
func ExampleSet_ContainsAny() {
	s := bitset.Of(1, 2, 64)

	fmt.Println(s.ContainsAny(-1, 64))
	fmt.Println(s.ContainsAny(-1, 3))
	fmt.Println(s.ContainsAny())
	// Output:
	// true
	// false
	// false
}

// Min returns the smallest value. The boolean is false for an empty set.
func ExampleSet_Min() {
	s := bitset.Of(200, 7, 64)

	fmt.Println(s.Min())
	fmt.Println(bitset.New().Min())
	// Output:
	// 7 true
	// 0 false
}

// Max returns the largest value in O(1). The boolean is false for an empty
// set.
func ExampleSet_Max() {
	s := bitset.Of(200, 7, 64)

	fmt.Println(s.Max())
	fmt.Println(bitset.New().Max())
	// Output:
	// 200 true
	// 0 false
}

// All returns a range-over-func iterator over the values in ascending order
// without consuming them. The set is left intact after iteration.
func ExampleSet_All() {
	s := bitset.Of(130, 3, 64)

	for v := range s.All() {
		fmt.Println(v)
	}

	fmt.Println("len unchanged:", s.Len())
	// Output:
	// 3
	// 64
	// 130
	// len unchanged: 3
}

// Backward iterates in descending order. Breaking out of the loop stops the
// iteration early.
func ExampleSet_Backward() {
	s := bitset.Of(130, 3, 64, 1)

	for v := range s.Backward() {
		if v < 10 {
			break
		}

		fmt.Println(v)
	}
	// Output:
	// 130
	// 64
}

// Clone returns an independent copy. Mutating one set does not affect the
// other, since they share no backing array.
func ExampleSet_Clone() {
	a := bitset.Of(1, 2)

	b := a.Clone()
	b.Remove(1) // removes 1 from b only

	fmt.Println(a.Slice())
	fmt.Println(b.Slice())
	// Output:
	// [1 2]
	// [2]
}

// Slice returns a copy of the values in ascending order. The returned slice
// is independent of the set.
func ExampleSet_Slice() {
	s := bitset.Of(3, 1, 2)

	out := s.Slice()
	fmt.Println(out)

	// The slice is detached: modifying it leaves the set untouched.
	out[0] = 99
	fmt.Println("set contains 1:", s.Contains(1))

	// An empty set returns nil.
	fmt.Println(bitset.New().Slice() == nil)
	// Output:
	// [1 2 3]
	// set contains 1: true
	// true
}

// Add reports whether the value was new. Adding an existing value is a no-op.
// Add panics on a negative value.
func ExampleSet_Add() {
	s := bitset.New()

	fmt.Println(s.Add(10))
	fmt.Println(s.Add(20))
	fmt.Println(s.Add(10)) // existing: no-op
	fmt.Println(s.Slice())
	// Output:
	// true
	// true
	// false
	// [10 20]
}

// AddN adds several values, extending the backing array at most once.
func ExampleSet_AddN() {
	s := bitset.Of(1)
	s.AddN(500, 1, 3, 500)

	fmt.Println(s.Slice())
	// Output: [1 3 500]
}

// AddSet adds the other set's values to the receiver in place, merging whole
// words at a time.
func ExampleSet_AddSet() {
	s := bitset.Of(1, 2)
	s.AddSet(bitset.Of(2, 3, 100))

	fmt.Println(s.Slice())
	// Output: [1 2 3 100]
}

// Remove deletes a value and reports whether it was present.
func ExampleSet_Remove() {
	s := bitset.Of(1, 2, 3)

	fmt.Println(s.Remove(2))
	fmt.Println(s.Remove(-1)) // never present
	fmt.Println(s.Slice())
	// Output:
	// true
	// false
	// [1 3]
}

// RemoveSet deletes every value of the other set from the receiver in place.
func ExampleSet_RemoveSet() {
	s := bitset.Of(1, 2, 3, 4, 5)
	s.RemoveSet(bitset.Of(4, 2))

	fmt.Println(s.Slice())
	// Output: [1 3 5]
}

// RetainSet keeps only the values also present in the other set.
func ExampleSet_RetainSet() {
	s := bitset.Of(1, 2, 3, 64)
	s.RetainSet(bitset.Of(2, 64, 99))

	fmt.Println(s.Slice())
	// Output: [2 64]
}

// RemoveFunc deletes every value the predicate selects in a single ascending
// pass and reports how many were removed. Deleting inside a range loop over
// the set is not supported; this is the way to do it.
func ExampleSet_RemoveFunc() {
	s := bitset.Of(1, 2, 3, 4, 5, 6, 7, 8)

	n := s.RemoveFunc(func(v int) bool {
		return v%2 == 0
	})

	fmt.Println(n, s.Slice())
	// Output: 4 [1 3 5 7]
}

// Union returns a new set with the values of every input set.
func ExampleSet_Union() {
	a := bitset.Of(2, 1)
	b := bitset.Of(3, 1, 100)

	fmt.Println(a.Union(b).Slice())
	// Output: [1 2 3 100]
}

// Intersection returns a new set with the values present in every input set.
func ExampleSet_Intersection() {
	a := bitset.Of(4, 3, 2, 1)
	b := bitset.Of(1, 2, 9)

	fmt.Println(a.Intersection(b).Slice())
	// Output: [1 2]
}

// Difference returns a new set with the receiver's values that are in none of
// the other sets.
func ExampleSet_Difference() {
	a := bitset.Of(1, 2, 3, 4, 5)

	fmt.Println(a.Difference(bitset.Of(4, 1), bitset.Of(5)).Slice())
	// Output: [2 3]
}

// SymmetricDifference returns a new set with the values in exactly one of the
// two sets.
func ExampleSet_SymmetricDifference() {
	a := bitset.Of(1, 2, 3, 4)
	b := bitset.Of(5, 3, 6, 1)

	fmt.Println(a.SymmetricDifference(b).Slice())
	// Output: [2 4 5 6]
}

// Equal reports whether two sets hold the same values, regardless of their
// capacity.
func ExampleSet_Equal() {
	a := bitset.Of(1, 2, 3)
	b := bitset.NewWithCap(4096)
	b.AddN(3, 2, 1)

	fmt.Println(a.Equal(b))
	fmt.Println(a.Equal(bitset.Of(1, 2)))
	// Output:
	// true
	// false
}

// IsSubset reports whether every value of the receiver is in the other set;
// IsSuperset is the reverse.
func ExampleSet_IsSubset() {
	a := bitset.Of(1, 2)
	b := bitset.Of(3, 2, 1)

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
	a := bitset.Of(1, 2)

	fmt.Println(a.IsDisjoint(bitset.Of(3, 64)))
	fmt.Println(a.IsDisjoint(bitset.Of(2, 3)))
	// Output:
	// true
	// false
}

// Clear empties the set and releases its backing array, and the set stays
// usable.
func ExampleSet_Clear() {
	s := bitset.NewWithCap(1024)
	s.AddN(1, 2)

	s.Clear()
	fmt.Println(s.Len(), s.Cap())

	s.Add(3)
	fmt.Println(s.Slice())
	// Output:
	// 0 0
	// [3]
}

// Reset, unlike Clear, empties the set but keeps the allocated backing array,
// so it can be refilled without a new allocation.
func ExampleSet_Reset() {
	s := bitset.NewWithCap(1024)
	s.AddN(1, 2)

	s.Reset()
	fmt.Println(s.Len(), s.Cap())
	// Output: 0 1024
}

// Shrink copies the words in use into a right-sized array and frees the old
// one immediately. Removing the largest values keeps the capacity, so a set
// that once held a large value keeps its memory until Shrink or Clear.
func ExampleSet_Shrink() {
	s := bitset.Of(1, 2, 100_000)
	fmt.Println(s.Cap() >= 100_000)

	s.Remove(100_000)
	s.Shrink() // copy into a right-sized array, free the old one now

	fmt.Println(s.Cap(), s.Slice())
	// Output:
	// true
	// 64 [1 2]
}

// Clip trims the spare capacity without copying. The tail memory is reclaimed
// lazily, when a later growth reallocates.
func ExampleSet_Clip() {
	s := bitset.NewWithCap(1024)
	s.AddN(1, 70)

	s.Clip()
	fmt.Println(s.Cap())
	// Output: 128
}

// Grow reserves room for the values 0 through n-1, the same bound NewWithCap
// takes, so later Adds below n do not reallocate.
func ExampleSet_Grow() {
	s := bitset.Of(1)

	s.Grow(1000)
	fmt.Println(s.Cap() >= 1000, s.Len())
	// Output: true 1
}

// A nil *Set behaves as an empty set for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a set
// created with New (or a zero value).
func Example_nilReceiver() {
	var s *bitset.Set // nil, never initialized

	fmt.Println(s.Len(), s.IsEmpty())

	ok := s.Remove(1)
	fmt.Println(ok)
	// Output:
	// 0 true
	// false
}

// The zero value of Set is usable as an empty set, so New is optional.
func Example_zeroValue() {
	var s bitset.Set
	s.Add(64)
	s.Add(1)

	fmt.Println(s.Len(), s.Slice())
	// Output: 2 [1 64]
}

// A practical example: the sieve of Eratosthenes. A bitset is a natural fit
// for dense small ints, and RemoveFunc deletes the composites of each prime
// in one pass.
func Example_sieve() {
	const limit = 50

	candidates := bitset.NewWithCap(limit)
	for v := 2; v < limit; v++ {
		candidates.Add(v)
	}

	for p := 2; p*p < limit; p++ {
		if !candidates.Contains(p) {
			continue
		}

		candidates.RemoveFunc(func(v int) bool {
			return v > p && v%p == 0
		})
	}

	fmt.Println(candidates.Slice())
	// Output: [2 3 5 7 11 13 17 19 23 29 31 37 41 43 47]
}
