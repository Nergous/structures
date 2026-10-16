# pqueue

A generic, heap-backed **keyed priority queue** for Go.

It is part of the `structures` data-structures library. Every key is unique and
has a priority that can be changed or removed through the key itself in
O(log n), replacing the indexed
[`container/heap`](https://pkg.go.dev/container/heap) pattern — an item type
with an `index` field, a `Swap` that maintains it, and a side map from keys to
items — with a single type, and uses Go 1.23 range-over-func iterators
([`iter.Seq2`](https://pkg.go.dev/iter#Seq2)).

## Install

```go
import "github.com/Nergous/structures/pqueue"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/pqueue"
)

func main() {
	q := pqueue.New[string, int]()
	q.Push("write docs", 3)
	q.Push("fix outage", 1)
	q.Push("review PR", 2)

	for !q.IsEmpty() {
		key, priority, _ := q.Pop()
		fmt.Println(priority, key) // 1 fix outage, 2 review PR, 3 write docs
	}
}
```

Build a priority queue from existing items in O(n). The queue copies the slice
and never aliases it; a key that appears more than once keeps the priority of
its last occurrence:

```go
q := pqueue.From([]pqueue.Item[string, int]{
	{Key: "b", Priority: 2},
	{Key: "a", Priority: 5},
	{Key: "a", Priority: 1},
})
fmt.Println(q.Sorted()) // [{a 1} {b 2}]
```

Pushing a key that is already queued changes its priority instead of adding a
duplicate, which is exactly the decrease-key step of Dijkstra's algorithm:

```go
dist := map[string]int{"A": 0}
q := pqueue.New[string, int]()
q.Push("A", 0)

for !q.IsEmpty() {
	v, d, _ := q.Pop()
	for _, e := range graph[v] {
		nd := d + e.weight
		if old, seen := dist[e.to]; !seen || nd < old {
			dist[e.to] = nd
			q.Push(e.to, nd) // inserts, or lowers the queued distance
		}
	}
}
```

Order any priority type with a comparator:

```go
q := pqueue.NewFunc[string](time.Time.Compare)
q.Push("send report", at.Add(3*time.Hour))
q.Push("backup", at.Add(time.Hour))

key, _, _ := q.Peek()
fmt.Println(key) // backup
```

## When to use `pqueue` vs `binheap`

| Need | Use |
| --- | --- |
| Change or remove a queued element by its identity | `pqueue` |
| Look up an element's current priority | `pqueue` |
| Stable order for equal priorities | `pqueue` |
| Duplicate elements, or elements that are not comparable | `binheap` |
| Maximum push/pop throughput | `binheap` |

`pqueue` keeps a map from every key to its heap position, updated on every
move inside the heap. That is what makes key lookups O(1) on average and
updates O(log n), and it is also why a push/pop cycle costs more than in
[`binheap`](../binheap) (see [Benchmarks](#benchmarks)).

## Ordering

The top of the queue is always the key with the **least** priority according to
the comparator. Throughout the docs, "least" is comparator-relative: for a
`NewMax` queue it is the largest priority.

| Constructors | Priority type | Order on top |
| --- | --- | --- |
| `New`, `NewWithCap`, `From` | `cmp.Ordered` | smallest priority (`cmp.Compare`) |
| `NewMax`, `NewMaxWithCap`, `FromMax` | `cmp.Ordered` | largest priority |
| `NewFunc`, `NewFuncWithCap`, `FromFunc` | any type | least per your comparator |

Keys are never ordered; they only need to be `comparable`. A comparator
follows the [`slices.SortFunc`](https://pkg.go.dev/slices#SortFunc) contract:
negative when `a` comes before `b`, positive when after, zero when they are
equivalent. It must be a strict weak ordering. The `Func` constructors panic on
a nil comparator.

The queue is **stable**: keys with equal priorities come out in the order they
were first pushed, for min, max, and custom queues alike. Changing a key's
priority keeps its original place among equal priorities; a key that is popped
or removed and pushed again counts as new. The `From` constructors produce the
same order as pushing the items one by one.

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[K comparable, P cmp.Ordered]() *Queue[K, P]` | Creates an empty priority queue with no preallocated capacity, smallest priority first. | O(1) |
| `NewWithCap` | `func NewWithCap[K comparable, P cmp.Ordered](n int) *Queue[K, P]` | Creates an empty priority queue with capacity for at least `n` keys, smallest priority first; a negative `n` is treated as 0. | O(n) |
| `From` | `func From[K comparable, P cmp.Ordered](items []Item[K, P]) *Queue[K, P]` | Creates a priority queue from a copy of `items`, smallest priority first; the last occurrence of a key wins. | O(n) |
| `NewMax` | `func NewMax[K comparable, P cmp.Ordered]() *Queue[K, P]` | Creates an empty priority queue with no preallocated capacity, largest priority first. | O(1) |
| `NewMaxWithCap` | `func NewMaxWithCap[K comparable, P cmp.Ordered](n int) *Queue[K, P]` | Creates an empty priority queue with capacity for at least `n` keys, largest priority first; a negative `n` is treated as 0. | O(n) |
| `FromMax` | `func FromMax[K comparable, P cmp.Ordered](items []Item[K, P]) *Queue[K, P]` | Creates a priority queue from a copy of `items`, largest priority first; the last occurrence of a key wins. | O(n) |
| `NewFunc` | `func NewFunc[K comparable, P any](compare func(a, b P) int) *Queue[K, P]` | Creates an empty priority queue ordered by `compare` with no preallocated capacity. | O(1) |
| `NewFuncWithCap` | `func NewFuncWithCap[K comparable, P any](n int, compare func(a, b P) int) *Queue[K, P]` | Creates an empty priority queue ordered by `compare` with capacity for at least `n` keys; a negative `n` is treated as 0. | O(n) |
| `FromFunc` | `func FromFunc[K comparable, P any](items []Item[K, P], compare func(a, b P) int) *Queue[K, P]` | Creates a priority queue ordered by `compare` from a copy of `items`; the last occurrence of a key wins. | O(n) |
| `Len` | `func (q *Queue[K, P]) Len() int` | Number of keys currently in the priority queue. | O(1) |
| `Cap` | `func (q *Queue[K, P]) Cap() int` | Capacity of the backing array before the next reallocation. | O(1) |
| `IsEmpty` | `func (q *Queue[K, P]) IsEmpty() bool` | Reports whether the priority queue has no keys. | O(1) |
| `Peek` | `func (q *Queue[K, P]) Peek() (K, P, bool)` | Returns the top key and its priority without removing it; `ok` is `false` if empty. | O(1) |
| `Contains` | `func (q *Queue[K, P]) Contains(key K) bool` | Reports whether `key` is queued. | O(1) avg |
| `Priority` | `func (q *Queue[K, P]) Priority(key K) (P, bool)` | Returns the current priority of `key`; `ok` is `false` if it is not queued. | O(1) avg |
| `All` | `func (q *Queue[K, P]) All() iter.Seq2[K, P]` | Iterator over keys and priorities in layout order (least first, rest unspecified), without consuming them. | O(1) per step |
| `Clone` | `func (q *Queue[K, P]) Clone() *Queue[K, P]` | Returns an independent **shallow** copy that shares the comparator; `nil` for a nil priority queue. | O(n) |
| `Slice` | `func (q *Queue[K, P]) Slice() []Item[K, P]` | Returns an independent copy of the items in layout order (same order as `All`); `nil` if empty. | O(n) |
| `Sorted` | `func (q *Queue[K, P]) Sorted() []Item[K, P]` | Returns an independent copy of the items in priority order, without consuming the priority queue; `nil` if empty. | O(n log n) |
| `Push` | `func (q *Queue[K, P]) Push(key K, priority P) bool` | Adds `key`, or changes its priority if already queued; reports whether `key` was new. | amortized O(log n) |
| `PushN` | `func (q *Queue[K, P]) PushN(items ...Item[K, P])` | Pushes `items` in order with `Push` semantics; grows the backing array at most once. | amortized O(k log(n+k)) |
| `Pop` | `func (q *Queue[K, P]) Pop() (K, P, bool)` | Removes and returns the top key and its priority; `ok` is `false` if empty. | O(log n) |
| `PopN` | `func (q *Queue[K, P]) PopN(n int) []Item[K, P]` | Removes and returns up to `n` top keys in priority order; `nil` if `n <= 0` or empty. | O(k log n) |
| `Remove` | `func (q *Queue[K, P]) Remove(key K) (P, bool)` | Removes `key` and returns its priority; `ok` is `false` if it was not queued. | O(log n) |
| `Clear` | `func (q *Queue[K, P]) Clear()` | Removes all keys **and releases** the backing array and the index (`Cap` drops to 0); keeps the comparator. | O(1) |
| `Reset` | `func (q *Queue[K, P]) Reset()` | Removes all keys but **keeps** the backing array and the index for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (q *Queue[K, P]) Shrink()` | Shrink-to-fit: **copies** into a right-sized array, frees the old one now, and rebuilds the index; no copy when `Cap == Len`. | O(n) |
| `Clip` | `func (q *Queue[K, P]) Clip()` | Reslices so `Cap == Len` **without copying**; the index is untouched. | O(1) |
| `Grow` | `func (q *Queue[K, P]) Grow(n int)` | Reserves capacity for at least `n` more keys; no-op when `n <= 0` or capacity already suffices. | O(n) |

`n` is the number of keys in the priority queue; `k` is the number of keys
pushed or popped by the call. `Item[K, P]` is a plain pair with `Key` and
`Priority` fields.

### The comma-ok contract

`Peek` and `Pop` return a key, a priority, and a boolean; `Priority` and
`Remove` return a priority and a boolean. The boolean is `true` when a key was
found and `false` when the priority queue is empty (or nil) or the key is not
queued; in that case the other results are zero values. `Push` returns `true`
when it added a new key and `false` when it changed the priority of a key that
was already queued.

```go
key, priority, ok := q.Pop()
if !ok {
	// queue was empty; key and priority are zero values
}
```

### Iteration with `All`

`All` returns an [`iter.Seq2[K, P]`](https://pkg.go.dev/iter#Seq2) — a Go 1.23
range-over-func iterator — that yields keys and priorities in the internal heap
**layout order**, the same order as `Slice`. The first pair has the least
priority; the order of the rest is **unspecified and not sorted**, which suits
order-independent work such as sums, counts, and serialization of a queue that
will be rebuilt with `From`. It does not consume the priority queue, and
breaking out of the loop early stops iteration cleanly.

`Sorted` returns an independent copy in priority order (the order repeated
`Pop` calls would produce, ties included) without consuming the priority
queue. It costs O(n log n).

```go
q := pqueue.FromMax([]pqueue.Item[string, int]{
	{Key: "a", Priority: 2},
	{Key: "b", Priority: 9},
	{Key: "c", Priority: 4},
})

fmt.Println(q.Sorted()) // [{b 9} {c 4} {a 2}]
fmt.Println(q.Len())    // 3

sum := 0
for _, priority := range q.All() { // layout order; early break is safe
	sum += priority
}
fmt.Println(sum) // 15
```

Mutating the priority queue while iterating over `All` is undefined behavior.

### Bulk push with `PushN`

`PushN` pushes a batch of items in order with the semantics of `Push`: a new
key is added, and a key that is already queued, or repeated within the batch,
has its priority changed in place, so its last occurrence wins. It reserves
room for the whole batch up front, so the backing array grows at most once. To
build a queue from scratch, `From`, `FromMax`, and `FromFunc` heapify a copy of
the items in O(n) instead:

```go
q := pqueue.New[string, int]()
q.Push("a", 5)
q.PushN(
	pqueue.Item[string, int]{Key: "b", Priority: 2},
	pqueue.Item[string, int]{Key: "a", Priority: 1}, // updates a
)

fmt.Println(q.Sorted()) // [{a 1} {b 2}]
```

With no arguments it is a no-op.

### Bulk pop with `PopN`

`PopN` removes up to `n` keys with the least priorities and returns them in
**priority order** — exactly as if you called `Pop` `n` times. If `n` exceeds
`Len`, it drains the priority queue; for `n <= 0`, or an empty or nil priority
queue, it returns `nil`. Vacated slots are zeroed and `Cap` is preserved:

```go
q := pqueue.From([]pqueue.Item[string, int]{
	{Key: "a", Priority: 3},
	{Key: "b", Priority: 1},
	{Key: "c", Priority: 2},
})

fmt.Println(q.PopN(2)) // [{b 1} {c 2}]
fmt.Println(q.Len())   // 1
```

### Copying out with `Clone` and `Slice`

`Clone` returns an independent priority queue that shares the comparator;
`Slice` returns a plain slice of `Item`. Both are detached from the original —
they share no backing array or index, so mutating one does not affect the
other:

```go
a := pqueue.FromMax([]pqueue.Item[string, int]{
	{Key: "x", Priority: 1},
	{Key: "y", Priority: 2},
})

b := a.Clone() // independent *Queue[string, int]
b.Pop()        // removes y from b only

fmt.Println(a.Sorted()) // [{y 2} {x 1}]
fmt.Println(b.Sorted()) // [{x 1}]
```

Both copies are **shallow**: when `K` or `P` is a pointer or contains
references, the pointed-to data is shared between the copies. The clone's
capacity equals its length. `Slice` returns `nil` for an empty priority queue,
and `Clone` of a nil priority queue returns `nil`.

## Capacity management

The priority queue grows automatically, but several methods let you control the
backing storage explicitly:

- **Reserve ahead** — the `WithCap` constructors preallocate the backing array
  and the index at construction; `Grow(n)` reserves array room for `n` more
  keys on an existing priority queue so subsequent pushes do not reallocate it.
- **Reclaim memory** — `Shrink` and `Clip` shrink the array capacity to the
  current length (see below); `Clear` releases both the array and the index
  entirely.
- **Reuse memory** — `Reset` empties the priority queue but keeps the array and
  the index buckets for refilling.

`Cap` is the capacity of the backing array. It does not include the index,
whose size Go does not expose; the index grows on demand and is not
preallocated by `Grow`. Go maps never shrink in place, so `Shrink` (or `Clear`)
is the only way to give a large index's memory back.

`Pop`, `PopN`, `Remove`, and `Reset` zero the slots they vacate, so the
priority queue never retains references to removed keys or priorities.

## `Shrink` vs `Clip`

Both reduce the array capacity to the current length, but they make opposite
trade-offs between work done now and memory reclaimed now:

| Operation | Copies? | Cost | Frees unused memory | Index map | Use when |
| --- | --- | --- | --- | --- | --- |
| `Shrink` | yes (new array) | O(n) | immediately | rebuilt (reclaimed) | The priority queue grew large, has shrunk, and you want the memory back now. |
| `Clip` | no (reslice) | O(1) | only on the next growth (reallocation) | left as is | You want the cheap version and can wait for lazy reclaim. |

```go
q := pqueue.NewWithCap[int, int](1024)
q.PushN(pqueue.Item[int, int]{Key: 1, Priority: 1})

q.Clip()
fmt.Println(q.Len(), q.Cap()) // 1 1  — O(1) reslice, old array still retained

q.Grow(2000)
q.Shrink()
fmt.Println(q.Len(), q.Cap()) // 1 1  — copied into a right-sized array, old one freed
```

Both are no-ops on the backing array when `Cap` already equals `Len`; `Shrink`
still rebuilds the index.

## `Clear` vs `Reset`

Both empty the priority queue and drop all references to the removed keys and
priorities (so they become eligible for garbage collection), and both keep the
comparator, so the priority queue stays usable. They differ in what happens to
the backing array and the index:

| Operation | Empties queue | Backing array and index | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the priority queue, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the priority queue and want to avoid reallocating. |

```go
q := pqueue.NewWithCap[string, int](8)
q.PushN(
	pqueue.Item[string, int]{Key: "a", Priority: 2},
	pqueue.Item[string, int]{Key: "b", Priority: 1},
)

q.Reset()
fmt.Println(q.Len(), q.Cap()) // 0 8  — capacity kept for reuse

q.Clear()
fmt.Println(q.Len(), q.Cap()) // 0 0  — backing array and index released
```

Both also restart the push sequence that breaks ties between equal priorities.

## Nil receiver and zero value

A nil `*Queue[K, P]` behaves as a valid **empty** priority queue for every
read-only method and for removal methods, so you can safely query a `nil`
priority queue without first calling a constructor:

```go
var q *pqueue.Queue[string, int] // nil

q.Len()              // 0
q.IsEmpty()          // true
q.Cap()              // 0
k, p, ok := q.Peek() // "", 0, false
q.Contains("a")      // false
q.Priority("a")      // 0, false
k, p, ok = q.Pop()   // "", 0, false (nothing to remove)
q.PopN(3)            // nil
q.Remove("a")        // 0, false
q.Slice()            // nil
q.Sorted()           // nil
for range q.All() { /* never runs */ }
q.Clone() // nil
```

`Clone` of a nil priority queue returns **nil**, not a usable priority queue.
Unlike the comparator-free containers in this library, a priority queue cannot
be conjured from nothing: the comparator is part of its state, and a queue
without one would panic on the first push, far from the actual bug.

The **mutating** methods — `Push`, `PushN`, `Clear`, `Reset`, `Shrink`, `Clip`,
and `Grow` — need a non-nil receiver, so they **panic** on a `nil` priority
queue with a message such as `pqueue: Push on nil receiver`.

The zero value `pqueue.Queue[K, P]{}` has no comparator. It reads and removes
like an empty priority queue, but `Push` and `PushN` panic with a message that
points at the missing constructor. Always create a queue with `New`, `NewMax`,
`NewFunc`, or one of their variants:

```go
var q pqueue.Queue[string, int] // zero value: no comparator
q.Len()                         // 0
q.Push("a", 1)                  // panics: pqueue: Push on zero-value Queue; ...
```

Copying a `Queue` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Queue` is **not safe for concurrent use**: it performs no internal locking.
If a priority queue is shared across goroutines, the caller must provide its
own synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `From` | build a priority queue from 4096 keys (copy, heapify, and index) | 241 280 | 246 257 | 23 |
| `Peek` | one `Peek` on 4096 keys | 1.4 | 0 | 0 |
| `Contains/hit` | `Contains` of a queued key, 4096 keys | 8.6 | 0 | 0 |
| `Contains/miss` | `Contains` of an absent key, 4096 keys | 7.9 | 0 | 0 |
| `All` | iterate 4096 keys with `All` | 8 774 | 0 | 0 |
| `Clone` | clone 4096 keys (array and index) | 99 787 | 246 216 | 21 |
| `Slice` | copy 4096 keys | 9 130 | 65 536 | 1 |
| `Sorted` | sorted copy of 4096 keys | 307 728 | 163 840 | 2 |
| `Push` | `Push` of new keys into a priority queue growing from empty to 4096 keys | 131 | 145 | 0 |
| `PushExisting` | `Push` of a queued key (priority changed), 4096 keys | 84.4 | 0 | 0 |
| `PushN/small-into-large` | `PushN` of 8 new keys into a priority queue growing from 4096 to 8192 keys | 565 | 0 | 0 |
| `PushN/into-empty` | `Reset` plus `PushN` of 4096 keys into an empty priority queue | 276 072 | 0 | 0 |
| `Pop` | one `Pop`, refilled outside the timing | 304 | 0 | 0 |
| `PopN` | one `PopN` of 128 keys, refilled outside the timing | 39 243 | 2 048 | 1 |
| `Remove` | one `Remove` of a queued key, refilled outside the timing | 110 | 0 | 0 |
| `PushPopChurn/pqueue` | one `Pop` + one `Push` of a new key with 4096 live keys | 404 | 0 | 0 |
| `PushPopChurn/binheap` | the same churn with `binheap` | 89.3 | 0 | 0 |
| `PushPopChurn/container-heap` | the same churn with an indexed `container/heap` | 238 | 24 | 1 |
| `PushRemoveChurn` | one `Remove` + one `Push` of the same key with 4096 live keys | 146 | 0 | 0 |
| `DijkstraChurn/pqueue` | one `Pop` + up to 4 decrease-key updates + one `Push` with 4096 live keys | 535 | 0 | 0 |
| `DijkstraChurn/container-heap` | the same mix with an indexed `container/heap` | 343 | 24 | 1 |

- `B/op` for `Push` is the amortized cost of growing the backing array and the index; `allocs/op` rounds down to 0 because growth happens once per many pushes. Pre-size with `NewWithCap` (array and index) or `Grow` (array only) to avoid it.
- `PushPopChurn/binheap` shows the price of keeping the key index up to date: use `binheap` when elements never need to be found, changed, or removed by identity.
- The indexed `container/heap` version allocates one item per push but updates a pointer field instead of a map entry on every move.

Run them with:

```sh
go test -bench=. -benchmem ./pqueue/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including every constructor, cloning, shrinking, nil-receiver use, and
Dijkstra's shortest paths — and are rendered alongside the API on the godoc
page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, comparing
`pqueue` with `binheap` on a plain push/pop cycle, and with the textbook indexed
`container/heap` queue on a push/pop cycle and a Dijkstra-style mix of pops and
decrease-key updates; their sources live in [`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/pqueue          # package overview
go doc github.com/Nergous/structures/pqueue Queue    # the Queue type and its methods
```

The same comments render on pkg.go.dev-style godoc.
