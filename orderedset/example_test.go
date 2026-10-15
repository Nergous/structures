package orderedset_test

import (
	"fmt"
	"slices"

	"github.com/Nergous/structures/orderedset"
)

// An ordered set de-duplicates a slice while keeping the first occurrence of
// every value in its original place, so the output is stable without sorting.
func Example() {
	tags := []string{"go", "db", "go", "api", "db", "cli"}

	fmt.Println(orderedset.From(tags).Slice())
	// Output: [go db api cli]
}

// The zero value of Set is an empty set ready to use; Add allocates on first
// insertion.
func Example_zeroValue() {
	var s orderedset.Set[string]
	fmt.Println(s.Len(), s.IsEmpty())

	s.Add("b")
	s.Add("a")
	fmt.Println(s.Len(), s.Slice())
	// Output:
	// 0 true
	// 2 [b a]
}

// New returns an empty set.
func ExampleNew() {
	s := orderedset.New[int]()
	fmt.Println(s.Len(), s.IsEmpty())

	s.Add(3)
	s.Add(1)
	fmt.Println(s.Len(), s.Slice())
	// Output:
	// 0 true
	// 2 [3 1]
}

// NewWithCap preallocates room for n values, so the first n insertions do not
// grow the backing storage.
func ExampleNewWithCap() {
	s := orderedset.NewWithCap[int](16)
	fmt.Println(s.Len(), s.Cap())

	s.AddN(1, 2, 3)
	fmt.Println(s.Len(), s.Cap())
	// Output:
	// 0 16
	// 3 16
}

// Of builds a set from explicit values. A duplicate keeps the position of its
// first occurrence.
func ExampleOf() {
	s := orderedset.Of("b", "a", "b", "c", "a")

	fmt.Println(s.Slice())
	// Output: [b a c]
}

// From builds a set from a slice. A duplicate keeps the position of its first
// occurrence.
func ExampleFrom() {
	s := orderedset.From([]int{3, 1, 3, 2, 1})

	fmt.Println(s.Slice())
	// Output: [3 1 2]
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

// All returns a range-over-func iterator from the oldest value to the newest.
// Unlike a plain set, the order is insertion order, every time.
func ExampleSet_All() {
	s := orderedset.Of("banana", "apple", "cherry")

	for v := range s.All() {
		fmt.Println(v)
	}
	// Output:
	// banana
	// apple
	// cherry
}

// Backward walks from the newest value to the oldest.
func ExampleSet_Backward() {
	s := orderedset.Of(1, 2, 3)

	for v := range s.Backward() {
		fmt.Println(v)
	}
	// Output:
	// 3
	// 2
	// 1
}

// Slice returns the values in insertion order. An empty set returns nil.
func ExampleSet_Slice() {
	s := orderedset.Of(3, 1, 2)
	fmt.Println(s.Slice())

	empty := orderedset.New[int]()
	fmt.Println(empty.Slice() == nil)
	// Output:
	// [3 1 2]
	// true
}

// Clone returns an independent copy with the same order; changing one set does
// not affect the other.
func ExampleSet_Clone() {
	s := orderedset.Of("a", "b")
	c := s.Clone()
	c.Add("c")
	c.Remove("a")

	fmt.Println(s.Slice())
	fmt.Println(c.Slice())
	// Output:
	// [a b]
	// [b c]
}

// First and Last peek at the oldest and the newest value without removing
// them. The boolean result is false for an empty set.
func ExampleSet_First() {
	s := orderedset.Of("a", "b", "c")

	v, _ := s.First()
	fmt.Println("oldest:", v)
	v, _ = s.Last()
	fmt.Println("newest:", v)

	_, ok := orderedset.New[string]().First()
	fmt.Println(ok)
	// Output:
	// oldest: a
	// newest: c
	// false
}

// Add plus PopFirst make a FIFO work queue that never holds the same job
// twice: enqueuing a job that is already waiting is a no-op.
func ExampleSet_PopFirst() {
	queue := orderedset.Of("fetch", "parse")

	for job, ok := queue.PopFirst(); ok; job, ok = queue.PopFirst() {
		fmt.Println("run", job)

		if job == "fetch" {
			queue.Add("parse") // already queued: ignored
			queue.Add("index")
		}
	}
	// Output:
	// run fetch
	// run parse
	// run index
}

// PopLast removes and returns the newest value, so the set can also be drained
// newest first.
func ExampleSet_PopLast() {
	s := orderedset.Of("a", "b", "c")

	for v, ok := s.PopLast(); ok; v, ok = s.PopLast() {
		fmt.Println(v)
	}
	fmt.Println(s.Len())
	// Output:
	// c
	// b
	// a
	// 0
}

// MoveToBack makes a value the newest one, which tracks recency: after each
// use the value moves to the back, so First is the least recently used value.
func ExampleSet_MoveToBack() {
	recent := orderedset.Of("a", "b", "c")

	fmt.Println(recent.MoveToBack("a"))   // "a" was just used
	fmt.Println(recent.MoveToBack("zzz")) // not present
	fmt.Println(recent.Slice())

	lru, _ := recent.First()
	fmt.Println("least recently used:", lru)
	// Output:
	// true
	// false
	// [b c a]
	// least recently used: b
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

// AddSet adds the other set's values in its order, skipping the ones already
// present.
func ExampleSet_AddSet() {
	a := orderedset.Of(1, 2)
	a.AddSet(orderedset.Of(2, 3, 1, 4))

	fmt.Println(a.Slice())
	// Output: [1 2 3 4]
}

// RemoveSet deletes every value of the other set; the rest keep their order.
func ExampleSet_RemoveSet() {
	a := orderedset.Of(1, 2, 3, 4, 5)
	a.RemoveSet(orderedset.Of(4, 2))

	fmt.Println(a.Slice())
	// Output: [1 3 5]
}

// RetainAll keeps only the values also present in the other set, in the
// receiver's order.
func ExampleSet_RetainAll() {
	a := orderedset.Of("x", "b", "y", "a")
	a.RetainAll(orderedset.Of("a", "b"))

	fmt.Println(a.Slice())
	// Output: [b a]
}

// Equal compares membership only and ignores order. To compare order as well,
// compare the slices.
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

// ContainsAll reports whether every given value is in the set. With no values
// it reports true.
func ExampleSet_ContainsAll() {
	perms := orderedset.Of("read", "write")

	fmt.Println(perms.ContainsAll("read", "write"))
	fmt.Println(perms.ContainsAll("read", "exec"))
	fmt.Println(perms.ContainsAll())
	// Output:
	// true
	// false
	// true
}

// ContainsAny reports whether at least one given value is in the set. With no
// values it reports false.
func ExampleSet_ContainsAny() {
	perms := orderedset.Of("read", "write")

	fmt.Println(perms.ContainsAny("exec", "write"))
	fmt.Println(perms.ContainsAny("exec"))
	fmt.Println(perms.ContainsAny())
	// Output:
	// true
	// false
	// false
}

// Clear removes all values and releases the backing storage, so Cap drops to
// 0. The set stays usable.
func ExampleSet_Clear() {
	s := orderedset.NewWithCap[int](8)
	s.AddN(1, 2, 3)

	s.Clear()
	fmt.Println(s.Len(), s.Cap())

	s.Add(4)
	fmt.Println(s.Slice())
	// Output:
	// 0 0
	// [4]
}

// Reset removes all values but keeps the backing storage for reuse, so Cap is
// preserved.
func ExampleSet_Reset() {
	s := orderedset.NewWithCap[int](8)
	s.AddN(1, 2, 3)

	s.Reset()
	fmt.Println(s.Len(), s.Cap())
	// Output: 0 8
}

// Shrink drops removed slots and unused capacity, so a set that grew large and
// then lost values releases the extra memory. Order is preserved.
func ExampleSet_Shrink() {
	s := orderedset.NewWithCap[int](100)
	s.AddN(1, 2, 3, 4, 5)
	s.Remove(2)
	fmt.Println(s.Len(), s.Cap())

	s.Shrink()
	fmt.Println(s.Len(), s.Cap())
	fmt.Println(s.Slice())
	// Output:
	// 4 100
	// 4 4
	// [1 3 4 5]
}

// Clip trims unused capacity without copying.
func ExampleSet_Clip() {
	s := orderedset.NewWithCap[int](10)
	s.AddN(1, 2, 3)
	fmt.Println(s.Len(), s.Cap())

	s.Clip()
	fmt.Println(s.Len(), s.Cap())
	// Output:
	// 3 10
	// 3 3
}

// Grow reserves room for at least n more values, so the next n insertions do
// not reallocate the backing storage.
func ExampleSet_Grow() {
	s := orderedset.Of(1, 2, 3)
	s.Grow(10)

	fmt.Println(s.Len())
	fmt.Println(s.Cap() >= s.Len()+10)
	// Output:
	// 3
	// true
}
