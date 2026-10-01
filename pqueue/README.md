# pqueue

A generic, keyed **priority queue** for Go. Every key is unique and has a
priority; the queue hands out the key with the least priority first and lets
you change or remove any key's priority by the key itself in O(log n).

It is part of the `structures` data-structures library. It replaces the
indexed `container/heap` pattern — an item type with an `index` field, a
`Swap` that maintains it, and a side map from keys to items — with a
single type. Keys only need to be `comparable`, priorities are ordered by
`cmp.Compare` or your own comparator, and iteration uses Go 1.23
range-over-func iterators ([`iter.Seq2`](https://pkg.go.dev/iter#Seq2)).

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
		task, priority, _ := q.Pop()
		fmt.Println(priority, task) // 1 fix outage, 2 review PR, 3 write docs
	}
}
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
q := pqueue.NewFunc[string](func(a, b time.Time) int { return a.Compare(b) })
q.Push("send report", at.Add(3*time.Hour))
q.Push("backup", at.Add(time.Hour))

job, _, _ := q.Peek()
fmt.Println(job) // backup
```

## When to use `pqueue` vs `binheap`

| Need                                                  | Use       |
| ----------------------------------------------------- | --------- |
| Change or remove a queued element by its identity     | `pqueue`  |
| Look up an element's current priority                 | `pqueue`  |
| Stable order for equal priorities                     | `pqueue`  |
| Duplicate elements, or elements that are not comparable | `binheap` |
| Maximum push/pop throughput                           | `binheap` |

`pqueue` keeps a map from every key to its heap position, updated on every
move inside the heap. That is what makes key lookups O(1) and updates
O(log n), and it is also why a push/pop cycle costs several times more than in
`binheap` (see [benchmarks](#benchmarks)).

## Ordering

The top of the queue is always the key with the **least** priority according to
the comparator. Throughout the docs, "least" is comparator-relative: for a
`NewMax` queue it is the largest priority.

| Constructors                   | Priority type | Order on top                      |
| ------------------------------ | ------------- | --------------------------------- |
| `New`, `NewWithCap`            | `cmp.Ordered` | smallest priority (`cmp.Compare`) |
| `NewMax`, `NewMaxWithCap`      | `cmp.Ordered` | largest priority                  |
| `NewFunc`, `NewFuncWithCap`    | any type      | least per your comparator         |

Keys are never ordered; they only need to be `comparable`. A comparator
follows the [`slices.SortFunc`](https://pkg.go.dev/slices#SortFunc)
contract and must be a strict weak ordering. The `Func` constructors panic on
a nil comparator.

The queue is **stable**: keys with equal priorities come out in the order they
were first pushed, for min, max, and custom queues alike. Changing a key's
priority keeps its original place among equal priorities; a key that is popped
or removed and pushed again counts as new.

## API reference

| Method / Function | Signature                                                                  | Description                                                                                   | Complexity         |
| ----------------- | -------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ------------------ |
| `New`             | `func New[K comparable, P cmp.Ordered]() *PriorityQueue[K, P]`                 | Creates an empty queue, smallest priority first.                                              | O(1)               |
| `NewWithCap`      | `func NewWithCap[K comparable, P cmp.Ordered](n int) *PriorityQueue[K, P]`     | Same, with room for at least `n` keys.                                                     | O(n)               |
| `NewMax`          | `func NewMax[K comparable, P cmp.Ordered]() *PriorityQueue[K, P]`              | Creates an empty queue, largest priority first.                                               | O(1)               |
| `NewMaxWithCap`   | `func NewMaxWithCap[K comparable, P cmp.Ordered](n int) *PriorityQueue[K, P]`  | Same, with room for at least `n` keys.                                                     | O(n)               |
| `NewFunc`         | `func NewFunc[K comparable, P any](compare func(a, b P) int) *PriorityQueue[K, P]` | Creates an empty queue ordered by `compare` on priorities.                               | O(1)               |
| `NewFuncWithCap`  | `func NewFuncWithCap[K comparable, P any](n int, compare func(a, b P) int) *PriorityQueue[K, P]` | Same, with room for at least `n` keys.                                    | O(n)               |
| `Len`             | `func (q *PriorityQueue[K, P]) Len() int`                                       | Number of keys in the queue.                                                                  | O(1)               |
| `Cap`             | `func (q *PriorityQueue[K, P]) Cap() int`                                       | Capacity of the backing array before the next reallocation.                                   | O(1)               |
| `IsEmpty`         | `func (q *PriorityQueue[K, P]) IsEmpty() bool`                                  | Reports whether the queue has no keys.                                                        | O(1)               |
| `Peek`            | `func (q *PriorityQueue[K, P]) Peek() (K, P, bool)`                             | Returns the top key and its priority without removing it.                                     | O(1)               |
| `Priority`        | `func (q *PriorityQueue[K, P]) Priority(key K) (P, bool)`                       | Returns the current priority of `key`.                                                     | O(1) avg           |
| `Contains`        | `func (q *PriorityQueue[K, P]) Contains(key K) bool`                            | Reports whether `key` is queued.                                                           | O(1) avg           |
| `Push`            | `func (q *PriorityQueue[K, P]) Push(key K, priority P) bool`                    | Adds `key`, or changes its priority if already queued; reports whether it was new.          | amortized O(log n) |
| `Pop`             | `func (q *PriorityQueue[K, P]) Pop() (K, P, bool)`                              | Removes and returns the top key and its priority.                                             | O(log n)           |
| `PopN`            | `func (q *PriorityQueue[K, P]) PopN(n int) []Item[K, P]`                        | Removes up to `n` top keys in priority order; `nil` when `n <= 0` or empty.                  | O(k log n)         |
| `Remove`          | `func (q *PriorityQueue[K, P]) Remove(key K) (P, bool)`                         | Removes `key` and returns its priority.                                                    | O(log n)           |
| `All`             | `func (q *PriorityQueue[K, P]) All() iter.Seq2[K, P]`                           | Iterator over keys and priorities in layout order, without consuming them.                   | O(1) per step      |
| `Slice`           | `func (q *PriorityQueue[K, P]) Slice() []Item[K, P]`                            | Copy in layout order, same as `All`; `nil` if empty.                                           | O(n)               |
| `Sorted`          | `func (q *PriorityQueue[K, P]) Sorted() []Item[K, P]`                           | Copy in priority order, without consuming the queue; `nil` if empty.                       | O(n log n)         |
| `Clone`           | `func (q *PriorityQueue[K, P]) Clone() *PriorityQueue[K, P]`                    | Independent shallow copy sharing the comparator; `nil` for a nil queue.                    | O(n)               |
| `Clear`           | `func (q *PriorityQueue[K, P]) Clear()`                                         | Removes all keys **and releases** storage (`Cap` drops to 0).                                  | O(1)               |
| `Reset`           | `func (q *PriorityQueue[K, P]) Reset()`                                         | Removes all keys but **keeps** storage for reuse.                                             | O(n)               |
| `Grow`            | `func (q *PriorityQueue[K, P]) Grow(n int)`                                     | Reserves backing-array capacity for at least `n` more keys.                                | O(n)               |
| `Shrink`          | `func (q *PriorityQueue[K, P]) Shrink()`                                        | Copies into right-sized storage and rebuilds the key index, freeing memory now.               | O(n)               |
| `Clip`            | `func (q *PriorityQueue[K, P]) Clip()`                                          | Reslices so `Cap == Len` **without copying**; the key index is untouched.                   | O(1)               |

`Item[K, P]` is a plain pair with `Key` and `Priority` fields.

### The comma-ok contract

`Peek` and `Pop` return `(key, priority, ok)`; `Priority` and `Remove`
return `(priority, ok)`. When `ok` is `false` — the queue is empty or the key
is missing — the other results are zero values.

`Push` returns `true` when it added a new key and `false` when it changed the
priority of a key that was already queued.

### Iteration order: `All`, `Slice`, and `Sorted`

`All` and `Slice` expose the internal heap layout. The first pair has the
least priority; the order of the rest is **unspecified and not sorted**.
`Sorted` returns an independent copy in the order repeated `Pop` calls would
produce, ties included, without consuming the queue.

```go
for task, priority := range q.All() { // layout order; early break is safe
	fmt.Println(task, priority)
}
```

Mutating the queue while iterating over `All` is undefined behavior.

## Capacity management

A queue stores its heap in a slice and its key index in a map. The
`WithCap` constructors size both. After construction:

- **Reserve ahead** — `Grow(n)` reserves backing-array room for `n` more keys;
  the map grows on demand.
- **Reclaim memory** — `Shrink` copies the heap into a right-sized array and
  rebuilds the map. Go maps never shrink in place, so `Shrink` (or `Clear`) is
  the way to give a large index's memory back. `Clip` only reslices the array
  in O(1) and leaves the map as is.
- **Reuse memory** — `Reset` empties the queue but keeps both the array and the
  map's buckets for refilling. `Clear` releases both.

`Pop`, `PopN`, `Remove`, and `Reset` zero the slots they vacate, so the
queue never retains references to removed keys or priorities.

## Nil receiver and zero value

The nil value of `*PriorityQueue[K, P]` behaves as a valid **empty** queue for
every read-only method and for removals:

```go
var q *pqueue.PriorityQueue[string, int] // nil

q.Len()                 // 0
q.IsEmpty()             // true
k, p, ok := q.Peek()    // "", 0, false
k, p, ok = q.Pop()      // "", 0, false
q.PopN(3)               // nil
q.Remove("a")           // 0, false
q.Priority("a")         // 0, false
q.Contains("a")         // false
q.Slice()               // nil
q.Sorted()              // nil
for range q.All() { /* never runs */ }
```

`Clone` on a nil receiver returns **nil**: the comparator is part of a queue's
state and cannot be recovered. The **mutating** methods — `Push`, `Grow`,
`Clear`, `Reset`, `Shrink`, and `Clip` — **panic** on a nil queue with a
message such as `pqueue: Push on nil receiver`.

The zero value `pqueue.PriorityQueue[K, P]{}` has no comparator. It reads and
removes like an empty queue, but `Push` panics with a message that points at
the missing constructor. Always create a queue with `New`, `NewMax`, `NewFunc`,
or one of their variants.

## Concurrency

A `PriorityQueue` is **not safe for concurrent use**: it performs no internal
locking. If a queue is shared across goroutines, the caller must provide its
own synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-01. Each value is the median of 5 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 5`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --------- | ---------------- | ----: | ---: | --------: |
| `Push` | push new keys, up to 4096 live | 62.6 | 0 | 0 |
| `PushUpdate` | `Push` of a queued key (change priority), 4096 keys | 31.9 | 0 | 0 |
| `PushPopCycle/pqueue` | one `Pop` + one `Push`, 4096 keys | 386 | 0 | 0 |
| `PushPopCycle/binheap` | the same cycle with `binheap` | 85.5 | 0 | 0 |
| `PushPopCycle/container-heap` | the same cycle with an indexed `container/heap` | 223 | 24 | 1 |
| `Remove` | `Remove` a key and push it back, 4096 keys | 131 | 0 | 0 |
| `DijkstraMix/pqueue` | one `Pop` + up to 4 decrease-key updates + one `Push` | 514 | 0 | 0 |
| `DijkstraMix/container-heap` | the same mix with an indexed `container/heap` | 332 | 24 | 1 |

- No operation allocates per call. The indexed `container/heap` version allocates one item per push but updates a pointer field instead of a map entry on every move, so it is faster in raw throughput.
- `pqueue` costs several times more than `binheap` on a plain push/pop cycle: that is the price of keeping a key-to-position map up to date. Use `binheap` when elements never need to be found, changed, or removed by identity.

Run them with:

```sh
go test -bench=. -benchmem ./pqueue/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including Dijkstra's shortest paths and deadline scheduling — and are rendered
alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above. They compare
`pqueue` with `binheap` on a plain push/pop cycle, and with the textbook indexed
`container/heap` queue on a push/pop cycle and a Dijkstra-style mix of pops and
decrease-key updates; their sources live in [`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/pqueue                # package overview
go doc github.com/Nergous/structures/pqueue PriorityQueue  # the type and its methods
```
