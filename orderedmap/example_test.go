package orderedmap_test

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Nergous/structures/orderedmap"
)

// An ordered map is a hash map that remembers insertion order: unlike a
// built-in map, it always yields its pairs from the oldest key to the newest.
func Example() {
	m := orderedmap.New[string, int]()
	m.Set("banana", 3)
	m.Set("apple", 5)
	m.Set("cherry", 7)

	for !m.IsEmpty() {
		k, v, _ := m.PopFirst()
		fmt.Println(k, v)
	}
	// Output:
	// banana 3
	// apple 5
	// cherry 7
}

// NewWithCap preallocates room for a known number of keys. Use it when the
// final size is known up front to avoid reallocations during Set.
func ExampleNewWithCap() {
	m := orderedmap.NewWithCap[string, int](3)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	fmt.Println(m.Len(), m.Cap())
	// Output: 3 3
}

// Get returns the value stored under a key. The boolean result is false when
// the key is absent.
func ExampleMap_Get() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)

	v, ok := m.Get("a")
	fmt.Println(v, ok)
	v, ok = m.Get("zzz") // not present
	fmt.Println(v, ok)
	// Output:
	// 1 true
	// 0 false
}

// First and Last return the oldest and the newest pair without removing them.
// The boolean result is false for an empty map.
func ExampleMap_First() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	k, v, _ := m.First()
	fmt.Println("oldest:", k, v)
	k, v, _ = m.Last()
	fmt.Println("newest:", k, v)
	fmt.Println("len unchanged:", m.Len())

	empty := orderedmap.New[string, int]()
	_, _, ok := empty.First()
	fmt.Println(ok)
	// Output:
	// oldest: a 1
	// newest: c 3
	// len unchanged: 3
	// false
}

// All returns a range-over-func iterator over the pairs from oldest to newest
// without consuming them. The map is left intact after iteration. Unlike a
// built-in map, the order is insertion order, every time.
func ExampleMap_All() {
	m := orderedmap.New[string, int]()
	m.Set("banana", 3)
	m.Set("apple", 5)
	m.Set("cherry", 7)

	for k, v := range m.All() {
		fmt.Println(k, v)
	}

	fmt.Println("len unchanged:", m.Len())
	// Output:
	// banana 3
	// apple 5
	// cherry 7
	// len unchanged: 3
}

// Backward returns a range-over-func iterator over the pairs from newest to
// oldest without consuming them. The map is left intact after iteration.
func ExampleMap_Backward() {
	m := orderedmap.New[string, int]()
	m.Set("banana", 3)
	m.Set("apple", 5)
	m.Set("cherry", 7)

	for k, v := range m.Backward() {
		fmt.Println(k, v)
	}

	fmt.Println("len unchanged:", m.Len())
	// Output:
	// cherry 7
	// apple 5
	// banana 3
	// len unchanged: 3
}

// Keys returns a range-over-func iterator over the keys from oldest to newest
// without consuming them. The map is left intact after iteration. Use
// slices.Collect to get the keys as a slice.
func ExampleMap_Keys() {
	m := orderedmap.New[string, int]()
	m.Set("b", 2)
	m.Set("a", 1)
	m.Set("c", 3)

	fmt.Println(slices.Collect(m.Keys()))
	fmt.Println("len unchanged:", m.Len())
	// Output:
	// [b a c]
	// len unchanged: 3
}

// Clone returns an independent shallow copy with the same order. Mutating one
// map does not affect the other, since they share no backing storage.
func ExampleMap_Clone() {
	a := orderedmap.New[string, int]()
	a.Set("x", 1)
	a.Set("y", 2)

	b := a.Clone()
	b.Remove("x") // removes "x" from b only

	fmt.Println(slices.Collect(a.Keys()))
	fmt.Println(slices.Collect(b.Keys()))
	// Output:
	// [x y]
	// [y]
}

// Set reports whether the key was new. Setting an existing key replaces its
// value and keeps its original place.
func ExampleMap_Set() {
	m := orderedmap.New[string, int]()

	fmt.Println(m.Set("a", 1))
	fmt.Println(m.Set("b", 2))
	fmt.Println(m.Set("a", 10)) // existing: value changes, place stays
	fmt.Println(slices.Collect(m.Keys()))
	v, _ := m.Get("a")
	fmt.Println(v)
	// Output:
	// true
	// true
	// false
	// [a b]
	// 10
}

// Remove deletes a key and returns its value. The boolean result reports
// whether the key was present; the remaining keys keep their order.
func ExampleMap_Remove() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	fmt.Println(m.Remove("b"))
	fmt.Println(m.Remove("zzz")) // not present
	fmt.Println(slices.Collect(m.Keys()))
	// Output:
	// 2 true
	// 0 false
	// [a c]
}

// RemoveFunc deletes every pair the predicate selects in a single pass and
// reports how many were removed. Deleting inside a range loop over the map is
// not supported; this is the way to do it.
func ExampleMap_RemoveFunc() {
	m := orderedmap.New[string, int]()
	for i, name := range strings.Fields("tmp:a keep:b tmp:c keep:d") {
		m.Set(name, i)
	}

	n := m.RemoveFunc(func(k string, _ int) bool {
		return strings.HasPrefix(k, "tmp:")
	})

	fmt.Println(n, slices.Collect(m.Keys()))
	// Output: 2 [keep:b keep:d]
}

// PopFirst removes and returns the oldest pair. The boolean result reports
// whether the map was non-empty; on an empty map it is false and the returned
// key and value are zero values.
func ExampleMap_PopFirst() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)

	k, v, ok := m.PopFirst()
	fmt.Println(k, v, ok)
	m.PopFirst()            // removes "b"
	_, _, ok = m.PopFirst() // map is now empty
	fmt.Println(ok)
	// Output:
	// a 1 true
	// false
}

// PopLast removes and returns the newest pair. The boolean result reports
// whether the map was non-empty; on an empty map it is false and the returned
// key and value are zero values.
func ExampleMap_PopLast() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)

	k, v, ok := m.PopLast()
	fmt.Println(k, v, ok)
	m.PopLast()            // removes "a"
	_, _, ok = m.PopLast() // map is now empty
	fmt.Println(ok)
	// Output:
	// b 2 true
	// false
}

// MoveToBack makes a key the newest one, without changing its value, and
// reports whether it was present. An absent key is not added.
func ExampleMap_MoveToBack() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	fmt.Println(m.MoveToBack("a"))
	fmt.Println(m.MoveToBack("zzz")) // not present
	fmt.Println(slices.Collect(m.Keys()))
	// Output:
	// true
	// false
	// [b c a]
}

// Clear empties the map and releases its backing storage: capacity drops back
// to zero, and the map stays usable.
func ExampleMap_Clear() {
	m := orderedmap.NewWithCap[int, int](8)
	m.Set(1, 10)
	m.Set(2, 20)

	m.Clear()
	fmt.Println(m.Len(), m.Cap())

	m.Set(3, 30)
	fmt.Println(slices.Collect(m.Keys()))
	// Output:
	// 0 0
	// [3]
}

// Reset, unlike Clear, empties the map but keeps the allocated backing
// storage, so it can be reused without a new allocation.
func ExampleMap_Reset() {
	m := orderedmap.NewWithCap[int, int](8)
	m.Set(1, 10)
	m.Set(2, 20)

	m.Reset()
	fmt.Println(m.Len(), m.Cap())
	// Output: 0 8
}

// Shrink copies the pairs into a right-sized array, rebuilds the index, and
// frees the old storage immediately. Grow reserves capacity up front, and Clip
// trims spare capacity without copying.
func ExampleMap_Shrink() {
	m := orderedmap.NewWithCap[int, int](1024)
	m.Set(1, 10)
	m.Set(2, 20)
	m.Set(3, 30)

	m.Grow(2000) // reserve room for 2000 more elements
	fmt.Println(m.Cap() >= 2003)

	m.Shrink() // copy into a right-sized array, free the old one now
	fmt.Println(m.Cap())

	m.Clip() // already tight: no-op
	fmt.Println(m.Cap())
	// Output:
	// true
	// 3
	// 3
}

// A nil *Map[K, V] behaves as an empty map for every read-only method and for
// removals, so it is safe to query. Mutating methods still require a map
// created with New (or a zero value).
func Example_nilReceiver() {
	var m *orderedmap.Map[int, int] // nil, never initialized

	fmt.Println(m.Len(), m.IsEmpty(), m.Cap())

	k, v, ok := m.PopFirst()
	fmt.Println(k, v, ok)
	// Output:
	// 0 true 0
	// 0 0 false
}

// The zero value of Map[K, V] is usable as an empty map, so New is optional.
func Example_zeroValue() {
	var m orderedmap.Map[string, int]
	m.Set("b", 2)
	m.Set("a", 1)

	fmt.Println(m.Len(), slices.Collect(m.Keys()))
	// Output: 2 [b a]
}

// A practical example: a least-recently-used cache. MoveToBack plus PopFirst
// is all it needs: a hit moves the key to the back, and when the cache is full
// the front key is the least recently used one.
func Example_lruCache() {
	const capacity = 2
	cache := orderedmap.New[string, int]()

	get := func(k string) {
		if v, ok := cache.Get(k); ok {
			cache.MoveToBack(k)
			fmt.Println("hit", k, v)
			return
		}

		fmt.Println("miss", k)
	}

	put := func(k string, v int) {
		cache.Set(k, v)
		cache.MoveToBack(k)
		if cache.Len() > capacity {
			evicted, _, _ := cache.PopFirst()
			fmt.Println("evict", evicted)
		}
	}

	put("a", 1)
	put("b", 2)
	get("a") // "a" is now the most recently used
	put("c", 3)
	get("b") // "b" was evicted
	get("a")
	// Output:
	// hit a 1
	// evict b
	// miss b
	// hit a 1
}
