# orderedset

A generic, insertion-ordered **set** of unique comparable values for Go.

It is part of the `structures` data-structures library. It mirrors the
unordered [`set`](../set) package, adds operations that rely on order, keeps
membership checks, adds, and removes at O(1) like a map-backed set, and uses
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
	s := orderedset.New[string]()
	s.Add("b")
	s.Add("a")
	s.Add("b") // already present: keeps its place

	for !s.IsEmpty() {
		v, _ := s.PopFirst()
		fmt.Println(v) // b, a
	}
}
```

Deduplicate a slice while keeping the first occurrence of every value in
place:

```go
tags := orderedset.From([]string{"go", "db", "go", "api", "db"})
fmt.Println(tags.Slice()) // [go db api]
```

Adding a value that is already present returns `false` and **keeps its place**.
Use `MoveToBack` to make a value the newest:

```go
s := orderedset.Of("a", "b", "c")
fmt.Println(s.Add("a")) // false
s.MoveToBack("a")
fmt.Println(s.Slice()) // [b c a]
```

`Add` and `PopFirst` make a work queue that never holds the same item twice:

```go
pending := orderedset.New[string]()
pending.AddN("job-1", "job-2", "job-1") // the second "job-1" is ignored

for !pending.IsEmpty() {
	job, _ := pending.PopFirst()
	fmt.Println(job) // job-1, job-2
}
```

Remove values selected by a predicate in one pass:

```go
s := orderedset.Of("tmp:a", "keep", "tmp:b")
removed := s.RemoveFunc(func(v string) bool {
	return strings.HasPrefix(v, "tmp:")
})
fmt.Println(removed, s.Slice()) // 2 [keep]
```

## When to use `orderedset` vs `set`

| Need | Use |
| --- | --- |
| Fastest adds, removes, and membership checks, order does not matter | `set` |
| Deterministic output, stable tests, deduplicated lists | `orderedset` |
| Oldest or newest value, FIFO of unique items, recency | `orderedset` |
| Fast full traversal | `orderedset` |

[`set`](../set) has the same set API without the methods that rely on
insertion order. `orderedset` keeps values contiguous in a slice next to its
index, so adds and removes cost somewhat more than in `set`, but traversal is
faster (see [Benchmarks](#benchmarks)).

## Ordering

The order is the order of **first insertion**:

| Operation | Effect on order |
| --- | --- |
| `Add` of a new value | value becomes the newest |
| `Add` of a present value | nothing changes, `Add` returns `false` |
| `MoveToBack` | value becomes the newest |
| `Remove`, `RemoveFunc`, `PopFirst`, `PopLast` | remaining values keep their relative order |
| a value removed and added again | counts as new: it becomes the newest |

Set algebra returns results in a defined order:

| Operation | Order of the result |
| --- | --- |
| `AddSet` | new values appended in the order of `other` |
| `RemoveSet`, `RetainSet` | remaining values keep their order |
| `Union` | values of `s`, then new values from each argument in argument order |
| `Intersection` | order of `s` |
| `Difference` | order of `s` |
| `SymmetricDifference` | values only in `s` in the order of `s`, then values only in `other` in its order |

`Equal`, `IsSubset`, `IsSuperset`, and `IsDisjoint` compare **contents only**
and ignore order: `{a, b}` equals `{b, a}`. To also compare order, use
`slices.Equal(a.Slice(), b.Slice())`.

There is deliberately **no `MoveToFront`**: it would cost O(n) in a
slice-backed set.

## How it works

`Set[T]` is a thin wrapper over
[`orderedmap.Map[T, struct{}]`](../orderedmap). The empty `struct{}` value
takes no memory, so each entry holds only the value and a liveness flag.
Values live in a slice in insertion order, next to a map from each value to
its position. Removing a value leaves a **hole** instead of shifting its
neighbors; once holes make up more than half of a slice of 16 or more entries,
the slice is compacted in place. Compaction is O(n) but runs only after O(n)
removals, so removes are amortized O(1). See the
[`orderedmap` README](../orderedmap/README.md#how-it-works) for details.

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[T comparable]() *Set[T]` | Creates an empty set with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[T comparable](n int) *Set[T]` | Creates an empty set with capacity for at least `n` values in the entry slice and the index; a negative `n` is treated as 0. | O(n) |
| `Of` | `func Of[T comparable](vs ...T) *Set[T]` | Creates a set of the values of `vs` in order of first occurrence. | O(n) |
| `From` | `func From[T comparable](s []T) *Set[T]` | Creates a set of the values of `s` in order of first occurrence; never aliases `s`. | O(n) |
| `Len` | `func (s *Set[T]) Len() int` | Number of values currently in the set. | O(1) |
| `Cap` | `func (s *Set[T]) Cap() int` | Capacity of the entry slice, holes included. | O(1) |
| `IsEmpty` | `func (s *Set[T]) IsEmpty() bool` | Reports whether the set has no values. | O(1) |
| `Contains` | `func (s *Set[T]) Contains(v T) bool` | Reports whether `v` is present. | O(1) avg |
| `ContainsAll` | `func (s *Set[T]) ContainsAll(vs ...T) bool` | Reports whether every value in `vs` is present; `true` with no arguments. | O(k) avg |
| `ContainsAny` | `func (s *Set[T]) ContainsAny(vs ...T) bool` | Reports whether any value in `vs` is present; `false` with no arguments. | O(k) avg |
| `First` | `func (s *Set[T]) First() (T, bool)` | Returns the oldest value without removing it; `ok` is `false` if empty. | O(1) |
| `Last` | `func (s *Set[T]) Last() (T, bool)` | Returns the newest value without removing it; `ok` is `false` if empty. | O(1) |
| `All` | `func (s *Set[T]) All() iter.Seq[T]` | Iterator over values from oldest to newest, without consuming them. | O(1) per step |
| `Backward` | `func (s *Set[T]) Backward() iter.Seq[T]` | Iterator over values from newest to oldest, without consuming them. | O(1) per step |
| `Clone` | `func (s *Set[T]) Clone() *Set[T]` | Returns an independent **shallow** copy; same order, no holes. | O(n) |
| `Slice` | `func (s *Set[T]) Slice() []T` | Returns an independent copy of the values from oldest to newest (same order as `All`); `nil` if empty. | O(n) |
| `Add` | `func (s *Set[T]) Add(v T) bool` | Adds `v`; keeps the place of a present value; reports whether `v` was new. | amortized O(1) |
| `AddN` | `func (s *Set[T]) AddN(vs ...T)` | Adds every value in `vs` in argument order; keeps the place of a present value. | amortized O(k) |
| `AddSet` | `func (s *Set[T]) AddSet(other *Set[T])` | In-place union with `other`. | amortized O(m) |
| `Remove` | `func (s *Set[T]) Remove(v T) bool` | Deletes `v`; reports whether `v` was present. | amortized O(1) |
| `RemoveSet` | `func (s *Set[T]) RemoveSet(other *Set[T])` | In-place difference with `other`. | amortized O(m) |
| `RetainSet` | `func (s *Set[T]) RetainSet(other *Set[T])` | In-place intersection with `other`. | O(n) |
| `RemoveFunc` | `func (s *Set[T]) RemoveFunc(del func(T) bool) int` | Deletes every value `del` selects, oldest to newest; returns how many; leaves no holes. | O(n) |
| `PopFirst` | `func (s *Set[T]) PopFirst() (T, bool)` | Removes and returns the oldest value; `ok` is `false` if empty. | amortized O(1) |
| `PopLast` | `func (s *Set[T]) PopLast() (T, bool)` | Removes and returns the newest value; `ok` is `false` if empty. | amortized O(1) |
| `MoveToBack` | `func (s *Set[T]) MoveToBack(v T) bool` | Makes `v` the newest value; reports whether `v` was present. | amortized O(1) |
| `Union` | `func (s *Set[T]) Union(others ...*Set[T]) *Set[T]` | New set with the values of all operands. | O(n+m) |
| `Intersection` | `func (s *Set[T]) Intersection(others ...*Set[T]) *Set[T]` | New set with the values of `s` present in every operand. | O(n·k) avg |
| `Difference` | `func (s *Set[T]) Difference(others ...*Set[T]) *Set[T]` | New set with the values of `s` absent from every operand. | O(n·k) avg |
| `SymmetricDifference` | `func (s *Set[T]) SymmetricDifference(other *Set[T]) *Set[T]` | New set with the values in exactly one of the two sets. | O(n+m) |
| `Equal` | `func (s *Set[T]) Equal(other *Set[T]) bool` | Reports whether both sets hold the same values; order is ignored. | O(n) avg |
| `IsSubset` | `func (s *Set[T]) IsSubset(other *Set[T]) bool` | Reports whether every value of `s` is in `other`. | O(n) avg |
| `IsSuperset` | `func (s *Set[T]) IsSuperset(other *Set[T]) bool` | Reports whether every value of `other` is in `s`. | O(m) avg |
| `IsDisjoint` | `func (s *Set[T]) IsDisjoint(other *Set[T]) bool` | Reports whether the sets share no values; iterates the smaller set. | O(min(n, m)) avg |
| `Clear` | `func (s *Set[T]) Clear()` | Removes all values **and releases** the entry slice and the index (`Cap` drops to 0). | O(1) |
| `Reset` | `func (s *Set[T]) Reset()` | Removes all values but **keeps** the entry slice and the index for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (s *Set[T]) Shrink()` | Shrink-to-fit: **copies** into a right-sized entry slice, frees the old one now, and rebuilds the index; no copy when `Cap == Len` and there are no holes. | O(n) |
| `Clip` | `func (s *Set[T]) Clip()` | Reslices the entry slice so `Cap == Len` **without copying**; holes and the index are untouched. | O(1) |
| `Grow` | `func (s *Set[T]) Grow(n int)` | Reserves entry-slice capacity for at least `n` more values; no-op when `n <= 0` or capacity already suffices. | O(n) |

`n` is the number of values in the set (or in the input for `Of` and `From`);
`m` is the total number of values in the set arguments; `k` is the number of
variadic arguments.

### The comma-ok contract

`First`, `Last`, `PopFirst`, and `PopLast` return a value and a boolean. The
boolean is `true` when a value was found and `false` when the set is empty (or
nil); in that case the value is the zero value of `T`. `Add` returns `true`
when the value was new; `Remove` and `MoveToBack` return whether the value was
present; `RemoveFunc` returns the number of removed values.

```go
v, ok := s.PopFirst()
if !ok {
	// set was empty; v is the zero value
}
```

### Iteration with `All`

`All` returns an [`iter.Seq[T]`](https://pkg.go.dev/iter#Seq) — a Go 1.23
range-over-func iterator — that yields values from **oldest to newest**;
`Backward` yields them from **newest to oldest**. Neither consumes the set, and
breaking out of the loop early stops iteration cleanly:

```go
for v := range s.All() {
	fmt.Println(v)
	if v == target {
		break // safe; the set is untouched
	}
}
fmt.Println(s.Len()) // unchanged

for v := range s.Backward() {
	fmt.Println(v) // newest to oldest
}
```

Mutating the set while iterating over `All` or `Backward` — including `Add` of
a new value, `Remove`, and `MoveToBack` from the loop body — is **undefined
behavior**: compaction can move entries under the iterator. Use `RemoveFunc` or
`RetainSet` to delete the values you select, or collect them first and mutate
afterwards.

### Bulk add with `AddN`

`AddN` adds several values at once **in argument order**, exactly as if you
called `Add` for each in turn; values already present keep their place:

```go
s := orderedset.Of("a")
s.AddN("b", "a", "c")
fmt.Println(s.Slice()) // [a b c]
```

With no arguments it is a no-op.

### Copying out with `Clone` and `Slice`

`Clone` returns an independent set with the same order and no holes; `Slice`
returns a plain slice. Both are detached from the original — they share no
entry slice or index, so mutating one does not affect the other:

```go
a := orderedset.Of(1, 2, 3)

b := a.Clone() // independent *Set[int]
b.Remove(1)    // removes 1 from b only

fmt.Println(a.Slice()) // [1 2 3]
fmt.Println(b.Slice()) // [2 3]
```

Both copies are **shallow**: when `T` is a pointer or contains references, the
pointed-to data is shared between the copies. The clone's capacity equals its
length. `Slice` returns `nil` for an empty set, and `Clone` of a nil set
returns a new, usable empty set.

### Set algebra

`Union`, `Intersection`, `Difference`, and `SymmetricDifference` return fresh
sets in a defined order (see [Ordering](#ordering)) and leave their operands
unchanged; nil operands are treated as empty sets:

```go
a := orderedset.Of(1, 2, 3)
b := orderedset.Of(3, 4)

fmt.Println(a.Union(b).Slice())               // [1 2 3 4]
fmt.Println(a.Intersection(b).Slice())        // [3]
fmt.Println(a.Difference(b).Slice())          // [1 2]
fmt.Println(a.SymmetricDifference(b).Slice()) // [1 2 4]
```

For hot paths, `AddSet`, `RemoveSet`, and `RetainSet` are the in-place forms of
`Union`, `Difference`, and `Intersection`: they mutate the receiver without
allocating a result set.

## Capacity management

The set grows automatically, but several methods let you control the backing
storage explicitly:

- **Reserve ahead** — `NewWithCap(n)` sizes both the entry slice and the index
  at construction; `Grow(n)` reserves entry-slice room for `n` more values on
  an existing set.
- **Reclaim memory** — `Shrink` removes holes, copies into a right-sized entry
  slice, and rebuilds the index; `Clip` only reslices the entry slice (see
  below); `Clear` releases both entirely.
- **Reuse memory** — `Reset` empties the set but keeps the entry slice and the
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
| `Shrink` | yes (packed, new slice) | O(n) | removed, memory freed now | rebuilt (reclaimed) | The set grew large, has shrunk, and you want the memory back now. |
| `Clip` | no (reslice) | O(1) | left as they are | left as is | You want the cheap version and can wait for lazy reclaim. |

```go
s := orderedset.NewWithCap[int](1024)
s.AddN(1, 2, 3)

s.Clip()
fmt.Println(s.Len(), s.Cap()) // 3 3  — O(1) reslice, old array still retained

s.Grow(2000)
s.Shrink()
fmt.Println(s.Len(), s.Cap()) // 3 3  — copied into a right-sized slice, old one freed
```

Both are no-ops on the entry slice when `Cap` already equals `Len` and there
are no holes; `Shrink` still rebuilds the index.

## `Clear` vs `Reset`

Both empty the set and drop all references to the removed values (so they
become eligible for garbage collection), but they differ in what happens to
the entry slice and the index:

| Operation | Empties set | Entry slice and index | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the set, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the set and want to avoid reallocating. |

```go
s := orderedset.NewWithCap[int](8)
s.AddN(1, 2)

s.Reset()
fmt.Println(s.Len(), s.Cap()) // 0 8  — capacity kept for reuse

s.Clear()
fmt.Println(s.Len(), s.Cap()) // 0 0  — entry slice and index released
```

## Nil receiver and zero value

A nil `*Set[T]` behaves as a valid **empty** set for every read-only method
and for removal methods, so you can safely query a `nil` set:

```go
var s *orderedset.Set[string] // nil

s.Len()              // 0
s.IsEmpty()          // true
s.Cap()              // 0
s.Contains("a")      // false
v, ok := s.First()   // "", false
s.Remove("a")        // false
s.RemoveFunc(f)      // 0
v, ok = s.PopFirst() // "", false (nothing to remove)
s.MoveToBack("a")    // false
s.Slice()            // nil
for range s.All() { /* never runs */ }
s.Clone() // new, usable empty set (never nil)
```

Set algebra treats nil operands as empty sets.

The **mutating** methods — `Add`, `AddN`, `AddSet`, `Clear`, `Reset`,
`Shrink`, `Clip`, and `Grow` — need a non-nil receiver, so they **panic** on a
`nil` set with a message such as `orderedset: Add on nil receiver`.
`RemoveFunc` panics with `orderedset: nil predicate` when `del` is nil, even on
a nil set.

The zero value of `Set[T]` is usable, so `New` is optional; `Add` and `AddN`
allocate the index on first insertion:

```go
var s orderedset.Set[string]
s.Add("a")
fmt.Println(s.Len()) // 1
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
| `From/orderedset` | build from 4096 values, each appearing twice | 74 835 | 213 408 | 20 |
| `From/set` | the same in `set` | 60 593 | 147 832 | 19 |
| `Contains/orderedset/hit` | `Contains` of a present value, 4096 values | 8.6 | 0 | 0 |
| `Contains/orderedset/miss` | `Contains` of an absent value, 4096 values | 7.8 | 0 | 0 |
| `Contains/set/hit` | the same present-value lookup in `set` | 8.5 | 0 | 0 |
| `Contains/set/miss` | the same absent-value lookup in `set` | 8.0 | 0 | 0 |
| `First` | one `First` on 4096 values | 1.4 | 0 | 0 |
| `Last` | one `Last` on 4096 values | 1.4 | 0 | 0 |
| `All/orderedset` | iterate 4096 values with `All` | 8 752 | 0 | 0 |
| `All/set` | the same iteration over `set` | 35 370 | 0 | 0 |
| `Backward` | iterate 4096 values with `Backward` | 8 851 | 0 | 0 |
| `Clone/orderedset` | clone 4096 values | 102 342 | 213 456 | 21 |
| `Clone/set` | the same clone of `set` | 125 174 | 147 833 | 19 |
| `Slice` | copy 4096 values | 6 951 | 32 768 | 1 |
| `Add/orderedset` | `Add` into a set growing from empty to 4096 values | 85.3 | 135 | 0 |
| `Add/set` | the same inserts into `set` | 70.6 | 72 | 0 |
| `AddExisting` | `Add` of a present value, 4096 values | 10.3 | 0 | 0 |
| `AddN` | one `AddN` of 256 new values into preallocated room | 6 026 | 0 | 0 |
| `Remove` | one `Remove` of a present value, refilled outside the timing | 61.5 | 0 | 0 |
| `RemoveFunc` | one `RemoveFunc` removing every other value of 4096, refilled outside the timing | 85 190 | 0 | 0 |
| `PopFirst` | one `PopFirst`, refilled outside the timing | 46.0 | 0 | 0 |
| `PopLast` | one `PopLast`, refilled outside the timing | 30.1 | 0 | 0 |
| `MoveToBack` | `MoveToBack` of a present value, 4096 values | 75.6 | 0 | 0 |
| `Union/orderedset/n=16` | union of two 16-value sets, half overlapping | 1 004 | 1 528 | 8 |
| `Union/orderedset/n=1024` | union of two 1024-value sets, half overlapping | 46 846 | 78 048 | 10 |
| `Union/orderedset/n=65536` | union of two 65536-value sets, half overlapping | 4 839 454 | 6 386 961 | 263 |
| `Union/set/n=16` | the same in `set` | 1 110 | 672 | 5 |
| `Union/set/n=1024` | the same in `set` | 59 947 | 37 000 | 7 |
| `Union/set/n=65536` | the same in `set` | 5 629 937 | 2 364 603 | 259 |
| `Intersection/orderedset/n=16` | intersection of two 16-value sets, half overlapping | 565 | 968 | 6 |
| `Intersection/orderedset/n=1024` | intersection of two 1024-value sets, half overlapping | 35 976 | 53 424 | 8 |
| `Intersection/orderedset/n=65536` | intersection of two 65536-value sets, half overlapping | 2 918 117 | 3 413 216 | 260 |
| `Intersection/set/n=16` | the same in `set` | 1 294 | 672 | 5 |
| `Intersection/set/n=1024` | the same in `set` | 66 947 | 37 000 | 7 |
| `Intersection/set/n=65536` | the same in `set` | 6 305 582 | 2 364 600 | 259 |
| `Difference/orderedset/n=16` | difference of two 16-value sets, half overlapping | 807 | 968 | 6 |
| `Difference/orderedset/n=1024` | difference of two 1024-value sets, half overlapping | 42 904 | 53 424 | 8 |
| `Difference/orderedset/n=65536` | difference of two 65536-value sets, half overlapping | 2 759 832 | 3 413 216 | 260 |
| `Difference/set/n=16` | the same in `set` | 1 224 | 672 | 5 |
| `Difference/set/n=1024` | the same in `set` | 71 308 | 37 000 | 7 |
| `Difference/set/n=65536` | the same in `set` | 6 313 668 | 2 364 600 | 259 |
| `AddRemoveChurn/orderedset` | one `Remove` + one `Add` of the same value with 4096 live values | 79.9 | 0 | 0 |
| `AddRemoveChurn/set` | the same churn in `set` | 44.7 | 0 | 0 |
| `PopFirstAddChurn` | one `PopFirst` + one `Add` of a new value with 4096 live values | 91.1 | 0 | 0 |

- `B/op` for `Add` is the amortized cost of growing the entry slice and the index. Pre-size with `NewWithCap` (entry slice and index) or `Grow` (entry slice only) to avoid it.
- `Union`, `Intersection`, and `Difference` allocate a result set, so their allocation counts grow with the input size; `AddSet`, `RemoveSet`, and `RetainSet` avoid that allocation.

Run them with:

```sh
go test -bench=. -benchmem ./orderedset/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including set algebra, `RemoveFunc`, `MoveToBack`, cloning, shrinking, nil and
zero-value use, and deduplication — and are rendered alongside the API on the
godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, with the
unordered `set` as the baseline; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/orderedset        # package overview
go doc github.com/Nergous/structures/orderedset Set    # the Set type and its methods
```

The same comments render on pkg.go.dev-style godoc.
