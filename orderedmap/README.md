# orderedmap

A generic, insertion-ordered **hash map** for Go.

It is part of the `structures` data-structures library. Lookups, inserts, and
deletes cost O(1) on average like a built-in map, but iteration always runs
from the oldest key to the newest and both ends are available in O(1), and it
uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq) and
[`iter.Seq2`](https://pkg.go.dev/iter#Seq2)).

## Install

```go
import "github.com/Nergous/structures/orderedmap"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/orderedmap"
)

func main() {
	m := orderedmap.New[string, int]()
	m.Set("banana", 3)
	m.Set("apple", 5)
	m.Set("cherry", 7)

	for k, v := range m.All() {
		fmt.Println(k, v) // banana 3, apple 5, cherry 7 — every time
	}
}
```

Setting an existing key replaces its value and **keeps its place**. Use
`MoveToBack` to make a key the newest:

```go
m.Set("banana", 30)    // value changes, banana is still first
m.MoveToBack("banana") // now banana is last

fmt.Println(slices.Collect(m.Keys())) // [apple cherry banana]
```

Remove pairs selected by a predicate in one pass:

```go
removed := m.RemoveFunc(func(k string, v int) bool {
	return strings.HasPrefix(k, "tmp:")
})
fmt.Println(removed) // 0
```

`MoveToBack` and `PopFirst` are all a least-recently-used cache needs:

```go
if _, ok := cache.Get(key); ok {
	cache.MoveToBack(key) // hit: now the most recently used
}
cache.Set(key, value)
cache.MoveToBack(key)
if cache.Len() > capacity {
	cache.PopFirst() // evict the least recently used
}
```

## Ordering

The order is the order of **first insertion**:

| Operation | Effect on order |
| --- | --- |
| `Set` of a new key | key becomes the newest |
| `Set` of an existing key | value replaced, place kept |
| `MoveToBack` | key becomes the newest, value unchanged |
| `Remove`, `RemoveFunc`, `PopFirst`, `PopLast` | remaining keys keep their relative order |
| a key removed and set again | counts as new: it becomes the newest |

There is deliberately **no `MoveToFront`**: it would cost O(n) in a slice-backed
map, and a cache does not need it.

## How it works

Entries live in a slice in insertion order, next to a map from each key to its
position in that slice. Deleting an entry leaves a **hole** in the slice instead
of shifting its neighbors, which keeps every delete O(1). Once holes make up
more than half of a slice of 16 or more entries, the slice is compacted in place
and positions are rewritten. Compaction is O(n) but runs only after O(n)
deletions, so deletes are amortized O(1), and it never changes `Cap`. Holes are
zeroed immediately, so deleted keys and values are not retained.

The slice layout is why traversal is fast, and why the order is simple: no
pointers per key, no linked list.

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[K comparable, V any]() *Map[K, V]` | Creates an empty map with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[K comparable, V any](n int) *Map[K, V]` | Creates an empty map with capacity for at least `n` keys in the entry slice and the index; a negative `n` is treated as 0. | O(n) |
| `Len` | `func (m *Map[K, V]) Len() int` | Number of keys currently in the map. | O(1) |
| `Cap` | `func (m *Map[K, V]) Cap() int` | Capacity of the entry slice, holes included. | O(1) |
| `IsEmpty` | `func (m *Map[K, V]) IsEmpty() bool` | Reports whether the map has no keys. | O(1) |
| `Get` | `func (m *Map[K, V]) Get(key K) (V, bool)` | Returns the value stored under `key`; `ok` is `false` if absent. | O(1) avg |
| `Contains` | `func (m *Map[K, V]) Contains(key K) bool` | Reports whether `key` is present. | O(1) avg |
| `First` | `func (m *Map[K, V]) First() (K, V, bool)` | Returns the oldest pair without removing it; `ok` is `false` if empty. | O(1) |
| `Last` | `func (m *Map[K, V]) Last() (K, V, bool)` | Returns the newest pair without removing it; `ok` is `false` if empty. | O(1) |
| `All` | `func (m *Map[K, V]) All() iter.Seq2[K, V]` | Iterator over pairs from oldest to newest, without consuming them. | O(1) per step |
| `Backward` | `func (m *Map[K, V]) Backward() iter.Seq2[K, V]` | Iterator over pairs from newest to oldest, without consuming them. | O(1) per step |
| `Keys` | `func (m *Map[K, V]) Keys() iter.Seq[K]` | Iterator over keys from oldest to newest, without consuming them. | O(1) per step |
| `Values` | `func (m *Map[K, V]) Values() iter.Seq[V]` | Iterator over values from the oldest key to the newest, without consuming them. | O(1) per step |
| `Clone` | `func (m *Map[K, V]) Clone() *Map[K, V]` | Returns an independent **shallow** copy; same order, no holes. | O(n) |
| `Set` | `func (m *Map[K, V]) Set(key K, value V) bool` | Stores `value` under `key`; keeps the place of an existing key; reports whether `key` was new. | amortized O(1) |
| `Remove` | `func (m *Map[K, V]) Remove(key K) (V, bool)` | Deletes `key` and returns its value; `ok` is `false` if absent. | amortized O(1) |
| `RemoveFunc` | `func (m *Map[K, V]) RemoveFunc(del func(K, V) bool) int` | Deletes every pair `del` selects, oldest to newest; returns how many; leaves no holes. | O(n) |
| `PopFirst` | `func (m *Map[K, V]) PopFirst() (K, V, bool)` | Removes and returns the oldest pair; `ok` is `false` if empty. | amortized O(1) |
| `PopLast` | `func (m *Map[K, V]) PopLast() (K, V, bool)` | Removes and returns the newest pair; `ok` is `false` if empty. | amortized O(1) |
| `MoveToBack` | `func (m *Map[K, V]) MoveToBack(key K) bool` | Makes `key` the newest key; reports whether `key` was present. | amortized O(1) |
| `Clear` | `func (m *Map[K, V]) Clear()` | Removes all pairs **and releases** the entry slice and the index (`Cap` drops to 0). | O(1) |
| `Reset` | `func (m *Map[K, V]) Reset()` | Removes all pairs but **keeps** the entry slice and the index for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (m *Map[K, V]) Shrink()` | Shrink-to-fit: **copies** into a right-sized entry slice, frees the old one now, and rebuilds the index; no copy when `Cap == Len` and there are no holes. | O(n) |
| `Clip` | `func (m *Map[K, V]) Clip()` | Reslices the entry slice so `Cap == Len` **without copying**; holes and the index are untouched. | O(1) |
| `Grow` | `func (m *Map[K, V]) Grow(n int)` | Reserves entry-slice capacity for at least `n` more keys; no-op when `n <= 0` or capacity already suffices. | O(n) |

### The comma-ok contract

`Get` and `Remove` return a value and a boolean; `First`, `Last`, `PopFirst`,
and `PopLast` return a key, a value, and a boolean. The boolean is `true` when
a pair was found and `false` when the map is empty (or nil) or the key is
absent; in that case the other results are zero values. `Set` returns `true`
when the key was new; `MoveToBack` returns whether the key was present;
`RemoveFunc` returns the number of removed pairs.

```go
v, ok := m.Get("a")
if !ok {
	// "a" is not in the map; v is the zero value
}
```

### Iteration with `All`

`All` returns an [`iter.Seq2[K, V]`](https://pkg.go.dev/iter#Seq2) — a Go 1.23
range-over-func iterator — that yields pairs from the **oldest key to the
newest**; `Backward` yields them from **newest to oldest**, and `Keys` and
`Values` yield one side of each pair. None of them consume the map, and
breaking out of the loop early stops iteration cleanly. To get slices, use
[`slices.Collect`](https://pkg.go.dev/slices#Collect):

```go
for k, v := range m.All() {
	fmt.Println(k, v)
	if k == target {
		break // safe; the map is untouched
	}
}
fmt.Println(m.Len()) // unchanged

for k, v := range m.Backward() {
	fmt.Println(k, v) // newest to oldest
}

keys := slices.Collect(m.Keys())     // oldest to newest
values := slices.Collect(m.Values()) // in the same order
```

Mutating the map while iterating over `All`, `Backward`, `Keys`, or `Values` —
including `Set` of a new key, `Remove`, and `MoveToBack` from the loop body —
is **undefined behavior**: compaction can move entries under the iterator. A
built-in map allows `delete` during `range`, and this one does not. Use
`RemoveFunc` to delete the pairs you select, or collect the keys first and
mutate afterwards.

### Copying out with `Clone`

`Clone` returns an independent map with the same order and no holes. It is
detached from the original — it shares no entry slice or index, so mutating
one does not affect the other:

```go
a := orderedmap.New[string, int]()
a.Set("x", 1)
a.Set("y", 2)

b := a.Clone() // independent *Map[string, int]
b.Remove("x")  // removes x from b only

fmt.Println(slices.Collect(a.Keys())) // [x y]
fmt.Println(slices.Collect(b.Keys())) // [y]
```

The copy is **shallow**: when `K` or `V` is a pointer or contains references,
the pointed-to data is shared between the copies. The clone's capacity equals
its length. `Clone` of a nil map returns a new, usable empty map.

## Capacity management

The map grows automatically, but several methods let you control the backing
storage explicitly:

- **Reserve ahead** — `NewWithCap(n)` sizes both the entry slice and the index
  at construction; `Grow(n)` reserves entry-slice room for `n` more keys on an
  existing map.
- **Reclaim memory** — `Shrink` removes holes, copies into a right-sized entry
  slice, and rebuilds the index; `Clip` only reslices the entry slice (see
  below); `Clear` releases both entirely.
- **Reuse memory** — `Reset` empties the map but keeps the entry slice and the
  index buckets for refilling.

`Cap` is the capacity of the entry slice, which counts live entries and holes
alike. It does not include the index, whose size Go does not expose; the index
grows on demand and is not preallocated by `Grow`. Go maps never shrink in
place, so `Shrink` (or `Clear`) is the only way to give a large index's memory
back.

## `Shrink` vs `Clip`

Both reduce the entry-slice capacity to the current length, but they make
opposite trade-offs between work done now and memory reclaimed now:

| Operation | Copies? | Cost | Holes and unused tail | Index map | Use when |
| --- | --- | --- | --- | --- | --- |
| `Shrink` | yes (packed, new slice) | O(n) | removed, memory freed now | rebuilt (reclaimed) | The map grew large, has shrunk, and you want the memory back now. |
| `Clip` | no (reslice) | O(1) | left as they are | left as is | You want the cheap version and can wait for lazy reclaim. |

```go
m := orderedmap.NewWithCap[int, int](1024)
m.Set(1, 10)
m.Set(2, 20)
m.Set(3, 30)

m.Clip()
fmt.Println(m.Len(), m.Cap()) // 3 3  — O(1) reslice, old array still retained

m.Grow(2000)
m.Shrink()
fmt.Println(m.Len(), m.Cap()) // 3 3  — copied into a right-sized slice, old one freed
```

Both are no-ops on the entry slice when `Cap` already equals `Len` and there
are no holes; `Shrink` still rebuilds the index.

## `Clear` vs `Reset`

Both empty the map and drop all references to the removed keys and values (so
they become eligible for garbage collection), but they differ in what happens
to the entry slice and the index:

| Operation | Empties map | Entry slice and index | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the map, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the map and want to avoid reallocating. |

```go
m := orderedmap.NewWithCap[int, int](8)
m.Set(1, 10)
m.Set(2, 20)

m.Reset()
fmt.Println(m.Len(), m.Cap()) // 0 8  — capacity kept for reuse

m.Clear()
fmt.Println(m.Len(), m.Cap()) // 0 0  — entry slice and index released
```

## Nil receiver and zero value

A nil `*Map[K, V]` behaves as a valid **empty** map for every read-only method
and for removal methods, so you can safely query a `nil` map:

```go
var m *orderedmap.Map[string, int] // nil

m.Len()                 // 0
m.IsEmpty()             // true
m.Cap()                 // 0
v, ok := m.Get("a")     // 0, false
m.Contains("a")         // false
k, v, ok := m.First()   // "", 0, false
m.Remove("a")           // 0, false
m.RemoveFunc(f)         // 0
k, v, ok = m.PopFirst() // "", 0, false (nothing to remove)
m.MoveToBack("a")       // false
for range m.All() { /* never runs */ }
m.Clone() // new, usable empty map (never nil)
```

The **mutating** methods — `Set`, `Clear`, `Reset`, `Shrink`, `Clip`, and
`Grow` — need a non-nil receiver, so they **panic** on a `nil` map with a
message such as `orderedmap: Set on nil receiver`. `RemoveFunc` panics with
`orderedmap: nil predicate` when `del` is nil, even on a nil map.

The zero value of `Map[K, V]` is usable, so `New` is optional; `Set` allocates
the index on first insertion:

```go
var m orderedmap.Map[string, int]
m.Set("a", 1)
fmt.Println(m.Len()) // 1
```

Copying a `Map` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Map` is **not safe for concurrent use**: it performs no internal locking.
If a map is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `Get/orderedmap` | `Get` of a present key, 4096 keys | 9.5 | 0 | 0 |
| `Get/builtin-map` | the same lookup in a built-in map | 8.6 | 0 | 0 |
| `Contains/orderedmap/hit` | `Contains` of a present key, 4096 keys | 8.7 | 0 | 0 |
| `Contains/orderedmap/miss` | `Contains` of an absent key, 4096 keys | 7.9 | 0 | 0 |
| `Contains/builtin-map/hit` | the same present-key lookup in a built-in map | 8.3 | 0 | 0 |
| `Contains/builtin-map/miss` | the same absent-key lookup in a built-in map | 7.7 | 0 | 0 |
| `First` | one `First` on 4096 keys | 1.4 | 0 | 0 |
| `Last` | one `Last` on 4096 keys | 1.5 | 0 | 0 |
| `All/orderedmap` | iterate 4096 pairs with `All` | 9 316 | 0 | 0 |
| `All/builtin-map` | the same iteration over a built-in map | 30 939 | 0 | 0 |
| `Backward` | iterate 4096 pairs with `Backward` | 8 812 | 0 | 0 |
| `Clone/orderedmap` | clone 4096 pairs | 105 472 | 246 176 | 20 |
| `Clone/builtin-map` | the same with `maps.Clone` on a built-in map | 25 666 | 147 824 | 18 |
| `Set/orderedmap` | `Set` of new keys into a map growing from empty to 4096 keys | 81.8 | 145 | 0 |
| `Set/builtin-map` | the same inserts into a built-in map | 65.9 | 72 | 0 |
| `SetExisting` | `Set` of an existing key (value replaced), 4096 keys | 14.0 | 0 | 0 |
| `Remove` | one `Remove` of a present key, refilled outside the timing | 63.0 | 0 | 0 |
| `RemoveFunc` | one `RemoveFunc` removing every other pair of 4096, refilled outside the timing | 83 238 | 0 | 0 |
| `PopFirst` | one `PopFirst`, refilled outside the timing | 45.5 | 0 | 0 |
| `PopLast` | one `PopLast`, refilled outside the timing | 29.7 | 0 | 0 |
| `MoveToBack` | `MoveToBack` of a present key, 4096 keys | 76.9 | 0 | 0 |
| `SetRemoveChurn/orderedmap` | one `Remove` + one `Set` of the same key with 4096 live keys | 81.1 | 0 | 0 |
| `SetRemoveChurn/builtin-map` | the same churn in a built-in map | 40.1 | 0 | 0 |
| `PopFirstSetChurn` | one `PopFirst` + one `Set` of a new key with 4096 live keys | 88.8 | 0 | 0 |

- `B/op` for `Set` is the amortized cost of growing the entry slice and the index. Pre-size with `NewWithCap` (entry slice and index) or `Grow` (entry slice only) to avoid it.

Run them with:

```sh
go test -bench=. -benchmem ./orderedmap/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including `RemoveFunc`, `MoveToBack`, cloning, shrinking, nil and zero-value
use, and an LRU cache — and are rendered alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, with a
built-in map as the baseline for the core operations; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/orderedmap        # package overview
go doc github.com/Nergous/structures/orderedmap Map    # the Map type and its methods
```

The same comments render on pkg.go.dev-style godoc.
