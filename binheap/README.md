# binheap

A generic, slice-backed **binary heap** for Go.

It is part of the `structures` data-structures library. It replaces
[`container/heap`](https://pkg.go.dev/container/heap) without the interface
boilerplate: the heap is min-first by default, supports max-heaps and custom
comparators, never boxes values in `any`, and uses Go 1.23 range-over-func
iterators ([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/binheap"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

The package is named `binheap`, not `heap`, so code migrating from
`container/heap` does not need an import alias.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/binheap"
)

func main() {
	h := binheap.New[int]()
	h.PushN(5, 1, 4, 2, 3)

	for !h.IsEmpty() {
		v, _ := h.Pop()
		fmt.Println(v) // 1, 2, 3, 4, 5
	}
}
```

Build a heap from existing data in O(n). The heap copies the slice and never
aliases it:

```go
h := binheap.From([]int{5, 3, 8, 1})
top, _ := h.Peek()
fmt.Println(top) // 1
```

Order any type with a comparator:

```go
type task struct {
	name     string
	priority int
}

h := binheap.NewFunc(func(a, b task) int {
	return cmp.Compare(a.priority, b.priority)
})
h.Push(task{name: "write docs", priority: 3})
h.Push(task{name: "fix outage", priority: 1})

next, _ := h.Pop()
fmt.Println(next.name) // fix outage
```

## When to use `binheap` vs `pqueue`

| Need | Use |
| --- | --- |
| Duplicate elements, or elements that are not comparable | `binheap` |
| Maximum push/pop throughput | `binheap` |
| Change or remove a queued element by its identity | `pqueue` |
| Look up an element's current priority | `pqueue` |
| Stable order for equal priorities | `pqueue` |

`binheap` stores nothing but the elements, in one slice, so a push/pop cycle
does no bookkeeping and does not allocate (see [Benchmarks](#benchmarks)). The
price is that finding an element is a linear scan: `Contains` and `Remove` run
in O(n). [`pqueue`](../pqueue) keeps a map from every key to its heap position,
which makes key lookups O(1) and priority updates O(log n), at the cost of
updating that map on every move inside the heap.

## Ordering

The top of the heap is always its **least** element according to the
comparator. Throughout the docs, "least" is comparator-relative: for a max-heap
it is the largest value.

| Constructors | Element type | Order on top |
| --- | --- | --- |
| `New`, `NewWithCap`, `From` | `cmp.Ordered` | smallest value (`cmp.Compare`) |
| `NewMax`, `NewMaxWithCap`, `FromMax` | `cmp.Ordered` | largest value |
| `NewFunc`, `NewFuncWithCap`, `FromFunc` | any type | least per your comparator |

A comparator follows the
[`slices.SortFunc`](https://pkg.go.dev/slices#SortFunc) contract: negative when
`a` comes before `b`, positive when after, zero when they are equivalent. It
must be a strict weak ordering. The `Func` constructors panic on a nil
comparator.

The heap is **not stable**: elements that compare equal come out in unspecified
order.

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[T cmp.Ordered]() *Heap[T]` | Creates an empty min-heap with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[T cmp.Ordered](n int) *Heap[T]` | Creates an empty min-heap with capacity for at least `n` elements; a negative `n` is treated as 0. | O(n) |
| `From` | `func From[T cmp.Ordered](s []T) *Heap[T]` | Creates a min-heap from a copy of `s`; never aliases `s`. | O(n) |
| `NewMax` | `func NewMax[T cmp.Ordered]() *Heap[T]` | Creates an empty max-heap with no preallocated capacity. | O(1) |
| `NewMaxWithCap` | `func NewMaxWithCap[T cmp.Ordered](n int) *Heap[T]` | Creates an empty max-heap with capacity for at least `n` elements; a negative `n` is treated as 0. | O(n) |
| `FromMax` | `func FromMax[T cmp.Ordered](s []T) *Heap[T]` | Creates a max-heap from a copy of `s`; never aliases `s`. | O(n) |
| `NewFunc` | `func NewFunc[T any](compare func(a, b T) int) *Heap[T]` | Creates an empty heap ordered by `compare` with no preallocated capacity. | O(1) |
| `NewFuncWithCap` | `func NewFuncWithCap[T any](n int, compare func(a, b T) int) *Heap[T]` | Creates an empty heap ordered by `compare` with capacity for at least `n` elements; a negative `n` is treated as 0. | O(n) |
| `FromFunc` | `func FromFunc[T any](s []T, compare func(a, b T) int) *Heap[T]` | Creates a heap ordered by `compare` from a copy of `s`; never aliases `s`. | O(n) |
| `Len` | `func (h *Heap[T]) Len() int` | Number of elements currently in the heap. | O(1) |
| `Cap` | `func (h *Heap[T]) Cap() int` | Capacity of the backing array before the next reallocation. | O(1) |
| `IsEmpty` | `func (h *Heap[T]) IsEmpty() bool` | Reports whether the heap has no elements. | O(1) |
| `Peek` | `func (h *Heap[T]) Peek() (T, bool)` | Returns the least element without removing it; `ok` is `false` if empty. | O(1) |
| `Contains` | `func (h *Heap[T]) Contains(v T) bool` | Reports whether an element compares equal to `v`. | O(n) |
| `All` | `func (h *Heap[T]) All() iter.Seq[T]` | Iterator over elements in layout order (least first, rest unspecified), without consuming them. | O(1) per step |
| `Clone` | `func (h *Heap[T]) Clone() *Heap[T]` | Returns an independent **shallow** copy that shares the comparator; `nil` for a nil heap. | O(n) |
| `Slice` | `func (h *Heap[T]) Slice() []T` | Returns an independent copy of the elements in layout order (same order as `All`); `nil` if empty. | O(n) |
| `Sorted` | `func (h *Heap[T]) Sorted() []T` | Returns an independent copy of the elements in priority order, without consuming the heap; `nil` if empty. | O(n log n) |
| `Push` | `func (h *Heap[T]) Push(v T)` | Adds `v` to the heap. | amortized O(log n) |
| `PushN` | `func (h *Heap[T]) PushN(vs ...T)` | Adds `vs`; grows the backing array at most once and picks sift-up or rebuild, whichever is cheaper. | O(min(k log(n+k), n+k)) |
| `Pop` | `func (h *Heap[T]) Pop() (T, bool)` | Removes and returns the least element; `ok` is `false` if empty. | O(log n) |
| `PopN` | `func (h *Heap[T]) PopN(n int) []T` | Removes and returns up to `n` least elements in priority order; `nil` if `n <= 0` or empty. | O(k log n) |
| `PushPop` | `func (h *Heap[T]) PushPop(v T) T` | Push then pop in one sift; returns `v` untouched when it would be popped first anyway. | O(log n) |
| `Replace` | `func (h *Heap[T]) Replace(v T) (T, bool)` | Pop then push in one sift; always stores `v`; `ok` reports whether an element was popped. | O(log n) |
| `Remove` | `func (h *Heap[T]) Remove(v T) bool` | Removes one element that compares equal to `v`; reports whether one was found. | O(n) |
| `Clear` | `func (h *Heap[T]) Clear()` | Removes all elements **and releases** the backing array (`Cap` drops to 0); keeps the comparator. | O(1) |
| `Reset` | `func (h *Heap[T]) Reset()` | Removes all elements but **keeps** the backing array for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (h *Heap[T]) Shrink()` | Shrink-to-fit: **copies** into a right-sized array and frees the old one now; no-op when `Cap == Len`. | O(n) |
| `Clip` | `func (h *Heap[T]) Clip()` | Reslices so `Cap == Len` **without copying**; unused memory is reclaimed only on the next growth. | O(1) |
| `Grow` | `func (h *Heap[T]) Grow(n int)` | Reserves capacity for at least `n` more elements; no-op when `n <= 0` or capacity already suffices. | O(n) |

`n` is the number of elements in the heap; `k` is the number of elements pushed
or popped by the call.

### The comma-ok contract

`Peek` and `Pop` return a value and a boolean. The boolean is `true` when an
element was found and `false` when the heap is empty (or nil); in that case the
value is the zero value of `T`. `Replace` uses the same shape, but its boolean
reports whether an element was **popped**: on an empty heap it stores `v` and
returns `(zero, false)`.

```go
v, ok := h.Pop()
if !ok {
	// heap was empty; v is the zero value
}
```

### Iteration with `All`

`All` returns an [`iter.Seq[T]`](https://pkg.go.dev/iter#Seq) — a Go 1.23
range-over-func iterator — that yields elements in the internal heap
**layout order**, the same order as `Slice`. The first element is the least
one; the order of the rest is **unspecified and not sorted**, which suits
order-independent work such as sums, counts, and serialization of a heap that
will be rebuilt with `From`. It does not consume the heap, and breaking out of
the loop early stops iteration cleanly.

`Sorted` returns an independent copy in priority order (the order repeated
`Pop` calls would produce) without consuming the heap. It costs O(n log n).

```go
h := binheap.FromMax([]int{2, 9, 4, 7})

fmt.Println(h.Sorted()) // [9 7 4 2]
fmt.Println(h.Len())    // 4

sum := 0
for v := range h.All() { // layout order; early break is safe
	sum += v
}
fmt.Println(sum) // 22
```

Mutating the heap while iterating over `All` is undefined behavior.

### Bulk push with `PushN`

`PushN` adds a batch and grows the backing array at most once. It compares the
worst-case cost of sifting each new element up against rebuilding the whole
heap and picks the cheaper one: small batches into a large heap are sifted in
O(k log(n+k)), and batches comparable to the heap size are rebuilt in O(n+k):

```go
h := binheap.New[int]()
h.PushN(7, 3, 9)

fmt.Println(h.Sorted()) // [3 7 9]
```

With no arguments it is a no-op.

### Bulk pop with `PopN`

`PopN` removes up to `n` least elements and returns them in **priority order**
— exactly as if you called `Pop` `n` times. If `n` exceeds `Len`, it drains the
heap; for `n <= 0`, or an empty or nil heap, it returns `nil`. Vacated slots
are zeroed and `Cap` is preserved:

```go
h := binheap.From([]int{7, 3, 9, 1, 5})

fmt.Println(h.PopN(3)) // [1 3 5]
fmt.Println(h.Len())   // 2
```

### Copying out with `Clone` and `Slice`

`Clone` returns an independent heap that shares the comparator; `Slice` returns
a plain slice. Both are detached from the original — they share no backing
array, so mutating one does not affect the other:

```go
a := binheap.FromMax([]int{1, 2, 3})

b := a.Clone() // independent *Heap[int]
b.Pop()        // removes 3 from b only

fmt.Println(a.Sorted()) // [3 2 1]
fmt.Println(b.Sorted()) // [2 1]
```

Both copies are **shallow**: when `T` is a pointer or contains references, the
pointed-to data is shared between the copies. The clone's capacity equals its
length. `Slice` returns `nil` for an empty heap, and `Clone` of a nil heap
returns `nil`.

### `PushPop` vs `Replace`

Both combine a push and a pop in a single sift, which is cheaper than calling
`Push` and `Pop` separately. They differ in which happens first:

| Operation | Order | When `v` is less than the top | On an empty heap | Grows the heap? |
| --- | --- | --- | --- | --- |
| `PushPop` | push, then pop | returns `v`; the heap is untouched | returns `v`; stores nothing | never |
| `Replace` | pop, then push | returns the old top; `v` is stored | stores `v`; returns `(zero, false)` | only when empty |

```go
h := binheap.From([]int{5, 10, 20})

fmt.Println(h.PushPop(1)) // 1 — would be popped first anyway
fmt.Println(h.PushPop(7)) // 5 — 7 is stored instead

old, ok := h.Replace(1) // the old top, even though 1 is smaller
fmt.Println(old, ok)    // 7 true
```

`PushPop` is the natural fit for "keep the k largest values": a min-heap of size
`k` holds the current top `k`, and `PushPop` drops the smallest of them whenever
a larger value arrives. `Replace` fits schedulers: pop the earliest deadline and
push its next occurrence.

### `Contains` and `Remove`

Both scan the heap linearly in O(n); a heap offers no faster lookup. They match
by **comparator equality** — an element matches when the comparator returns 0 —
not by identity. For a comparator that orders by a key, such as a priority
field, any element with the same key matches, and `Remove` deletes one of them,
which one being unspecified.

```go
h := binheap.From([]int{1, 2, 3, 4})

fmt.Println(h.Remove(3))  // true
fmt.Println(h.Remove(42)) // false
fmt.Println(h.Sorted())   // [1 2 4]
```

If you need to update or remove elements by key often, a heap is the wrong tool;
[`pqueue`](../pqueue) keeps element positions and does both in O(log n).

## Capacity management

The heap grows automatically, but several methods let you control the backing
array explicitly:

- **Reserve ahead** — the `WithCap` constructors preallocate at construction;
  `Grow(n)` reserves room for `n` more elements on an existing heap so
  subsequent pushes do not reallocate.
- **Reclaim memory** — `Shrink` and `Clip` shrink the capacity to the current
  length (see below); `Clear` releases the backing array entirely.
- **Reuse memory** — `Reset` empties the heap but keeps the capacity for
  refilling.

`Pop`, `PopN`, `Remove`, and `Reset` zero the slots they vacate, so the heap
never retains references to removed elements.

## `Shrink` vs `Clip`

Both reduce the capacity to the current length, but they make opposite
trade-offs between work done now and memory reclaimed now:

| Operation | Copies? | Cost | Frees unused memory | Use when |
| --- | --- | --- | --- | --- |
| `Shrink` | yes (new array) | O(n) | immediately | The heap grew large, has shrunk, and you want the memory back now. |
| `Clip` | no (reslice) | O(1) | only on the next growth (reallocation) | You want the cheap version and can wait for lazy reclaim. |

```go
h := binheap.NewWithCap[int](1024)
h.PushN(1, 2, 3)

h.Clip()
fmt.Println(h.Len(), h.Cap()) // 3 3  — O(1) reslice, old array still retained

h.Grow(2000)
h.Shrink()
fmt.Println(h.Len(), h.Cap()) // 3 3  — copied into a right-sized array, old one freed
```

Both are no-ops when `Cap` already equals `Len`.

## `Clear` vs `Reset`

Both empty the heap and drop all references to the removed elements (so they
become eligible for garbage collection), and both keep the comparator, so the
heap stays usable. They differ in what happens to the allocated backing array:

| Operation | Empties heap | Backing array | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the heap, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the heap and want to avoid reallocating. |

```go
h := binheap.NewWithCap[int](8)
h.PushN(2, 1)

h.Reset()
fmt.Println(h.Len(), h.Cap()) // 0 8  — capacity kept for reuse

h.Clear()
fmt.Println(h.Len(), h.Cap()) // 0 0  — backing array released
```

## Nil receiver and zero value

A nil `*Heap[T]` behaves as a valid **empty** heap for every read-only method
and for removal methods, so you can safely query a `nil` heap without first
calling a constructor:

```go
var h *binheap.Heap[int] // nil

h.Len()           // 0
h.IsEmpty()       // true
h.Cap()           // 0
v, ok := h.Peek() // 0, false
h.Contains(1)     // false
v, ok = h.Pop()   // 0, false (nothing to remove)
h.PopN(3)         // nil
h.PushPop(5)      // 5 (nothing to pop against)
h.Remove(1)       // false
h.Slice()         // nil
h.Sorted()        // nil
for range h.All() { /* never runs */ }
h.Clone() // nil
```

`Clone` of a nil heap returns **nil**, not a usable heap. Unlike the
comparator-free containers in this library, a heap cannot be conjured from
nothing: the comparator is part of its state, and a heap without one would
panic on the first push, far from the actual bug.

The **mutating** methods — `Push`, `PushN`, `Replace`, `Clear`, `Reset`,
`Shrink`, `Clip`, and `Grow` — need a non-nil receiver, so they **panic** on a
`nil` heap with a message such as `binheap: Push on nil receiver`.

The zero value `binheap.Heap[T]{}` has no comparator. It reads and removes like
an empty heap, but `Push`, `PushN`, and `Replace` panic with a message that
points at the missing constructor. Always create a heap with `New`, `NewMax`,
`NewFunc`, or one of their variants:

```go
var h binheap.Heap[int] // zero value: no comparator
h.Len()                 // 0
h.Push(1)               // panics: binheap: Push on zero-value Heap; ...
```

Copying a `Heap` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Heap` is **not safe for concurrent use**: it performs no internal locking.
If a heap is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `From` | build a heap from 4096 elements (copy and heapify) | 24 842 | 32 816 | 3 |
| `Peek` | one `Peek` on 4096 elements | 1.4 | 0 | 0 |
| `Contains/hit` | linear `Contains` of a present element, 4096 elements | 4 335 | 0 | 0 |
| `Contains/miss` | linear `Contains` of an absent element, 4096 elements | 8 339 | 0 | 0 |
| `All` | iterate 4096 elements with `All` | 8 746 | 0 | 0 |
| `Clone` | clone 4096 elements | 3 868 | 32 800 | 2 |
| `Slice` | copy 4096 elements | 3 620 | 32 768 | 1 |
| `Sorted` | sorted copy of 4096 elements | 220 791 | 32 768 | 1 |
| `Push` | `Push` into a heap growing from empty to 4096 elements | 15.1 | 31 | 0 |
| `PushN/small-into-large` | `PushN` of 8 elements into a heap growing from 4096 to 8192 elements (sift-up) | 76.5 | 0 | 0 |
| `PushN/into-empty` | `Reset` plus `PushN` of 4096 elements into an empty heap (rebuild) | 30 062 | 0 | 0 |
| `Pop` | one `Pop`, refilled outside the timing | 90.9 | 0 | 0 |
| `PopN` | one `PopN` of 128 elements, refilled outside the timing | 11 683 | 1 024 | 1 |
| `PushPop` | one `PushPop` on 4096 elements | 84.6 | 0 | 0 |
| `Replace` | one `Replace` on 4096 elements | 81.5 | 0 | 0 |
| `Remove` | one `Remove` of a present element, refilled outside the timing | 3 428 | 0 | 0 |
| `PushPopChurn/binheap` | one `Pop` + one `Push` with 4096 live elements | 93.7 | 0 | 0 |
| `PushPopChurn/container-heap` | the same churn with `container/heap` | 138 | 16 | 1 |

- `B/op` for `Push` is the amortized cost of growing the backing array; `allocs/op` rounds down to 0 because growth happens once per many pushes. Pre-size with `NewWithCap` or `Grow` to avoid it.
- `container/heap` boxes every value in `any` and dispatches through an interface; `binheap` does neither.

Run them with:

```sh
go test -bench=. -benchmem ./binheap/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including every constructor, `PushPop` and `Replace`, cloning, shrinking,
nil-receiver use, and a top-k stream filter — and are rendered alongside the
API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, including
a push/pop churn comparison against `container/heap`; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/binheap         # package overview
go doc github.com/Nergous/structures/binheap Heap    # the Heap type and its methods
```

The same comments render on pkg.go.dev-style godoc.
