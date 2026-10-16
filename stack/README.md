# stack

A generic, slice-backed **LIFO (last-in, first-out) stack** for Go.

It is part of the `structures` data-structures library. The stack keeps its top
at the end of a single slice, so `Push` and `Pop` never move other elements,
and uses Go 1.23 range-over-func iterators
([`iter.Seq`](https://pkg.go.dev/iter#Seq)).

## Install

```go
import "github.com/Nergous/structures/stack"
```

Requires Go 1.23+ (for range-over-func). The module targets Go 1.26.

## Usage

```go
package main

import (
	"fmt"

	"github.com/Nergous/structures/stack"
)

func main() {
	s := stack.New[int]()
	s.Push(1)
	s.Push(2)
	s.Push(3)

	for !s.IsEmpty() {
		v, _ := s.Pop()
		fmt.Println(v) // 3, 2, 1
	}
}
```

Preallocate when the maximum size is known up front to avoid reallocations
while pushing:

```go
s := stack.NewWithCap[string](3)
s.Push("a")
s.Push("b")
s.Push("c")
fmt.Println(s.Len(), s.Cap()) // 3 3
```

## Ordering

A stack is **LIFO**: the top is the most recently pushed element, and it is the
first one to come out. `Push` and `PushN` add to the top; `Pop`, `PopN`, and
`Peek` work on the top. Every method that reports several elements does so from
**top to bottom**, the order in which repeated `Pop` calls would remove them:

- `All` yields elements from top to bottom.
- `Slice` returns them from top to bottom, the same order as `All`.
- `PopN` returns them from top to bottom, so the first element is the former
  top.
- `PushN(1, 2, 3)` leaves `3` on top, exactly like `Push(1)`, `Push(2)`,
  `Push(3)`.

```go
s := stack.New[int]()
s.PushN(1, 2, 3)

fmt.Println(s.Slice()) // [3 2 1]
fmt.Println(s.PopN(2)) // [3 2]
top, _ := s.Peek()
fmt.Println(top) // 1
```

## API reference

| Method / Function | Signature | Description | Complexity |
| --- | --- | --- | --- |
| `New` | `func New[T any]() *Stack[T]` | Creates an empty stack with no preallocated capacity. | O(1) |
| `NewWithCap` | `func NewWithCap[T any](n int) *Stack[T]` | Creates an empty stack with capacity for at least `n` elements; a negative `n` is treated as 0. | O(n) |
| `Len` | `func (s *Stack[T]) Len() int` | Number of elements currently on the stack. | O(1) |
| `Cap` | `func (s *Stack[T]) Cap() int` | Capacity of the backing array before the next reallocation. | O(1) |
| `IsEmpty` | `func (s *Stack[T]) IsEmpty() bool` | Reports whether the stack has no elements. | O(1) |
| `Peek` | `func (s *Stack[T]) Peek() (T, bool)` | Returns the top element without removing it; `ok` is `false` if empty. | O(1) |
| `All` | `func (s *Stack[T]) All() iter.Seq[T]` | Iterator over elements from top to bottom, without consuming them. | O(1) per step |
| `Clone` | `func (s *Stack[T]) Clone() *Stack[T]` | Returns an independent **shallow** copy; the two stacks share no backing array. | O(n) |
| `Slice` | `func (s *Stack[T]) Slice() []T` | Returns an independent copy of the elements from top to bottom (same order as `All`); `nil` if empty. | O(n) |
| `Push` | `func (s *Stack[T]) Push(v T)` | Adds `v` to the top of the stack. | amortized O(1) |
| `PushN` | `func (s *Stack[T]) PushN(vs ...T)` | Adds `vs` in argument order (last argument ends up on top); grows the backing array at most once. | amortized O(k) |
| `Pop` | `func (s *Stack[T]) Pop() (T, bool)` | Removes and returns the top element; `ok` is `false` if empty. | O(1) |
| `PopN` | `func (s *Stack[T]) PopN(n int) []T` | Removes and returns up to `n` elements from top to bottom; `nil` if `n <= 0` or empty. | O(k) |
| `Clear` | `func (s *Stack[T]) Clear()` | Removes all elements **and releases** the backing array (`Cap` drops to 0). | O(1) |
| `Reset` | `func (s *Stack[T]) Reset()` | Removes all elements but **keeps** the backing array for reuse (`Cap` preserved). | O(n) |
| `Shrink` | `func (s *Stack[T]) Shrink()` | Shrink-to-fit: **copies** into a right-sized array and frees the old one now; no-op when `Cap == Len`. | O(n) |
| `Clip` | `func (s *Stack[T]) Clip()` | Reslices so `Cap == Len` **without copying**; unused memory is reclaimed only on the next growth. | O(1) |
| `Grow` | `func (s *Stack[T]) Grow(n int)` | Reserves capacity for at least `n` more elements; no-op when `n <= 0` or capacity already suffices. | O(n) |

`n` is the number of elements on the stack; `k` is the number of elements
pushed or popped by the call.

### The comma-ok contract

`Peek` and `Pop` return a value and a boolean. The boolean is `true` when an
element was found and `false` when the stack is empty (or nil); in that case
the value is the zero value of `T`.

```go
v, ok := s.Pop()
if !ok {
	// stack was empty; v is the zero value
}
```

### Iteration with `All`

`All` returns an [`iter.Seq[T]`](https://pkg.go.dev/iter#Seq) — a Go 1.23
range-over-func iterator — that yields elements from **top to bottom**. It
does not consume the stack, and breaking out of the loop early stops iteration
cleanly:

```go
for v := range s.All() {
	fmt.Println(v)
	if v == target {
		break // safe; the stack is untouched
	}
}
fmt.Println(s.Len()) // unchanged
```

Mutating the stack while iterating over `All` is undefined behavior.

### Bulk push with `PushN`

`PushN` adds several elements at once **in argument order**, so the last
argument ends up on top — exactly as if you called `Push` for each in turn,
but it grows the backing array at most once instead of on every element:

```go
s := stack.New[int]()
s.PushN(1, 2, 3) // 3 is now on top, same as Push(1); Push(2); Push(3)

top, _ := s.Peek()
fmt.Println(top) // 3
```

With no arguments it is a no-op.

### Bulk pop with `PopN`

`PopN` removes up to `n` elements from the top and returns them from **top to
bottom**, so the first element of the result is the former top — exactly as if
you called `Pop` `n` times. If `n` exceeds `Len`, it drains the stack; for
`n <= 0`, or an empty or nil stack, it returns `nil`. Vacated slots are zeroed
and `Cap` is preserved:

```go
s := stack.New[int]()
s.PushN(1, 2, 3, 4, 5)

fmt.Println(s.PopN(3)) // [5 4 3]
fmt.Println(s.Slice()) // [2 1]
```

### Copying out with `Clone` and `Slice`

`Clone` returns an independent stack; `Slice` returns a plain slice. Both are
detached from the original — they share no backing array, so mutating one does
not affect the other:

```go
a := stack.New[int]()
a.PushN(1, 2, 3)

b := a.Clone() // independent *Stack[int]
b.Pop()        // removes 3 from b only

fmt.Println(a.Slice()) // [3 2 1]
fmt.Println(b.Slice()) // [2 1]
```

Both copies are **shallow**: when `T` is a pointer or contains references, the
pointed-to data is shared between the copies. The clone's capacity equals its
length. `Slice` returns `nil` for an empty stack, and `Clone` of a nil stack
returns a new, usable empty stack.

## Capacity management

The stack grows automatically, but several methods let you control the backing
array explicitly:

- **Reserve ahead** — `NewWithCap(n)` preallocates at construction; `Grow(n)`
  reserves room for `n` more elements on an existing stack so subsequent pushes
  do not reallocate.
- **Reclaim memory** — `Shrink` and `Clip` shrink the capacity to the current
  length (see below); `Clear` releases the backing array entirely.
- **Reuse memory** — `Reset` empties the stack but keeps the capacity for
  refilling.

## `Shrink` vs `Clip`

Both reduce the capacity to the current length, but they make opposite
trade-offs between work done now and memory reclaimed now:

| Operation | Copies? | Cost | Frees unused memory | Use when |
| --- | --- | --- | --- | --- |
| `Shrink` | yes (new array) | O(n) | immediately | The stack grew large, has shrunk, and you want the memory back now. |
| `Clip` | no (reslice) | O(1) | only on the next growth (reallocation) | You want the cheap version and can wait for lazy reclaim. |

```go
s := stack.NewWithCap[int](1024)
s.PushN(1, 2, 3)

s.Clip()
fmt.Println(s.Len(), s.Cap()) // 3 3  — O(1) reslice, old array still retained

s.Grow(2000)
s.Shrink()
fmt.Println(s.Len(), s.Cap()) // 3 3  — copied into a right-sized array, old one freed
```

Both are no-ops when `Cap` already equals `Len`.

## `Clear` vs `Reset`

Both empty the stack and drop all references to the removed elements (so they
become eligible for garbage collection), but they differ in what happens to the
allocated backing array:

| Operation | Empties stack | Backing array | `Cap` afterwards | Use when |
| --- | --- | --- | --- | --- |
| `Clear` | yes | released (eligible for GC) | `0` | You are done with the stack, or want to free its memory. |
| `Reset` | yes | retained for reuse | unchanged | You will refill the stack and want to avoid reallocating. |

```go
s := stack.NewWithCap[int](8)
s.PushN(1, 2)

s.Reset()
fmt.Println(s.Len(), s.Cap()) // 0 8  — capacity kept for reuse

s.Clear()
fmt.Println(s.Len(), s.Cap()) // 0 0  — backing array released
```

## Nil receiver and zero value

A nil `*Stack[T]` behaves as a valid **empty** stack for every read-only
method and for removal methods, so you can safely query a `nil` stack:

```go
var s *stack.Stack[int] // nil

s.Len()           // 0
s.IsEmpty()       // true
s.Cap()           // 0
v, ok := s.Peek() // 0, false
v, ok = s.Pop()   // 0, false (nothing to remove)
s.PopN(3)         // nil
s.Slice()         // nil
for range s.All() { /* never runs */ }
s.Clone() // new, usable empty stack (never nil)
```

The **mutating** methods — `Push`, `PushN`, `Clear`, `Reset`, `Shrink`, `Clip`,
and `Grow` — need a non-nil receiver, so they **panic** on a `nil` stack with a
message such as `stack: Push on nil receiver`.

The zero value of `Stack[T]` is usable, so `New` is optional:

```go
var s stack.Stack[int]
s.Push(1)
fmt.Println(s.Len()) // 1
```

Copying a `Stack` value (`b := *a`) shares its storage with the original; use
`Clone` for an independent copy.

## Concurrency

A `Stack` is **not safe for concurrent use**: it performs no internal locking.
If a stack is shared across goroutines, the caller must provide its own
synchronization (for example, a `sync.Mutex`).

## Benchmarks

Measured with Go 1.26.5 on windows/amd64, AMD Ryzen 7 5800X 8-Core Processor, 2026-10-15. Each value is the median of 6 runs of `go test -run NONE -bench . -benchmem -benchtime 200ms -count 6`. Absolute numbers depend on hardware, Go version, and system load, so compare rows with each other rather than with other machines.

| Benchmark | What it measures | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| `Peek` | one `Peek` on 4096 elements | 1.4 | 0 | 0 |
| `All` | iterate 4096 elements with `All` | 7 816 | 0 | 0 |
| `Clone` | clone 4096 elements | 4 094 | 32 792 | 2 |
| `Slice` | copy 4096 elements | 4 975 | 32 768 | 1 |
| `Push/stack` | `Push` into a stack growing from empty to 4096 elements | 4.5 | 31 | 0 |
| `Push/slice` | the same with `append` to a plain `[]int` | 4.4 | 31 | 0 |
| `PushN` | one `PushN` of 256 elements into preallocated room | 95.1 | 0 | 0 |
| `Pop/stack` | one `Pop`, refilled outside the timing | 1.6 | 0 | 0 |
| `Pop/slice` | the same pop from a plain `[]int` | 1.6 | 0 | 0 |
| `PopN` | one `PopN` of 128 elements, refilled outside the timing | 186 | 1 024 | 1 |
| `PushPopChurn/stack` | one `Pop` + one `Push` with 16 live elements | 2.2 | 0 | 0 |
| `PushPopChurn/slice` | the same churn on a plain `[]int` | 1.9 | 0 | 0 |

- `B/op` for `Push` is the amortized cost of growing the backing array; `allocs/op` rounds down to 0 because growth happens once per many pushes. Pre-size with `NewWithCap` or `Grow` to avoid it.

Run them with:

```sh
go test -bench=. -benchmem ./stack/...
```

The benchmark sources live in [`bench_test.go`](./bench_test.go).

## More examples and docs

Runnable, verified examples live in [`example_test.go`](./example_test.go) —
including bulk push and pop, cloning, shrinking, nil and zero-value use, and
bracket balancing — and are rendered alongside the API on the godoc page.

Benchmark results are in the [Benchmarks](#benchmarks) section above, including
`Push`, `Pop`, and a push/pop churn against a plain `[]int`; their sources
live in [`bench_test.go`](./bench_test.go).

View the documentation locally:

```sh
go doc github.com/Nergous/structures/stack          # package overview
go doc github.com/Nergous/structures/stack Stack    # the Stack type and its methods
```

The same comments render on pkg.go.dev-style godoc.
