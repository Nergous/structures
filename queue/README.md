# queue

A generic, slice-backed ring-buffer **FIFO (first-in, first-out) queue** for Go.

It is part of the `structures` data-structures library. It reuses popped slots
for later pushes instead of sliding or reallocating its backing buffer, and
uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/queue"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/queue"
)

func main() {
	q := queue.New[int]()
	q.Push(1)
	q.Push(2)
	q.Push(3)

	for !q.IsEmpty() {
		v, _ := q.Pop()
		fmt.Println(v) // 1, 2, 3
	}
}
```

Preallocate when the maximum size is known up front to avoid reallocations
while pushing:

```go
q := queue.NewWithCap[string](3)
q.Push("a")
q.Push("b")
q.Push("c")
fmt.Println(q.Len(), q.Cap()) // 3 3
```

## Ordering

A queue removes elements in the same order they were added: `Push` and `PushN`
append to the back, while `Peek`, `Pop`, and `PopN` operate on the front (the
oldest element still present). `All` and `Slice` report the elements from
front to back, the order in which `Pop` would return them.

```go
q := queue.New[string]()
q.Push("first")
q.Push("last")

v, _ := q.Pop()
fmt.Println(v) // first
```

For pushes and pops at both ends, use [`deque`](../deque).

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[T any]() *Queue[T]` | Creates an empty queue with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[T any](n int) *Queue[T]` | Creates an empty queue with capacity for at least `n` elements; a negative `n` is treated as 0. | O(n) |
| `Len` | `func (q *Queue[T]) Len() int` | Number of elements currently in the queue. | O(1) |
| `Cap` | `func (q *Queue[T]) Cap() int` | Capacity of the backing buffer before the next reallocation. | O(1) |
| `IsEmpty` | `func (q *Queue[T]) IsEmpty() bool` | Reports whether the queue has no elements. | O(1) |
| `Peek` | `func (q *Queue[T]) Peek() (T, bool)` | Returns the front element (the oldest) without removing it; `ok` is `false` if empty. | O(1) |
| `All` | `func (q *Queue[T]) All() iter.Seq[T]` | Iterator over elements from front to back, without consuming them. | O(1) per step |
| `Clone` | `func (q *Queue[T]) Clone() *Queue[T]` | Returns an independent **shallow** copy; the two queues share no backing buffer. | O(n) |
| `Slice` | `func (q *Queue[T]) Slice() []T` | Returns an independent copy of the elements from front to back (same order as `All`); `nil` if empty. | O(n) |
| `Push` | `func (q *Queue[T]) Push(v T)` | Adds `v` to the back of the queue. | amortized O(1) |
| `PushN` | `func (q *Queue[T]) PushN(vs ...T)` | Adds `vs` to the back in argument order (last argument at the back); grows the backing buffer at most once. | amortized O(k) |
| `Pop` | `func (q *Queue[T]) Pop() (T, bool)` | Removes and returns the front element (the oldest); `ok` is `false` if empty. | O(1) |
| `PopN` | `func (q *Queue[T]) PopN(n int) []T` | Removes and returns up to `n` front elements in front-to-back order; `nil` if `n <= 0` or empty. | O(k) |
| `Clear` | `func (q *Queue[T]) Clear()` | Removes all elements **and releases** the backing buffer (`Cap` drops to 0). | O(1) |
| `Reset` | `func (q *Queue[T]) Reset()` | Removes all elements but **keeps** the backing buffer for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (q *Queue[T]) Shrink()` | Shrink-to-fit: **copies** into a right-sized buffer and frees the old one now; no-op when `Cap == Len` and the elements do not wrap. | O(n) |
| `Clip` | `func (q *Queue[T]) Clip()` | Trims reported capacity to `Len`; reslices without copying unless the elements wrap around the buffer end. | O(1), O(n) if wrapped |
| `Grow` | `func (q *Queue[T]) Grow(n int)` | Reserves capacity for at least `n` more elements; no-op when `n <= 0` or capacity already suffices. | O(n) |

`n` is the number of elements in the queue; `k` is the number of elements
pushed or popped by the call.

### The comma-ok contract

`Peek` and `Pop` return a value and a boolean. The boolean is `true` when an
element was found and `false` when the queue is empty (or nil); in that case
the value is the zero value of `T`.

```go
v, ok := q.Pop()
if !ok {
	// queue was empty; v is the zero value
}
```

### Iteration with `All`

`All` returns an [`iter.Seq[T]`](https://pkg.go.dev/iter#Seq) — a Go 1.23
range-over-func iterator — that yields elements from **front to back** (FIFO
order). It does not consume the queue, and breaking out of the loop early stops
iteration cleanly:

```go
for v := range q.All() {
	fmt.Println(v)
	if v == target {
		break // safe; the queue is untouched
	}
}
fmt.Println(q.Len()) // unchanged
```

Mutating the queue while iterating over `All` is undefined behavior.

### Bulk push with `PushN`

`PushN` adds several elements at once **in argument order**, so the last
argument ends up at the back — exactly as if you called `Push` for each in turn,
but it grows the backing buffer at most once instead of on every element:

```go
q := queue.New[int]()
q.PushN(1, 2, 3) // same as Push(1); Push(2); Push(3)

front, _ := q.Peek()
fmt.Println(front) // 1
```

With no arguments it is a no-op.

### Bulk pop with `PopN`

`PopN` removes up to `n` elements from the front and returns them in **FIFO
order**, so the first element of the result is the former front — exactly as
if you called `Pop` `n` times. If `n` exceeds `Len`, it drains the queue; for
`n <= 0`, or an empty or nil queue, it returns `nil`. Vacated slots are zeroed
and `Cap` is preserved:

```go
q := queue.New[int]()
q.PushN(1, 2, 3, 4)

fmt.Println(q.PopN(3)) // [1 2 3]
fmt.Println(q.Slice()) // [4]
```

### Copying out with `Clone` and `Slice`

`Clone` returns an independent queue; `Slice` returns a plain slice. Both are
detached from the original — they share no backing buffer, so mutating one does
not affect the other:

```go
a := queue.New[int]()
a.PushN(1, 2, 3)

b := a.Clone() // independent *Queue[int]
b.Pop()        // removes 1 from b only

fmt.Println(a.Slice()) // [1 2 3]
fmt.Println(b.Slice()) // [2 3]
```

Both copies are **shallow**: when `T` is a pointer or contains references, the
pointed-to data is shared between the copies. The clone's capacity equals its
length. `Slice` returns `nil` for an empty queue, and `Clone` of a nil queue
returns a new, usable empty queue.

## Capacity management

The queue grows automatically, but several methods let you control the backing
buffer explicitly:

- **Reserve ahead** — `NewWithCap(n)` preallocates at construction; `Grow(n)`
  reserves room for `n` more elements on an existing queue so subsequent pushes
  do not reallocate.
- **Reclaim memory** — `Shrink` and `Clip` shrink the capacity to the current
  length (see below); `Clear` releases the backing buffer entirely.
- **Reuse memory** — `Reset` empties the queue but keeps the capacity for
  refilling.

The backing buffer is a ring: popped slots become available for later pushes,
so the buffer never has to slide or shrink. `Pop` and `PopN` therefore do not
reduce the value reported by `Cap`, and `Reset` preserves the full allocation
even after many removals.

## `Shrink` vs `Clip`

Both reduce the capacity to the current length, but they make opposite
trade-offs between work done now and memory reclaimed now:

| Operation | Copies? | Cost | Frees unused memory | Use when |
| --- | --- | --- | --- | --- |
| `Shrink` | yes (new buffer) | O(n) | immediately | The queue grew large, has shrunk, and you want the memory back now. |
| `Clip` | only if the elements wrap | O(1), O(n) if wrapped | only on the next growth (reallocation) | You want the cheap version and can wait for lazy reclaim. |

```go
q := queue.NewWithCap[int](1024)
q.PushN(1, 2, 3)

q.Clip()
fmt.Println(q.Len(), q.Cap()) // 3 3  — O(1) reslice, old buffer still retained

q.Grow(2000)
q.Shrink()
fmt.Println(q.Len(), q.Cap()) // 3 3  — copied into a right-sized buffer, old one freed
```

Both are no-ops when `Cap` already equals `Len` and the elements do not wrap.

## `Clear` vs `Reset`

Both empty the queue and drop all references to the removed elements (so they
become eligible for garbage collection), but they differ in what happens to the
allocated backing buffer:

| Operation | Empties queue | Backing buffer | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the queue, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the queue and want to avoid reallocating. |

```go
q := queue.NewWithCap[int](8)
q.PushN(1, 2)

q.Reset()
fmt.Println(q.Len(), q.Cap()) // 0 8  — capacity kept for reuse

q.Clear()
fmt.Println(q.Len(), q.Cap()) // 0 0  — backing buffer released
```

## Nil receiver and zero value

A nil `*Queue[T]` behaves as a valid **empty** queue for every read-only
method and for removal methods, so you can safely query a `nil` queue:

```go
var q *queue.Queue[int] // nil

q.Len()           // 0
q.IsEmpty()       // true
q.Cap()           // 0
v, ok := q.Peek() // 0, false
v, ok = q.Pop()   // 0, false (nothing to remove)
q.PopN(3)         // nil
q.Slice()         // nil
for range q.All() { /* never runs */ }
q.Clone() // new, usable empty queue (never nil)
```

The **mutating** methods — `Push`, `PushN`, `Clear`, `Reset`, `Shrink`, `Clip`,
and `Grow` — need a non-nil receiver, so they **panic** on a `nil` queue with a
message such as `queue: Push on nil receiver`.

The zero value of `Queue[T]` is usable, so `New` is optional:

```go
var q queue.Queue[int]
q.Push(1)
fmt.Println(q.Len()) // 1
```

Copying a `Queue` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Queue` is **not safe for concurrent use**: it performs no internal locking.
If a queue is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `Peek` | one `Peek` on 4096 elements | 1.4 | 0 | 0 |
| `All` | iterate 4096 elements with `All` | 9 001 | 0 | 0 |
| `Clone` | clone 4096 elements | 3 978 | 32 816 | 2 |
| `Slice` | copy 4096 elements | 3 728 | 32 768 | 1 |
| `Push` | `Push` into a queue growing from empty to 4096 elements | 8.4 | 43 | 0 |
| `PushN` | one `PushN` of 256 elements into preallocated room | 282 | 0 | 0 |
| `Pop` | one `Pop`, refilled outside the timing | 2.7 | 0 | 0 |
| `PopN` | one `PopN` of 128 elements, refilled outside the timing | 279 | 1 024 | 1 |
| `PushPopChurn/queue` | one `Pop` + one `Push` with 16 live elements | 5.8 | 0 | 0 |
| `PushPopChurn/container-list` | the same churn on [`container/list`](https://pkg.go.dev/container/list) | 39.1 | 55 | 1 |

- `B/op` for `Push` is the amortized cost of growing the ring buffer; `allocs/op` rounds down to 0 because growth happens once per many pushes. Pre-size with `NewWithCap` or `Grow` to avoid it.

Run them with:

```sh
go test -bench=. -benchmem ./queue/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including bulk push and pop, cloning, shrinking, nil and zero-value use, and
breadth-first (level-order) tree traversal — and are rendered alongside the API
on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, including
a churn comparison against `container/list`; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/queue          # package overview
go doc github.com/Nergous/structures/queue Queue    # the Queue type and its methods
```

The same comments render on pkg.go.dev-style godoc.
