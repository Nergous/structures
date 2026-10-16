# deque

A generic, slice-backed ring-buffer **double-ended queue** for Go.

It is part of the `structures` data-structures library. It pushes and pops at
both ends in amortized O(1), reusing freed slots instead of sliding or
reallocating its backing buffer, and uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/deque"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/deque"
)

func main() {
	d := deque.New[int]()
	d.PushBack(2)
	d.PushFront(1)
	d.PushBack(3)

	for !d.IsEmpty() {
		v, _ := d.PopFront()
		fmt.Println(v) // 1, 2, 3
	}
}
```

Preallocate when the maximum size is known up front to avoid reallocations
while pushing:

```go
d := deque.NewWithCap[string](3)
d.PushBack("a")
d.PushBack("b")
d.PushBack("c")
fmt.Println(d.Len(), d.Cap()) // 3 3
```

## Ordering

A deque operates on either end:

- **Front** — `PushFront`, `PushFrontN`, `Front`, `PopFront`, and `PopFrontN`.
- **Back** — `PushBack`, `PushBackN`, `Back`, `PopBack`, and `PopBackN`.

`All` and `Slice` report the elements from front to back; `Backward` reports
them from back to front. Used only through `PushBack` and `PopFront` it is a
FIFO queue; used only through one end it is a LIFO stack.

```go
d := deque.New[string]()
d.PushBack("normal")
d.PushFront("urgent")

front, _ := d.PopFront()
back, _ := d.PopBack()
fmt.Println(front, back) // urgent normal
```

For a FIFO-only API, use [`queue`](../queue); for a LIFO-only API, use
[`stack`](../stack).

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[T any]() *Deque[T]` | Creates an empty deque with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[T any](n int) *Deque[T]` | Creates an empty deque with capacity for at least `n` elements; a negative `n` is treated as 0. | O(n) |
| `Len` | `func (d *Deque[T]) Len() int` | Number of elements currently in the deque. | O(1) |
| `Cap` | `func (d *Deque[T]) Cap() int` | Capacity of the backing buffer before the next reallocation. | O(1) |
| `IsEmpty` | `func (d *Deque[T]) IsEmpty() bool` | Reports whether the deque has no elements. | O(1) |
| `Front` | `func (d *Deque[T]) Front() (T, bool)` | Returns the front element without removing it; `ok` is `false` if empty. | O(1) |
| `Back` | `func (d *Deque[T]) Back() (T, bool)` | Returns the back element without removing it; `ok` is `false` if empty. | O(1) |
| `All` | `func (d *Deque[T]) All() iter.Seq[T]` | Iterator over elements from front to back, without consuming them. | O(1) per step |
| `Backward` | `func (d *Deque[T]) Backward() iter.Seq[T]` | Iterator over elements from back to front, without consuming them. | O(1) per step |
| `Clone` | `func (d *Deque[T]) Clone() *Deque[T]` | Returns an independent **shallow** copy; the two deques share no backing buffer. | O(n) |
| `Slice` | `func (d *Deque[T]) Slice() []T` | Returns an independent copy of the elements from front to back (same order as `All`); `nil` if empty. | O(n) |
| `PushFront` | `func (d *Deque[T]) PushFront(v T)` | Adds `v` to the front of the deque. | amortized O(1) |
| `PushBack` | `func (d *Deque[T]) PushBack(v T)` | Adds `v` to the back of the deque. | amortized O(1) |
| `PushFrontN` | `func (d *Deque[T]) PushFrontN(vs ...T)` | Adds `vs` to the front in argument order (first argument at the front); grows the backing buffer at most once. | amortized O(k) |
| `PushBackN` | `func (d *Deque[T]) PushBackN(vs ...T)` | Adds `vs` to the back in argument order (last argument at the back); grows the backing buffer at most once. | amortized O(k) |
| `PopFront` | `func (d *Deque[T]) PopFront() (T, bool)` | Removes and returns the front element; `ok` is `false` if empty. | O(1) |
| `PopBack` | `func (d *Deque[T]) PopBack() (T, bool)` | Removes and returns the back element; `ok` is `false` if empty. | O(1) |
| `PopFrontN` | `func (d *Deque[T]) PopFrontN(n int) []T` | Removes and returns up to `n` front elements in front-to-back order; `nil` if `n <= 0` or empty. | O(k) |
| `PopBackN` | `func (d *Deque[T]) PopBackN(n int) []T` | Removes and returns up to `n` back elements in back-to-front order; `nil` if `n <= 0` or empty. | O(k) |
| `Clear` | `func (d *Deque[T]) Clear()` | Removes all elements **and releases** the backing buffer (`Cap` drops to 0). | O(1) |
| `Reset` | `func (d *Deque[T]) Reset()` | Removes all elements but **keeps** the backing buffer for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (d *Deque[T]) Shrink()` | Shrink-to-fit: **copies** into a right-sized buffer and frees the old one now; no-op when `Cap == Len` and the elements do not wrap. | O(n) |
| `Clip` | `func (d *Deque[T]) Clip()` | Trims reported capacity to `Len`; reslices without copying unless the elements wrap around the buffer end. | O(1), O(n) if wrapped |
| `Grow` | `func (d *Deque[T]) Grow(n int)` | Reserves capacity for at least `n` more elements; no-op when `n <= 0` or capacity already suffices. | O(n) |

`n` is the number of elements in the deque; `k` is the number of elements
pushed or popped by the call.

### The comma-ok contract

`Front`, `Back`, `PopFront`, and `PopBack` return a value and a boolean. The
boolean is `true` when an element was found and `false` when the deque is empty
(or nil); in that case the value is the zero value of `T`.

```go
v, ok := d.PopFront()
if !ok {
	// deque was empty; v is the zero value
}
```

### Iteration with `All`

`All` returns an [`iter.Seq[T]`](https://pkg.go.dev/iter#Seq) — a Go 1.23
range-over-func iterator — that yields elements from **front to back**;
`Backward` yields them from **back to front**. Neither consumes the deque, and
breaking out of the loop early stops iteration cleanly:

```go
for v := range d.All() {
	fmt.Println(v)
	if v == target {
		break // safe; the deque is untouched
	}
}
fmt.Println(d.Len()) // unchanged

for v := range d.Backward() {
	fmt.Println(v) // back to front
}
```

Mutating the deque while iterating over `All` or `Backward` is undefined
behavior.

### Bulk push with `PushFrontN` and `PushBackN`

`PushFrontN` and `PushBackN` add several elements at once **in argument
order** and grow the backing buffer at most once instead of on every element:

```go
d := deque.New[int]()
d.PushBackN(4, 5)
d.PushFrontN(1, 2, 3)

fmt.Println(d.Slice()) // [1 2 3 4 5]
```

`PushFrontN` keeps argument order at the front: the first argument becomes the
new front. `PushBackN` keeps argument order at the back: the last argument
becomes the new back. With no arguments, both are no-ops.

### Bulk pop with `PopFrontN` and `PopBackN`

`PopFrontN` removes up to `n` elements from the front and returns them in
**front-to-back** order; `PopBackN` removes up to `n` elements from the back
and returns them in **back-to-front** order — exactly as if you called
`PopFront` or `PopBack` `n` times. If `n` exceeds `Len`, they drain the deque;
for `n <= 0`, or an empty or nil deque, they return `nil`. Vacated slots are
zeroed and `Cap` is preserved:

```go
d := deque.New[int]()
d.PushBackN(1, 2, 3, 4)

fmt.Println(d.PopFrontN(2)) // [1 2]
fmt.Println(d.PopBackN(2))  // [4 3]
```

### Copying out with `Clone` and `Slice`

`Clone` returns an independent deque; `Slice` returns a plain slice. Both are
detached from the original — they share no backing buffer, so mutating one
does not affect the other:

```go
a := deque.New[int]()
a.PushBackN(1, 2, 3)

b := a.Clone() // independent *Deque[int]
b.PopFront()   // removes 1 from b only

fmt.Println(a.Slice()) // [1 2 3]
fmt.Println(b.Slice()) // [2 3]
```

Both copies are **shallow**: when `T` is a pointer or contains references, the
pointed-to data is shared between the copies. The clone's capacity equals its
length. `Slice` returns `nil` for an empty deque, and `Clone` of a nil deque
returns a new, usable empty deque.

## Capacity management

The deque grows automatically, but several methods let you control the backing
buffer explicitly:

- **Reserve ahead** — `NewWithCap(n)` preallocates at construction; `Grow(n)`
  reserves room for `n` more elements on an existing deque so subsequent pushes
  do not reallocate.
- **Reclaim memory** — `Shrink` and `Clip` shrink the capacity to the current
  length (see below); `Clear` releases the backing buffer entirely.
- **Reuse memory** — `Reset` empties the deque but keeps the capacity for
  refilling.

The backing buffer is a ring: slots popped at either end become available for
later pushes, so the buffer never has to slide or shrink. `PopFront`,
`PopBack`, `PopFrontN`, and `PopBackN` therefore do not reduce the value
reported by `Cap`, and `Reset` preserves the full allocation even after many
removals.

## `Shrink` vs `Clip`

Both reduce the capacity to the current length, but they make opposite
trade-offs between work done now and memory reclaimed now:

| Operation | Copies? | Cost | Frees unused memory | Use when |
| --- | --- | --- | --- | --- |
| `Shrink` | yes (new buffer) | O(n) | immediately | The deque grew large, has shrunk, and you want the memory back now. |
| `Clip` | only if the elements wrap | O(1), O(n) if wrapped | only on the next growth (reallocation) | You want the cheap version and can wait for lazy reclaim. |

```go
d := deque.NewWithCap[int](1024)
d.PushBackN(1, 2, 3)

d.Clip()
fmt.Println(d.Len(), d.Cap()) // 3 3  — O(1) reslice, old buffer still retained

d.Grow(2000)
d.Shrink()
fmt.Println(d.Len(), d.Cap()) // 3 3  — copied into a right-sized buffer, old one freed
```

Both are no-ops when `Cap` already equals `Len` and the elements do not wrap.

## `Clear` vs `Reset`

Both empty the deque and drop all references to the removed elements (so they
become eligible for garbage collection), but they differ in what happens to the
allocated backing buffer:

| Operation | Empties deque | Backing buffer | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the deque, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the deque and want to avoid reallocating. |

```go
d := deque.NewWithCap[int](8)
d.PushBackN(1, 2)

d.Reset()
fmt.Println(d.Len(), d.Cap()) // 0 8  — capacity kept for reuse

d.Clear()
fmt.Println(d.Len(), d.Cap()) // 0 0  — backing buffer released
```

## Nil receiver and zero value

A nil `*Deque[T]` behaves as a valid **empty** deque for every read-only
method and for removal methods, so you can safely query a `nil` deque:

```go
var d *deque.Deque[int] // nil

d.Len()              // 0
d.IsEmpty()          // true
d.Cap()              // 0
v, ok := d.Front()   // 0, false
v, ok = d.Back()     // 0, false
v, ok = d.PopFront() // 0, false (nothing to remove)
v, ok = d.PopBack()  // 0, false (nothing to remove)
d.PopFrontN(3)       // nil
d.PopBackN(3)        // nil
d.Slice()            // nil
for range d.All() { /* never runs */ }
for range d.Backward() { /* never runs */ }
d.Clone() // new, usable empty deque (never nil)
```

The **mutating** methods — `PushFront`, `PushBack`, `PushFrontN`, `PushBackN`,
`Clear`, `Reset`, `Shrink`, `Clip`, and `Grow` — need a non-nil receiver, so
they **panic** on a `nil` deque with a message such as
`deque: PushBack on nil receiver`.

The zero value of `Deque[T]` is usable, so `New` is optional:

```go
var d deque.Deque[int]
d.PushBack(1)
fmt.Println(d.Len()) // 1
```

Copying a `Deque` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Deque` is **not safe for concurrent use**: it performs no internal locking.
If a deque is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `Front` | one `Front` on 4096 elements | 1.5 | 0 | 0 |
| `Back` | one `Back` on 4096 elements | 1.7 | 0 | 0 |
| `All` | iterate 4096 elements with `All` | 9 364 | 0 | 0 |
| `Backward` | iterate 4096 elements with `Backward` | 7 814 | 0 | 0 |
| `Clone` | clone 4096 elements | 4 108 | 32 816 | 2 |
| `Slice` | copy 4096 elements | 4 112 | 32 768 | 1 |
| `PushFront` | `PushFront` into a deque growing from empty to 4096 elements | 9.7 | 43 | 0 |
| `PushBack` | `PushBack` into a deque growing from empty to 4096 elements | 9.0 | 43 | 0 |
| `PushFrontN` | one `PushFrontN` of 256 elements into preallocated room | 273 | 0 | 0 |
| `PushBackN` | one `PushBackN` of 256 elements into preallocated room | 281 | 0 | 0 |
| `PopFront` | one `PopFront`, refilled outside the timing | 3.1 | 0 | 0 |
| `PopBack` | one `PopBack`, refilled outside the timing | 3.8 | 0 | 0 |
| `PopFrontN` | one `PopFrontN` of 128 elements, refilled outside the timing | 311 | 1 024 | 1 |
| `PopBackN` | one `PopBackN` of 128 elements, refilled outside the timing | 416 | 1 024 | 1 |
| `PushPopFrontBackChurn/deque` | one `PopFront` + `PushBack` and one `PopBack` + `PushFront` with 16 live elements | 10.9 | 0 | 0 |
| `PushPopFrontBackChurn/container-list` | the same churn on [`container/list`](https://pkg.go.dev/container/list) | 76.8 | 111 | 3 |

- `B/op` for `PushFront` and `PushBack` is the amortized cost of growing the ring buffer; `allocs/op` rounds down to 0 because growth happens once per many pushes. Pre-size with `NewWithCap` or `Grow` to avoid it.

Run them with:

```sh
go test -bench=. -benchmem ./deque/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including pushes and pops at both ends, reverse iteration with `Backward`, bulk
push and pop, cloning, shrinking, nil and zero-value use, and an urgent-first
task queue — and are rendered alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, including
a churn comparison against `container/list`; their sources live in
[`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/deque          # package overview
go doc github.com/Nergous/structures/deque Deque    # the Deque type and its methods
```

The same comments render on pkg.go.dev-style godoc.
