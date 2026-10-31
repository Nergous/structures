# structures

Generic, idiomatic data structures for Go — small, allocation-friendly, and
built on the Go 1.23+ standard library, including range-over-func iterators.

![Structures containers overview](./assets/structures-overview.svg)

![Structures benchmark snapshot](./assets/structures-benchmarks.svg)

`structures` is a growing collection of generic container types. Each package is
self-contained, dependency-free, and documented with runnable examples.

**See every method animated step by step in the
[visual field guide](https://nergous.github.io/structures/).**

## About Structures

Structures aims for containers with predictable behavior, small APIs, and
explicit trade-offs instead of hidden machinery.

The library is intentionally practical. It does not try to wrap every possible
data-structure pattern or replace Go slices and maps. Instead, it focuses on the
cases where a named container makes code easier to read: a LIFO stack, a FIFO
queue, a set, a heap, and other compact structures where the order of
operations or membership matters.

The project values:

- **Order** — iteration and removal order are part of each package contract, not
  an implementation detail.
- **Balance** — APIs stay small enough to remember, but still expose the memory
  controls needed in long-running programs.
- **Clarity** — empty reads use the `(value, ok)` convention, examples are
  runnable, and package docs explain nil receivers and memory behavior.
- **Predictability** — containers are backed by slices or maps,
  dependency-free, and explicit about when capacity is reused or released.

## Requirements

Go 1.23+ (for range-over-func iterators). The module targets Go 1.26.

## Install

```go
import "github.com/Nergous/structures/stack"
```

## Packages

| Package | Description | Docs |
| --- | --- | --- |
| [`stack`](./stack) | Generic, slice-backed LIFO stack with amortized O(1) push and O(1) pop. | [README](./stack/README.md) |
| [`queue`](./queue) | Generic, slice-backed ring-buffer FIFO queue with amortized O(1) push and O(1) pop. | [README](./queue/README.md) |
| [`deque`](./deque) | Generic, slice-backed ring-buffer double-ended queue with amortized O(1) push and O(1) pop at both ends. | [README](./deque/README.md) |
| [`set`](./set) | Generic, map-backed set with unordered iteration and set algebra. | [README](./set/README.md) |
| [`binheap`](./binheap) | Generic, slice-backed binary heap with min, max, or custom order, a type-safe `container/heap` replacement. | [README](./binheap/README.md) |
| [`pqueue`](./pqueue) | Generic, heap-backed keyed priority queue with O(log n) priority updates and removal by key, stable for equal priorities. | [README](./pqueue/README.md) |
| [`orderedmap`](./orderedmap) | Generic, insertion-ordered hash map with O(1) access to both ends and `MoveToBack`. | [README](./orderedmap/README.md) |
| [`orderedset`](./orderedset) | Generic, insertion-ordered set with set algebra in a defined order and O(1) access to both ends. | [README](./orderedset/README.md) |
| [`bitset`](./bitset) | Set of non-negative ints backed by a bit vector, with O(1) membership, ascending iteration, and word-wise set algebra. | [README](./bitset/README.md) |

More containers are on the way.

## When to use it

Use Structures when the container itself is part of the program's meaning:

- a stack for undo history, parser state, tree traversal, or backtracking;
- a queue for breadth-first traversal, work scheduling, buffering, or staged
  processing;
- a deque for sliding windows, work stealing, or histories that grow and trim
  from both ends;
- a set for membership tests, deduplication, and set algebra;
- a heap for priority scheduling, top-k selection, shortest paths, or merging
  sorted streams;
- a priority queue when queued items must be re-prioritized or cancelled by
  identity, as in Dijkstra's algorithm, A*, or job schedulers;
- an ordered map when output must be deterministic, or as the core of a FIFO
  or least-recently-used cache;
- an ordered set when unique values must keep insertion order, as in
  deterministic output or unique work queues;
- a bitset for dense small ints: visited marks, flags, sieves, and fast set
  algebra over IDs;
- future containers where a compact, documented abstraction reads better than
  open-coded slice manipulation.

Plain slices and maps are still the right choice for many jobs. Structures is
for the spots where naming the behavior makes the code more obvious and where
tested edge cases — empty reads, nil receivers, cloning, slicing, capacity
control — are worth having in one place.

## Quick start

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

## Design goals

- **Generic** — type-safe containers via Go generics, no `any` boxing.
- **Idiomatic** — small APIs that follow standard-library conventions, such as
  the `(value, ok)` contract and [`iter.Seq`](https://pkg.go.dev/iter#Seq)
  iterators.
- **Allocation-friendly** — capacity preallocation where it matters, with
  explicit control over when backing memory is released.
- **Dependency-free** — standard library only.
- **Documented by examples** — exported behavior is backed by runnable examples
  that double as package documentation.

## API promises

Structures packages are designed around a few consistent rules:

- Read-only methods on nil receivers behave like empty containers.
- Operations on empty containers are not errors. Methods such as `Peek`, `Pop`,
  `PopN`, `Remove`, and lookup-style predicates return the zero value,
  `false`, or `nil` according to Go's usual comma-ok conventions.
- Apart from documented argument checks (a nil predicate passed to
  `RemoveFunc`, a nil comparator passed to a `Func` constructor) and zero-value
  heaps, methods panic only when the call cannot be completed because the
  receiver is nil and the method needs to store, resize, or release state.
  Examples include `Push`, `Add`, `Grow`, `Clear`, `Reset`, `Shrink`, and
  `Clip`.
- Non-nil containers, including zero-value structs where a package documents
  them as usable, remain valid after empty reads, failed removals, `Clear`, or
  `Reset`.
- The zero value of `stack.Stack`, `queue.Queue`, `deque.Deque`, `set.Set`,
  `orderedmap.Map`, `orderedset.Set`, and `bitset.Set` is an empty, usable
  container.
  `binheap.Heap` and `pqueue.Queue` need a constructor because they hold a
  comparator; their zero value reads as empty and panics on `Push`.
- Nil-receiver panics share one message format: `pkg: Method on nil receiver`.
- `Clone` returns an independent shallow copy. Cloning a nil receiver returns
  a new, usable empty container, except for `binheap.Heap` and `pqueue.Queue`:
  a nil heap has no comparator to copy, so its `Clone` returns nil.
- `Slice` returns a detached slice in the same order as iteration.
- `Reset` empties a container while keeping capacity for reuse.
- `Clear`, `Shrink`, and `Clip` are explicit tools for releasing or reducing
  backing memory.
- Containers are not internally synchronized; callers that share a value across
  goroutines must provide synchronization.

## Documentation

Each package ships a detailed README and verified `example_test.go` files that
render on godoc:

```sh
go doc github.com/Nergous/structures/stack          # package overview
go doc github.com/Nergous/structures/stack Stack    # a type and its methods
```

The [visual field guide](https://nergous.github.io/structures/) animates every
method of every package against its internals (backing arrays, ring buffer,
heap sifts, holes and compaction, bit words); its source lives in
[`docs/`](./docs).

Release notes live in [`CHANGELOG.md`](./CHANGELOG.md).

## License

[MIT](./LICENSE) © Nergous
