# orderedset

A generic **set that remembers insertion order** for Go. Membership checks,
adds, and removes are O(1) like a map-backed set, but iteration always runs from
the oldest value to the newest, and both ends are available in O(1).

It is part of the `structures` data-structures library. The API mirrors the
unordered [`set`](../set) package, adds operations that rely on order, and uses
Go 1.23 range-over-func iterators ([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/orderedset"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/orderedset"
)

func main() {
	tags := orderedset.From([]string{"go", "db", "go", "api", "db"})

	fmt.Println(tags.Slice()) // [go db api] — duplicates dropped, first order kept

	for t := range tags.All() {
		fmt.Println(t) // go, db, api — every time
	}
}
```

Adding a value that is already present returns `false` and **keeps its place**.
Use `MoveToBack` to make a value the newest:

```go
s := orderedset.Of("a", "b", "c")
s.Add("a")        // false, order is still [a b c]
s.MoveToBack("a") // order is now [b c a]
```

`Add` and `PopFirst` make a work queue that never holds the same item twice:

```go
pending := orderedset.New[string]()
pending.Add("job-1")
pending.Add("job-2")
pending.Add("job-1") // already queued, ignored

for !pending.IsEmpty() {
	job, _ := pending.PopFirst()
	fmt.Println(job) // job-1, job-2
}
```

Remove values selected by a predicate in one pass:

```go
removed := s.RemoveFunc(func(v string) bool {
	return strings.HasPrefix(v, "tmp:")
})
```

## When to use `orderedset` vs `set`

| Need                                                    | Use          |
| ------------------------------------------------------- | ------------ |
| Fastest membership checks, order does not matter        | `set`        |
| Deterministic output, stable tests, deduplicated lists  | `orderedset` |
| Oldest or newest value, FIFO of unique items, recency   | `orderedset` |
| Fast full traversal                                     | `orderedset` |

`orderedset` adds and removes values somewhat slower than `set`, but traverses
several times faster, because values are contiguous in a slice. See
[Benchmarks](#benchmarks).

## Ordering

The order is the order of **first insertion**:

| Operation                                          | Effect on order                           |
| -------------------------------------------------- | ----------------------------------------- |
| `Add` of a new value                               | value becomes the newest                  |
| `Add` of a present value                           | nothing changes, `Add` returns `false`    |
| `MoveToBack`                                       | value becomes the newest                  |
| `Remove`, `PopFirst`, `PopLast`, `RemoveFunc`      | remaining values keep their relative order |
| a value removed and added again                    | counts as new: it becomes the newest      |

Set algebra returns results in a defined order:

| Operation             | Order of the result                                                       |
| --------------------- | ------------------------------------------------------------------------- |
| `Union`               | values of `s`, then new values from each argument in argument order       |
| `Intersection`        | order of `s`                                                              |
| `Difference`          | order of `s`                                                              |
| `SymmetricDifference` | values only in `s` in the order of `s`, then values only in `other` in its order |
| `AddSet`              | new values appended in the order of `other`                               |
| `RemoveSet`, `RetainAll` | remaining values keep their order                                      |

`Equal`, `IsSubset`, `IsSuperset`, and `IsDisjoint` compare **contents only** and
ignore order: `{a, b}` equals `{b, a}`. To also compare order, use
`slices.Equal(a.Slice(), b.Slice())`.

There is deliberately **no `MoveToFront`**: it would cost O(n) in a slice-backed
set.

## How it works

`Set[T]` is a thin wrapper over
[`orderedmap.Map[T, struct{}]`](../orderedmap). The empty `struct{}` value takes
no memory, so each entry holds only the value and a liveness flag. Values live
in a slice in insertion order, next to a map from each value to its position.
Removing a value leaves a **hole** instead of shifting its neighbors; once holes
make up more than half of a slice of 16 or more entries, the slice is compacted
in place. Compaction is O(n) but runs only after O(n) removals, so removes are
O(1) amortized. See the [`orderedmap` README](../orderedmap/README.md#how-it-works)
for details.

## API reference

| Method / Function     | Signature                                                     | Description                                                                                  | Complexity     |
| --------------------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------- |
| `New`                 | `func New[T comparable]() *Set[T]`                            | Creates an empty set.                                                                        | O(1)           |
| `NewWithCap`          | `func NewWithCap[T comparable](n int) *Set[T]`                | Same, with room for at least `n` values in the slice and the index.                          | O(n)           |
| `Of`                  | `func Of[T comparable](vs ...T) *Set[T]`                      | Set of `vs` in order of first occurrence.                                                    | O(n)           |
| `From`                | `func From[T comparable](s []T) *Set[T]`                      | Set of the values of `s` in order of first occurrence; does not retain `s`.                  | O(n)           |
| `Len`                 | `func (s *Set[T]) Len() int`                                  | Number of values.                                                                            | O(1)           |
| `Cap`                 | `func (s *Set[T]) Cap() int`                                  | Capacity of the entry slice, holes included.                                                 | O(1)           |
| `IsEmpty`             | `func (s *Set[T]) IsEmpty() bool`                             | Reports whether the set has no values.                                                       | O(1)           |
| `Contains`            | `func (s *Set[T]) Contains(v T) bool`                         | Reports whether `v` is present.                                                              | O(1) avg       |
| `ContainsAll`         | `func (s *Set[T]) ContainsAll(vs ...T) bool`                  | Reports whether every value is present; `true` with no arguments.                            | O(k) avg       |
| `ContainsAny`         | `func (s *Set[T]) ContainsAny(vs ...T) bool`                  | Reports whether any value is present; `false` with no arguments.                             | O(k) avg       |
| `First`               | `func (s *Set[T]) First() (T, bool)`                          | Oldest value, without removing it.                                                           | O(1)           |
| `Last`                | `func (s *Set[T]) Last() (T, bool)`                           | Newest value, without removing it.                                                           | O(1)           |
| `Add`                 | `func (s *Set[T]) Add(v T) bool`                              | Adds `v`; keeps the place of a present value; reports whether `v` was new.                   | O(1) amortized |
| `AddN`                | `func (s *Set[T]) AddN(vs ...T)`                              | Adds `vs` in argument order.                                                                 | O(k) amortized |
| `Remove`              | `func (s *Set[T]) Remove(v T) bool`                           | Deletes `v`; reports whether it was present.                                                 | O(1) amortized |
| `RemoveFunc`          | `func (s *Set[T]) RemoveFunc(del func(T) bool) int`           | Deletes every value `del` selects, oldest to newest; returns how many; leaves no holes.      | O(n)           |
| `PopFirst`            | `func (s *Set[T]) PopFirst() (T, bool)`                       | Removes and returns the oldest value.                                                        | O(1) amortized |
| `PopLast`             | `func (s *Set[T]) PopLast() (T, bool)`                        | Removes and returns the newest value.                                                        | O(1) amortized |
| `MoveToBack`          | `func (s *Set[T]) MoveToBack(v T) bool`                       | Makes `v` the newest value; reports whether it was present.                                  | O(1) amortized |
| `All`                 | `func (s *Set[T]) All() iter.Seq[T]`                          | Values from oldest to newest.                                                                | O(1) per step  |
| `Backward`            | `func (s *Set[T]) Backward() iter.Seq[T]`                     | Values from newest to oldest.                                                                | O(1) per step  |
| `Slice`               | `func (s *Set[T]) Slice() []T`                                | Independent copy of the values in `All` order; `nil` when empty.                             | O(n)           |
| `Clone`               | `func (s *Set[T]) Clone() *Set[T]`                            | Independent shallow copy with the same order, no holes.                                      | O(n)           |
| `AddSet`              | `func (s *Set[T]) AddSet(other *Set[T])`                      | In-place union.                                                                              | O(m)           |
| `RemoveSet`           | `func (s *Set[T]) RemoveSet(other *Set[T])`                   | In-place difference.                                                                         | O(m)           |
| `RetainAll`           | `func (s *Set[T]) RetainAll(other *Set[T])`                   | In-place intersection.                                                                       | O(n)           |
| `Union`               | `func (s *Set[T]) Union(others ...*Set[T]) *Set[T]`           | New set with the values of all operands.                                                     | O(n+m)         |
| `Intersection`        | `func (s *Set[T]) Intersection(others ...*Set[T]) *Set[T]`    | New set with the values of `s` present in every operand.                                     | O(n·k)         |
| `Difference`          | `func (s *Set[T]) Difference(others ...*Set[T]) *Set[T]`      | New set with the values of `s` absent from every operand.                                    | O(n·k)         |
| `SymmetricDifference` | `func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T]` | New set with the values in exactly one of the two sets.                                      | O(n+m)         |
| `Equal`               | `func (s *Set[T]) Equal(other *Set[T]) bool`                  | Same contents, order ignored.                                                                | O(n)           |
| `IsSubset`            | `func (s *Set[T]) IsSubset(other *Set[T]) bool`               | Every value of `s` is in `other`.                                                            | O(n)           |
| `IsSuperset`          | `func (s *Set[T]) IsSuperset(other *Set[T]) bool`             | Every value of `other` is in `s`.                                                            | O(m)           |
| `IsDisjoint`          | `func (s *Set[T]) IsDisjoint(other *Set[T]) bool`             | No common values; iterates the smaller set.                                                  | O(min(n, m))   |
| `Clear`               | `func (s *Set[T]) Clear()`                                    | Removes all values **and releases** storage (`Cap` drops to 0).                              | O(1)           |
| `Reset`               | `func (s *Set[T]) Reset()`                                    | Removes all values but **keeps** storage for reuse.                                          | O(n)           |
| `Grow`                | `func (s *Set[T]) Grow(n int)`                                | Reserves entry-slice capacity for at least `n` more values.                                  | O(n)           |
| `Shrink`              | `func (s *Set[T]) Shrink()`                                   | Removes holes, copies into right-sized storage, and rebuilds the index, freeing memory now.  | O(n)           |
| `Clip`                | `func (s *Set[T]) Clip()`                                     | Reslices so `Cap == Len` of the slice **without copying**; holes and index are untouched.    | O(1)           |

`n` is the size of the receiver, `m` the size of the argument, and `k` the number
of arguments.

### The comma-ok contract

`First`, `Last`, `PopFirst`, and `PopLast` report whether a value was found. When
`ok` is `false` the value is the zero value of `T`. `Add` returns `true` when the
value was new; `Remove` and `MoveToBack` return whether the value was present;
`RemoveFunc` returns the number of removed values.

### Mutating while iterating

Mutating the set while ranging over it — including `Add` of a new value,
`Remove`, and `MoveToBack` from the loop body — is **undefined behavior**:
compaction can move entries under the iterator. Use `RemoveFunc` or `RetainAll`
to delete the values you select, or collect them first and mutate afterwards.

## Capacity management

- **Reserve ahead** — `NewWithCap` sizes both the entry slice and the index;
  `Grow(n)` reserves entry-slice room for `n` more values, and the index grows
  on demand.
- **Reclaim memory** — `Shrink` removes holes, copies into a right-sized slice,
  and rebuilds the index. Go maps never shrink in place, so `Shrink` (or
  `Clear`) is how to give a large index's memory back. `Clip` only reslices the
  entry slice in O(1) and leaves holes and the index as they are.
- **Reuse memory** — `Reset` empties the set but keeps the slice and the index
  buckets for refilling. `Clear` releases both.

`Cap` is the capacity of the entry slice, which counts live values and holes
alike. It does not include the index, whose size Go does not expose.

## `Shrink` vs `Clip`

Both release unused capacity, but they trade work against reclaimed memory
differently:

| Operation | Copies?                 | Cost | Holes and unused tail     | Index map           | Use when                                                          |
| --------- | ----------------------- | ---- | ------------------------- | ------------------- | ----------------------------------------------------------------- |
| `Shrink`  | yes (packed, new slice) | O(n) | removed, memory freed now | rebuilt (reclaimed) | The set grew large, has shrunk, and you want the memory back now. |
| `Clip`    | no (reslice)            | O(1) | left as they are          | left as is          | You want the cheap version and can wait for lazy reclaim.         |

## `Clear` vs `Reset`

Both empty the set and drop all references to the removed values, and the set
stays usable. They differ in what happens to the storage:

| Operation | Empties set | Entry slice and index      | `Cap` afterwards | Use when                                                |
| --------- | ----------- | -------------------------- | ---------------- | ------------------------------------------------------- |
| `Clear`   | yes         | released (eligible for GC) | `0`              | You are done with the set, or want to free its memory.  |
| `Reset`   | yes         | retained for reuse         | unchanged        | You will refill the set and want to avoid reallocating. |

## Nil receiver and zero value

The nil value of `*Set[T]` behaves as a valid **empty** set for every read-only
method, for predicates, for iteration, and for removals:

```go
var s *orderedset.Set[string] // nil

s.Len()             // 0
s.IsEmpty()         // true
s.Contains("a")     // false
v, ok := s.First()  // "", false
v, ok = s.PopFirst() // "", false
s.Remove("a")       // false
s.MoveToBack("a")   // false
s.RemoveFunc(f)     // 0
s.Slice()           // nil
for range s.All() { /* never runs */ }
```

Set algebra treats nil operands as empty sets, and `Clone` on a nil receiver
returns a new, usable **empty** set. The **mutating** methods — `Add`, `AddN`,
`AddSet`, `Grow`, `Clear`, `Reset`, `Shrink`, and `Clip` — **panic** on a nil set
with a message such as `orderedset: Add on nil receiver`.

The **zero value** `orderedset.Set[T]{}` is ready to use:

```go
var s orderedset.Set[string]
s.Add("a")
fmt.Println(s.Len()) // 1
```

Copying a `Set` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Set` is **not safe for concurrent use**: it performs no internal locking. If a
set is shared across goroutines, the caller must provide its own synchronization
(for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 5 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 5`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --------- | ---------------- | ----: | ---: | --------: |
| `Add/orderedset` | `Add` of new values, up to 4096 live | 22.3 | 0 | 0 |
| `Add/set` | the same adds into `set` | 19.3 | 0 | 0 |
| `AddExisting` | `Add` of a present value, 4096 values | 9.6 | 0 | 0 |
| `Contains/orderedset` | `Contains` of a present value, 4096 values | 8.2 | 0 | 0 |
| `Contains/set` | the same lookup in `set` | 8.2 | 0 | 0 |
| `Remove` | `Remove` of the oldest value, refilled outside the timing | 61.0 | 0 | 0 |
| `AddRemoveChurn/orderedset` | `Remove` a value and `Add` it again, 4096 values | 78.4 | 0 | 0 |
| `AddRemoveChurn/set` | the same in `set` | 44.5 | 0 | 0 |
| `PopFirst` | `PopFirst`, refilled outside the timing | 44.2 | 0 | 0 |
| `PopLast` | `PopLast`, refilled outside the timing | 29.1 | 0 | 0 |
| `MoveToBack` | `MoveToBack`, 4096 values | 73.5 | 0 | 0 |
| `All/orderedset` | iterate 1024 values with `All` | 2 171 | 0 | 0 |
| `All/set` | iterate 1024 values of `set` | 8 621 | 0 | 0 |
| `Backward` | iterate 1024 values newest to oldest | 2 108 | 0 | 0 |
| `Slice` | copy 1024 values | 1 757 | 8 192 | 1 |
| `Clone` | `Clone` of 4096 values | 97 051 | 213 456 | 21 |
| `RemoveFunc` | `RemoveFunc` removing every other value of 4096 (set rebuilt outside the timing) | 84 738 | 0 | 0 |
| `Union/orderedset` | union of two 1024-value sets with 50% overlap | 44 233 | 78 048 | 10 |
| `Union/set` | the same in `set` | 53 386 | 37 000 | 7 |
| `Intersection/orderedset` | intersection of the same sets | 23 712 | 53 424 | 8 |
| `Intersection/set` | the same in `set` | 64 082 | 37 000 | 7 |
| `Difference/orderedset` | difference of the same sets | 25 447 | 53 424 | 8 |
| `Difference/set` | the same in `set` | 62 075 | 37 000 | 7 |
| `From/orderedset` | build from 1024 values, each appearing twice | 18 078 | 53 424 | 8 |
| `From/set` | the same in `set` | 14 600 | 37 000 | 7 |

- Lookups cost the same as `set`; adds and removes cost about 1.1–1.8x more, because the entry slice is kept in order.
- Traversal is about 4x faster than ranging over `set`, because values are contiguous in a slice. That also makes `Intersection` and `Difference` about 2.5x faster than in `set`, even though they allocate more for the ordered result.
- Only `Clone`, `Slice`, set algebra, constructors, and slice growth allocate.

Run them with:

```sh
go test -bench=. -benchmem ./orderedset/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including deduplication, a unique work queue, and recency tracking — and are
rendered alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, with the
unordered `set` as the baseline; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/orderedset       # package overview
go doc github.com/Nergous/structures/orderedset Set   # the Set type and its methods
```
