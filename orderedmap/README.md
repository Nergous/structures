# orderedmap

A generic **hash map that remembers insertion order** for Go. Lookups, inserts,
and deletes are O(1) like a built-in map, but iteration always runs from the
oldest key to the newest, and both ends are available in O(1).

It is part of the `structures` data-structures library. Keys only need to be
`comparable`, values can be anything, and iteration uses Go 1.23
range-over-func iterators ([`iter.Seq2`](https://pkg.go.dev/iter#Seq2)).

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
m.Set("banana", 30)      // value changes, banana is still first
m.MoveToBack("banana")   // now banana is last
```

Remove entries selected by a predicate in one pass:

```go
removed := m.RemoveFunc(func(k string, v int) bool {
	return strings.HasPrefix(k, "tmp:")
})
```

`MoveToBack` and `PopFirst` are all a least-recently-used cache needs:

```go
if v, ok := cache.Get(key); ok {
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

| Operation                         | Effect on order                              |
| --------------------------------- | -------------------------------------------- |
| `Set` of a new key                | key becomes the newest                       |
| `Set` of an existing key          | value replaced, place kept                   |
| `MoveToBack`                      | key becomes the newest, value unchanged      |
| `Remove`, `PopFirst`, `PopLast`, `RemoveFunc` | remaining keys keep their relative order |
| a key removed and set again       | counts as new: it becomes the newest         |

There is deliberately **no `MoveToFront`**: it would cost O(n) in a slice-backed
map, and a cache does not need it.

## How it works

Entries live in a slice in insertion order, next to a map from each key to its
position in that slice. Deleting an entry leaves a **hole** in the slice instead
of shifting its neighbors, which keeps every delete O(1). Once holes make up
more than half of a slice of 16 or more entries, the slice is compacted in place
and positions are rewritten. Compaction is O(n) but runs only after O(n)
deletions, so deletes are O(1) amortized, and it never changes `Cap`. Holes are
zeroed immediately, so deleted keys and values are not retained.

The slice layout is why traversal is fast, and why the order is simple: no
pointers per key, no linked list.

## API reference

| Method / Function | Signature                                                      | Description                                                                                   | Complexity     |
| ----------------- | -------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | -------------- |
| `New`             | `func New[K comparable, V any]() *Map[K, V]`                   | Creates an empty map.                                                                         | O(1)           |
| `NewWithCap`      | `func NewWithCap[K comparable, V any](n int) *Map[K, V]`       | Same, with room for at least `n` keys in the slice and the index.                             | O(n)           |
| `Len`             | `func (m *Map[K, V]) Len() int`                                | Number of pairs.                                                                              | O(1)           |
| `Cap`             | `func (m *Map[K, V]) Cap() int`                                | Capacity of the entry slice, holes included.                                                  | O(1)           |
| `IsEmpty`         | `func (m *Map[K, V]) IsEmpty() bool`                           | Reports whether the map has no pairs.                                                         | O(1)           |
| `Get`             | `func (m *Map[K, V]) Get(k K) (V, bool)`                       | Returns the value for `k`.                                                                    | O(1) avg       |
| `Contains`        | `func (m *Map[K, V]) Contains(k K) bool`                       | Reports whether `k` is present.                                                               | O(1) avg       |
| `First`           | `func (m *Map[K, V]) First() (K, V, bool)`                     | Oldest pair, without removing it.                                                             | O(1)           |
| `Last`            | `func (m *Map[K, V]) Last() (K, V, bool)`                      | Newest pair, without removing it.                                                             | O(1)           |
| `Set`             | `func (m *Map[K, V]) Set(k K, v V) bool`                       | Stores `v`; keeps the place of an existing key; reports whether `k` was new.                  | O(1) amortized |
| `Remove`          | `func (m *Map[K, V]) Remove(k K) (V, bool)`                    | Deletes `k` and returns its value.                                                            | O(1) amortized |
| `RemoveFunc`      | `func (m *Map[K, V]) RemoveFunc(del func(K, V) bool) int`      | Deletes every pair `del` selects, oldest to newest; returns how many; leaves no holes.        | O(n)           |
| `PopFirst`        | `func (m *Map[K, V]) PopFirst() (K, V, bool)`                  | Removes and returns the oldest pair.                                                          | O(1) amortized |
| `PopLast`         | `func (m *Map[K, V]) PopLast() (K, V, bool)`                   | Removes and returns the newest pair.                                                          | O(1) amortized |
| `MoveToBack`      | `func (m *Map[K, V]) MoveToBack(k K) bool`                     | Makes `k` the newest key; reports whether it was present.                                     | O(1) amortized |
| `All`             | `func (m *Map[K, V]) All() iter.Seq2[K, V]`                    | Pairs from oldest to newest.                                                                  | O(1) per step  |
| `Backward`        | `func (m *Map[K, V]) Backward() iter.Seq2[K, V]`               | Pairs from newest to oldest.                                                                  | O(1) per step  |
| `Keys`            | `func (m *Map[K, V]) Keys() iter.Seq[K]`                       | Keys from oldest to newest.                                                                   | O(1) per step  |
| `Values`          | `func (m *Map[K, V]) Values() iter.Seq[V]`                     | Values from the oldest key to the newest.                                                     | O(1) per step  |
| `Clone`           | `func (m *Map[K, V]) Clone() *Map[K, V]`                       | Independent shallow copy with the same order, no holes.                                       | O(n)           |
| `Clear`           | `func (m *Map[K, V]) Clear()`                                  | Removes all pairs **and releases** storage (`Cap` drops to 0).                                | O(1)           |
| `Reset`           | `func (m *Map[K, V]) Reset()`                                  | Removes all pairs but **keeps** storage for reuse.                                            | O(n)           |
| `Grow`            | `func (m *Map[K, V]) Grow(n int)`                              | Reserves entry-slice capacity for at least `n` more keys.                                     | O(n)           |
| `Shrink`          | `func (m *Map[K, V]) Shrink()`                                 | Removes holes, copies into right-sized storage, and rebuilds the index, freeing memory now.   | O(n)           |
| `Clip`            | `func (m *Map[K, V]) Clip()`                                   | Reslices so `Cap == Len` of the slice **without copying**; holes and index are untouched.     | O(1)           |

### The comma-ok contract

`Get`, `First`, `Last`, `Remove`, `PopFirst`, and `PopLast` report whether a pair
was found. When `ok` is `false` the other results are zero values. `Set`
returns `true` when the key was new; `MoveToBack` returns whether the key was
present; `RemoveFunc` returns the number of removed pairs.

### Collecting keys and values

`Keys` and `Values` are iterators. To get slices, use the standard library:

```go
keys := slices.Collect(m.Keys())
values := slices.Collect(m.Values())
```

### Mutating while iterating

Mutating the map while ranging over it — including `Set` of a new key, `Remove`,
and `MoveToBack` from the loop body — is **undefined behavior**: compaction can
move entries under the iterator. A built-in map allows `delete` during `range`,
and this one does not. Use `RemoveFunc` to delete the pairs you select, or
collect the keys first and mutate afterwards.

## Capacity management

- **Reserve ahead** — `NewWithCap` sizes both the entry slice and the index;
  `Grow(n)` reserves entry-slice room for `n` more keys, and the index grows on
  demand.
- **Reclaim memory** — `Shrink` removes holes, copies into a right-sized slice,
  and rebuilds the index. Go maps never shrink in place, so `Shrink` (or `Clear`)
  is how to give a large index's memory back. `Clip` only reslices the entry
  slice in O(1) and leaves holes and the index as they are.
- **Reuse memory** — `Reset` empties the map but keeps the slice and the index
  buckets for refilling. `Clear` releases both.

`Cap` is the capacity of the entry slice, which counts live entries and holes
alike. It does not include the index, whose size Go does not expose.

## Nil receiver and zero value

The nil value of `*Map[K, V]` behaves as a valid **empty** map for every
read-only method, for iteration, and for removals:

```go
var m *orderedmap.Map[string, int] // nil

m.Len()                 // 0
m.IsEmpty()             // true
v, ok := m.Get("a")     // 0, false
k, v, ok := m.First()   // "", 0, false
k, v, ok = m.PopFirst() // "", 0, false
m.Remove("a")           // 0, false
m.MoveToBack("a")       // false
m.RemoveFunc(f)         // 0
for range m.All() { /* never runs */ }
```

`Clone` on a nil receiver returns a new, usable **empty** map. The **mutating**
methods — `Set`, `Grow`, `Clear`, `Reset`, `Shrink`, and `Clip` — **panic** on a nil
map with a message such as `orderedmap: Set on nil receiver`.

The **zero value** `orderedmap.Map[K, V]{}` is ready to use: `Set` allocates the
index on first insertion.

## Concurrency

A `Map` is **not safe for concurrent use**: it performs no internal locking. If a
map is shared across goroutines, the caller must provide its own synchronization
(for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-01. Each value is the median of 5 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 5`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --------- | ---------------- | ----: | ---: | --------: |
| `Set/orderedmap` | `Set` of new keys, up to 4096 live | 21.8 | 0 | 0 |
| `Set/builtin-map` | the same inserts into a built-in map | 16.7 | 0 | 0 |
| `SetExisting` | `Set` of an existing key (replace value), 4096 keys | 13.7 | 0 | 0 |
| `Get/orderedmap` | `Get` of a present key, 4096 keys | 9.0 | 0 | 0 |
| `Get/builtin-map` | the same lookup in a built-in map | 7.9 | 0 | 0 |
| `RemoveSet/orderedmap` | `Remove` a key and `Set` it again, 4096 keys | 77.5 | 0 | 0 |
| `RemoveSet/builtin-map` | the same in a built-in map | 39.0 | 0 | 0 |
| `MoveToBack` | `MoveToBack`, 4096 keys | 72.4 | 0 | 0 |
| `Queue` | `PopFirst` + `Set` at a steady 4096 keys | 87.6 | 0 | 0 |
| `All/orderedmap` | iterate 4096 pairs with `All` | 8 950 | 0 | 0 |
| `All/builtin-map` | range over a 4096-entry built-in map | 30 311 | 0 | 0 |
| `RemoveFunc` | `RemoveFunc` removing every other pair of 4096 (map rebuilt outside the timing) | 81 698 | 0 | 0 |
| `Clone` | `Clone` of 4096 pairs | 96 617 | 246 176 | 20 |

- Ordered operations cost roughly 1.1–2x a built-in map, and traversal is about 3x faster than ranging over a built-in map, because entries are contiguous in a slice.
- Only `Clone`, `Shrink`, and slice growth allocate.

Run them with:

```sh
go test -bench=. -benchmem ./orderedmap/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including an LRU cache and in-place deletion — and are rendered alongside the
API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, with a
built-in map as the baseline for the core operations; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/orderedmap       # package overview
go doc github.com/Nergous/structures/orderedmap Map   # the Map type and its methods
```
