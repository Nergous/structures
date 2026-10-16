# set

A generic, map-backed **set** of unique comparable values for Go.

It is part of the `structures` data-structures library. Values are stored as
keys of a Go map, so adds, removes, and membership checks run in O(1) on
average, and it uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/set"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"
	"slices"

	"github.com/Nergous/structures/set"
)

func main() {
	s := set.Of("go", "structures", "go")

	fmt.Println(s.Contains("structures")) // true

	out := s.Slice()
	slices.Sort(out) // set iteration order is unspecified
	fmt.Println(out) // [go structures]
}
```

Preallocate when the maximum size is known up front to avoid reallocations
while adding:

```go
s := set.NewWithCap[int](1000)
for i := range 1000 {
	s.Add(i)
}
fmt.Println(s.Len()) // 1000
```

Remove values selected by a predicate in one pass:

```go
removed := s.RemoveFunc(func(v int) bool {
	return v%2 == 0
})
fmt.Println(removed, s.Len()) // 500 500
```

## When to use `set` vs `orderedset`

| Need | Use |
| --- | --- |
| Fastest adds, removes, and membership checks, order does not matter | `set` |
| Deterministic output, stable tests, deduplicated lists | `orderedset` |
| Oldest or newest value, FIFO of unique items, recency | `orderedset` |
| Fast full traversal | `orderedset` |

[`orderedset`](../orderedset) has the same set API plus the methods that rely
on insertion order. It keeps values contiguous in a slice next to its index,
so adds and removes cost somewhat more than in `set`, but traversal is faster
(see [Benchmarks](#benchmarks)).

## Ordering

Iteration order is **unspecified**. `All` and `Slice` follow Go map iteration
order, which may differ between calls, even on an unchanged set. Sort the
result when stable output matters:

```go
out := s.Slice()
slices.Sort(out) // caller chooses ordering when needed
fmt.Println(out)
```

Use [`orderedset`](../orderedset) when values must come back in insertion
order.

## Comparable values

`Set[T]` requires `T comparable` because values are stored as Go map keys. When
`T` is an interface type, inserting a dynamic value that is not comparable (for
example, a slice, map, or function) panics just like assigning that value as a
map key.

Float NaN values behave like Go map keys: because `NaN != NaN`, a NaN value may
not be found or removed by a later lookup.

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[T comparable]() *Set[T]` | Creates an empty set with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[T comparable](n int) *Set[T]` | Creates an empty set with capacity for at least `n` values; a negative `n` is treated as 0. | O(n) |
| `Of` | `func Of[T comparable](vs ...T) *Set[T]` | Creates a set of the values of `vs`, each stored once. | O(n) |
| `From` | `func From[T comparable](s []T) *Set[T]` | Creates a set of the values of `s`, each stored once; never aliases `s`. | O(n) |
| `Len` | `func (s *Set[T]) Len() int` | Number of values currently in the set. | O(1) |
| `IsEmpty` | `func (s *Set[T]) IsEmpty() bool` | Reports whether the set has no values. | O(1) |
| `Contains` | `func (s *Set[T]) Contains(v T) bool` | Reports whether `v` is present. | O(1) avg |
| `ContainsAll` | `func (s *Set[T]) ContainsAll(vs ...T) bool` | Reports whether every value in `vs` is present; `true` with no arguments. | O(k) avg |
| `ContainsAny` | `func (s *Set[T]) ContainsAny(vs ...T) bool` | Reports whether any value in `vs` is present; `false` with no arguments. | O(k) avg |
| `All` | `func (s *Set[T]) All() iter.Seq[T]` | Iterator over values in unspecified order, without consuming them. | O(1) per step |
| `Clone` | `func (s *Set[T]) Clone() *Set[T]` | Returns an independent **shallow** copy; the two sets share no backing map. | O(n) |
| `Slice` | `func (s *Set[T]) Slice() []T` | Returns an independent copy of the values in unspecified order; `nil` if empty. | O(n) |
| `Add` | `func (s *Set[T]) Add(v T) bool` | Adds `v`; reports whether `v` was new. | O(1) avg |
| `AddN` | `func (s *Set[T]) AddN(vs ...T)` | Adds every value in `vs`, each stored once. | O(k) avg |
| `AddSet` | `func (s *Set[T]) AddSet(other *Set[T])` | In-place union with `other`. | O(m) avg |
| `Remove` | `func (s *Set[T]) Remove(v T) bool` | Deletes `v`; reports whether `v` was present. | O(1) avg |
| `RemoveSet` | `func (s *Set[T]) RemoveSet(other *Set[T])` | In-place difference with `other`. | O(m) avg |
| `RetainSet` | `func (s *Set[T]) RetainSet(other *Set[T])` | In-place intersection with `other`. | O(n) avg |
| `RemoveFunc` | `func (s *Set[T]) RemoveFunc(del func(T) bool) int` | Deletes every value `del` selects; returns how many. | O(n) |
| `Union` | `func (s *Set[T]) Union(others ...*Set[T]) *Set[T]` | New set with the values of all operands. | O(n+m) avg |
| `Intersection` | `func (s *Set[T]) Intersection(others ...*Set[T]) *Set[T]` | New set with the values of `s` present in every operand. | O(n·k) avg |
| `Difference` | `func (s *Set[T]) Difference(others ...*Set[T]) *Set[T]` | New set with the values of `s` absent from every operand. | O(n·k) avg |
| `SymmetricDifference` | `func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T]` | New set with the values in exactly one of the two sets. | O(n+m) avg |
| `Equal` | `func (s *Set[T]) Equal(other *Set[T]) bool` | Reports whether both sets hold the same values. | O(n) avg |
| `IsSubset` | `func (s *Set[T]) IsSubset(other *Set[T]) bool` | Reports whether every value of `s` is in `other`. | O(n) avg |
| `IsSuperset` | `func (s *Set[T]) IsSuperset(other *Set[T]) bool` | Reports whether every value of `other` is in `s`. | O(m) avg |
| `IsDisjoint` | `func (s *Set[T]) IsDisjoint(other *Set[T]) bool` | Reports whether the sets share no values; iterates the smaller set. | O(min(n, m)) avg |
| `Clear` | `func (s *Set[T]) Clear()` | Removes all values **and releases** the backing map. | O(1) |
| `Reset` | `func (s *Set[T]) Reset()` | Removes all values but **keeps** the backing map for reuse. | O(n) |
| `Shrink` | `func (s *Set[T]) Shrink()` | Shrink-to-fit: **copies** into a right-sized map and frees the old one now; an empty set releases its map. | O(n) |

`n` is the number of values in the set (or in the input for `Of` and `From`);
`m` is the total number of values in the set arguments; `k` is the number of
variadic arguments.

### The comma-ok contract

`Set` has no `(T, bool)` lookups, but its mutators follow the same idea: the
boolean reports whether the operation found what it looked for. `Add` returns
`true` when the value was new and `false` when it was already present;
`Remove` returns `true` when the value was present and `false` when it was
already absent. In both `false` cases the set is unchanged. `RemoveFunc`
returns the number of removed values.

```go
s := set.New[int]()
fmt.Println(s.Add(1))    // true
fmt.Println(s.Add(1))    // false, already present
fmt.Println(s.Remove(1)) // true
fmt.Println(s.Remove(1)) // false, already absent
```

`ContainsAll()` with no arguments returns `true`; `ContainsAny()` with no
arguments returns `false`.

### Iteration with `All`

`All` returns an [`iter.Seq[T]`](https://pkg.go.dev/iter#Seq) — a Go 1.23
range-over-func iterator — that yields every value once, in **unspecified**
order. It does not consume the set, and breaking out of the loop early stops
iteration cleanly:

```go
for v := range s.All() {
	fmt.Println(v)
	if v == target {
		break // safe; the set is untouched
	}
}
fmt.Println(s.Len()) // unchanged
```

Mutating the set while iterating over `All` is undefined behavior. Use
`RemoveFunc` or `RetainSet` to delete the values you select, or collect them
first and mutate afterwards.

### Bulk add with `AddN`

`AddN` adds several values at once; duplicates, both among the arguments and
against the set, are stored once. On a zero-value set it sizes the new backing
map for the arguments:

```go
var s set.Set[string]
s.AddN("a", "b", "a")
fmt.Println(s.Len()) // 2
```

With no arguments it is a no-op.

### Copying out with `Clone` and `Slice`

`Clone` returns an independent set; `Slice` returns a plain slice. Both are
detached from the original — they share no backing map, so mutating one does
not affect the other:

```go
a := set.Of(1, 2, 3)

b := a.Clone() // independent *Set[int]
b.Remove(1)    // removes 1 from b only

fmt.Println(slices.Sorted(a.All())) // [1 2 3]
fmt.Println(slices.Sorted(b.All())) // [2 3]
```

Both copies are **shallow**: when `T` is a pointer or contains references, the
pointed-to data is shared between the copies. `Slice` returns `nil` for an
empty set, and `Clone` of a nil set returns a new, usable empty set.

### Set algebra

`Union`, `Intersection`, `Difference`, and `SymmetricDifference` return fresh
sets and leave their operands unchanged; nil operands are treated as empty
sets:

```go
a := set.Of(1, 2, 3)
b := set.Of(3, 4)

fmt.Println(slices.Sorted(a.Union(b).All()))               // [1 2 3 4]
fmt.Println(slices.Sorted(a.Intersection(b).All()))        // [3]
fmt.Println(slices.Sorted(a.Difference(b).All()))          // [1 2]
fmt.Println(slices.Sorted(a.SymmetricDifference(b).All())) // [1 2 4]
```

For hot paths, `AddSet`, `RemoveSet`, and `RetainSet` are the in-place forms of
`Union`, `Difference`, and `Intersection`: they mutate the receiver without
allocating a result set.

## Capacity management

The set grows automatically, but several methods let you control the backing
map explicitly:

- **Reserve ahead** — `NewWithCap(n)` sizes the backing map for `n` values at
  construction, so adding up to `n` values does not grow it.
- **Reclaim memory** — `Shrink` copies the values into a right-sized map and
  frees the old one now; `Clear` releases the backing map entirely.
- **Reuse memory** — `Reset` empties the set but keeps the backing map for
  refilling.

Go maps do not expose their capacity, and they never shrink in place: a map
that once held a million values keeps that memory after the values are
removed. So `set` intentionally has no `Cap`, `Grow`, or `Clip`, and there is
no `Shrink` vs `Clip` choice: `Shrink` is the only way to give that memory
back short of `Clear`. It always rebuilds the map, because whether the map is
already right-sized cannot be observed.

```go
s := set.NewWithCap[int](1 << 20)
// ... fill, then remove most values ...
s.Shrink() // copy the survivors into a right-sized map, free the old one now
```

## `Clear` vs `Reset`

Both empty the set and drop all references to the removed values (so they
become eligible for garbage collection), but they differ in what happens to the
allocated backing map:

| Operation | Empties set | Backing map | Use when |
| --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | You are done with the set, or want to free its memory. |
| `Reset` | yes | retained for reuse | You will refill the set and want to avoid reallocating. |

```go
s := set.NewWithCap[int](8)
s.AddN(1, 2)

s.Reset()
fmt.Println(s.Len()) // 0  — capacity kept for reuse

s.Clear()
fmt.Println(s.Len()) // 0  — backing map released
```

## Nil receiver and zero value

A nil `*Set[T]` behaves as a valid **empty** set for every read-only method and
for removal methods, so you can safely query a `nil` set:

```go
var s *set.Set[int] // nil

s.Len()         // 0
s.IsEmpty()     // true
s.Contains(1)   // false
s.Remove(1)     // false (nothing to remove)
s.RemoveFunc(f) // 0
s.Slice()       // nil
for range s.All() { /* never runs */ }
s.Clone() // new, usable empty set (never nil)
```

Set algebra treats nil operands as empty sets.

The **mutating** methods — `Add`, `AddN`, `AddSet`, `Clear`, `Reset`, and
`Shrink` — need a non-nil receiver, so they **panic** on a `nil` set with a
message such as `set: Add on nil receiver`. `RemoveFunc` panics with
`set: nil predicate` when `del` is nil, even on a nil set.

The zero value of `Set[T]` is usable, so `New` is optional; `Add` and `AddN`
allocate the backing map on first insertion:

```go
var s set.Set[int]
s.Add(1)
fmt.Println(s.Contains(1)) // true
```

Copying a `Set` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Set` is **not safe for concurrent use**: it performs no internal locking.
If a set is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `From` | build from 4096 values, each appearing twice | 61 044 | 147 832 | 19 |
| `Contains/set/hit` | `Contains` of a present value, 4096 values | 8.5 | 0 | 0 |
| `Contains/set/miss` | `Contains` of an absent value, 4096 values | 7.8 | 0 | 0 |
| `Contains/builtin-map/hit` | the same present-value lookup in a `map[int]struct{}` | 8.4 | 0 | 0 |
| `Contains/builtin-map/miss` | the same absent-value lookup in a `map[int]struct{}` | 7.4 | 0 | 0 |
| `All/set` | iterate 4096 values with `All` | 36 180 | 0 | 0 |
| `All/builtin-map` | the same iteration over a `map[int]struct{}` | 30 898 | 0 | 0 |
| `Clone/set` | clone 4096 values | 125 710 | 147 833 | 19 |
| `Clone/builtin-map` | the same with `maps.Clone` on a `map[int]struct{}` | 26 032 | 147 824 | 18 |
| `Slice` | copy 4096 values | 39 168 | 32 768 | 1 |
| `Add/set` | `Add` into a set growing from empty to 4096 values | 68.7 | 72 | 0 |
| `Add/builtin-map` | the same inserts into a `map[int]struct{}` | 60.7 | 72 | 0 |
| `AddExisting` | `Add` of a present value, 4096 values | 8.7 | 0 | 0 |
| `AddN` | one `AddN` of 256 new values into preallocated room | 4 120 | 0 | 0 |
| `Remove` | one `Remove` of a present value, refilled outside the timing | 29.0 | 0 | 0 |
| `RemoveFunc` | one `RemoveFunc` removing every other value of 4096, refilled outside the timing | 75 730 | 0 | 0 |
| `Union/n=16` | union of two 16-value sets, half overlapping | 986 | 672 | 5 |
| `Union/n=1024` | union of two 1024-value sets, half overlapping | 57 014 | 37 000 | 7 |
| `Union/n=65536` | union of two 65536-value sets, half overlapping | 5 972 745 | 2 364 601 | 259 |
| `Intersection/n=16` | intersection of two 16-value sets, half overlapping | 1 126 | 672 | 5 |
| `Intersection/n=1024` | intersection of two 1024-value sets, half overlapping | 66 046 | 37 000 | 7 |
| `Intersection/n=65536` | intersection of two 65536-value sets, half overlapping | 6 337 341 | 2 364 601 | 259 |
| `Difference/n=16` | difference of two 16-value sets, half overlapping | 1 176 | 672 | 5 |
| `Difference/n=1024` | difference of two 1024-value sets, half overlapping | 66 669 | 37 000 | 7 |
| `Difference/n=65536` | difference of two 65536-value sets, half overlapping | 6 435 450 | 2 364 601 | 259 |
| `AddRemoveChurn/set` | one `Remove` + one `Add` of the same value with 4096 live values | 45.0 | 0 | 0 |
| `AddRemoveChurn/builtin-map` | the same churn in a `map[int]struct{}` | 31.8 | 0 | 0 |

- `B/op` for `Add` is the amortized cost of growing the backing map. Pre-size with `NewWithCap` to avoid it.
- `Union`, `Intersection`, and `Difference` allocate a result set, so their allocation counts grow with the input size; `AddSet`, `RemoveSet`, and `RetainSet` avoid that allocation.

Run them with:

```sh
go test -bench=. -benchmem ./set/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including set algebra, `RemoveFunc`, cloning, shrinking, nil and zero-value
use, and deduplication — and are rendered alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, with a
built-in `map[int]struct{}` as the baseline; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/set        # package overview
go doc github.com/Nergous/structures/set Set    # the Set type and its methods
```

The same comments render on pkg.go.dev-style godoc.
