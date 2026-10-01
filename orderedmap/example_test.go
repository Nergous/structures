package orderedmap_test

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Nergous/structures/orderedmap"
)

// Unlike a built-in map, an ordered map iterates in insertion order, every
// time.
func Example() {
	m := orderedmap.New[string, int]()
	m.Set("banana", 3)
	m.Set("apple", 5)
	m.Set("cherry", 7)

	for k, v := range m.All() {
		fmt.Println(k, v)
	}
	// Output:
	// banana 3
	// apple 5
	// cherry 7
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

// MoveToBack makes a key the newest one without changing its value.
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

// First and Last peek at the oldest and the newest pair; PopFirst and PopLast
// remove them.
func ExampleMap_PopFirst() {
	m := orderedmap.New[string, int]()
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)

	k, v, _ := m.First()
	fmt.Println("oldest:", k, v)
	k, v, _ = m.Last()
	fmt.Println("newest:", k, v)

	k, v, _ = m.PopFirst()
	fmt.Println("popped:", k, v)
	fmt.Println(m.Len())
	// Output:
	// oldest: a 1
	// newest: c 3
	// popped: a 1
	// 2
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

// Backward walks from the newest pair to the oldest.
func ExampleMap_Backward() {
	m := orderedmap.New[int, string]()
	m.Set(1, "one")
	m.Set(2, "two")
	m.Set(3, "three")

	for k, v := range m.Backward() {
		fmt.Println(k, v)
	}
	// Output:
	// 3 three
	// 2 two
	// 1 one
}

// MoveToBack plus PopFirst is all a least-recently-used cache needs: a hit
// moves the key to the back, and when the cache is full the front key is the
// least recently used one.
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
	get("a") // a is now the most recently used
	put("c", 3)
	get("b") // b was evicted
	get("a")
	// Output:
	// hit a 1
	// evict b
	// miss b
	// hit a 1
}
