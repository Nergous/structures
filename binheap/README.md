# binheap

A generic, slice-backed **binary heap** for Go — a type-safe replacement for
`container/heap` without the interface boilerplate.

It is part of the `structures` data-structures library. The heap is min-first by
default, supports max-heaps and custom comparators, never boxes values in
`any`, and uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq)) for non-consuming iteration.

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

## Ordering

The top of the heap is always its **least** element according to the
comparator. Throughout the docs, "least" is comparator-relative: for a max-heap
it is the largest value.

| Constructors                              | Element type      | Order on top                  |
| ----------------------------------------- | ----------------- | ----------------------------- |
| `New`, `NewWithCap`, `From`               | `cmp.Ordered`     | smallest value (`cmp.Compare`) |
| `NewMax`, `NewMaxWithCap`, `FromMax`      | `cmp.Ordered`     | largest value                 |
| `NewFunc`, `NewFuncWithCap`, `FromFunc`   | any type          | least per your comparator     |

A comparator follows the [`slices.SortFunc`](https://pkg.go.dev/slices#SortFunc)
contract: negative when `a` comes before `b`, positive when after, zero when they
are equivalent. It must be a strict weak ordering. The `Func` constructors panic
on a nil comparator.

The heap is **not stable**: elements that compare equal come out in unspecified
order.

## API reference

| Method / Function | Signature                                                   | Description                                                                                                   | Complexity                  |
| ----------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | --------------------------- |
| `New`             | `func New[T cmp.Ordered]() *Heap[T]`                         | Creates an empty min-heap.                                                                                    | O(1)                        |
| `NewWithCap`      | `func NewWithCap[T cmp.Ordered](n int) *Heap[T]`             | Creates an empty min-heap with capacity for at least `n` elements.                                           | O(n)                        |
| `From`            | `func From[T cmp.Ordered](s []T) *Heap[T]`                   | Creates a min-heap from a copy of `s`.                                                                       | O(n)                        |
| `NewMax`          | `func NewMax[T cmp.Ordered]() *Heap[T]`                      | Creates an empty max-heap.                                                                                    | O(1)                        |
| `NewMaxWithCap`   | `func NewMaxWithCap[T cmp.Ordered](n int) *Heap[T]`          | Creates an empty max-heap with capacity for at least `n` elements.                                           | O(n)                        |
| `FromMax`         | `func FromMax[T cmp.Ordered](s []T) *Heap[T]`                | Creates a max-heap from a copy of `s`.                                                                       | O(n)                        |
| `NewFunc`         | `func NewFunc[T any](compare func(a, b T) int) *Heap[T]`     | Creates an empty heap ordered by `compare`.                                                                  | O(1)                        |
| `NewFuncWithCap`  | `func NewFuncWithCap[T any](n int, compare func(a, b T) int) *Heap[T]` | Same, with capacity for at least `n` elements.                                                     | O(n)                        |
| `FromFunc`        | `func FromFunc[T any](s []T, compare func(a, b T) int) *Heap[T]` | Creates a heap ordered by `compare` from a copy of `s`.                                                 | O(n)                        |
| `Len`             | `func (h *Heap[T]) Len() int`                                | Number of elements in the heap.                                                                               | O(1)                        |
| `Cap`             | `func (h *Heap[T]) Cap() int`                                | Capacity of the backing array before the next reallocation.                                                   | O(1)                        |
| `IsEmpty`         | `func (h *Heap[T]) IsEmpty() bool`                           | Reports whether the heap has no elements.                                                                     | O(1)                        |
| `Peek`            | `func (h *Heap[T]) Peek() (T, bool)`                         | Returns the least element without removing it; `ok` is `false` if empty.                                      | O(1)                        |
| `Push`            | `func (h *Heap[T]) Push(v T)`                                | Adds `v`.                                                                                                     | amortized O(log n)          |
| `PushN`           | `func (h *Heap[T]) PushN(vs ...T)`                           | Adds `vs`; grows the backing array at most once and picks sift-up or rebuild, whichever is cheaper.          | min(O(k log(n+k)), O(n+k))  |
| `Pop`             | `func (h *Heap[T]) Pop() (T, bool)`                          | Removes and returns the least element; `ok` is `false` if empty.                                              | O(log n)                    |
| `PopN`            | `func (h *Heap[T]) PopN(n int) []T`                          | Removes up to `n` least elements and returns them in priority order; `nil` when `n <= 0` or empty.           | O(k log n)                  |
| `PushPop`         | `func (h *Heap[T]) PushPop(v T) T`                           | Push then pop in one sift; returns `v` untouched when it would be popped first anyway.                        | O(log n)                    |
| `Replace`         | `func (h *Heap[T]) Replace(v T) (T, bool)`                   | Pop then push in one sift; always stores `v`; `ok` reports whether an element was popped.                    | O(log n)                    |
| `Contains`        | `func (h *Heap[T]) Contains(v T) bool`                       | Reports whether an element compares equal to `v`.                                                             | O(n)                        |
| `Remove`          | `func (h *Heap[T]) Remove(v T) bool`                         | Removes one element that compares equal to `v`; reports whether one was found.                               | O(n)                        |
| `All`             | `func (h *Heap[T]) All() iter.Seq[T]`                        | Iterator over elements in layout order (least first, rest unspecified) without consuming them.               | O(1) per step               |
| `Slice`           | `func (h *Heap[T]) Slice() []T`                              | Copy of the elements in layout order, same as `All`; `nil` if empty.                                          | O(n)                        |
| `Sorted`          | `func (h *Heap[T]) Sorted() []T`                             | Copy of the elements in priority order, without consuming the heap; `nil` if empty.                          | O(n log n)                  |
| `Clone`           | `func (h *Heap[T]) Clone() *Heap[T]`                         | Independent shallow copy sharing the comparator; `nil` for a nil heap.                                        | O(n)                        |
| `Clear`           | `func (h *Heap[T]) Clear()`                                  | Removes all elements **and releases** the backing array (`Cap` drops to 0); keeps the comparator.             | O(1)                        |
| `Reset`           | `func (h *Heap[T]) Reset()`                                  | Removes all elements but **keeps** the backing array for reuse (`Cap` preserved).                            | O(n)                        |
| `Grow`            | `func (h *Heap[T]) Grow(n int)`                              | Reserves capacity for at least `n` more elements; no-op when `n <= 0` or capacity already suffices.           | O(n)                        |
| `Shrink`          | `func (h *Heap[T]) Shrink()`                                 | Shrink-to-fit: **copies** into a right-sized array and frees the old one now; no-op when `Cap == Len`.       | O(n)                        |
| `Clip`            | `func (h *Heap[T]) Clip()`                                   | Reslices so `Cap == Len` **without copying**; unused memory is reclaimed only on the next growth.            | O(1)                        |

### The `(T, bool)` contract

`Peek` and `Pop` return two values. The boolean reports whether the operation
found an element: it is `true` on success and `false` when the heap is empty.
When it is `false`, the first return is the zero value of `T`.

```go
v, ok := h.Pop()
if !ok {
	// heap was empty; v is the zero value
}
```

`Replace` uses the same shape, but its boolean reports whether an element was
**popped**: on an empty heap it stores `v` and returns `(zero, false)`.

### `PushPop` vs `Replace`

Both combine a push and a pop in a single sift, which is cheaper than calling
`Push` and `Pop` separately. They differ in which happens first:

| Operation | Order         | When `v` is less than the top          | On an empty heap          | Grows the heap? |
| --------- | ------------- | -------------------------------------- | ------------------------- | --------------- |
| `PushPop` | push, then pop | returns `v`; the heap is untouched     | returns `v`; stores nothing | never         |
| `Replace` | pop, then push | returns the old top; `v` is stored     | stores `v`; returns `(zero, false)` | only when empty |

```go
h := binheap.From([]int{5, 10, 20})

fmt.Println(h.PushPop(1)) // 1 — would be popped first anyway
fmt.Println(h.PushPop(7)) // 5 — 7 is stored instead

old, ok := h.Replace(1)   // 7 true — the old top, even though 1 is smaller
fmt.Println(old, ok)
```

`PushPop` is the natural fit for "keep the k largest values": a min-heap of size
`k` holds the current top `k`, and `PushPop` drops the smallest of them whenever
a larger value arrives. `Replace` fits schedulers: pop the earliest deadline and
push its next occurrence.

### Bulk operations with `PushN` and `PopN`

`PushN` adds a batch and grows the backing array at most once. It compares the
worst-case cost of sifting each new element up against rebuilding the whole
heap and picks the cheaper one: small batches into a large heap are sifted in
O(k log(n+k)), and batches comparable to the heap size are rebuilt in O(n+k).
With no arguments it is a no-op.

`PopN` removes up to `n` least elements and returns them in priority order. If
`n` exceeds the length, it drains the heap. If `n <= 0`, or the heap is empty or
nil, it returns `nil`.

```go
h := binheap.New[int]()
h.PushN(7, 3, 9, 1, 5)

fmt.Println(h.PopN(3)) // [1 3 5]
fmt.Println(h.Len())   // 2
```

### Iteration order: `All`, `Slice`, and `Sorted`

`All` and `Slice` expose the internal heap layout. The first element is the
least one; the order of the rest is **unspecified and not sorted**. They suit
order-independent work such as sums, counts, and serialization of a heap that
will be rebuilt with `From`.

`Sorted` returns an independent copy in priority order — the order repeated `Pop`
calls would produce — without consuming the heap. It costs O(n log n).

```go
h := binheap.FromMax([]int{2, 9, 4, 7})

fmt.Println(h.Sorted()) // [9 7 4 2]
fmt.Println(h.Len())    // 4 — unchanged

sum := 0
for v := range h.All() { // layout order; early break is safe
	sum += v
}
```

Mutating the heap while iterating over `All` is undefined behavior.

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
an indexed heap keeps element positions and does both in O(log n).

### Copying out with `Clone` and `Slice`

`Clone` returns an independent heap that shares the comparator; `Slice` returns
a plain slice. Both are detached from the original: they share no backing array,
so mutating one does not affect the other. Both copies are **shallow**: when `T`
is a pointer or contains references, the pointed-to data is shared.

```go
a := binheap.FromMax([]int{1, 2, 3})

b := a.Clone()
b.Pop() // does not touch a

fmt.Println(a.Sorted(), b.Sorted()) // [3 2 1] [2 1]
```

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

| Operation | Copies?         | Cost | Frees unused memory                    | Use when                                                          |
| --------- | --------------- | ---- | -------------------------------------- | ----------------------------------------------------------------- |
| `Shrink`  | yes (new array) | O(n) | immediately                            | The heap grew large, has shrunk, and you want the memory back now. |
| `Clip`    | no (reslice)    | O(1) | only on the next growth (reallocation) | You want the cheap version and can wait for lazy reclaim.         |

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

Both empty the heap and drop all references to the removed elements, and both
keep the comparator, so the heap stays usable. They differ in what happens to
the backing array:

| Operation | Empties heap | Backing array              | `Cap` afterwards | Use when                                                 |
| --------- | ------------ | -------------------------- | ---------------- | -------------------------------------------------------- |
| `Clear`   | yes          | released (eligible for GC) | `0`              | You are done with the heap, or want to free its memory.  |
| `Reset`   | yes          | retained for reuse         | unchanged        | You will refill the heap and want to avoid reallocating. |

```go
h := binheap.NewWithCap[int](8)
h.PushN(2, 1)

h.Reset()
fmt.Println(h.Len(), h.Cap()) // 0 8  — capacity kept for reuse

h.Clear()
fmt.Println(h.Len(), h.Cap()) // 0 0  — backing array released
```

## Nil receiver and zero value

The nil value of `*Heap[T]` behaves as a valid **empty** heap for every
read-only method and for removals, so you can safely query a `nil` heap without
first calling a constructor:

```go
var h *binheap.Heap[int] // nil

h.Len()              // 0
h.IsEmpty()          // true
h.Cap()              // 0
v, ok := h.Peek()    // 0, false
v, ok = h.Pop()      // 0, false (nothing to remove)
h.PopN(3)            // nil
h.PushPop(5)         // 5 (nothing to pop against)
h.Contains(1)        // false
h.Remove(1)          // false
h.Slice()            // nil
h.Sorted()           // nil
for range h.All() { /* never runs */ }
```

`Clone` on a nil receiver returns **nil**, not a usable heap. Unlike the
comparator-free containers in this library, a heap cannot be conjured from
nothing: the comparator is part of its state, and a heap without one would
panic on the first push, far from the actual bug.

The **mutating** methods — `Push`, `PushN`, `Replace`, `Grow`, `Clear`, `Reset`,
`Shrink`, and `Clip` — need a non-nil receiver, so they **panic** on a `nil`
heap with a message such as `binheap: Push on nil receiver`.

The zero value `binheap.Heap[T]{}` has no comparator. It reads and removes like
an empty heap, but `Push`, `PushN`, and `Replace` panic with a message that
points at the missing constructor. Always create a heap with `New`, `NewMax`,
`NewFunc`, or one of their variants.

## Concurrency

A `Heap` is **not safe for concurrent use**: it performs no internal locking.
If a heap is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including a top-k stream filter and a priority task queue — and are rendered
alongside the API on the godoc page.

Benchmarks backing the complexity and allocation claims live in
[`bench_test.go`](./bench_test.go), including a Push+Pop comparison against
`container/heap`. Run them with:

```sh
go test -bench=. -benchmem ./binheap/...
```

View the documentation locally:

```sh
go doc github.com/Nergous/structures/binheap         # package overview
go doc github.com/Nergous/structures/binheap Heap    # the Heap type and its methods
```

The same comments render on pkg.go.dev-style godoc.
