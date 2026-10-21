# bitset

A **set of non-negative ints** backed by a bit vector, for Go.

It is part of the `structures` data-structures library. Value `v` is bit
`v%64` of word `v/64`, so membership checks, adds, and removes are a shift and
a mask, set algebra works on 64 values at a time, and iteration runs in
ascending order. It uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/bitset"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/bitset"
)

func main() {
	s := bitset.Of(64, 3, 64, 0)

	fmt.Println(s.Contains(3)) // true
	fmt.Println(s.Slice())     // [0 3 64], always ascending
	fmt.Println(s.Max())       // 64 true
}
```

Preallocate when the largest value is known up front to avoid reallocations
while adding:

```go
s := bitset.NewWithCap(1000) // covers the values 0..999
for i := range 1000 {
	s.Add(i)
}
fmt.Println(s.Len(), s.Cap()) // 1000 1024
```

Remove values selected by a predicate in one ascending pass:

```go
removed := s.RemoveFunc(func(v int) bool {
	return v%2 == 0
})
fmt.Println(removed, s.Len()) // 500 500
```

## When to use `bitset` vs `set`

| Need | Use |
| --- | --- |
| Dense small ints: IDs, indexes, flags, sieve, graph visited marks | `bitset` |
| Fast set algebra over many values | `bitset` |
| Values in ascending order, `Min` / `Max` | `bitset` |
| Sparse or very large ints, such as `1 << 40` | `set` |
| Negative ints, strings, or any other comparable type | `set` |

Memory grows with the **largest** stored value, not with the number of
values: a `bitset` holding only `1 << 20` still uses 128 KiB, while a
[`set`](../set) holding the same value uses a few dozen bytes. For dense
values the trade flips: 4096 values take 512 bytes here and about 150 KiB in a
map.

## Ordering

Iteration order is **ascending** and stable between calls. `All` and `Slice`
yield the smallest value first, and `Backward` yields the values in
descending order. No sorting is needed for deterministic output:

```go
s := bitset.Of(130, 3, 64)
for v := range s.All() {
	fmt.Println(v) // 3, 64, 130
}
```

## Negative values

A bitset stores only non-negative ints. Negative values are never present:
`Contains`, `ContainsAll`, `ContainsAny`, and `Remove` report `false` for them
instead of panicking. `Add`, `AddN`, `Of`, and `From` **panic** on a negative
value with `bitset: Add of negative value` or `bitset: AddN of negative
value`; `AddN` checks every value first, so the set is left unchanged.

## How it works

The set keeps a slice of `uint64` words and a cached count of set bits, so
`Len` is O(1). The slice never ends in an empty word: removing the largest
values trims the trailing empty words (keeping the capacity), which makes
`Max` O(1) and lets `Equal` compare the word slices directly. Words in the
spare capacity may hold stale bits; they are cleared when a later `Add`
reuses them, so `Reset` is O(1).

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New() *Set` | Creates an empty set with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap(n int) *Set` | Creates an empty set covering the values `0..n-1`, rounded up to a multiple of 64; a negative `n` is treated as 0. | O(n/64) |
| `Of` | `func Of(vs ...int) *Set` | Creates a set of the values of `vs`, each stored once; panics on a negative value. | O(k + m/64) |
| `From` | `func From(s []int) *Set` | Creates a set of the values of `s`, each stored once; never aliases `s`; panics on a negative value. | O(k + m/64) |
| `Len` | `func (s *Set) Len() int` | Number of values currently in the set. | O(1) |
| `Cap` | `func (s *Set) Cap() int` | Number of values the backing array covers: `0..Cap()-1` can be added without reallocating. | O(1) |
| `IsEmpty` | `func (s *Set) IsEmpty() bool` | Reports whether the set has no values. | O(1) |
| `Contains` | `func (s *Set) Contains(v int) bool` | Reports whether `v` is present; `false` for negative `v`. | O(1) |
| `ContainsAll` | `func (s *Set) ContainsAll(vs ...int) bool` | Reports whether every value in `vs` is present; `true` with no arguments. | O(k) |
| `ContainsAny` | `func (s *Set) ContainsAny(vs ...int) bool` | Reports whether any value in `vs` is present; `false` with no arguments. | O(k) |
| `Min` | `func (s *Set) Min() (int, bool)` | Smallest value; `(0, false)` if empty. | O(min/64) |
| `Max` | `func (s *Set) Max() (int, bool)` | Largest value; `(0, false)` if empty. | O(1) |
| `All` | `func (s *Set) All() iter.Seq[int]` | Iterator over values in ascending order, without consuming them. | O(m/64 + n) |
| `Backward` | `func (s *Set) Backward() iter.Seq[int]` | Iterator over values in descending order, without consuming them. | O(m/64 + n) |
| `Clone` | `func (s *Set) Clone() *Set` | Returns an independent copy without the original's spare capacity. | O(m/64) |
| `Slice` | `func (s *Set) Slice() []int` | Returns an independent copy of the values in ascending order; `nil` if empty. | O(m/64 + n) |
| `Add` | `func (s *Set) Add(v int) bool` | Adds `v`; reports whether `v` was new; panics on a negative `v`. | O(1), O(v/64) when growing |
| `AddN` | `func (s *Set) AddN(vs ...int)` | Adds every value in `vs`, growing the backing array at most once. | O(k + m/64) |
| `AddSet` | `func (s *Set) AddSet(other *Set)` | In-place union with `other`, word by word. | O(m/64) |
| `Remove` | `func (s *Set) Remove(v int) bool` | Deletes `v`; reports whether `v` was present. | amortized O(1) |
| `RemoveSet` | `func (s *Set) RemoveSet(other *Set)` | In-place difference with `other`, word by word. | O(m/64) |
| `RetainSet` | `func (s *Set) RetainSet(other *Set)` | In-place intersection with `other`, word by word. | O(m/64) |
| `RemoveFunc` | `func (s *Set) RemoveFunc(del func(int) bool) int` | Deletes every value `del` selects, in ascending order; returns how many. | O(m/64 + n) |
| `Union` | `func (s *Set) Union(others ...*Set) *Set` | New set with the values of all operands. | O(k·m/64) |
| `Intersection` | `func (s *Set) Intersection(others ...*Set) *Set` | New set with the values of `s` present in every operand. | O(k·m/64) |
| `Difference` | `func (s *Set) Difference(others ...*Set) *Set` | New set with the values of `s` absent from every operand. | O(k·m/64) |
| `SymmetricDifference` | `func (s *Set) SymmetricDifference(other *Set) *Set` | New set with the values in exactly one of the two sets. | O(m/64) |
| `Equal` | `func (s *Set) Equal(other *Set) bool` | Reports whether both sets hold the same values, regardless of capacity. | O(m/64) |
| `IsSubset` | `func (s *Set) IsSubset(other *Set) bool` | Reports whether every value of `s` is in `other`. | O(m/64) |
| `IsSuperset` | `func (s *Set) IsSuperset(other *Set) bool` | Reports whether every value of `other` is in `s`. | O(m/64) |
| `IsDisjoint` | `func (s *Set) IsDisjoint(other *Set) bool` | Reports whether the sets share no values. | O(m/64) |
| `Clear` | `func (s *Set) Clear()` | Removes all values **and releases** the backing array. | O(1) |
| `Reset` | `func (s *Set) Reset()` | Removes all values but **keeps** the backing array for reuse. | O(1) |
| `Shrink` | `func (s *Set) Shrink()` | Shrink-to-fit: **copies** the words in use into a right-sized array and frees the old one now. | O(m/64) |
| `Clip` | `func (s *Set) Clip()` | Trims spare capacity **without copying**; the tail is reclaimed lazily. | O(1) |
| `Grow` | `func (s *Set) Grow(n int)` | Ensures the backing array covers the values `0..n-1`. | O(n/64) |

`n` is the number of values in the set; `m` is the largest value involved;
`k` is the number of variadic arguments or operands.

### The comma-ok contract

`Min` and `Max` return `(value, true)`, or `(0, false)` on an empty set. The
mutators follow the same idea: `Add` returns `true` when the value was new and
`false` when it was already present; `Remove` returns `true` when the value
was present and `false` when it was absent or negative. In both `false` cases
the set is unchanged. `RemoveFunc` returns the number of removed values.

```go
s := bitset.New()
fmt.Println(s.Add(1))     // true
fmt.Println(s.Add(1))     // false, already present
fmt.Println(s.Remove(1))  // true
fmt.Println(s.Remove(-1)) // false, never present
fmt.Println(s.Min())      // 0 false
```

`ContainsAll()` with no arguments returns `true`; `ContainsAny()` with no
arguments returns `false`.

### Iteration with `All` and `Backward`

`All` and `Backward` return an [`iter.Seq[int]`](https://pkg.go.dev/iter#Seq)
— a Go 1.23 range-over-func iterator — that yields every value once, in
ascending or descending order. They do not consume the set, and breaking out
of the loop early stops iteration cleanly:

```go
for v := range s.Backward() {
	if v < limit {
		break // safe; the set is untouched
	}
	fmt.Println(v)
}
```

Mutating the set while iterating is undefined behavior. Use `RemoveFunc` or
`RetainSet` to delete the values you select, or collect them first and mutate
afterwards.

### Bulk add with `AddN`

`AddN` adds several values at once; duplicates, both among the arguments and
against the set, are stored once. It finds the largest value first and grows
the backing array at most once:

```go
var s bitset.Set
s.AddN(500, 1, 3, 500)
fmt.Println(s.Slice()) // [1 3 500]
```

With no arguments it is a no-op.

### Copying out with `Clone` and `Slice`

`Clone` returns an independent set; `Slice` returns a plain slice in ascending
order. Both are detached from the original, so mutating one does not affect
the other:

```go
a := bitset.Of(1, 2, 3)

b := a.Clone() // independent *Set
b.Remove(1)    // removes 1 from b only

fmt.Println(a.Slice()) // [1 2 3]
fmt.Println(b.Slice()) // [2 3]
```

The clone covers exactly the words in use, without the original's spare
capacity. `Slice` returns `nil` for an empty set, and `Clone` of a nil set
returns a new, usable empty set.

### Set algebra

`Union`, `Intersection`, `Difference`, and `SymmetricDifference` return fresh
sets and leave their operands unchanged; nil operands are treated as empty
sets. They combine 64 values per machine word:

```go
a := bitset.Of(1, 2, 3)
b := bitset.Of(3, 4)

fmt.Println(a.Union(b).Slice())               // [1 2 3 4]
fmt.Println(a.Intersection(b).Slice())        // [3]
fmt.Println(a.Difference(b).Slice())          // [1 2]
fmt.Println(a.SymmetricDifference(b).Slice()) // [1 2 4]
```

For hot paths, `AddSet`, `RemoveSet`, and `RetainSet` are the in-place forms of
`Union`, `Difference`, and `Intersection`: they mutate the receiver without
allocating a result set.

## Capacity management

Capacity in a bitset bounds the **largest value**, not the number of values:
`Cap()` reports how many values from 0 up the backing array covers, always a
multiple of 64.

- **Reserve ahead** — `NewWithCap(n)` at construction, or `Grow(n)` later,
  covers the values `0..n-1`, so adding them does not reallocate. Both take
  the same value bound.
- **Reclaim memory** — `Shrink` copies the words in use into a right-sized
  array and frees the old one now; `Clip` trims the reported capacity without
  copying; `Clear` releases the backing array entirely.
- **Reuse memory** — `Reset` empties the set but keeps the backing array for
  refilling.

Removing the largest values trims the words in use but keeps the capacity, so
a set that once held a large value keeps its memory until `Shrink`, `Clip`, or
`Clear`:

```go
s := bitset.Of(1, 2, 100_000)
s.Remove(100_000)
s.Shrink()
fmt.Println(s.Cap()) // 64
```

## `Shrink` vs `Clip`

| Operation | Copies | Allocates | Memory reclaimed | Complexity |
| --- | --- | --- | --- | --- |
| `Shrink` | yes | once, unless already right-sized | immediately | O(m/64) |
| `Clip` | no | no | lazily, on the next growth | O(1) |

An empty set releases its backing array on `Shrink`, as after `Clear`.

## `Clear` vs `Reset`

Both empty the set, but they differ in what happens to the allocated backing
array. A bitset stores no references, so neither has values to release for
garbage collection:

| Operation | Empties set | Backing array | Use when |
| --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | You are done with the set, or want to free its memory. |
| `Reset` | yes | retained for reuse | You will refill the set and want to avoid reallocating. |

```go
s := bitset.NewWithCap(1024)
s.AddN(1, 2)

s.Reset()
fmt.Println(s.Len(), s.Cap()) // 0 1024 — capacity kept for reuse

s.Clear()
fmt.Println(s.Len(), s.Cap()) // 0 0    — backing array released
```

## Nil receiver and zero value

A nil `*Set` behaves as a valid **empty** set for every read-only method and
for removal methods, so you can safely query a `nil` set:

```go
var s *bitset.Set // nil

s.Len()         // 0
s.Cap()         // 0
s.IsEmpty()     // true
s.Contains(1)   // false
s.Max()         // 0, false
s.Remove(1)     // false (nothing to remove)
s.RemoveFunc(f) // 0
s.Slice()       // nil
for range s.All() { /* never runs */ }
s.Clone() // new, usable empty set (never nil)
```

Set algebra treats nil operands as empty sets.

The **mutating** methods — `Add`, `AddN`, `AddSet`, `Clear`, `Reset`,
`Shrink`, `Clip`, and `Grow` — need a non-nil receiver, so they **panic** on a
`nil` set with a message such as `bitset: Add on nil receiver`. `RemoveFunc`
panics with `bitset: nil predicate` when `del` is nil, even on a nil set.

The zero value of `Set` is usable, so `New` is optional; `Add` and `AddN`
allocate the backing array on first insertion:

```go
var s bitset.Set
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

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-21. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `From` | build from 4096 values, each appearing twice | 6 123 | 288 | 2 |
| `Contains/bitset/hit` | `Contains` of a present value, 4096 values | 1.7 | 0 | 0 |
| `Contains/bitset/miss` | `Contains` of an absent value, 4096 values | 1.8 | 0 | 0 |
| `Contains/builtin-map/hit` | the same present-value lookup in a `map[int]struct{}` | 8.1 | 0 | 0 |
| `Contains/builtin-map/miss` | the same absent-value lookup in a `map[int]struct{}` | 7.5 | 0 | 0 |
| `Min` | `Min` with the smallest value in the last of 64 words | 21.4 | 0 | 0 |
| `All/bitset` | iterate 4096 values with `All` | 10 744 | 0 | 0 |
| `All/builtin-map` | the same iteration over a `map[int]struct{}` | 30 120 | 0 | 0 |
| `Clone/bitset` | clone 4096 values | 110 | 544 | 2 |
| `Clone/builtin-map` | the same with `maps.Clone` on a `map[int]struct{}` | 23 422 | 147 824 | 18 |
| `Slice` | copy 4096 values | 7 861 | 32 768 | 1 |
| `Add/bitset` | `Add` into a set growing from empty to 4096 values | 3.5 | 0 | 0 |
| `Add/builtin-map` | the same inserts into a `map[int]struct{}` | 58.8 | 72 | 0 |
| `AddExisting` | `Add` of a present value, 4096 values | 2.9 | 0 | 0 |
| `AddN` | one `AddN` of 256 new values into preallocated room | 452 | 0 | 0 |
| `AddSet` | merge 4096 values, half overlapping, into preallocated room | 118 | 0 | 0 |
| `Remove` | one `Remove` of a present value, refilled outside the timing | 2.8 | 0 | 0 |
| `RemoveFunc` | one `RemoveFunc` removing every other value of 4096, refilled outside the timing | 9 647 | 0 | 0 |
| `Union/n=16` | union of two 16-value sets, half overlapping | 45.7 | 40 | 2 |
| `Union/n=1024` | union of two 1024-value sets, half overlapping | 105 | 224 | 2 |
| `Union/n=65536` | union of two 65536-value sets, half overlapping | 3 014 | 12 320 | 2 |
| `Intersection/n=16` | intersection of two 16-value sets, half overlapping | 37.1 | 40 | 2 |
| `Intersection/n=1024` | intersection of two 1024-value sets, half overlapping | 79.1 | 160 | 2 |
| `Intersection/n=65536` | intersection of two 65536-value sets, half overlapping | 2 498 | 8 224 | 2 |
| `Difference/n=16` | difference of two 16-value sets, half overlapping | 37.4 | 40 | 2 |
| `Difference/n=1024` | difference of two 1024-value sets, half overlapping | 74.3 | 160 | 2 |
| `Difference/n=65536` | difference of two 65536-value sets, half overlapping | 2 576 | 8 224 | 2 |
| `IsSubset` | subset check of 4096 values against 8192, every word compared | 23.1 | 0 | 0 |
| `AddRemoveChurn/bitset` | one `Remove` + one `Add` of the same value with 4096 live values | 4.8 | 0 | 0 |
| `AddRemoveChurn/builtin-map` | the same churn in a `map[int]struct{}` | 31.1 | 0 | 0 |

- `Add` grows the backing array with amortized doubling; at 4096 values the growth cost rounds to 0 B/op. Pre-size with `NewWithCap` to avoid it entirely.
- `Union`, `Intersection`, and `Difference` allocate one result set and its array, so their allocation count stays at 2 for every size; `AddSet`, `RemoveSet`, and `RetainSet` avoid that allocation.

Run them with:

```sh
go test -bench=. -benchmem ./bitset/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including set algebra, `RemoveFunc`, `Min` / `Max`, descending iteration,
capacity controls, nil and zero-value use, and a sieve of Eratosthenes — and
are rendered alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, with a
built-in `map[int]struct{}` as the baseline; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/bitset        # package overview
go doc github.com/Nergous/structures/bitset Set    # the Set type and its methods
```

The same comments render on pkg.go.dev-style godoc.
