# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is 0, a minor release may contain breaking changes;
they are always listed under **Breaking**.

## [Unreleased]

### Breaking

- `pqueue`: the `PriorityQueue` type is renamed to `Queue`, so it reads as
  `pqueue.Queue[K, P]`. There is no alias; replace `pqueue.PriorityQueue` with
  `pqueue.Queue`.
- `set` and `orderedset`: `RetainAll` is renamed to `RetainSet`, matching
  `AddSet` and `RemoveSet`. There is no alias; replace `RetainAll` with
  `RetainSet`.

### Added

- `bitset`: set of non-negative ints backed by a bit vector. Membership, `Add`,
  `Remove`, and `Max` run in O(1), iteration is ascending (`All`, `Backward`,
  `Slice`, `Min`), set algebra and in-place `AddSet`, `RemoveSet`, and
  `RetainSet` work 64 values per word, and the full memory block (`Cap`,
  `Clear`, `Reset`, `Shrink`, `Clip`, `Grow`) is available, with capacity
  bounding the largest value.
- `set`: `RemoveFunc(del)` deletes every value for which `del` returns true
  and reports how many were removed, with the same contract as
  `orderedset.RemoveFunc`.
- `set`: `Shrink` rebuilds the backing map right-sized. Go maps never shrink
  in place, so it is the only way to reclaim memory short of `Clear`.
- `deque`: `Backward` iterates from back to front without consuming elements.
- `pqueue`: `From`, `FromMax`, and `FromFunc` build a queue from a slice of
  `Item[K, P]` in O(n), mirroring the `binheap` constructors. When a key
  repeats, its last occurrence wins.
- `pqueue`: `PushN` pushes several items with `Push` semantics, so an existing
  key has its priority updated.

### Changed

- Documentation, tests, and benchmarks consistency pass across all packages:
  one declaration order, shared godoc and README templates, uniform test
  layout with nil-receiver and zero-value checks, parallel runnable examples,
  and benchmarks built on one skeleton (`benchSize`, `b.Loop`, bulk state
  restores instead of per-iteration timer stops).
- Root `README.md`, `ROADMAP.md`, and `AGENTS.md` describe every package with
  one template and document the naming conventions and deliberate API
  omissions.

## [0.2.0] - 2026-10-15

The first release under the `github.com/Nergous/structures` module path. It
grows the library from a single stack into eight container packages that share
one set of conventions.

### Breaking

- The module path changed from `github.com/Nergous/argos` to
  `github.com/Nergous/structures` (by way of `github.com/Nergous/maat`). Update
  imports, for example `github.com/Nergous/structures/stack`. The `v0.1.0` tag
  declares the old path and cannot be fetched under the new one.

### Added

- `queue`: FIFO queue on a growable ring buffer with `Push`, `PushN`, `Pop`,
  `PopN`, `Peek`, iteration, and capacity controls.
- `deque`: double-ended queue on a growable ring buffer with O(1) push and pop
  at both ends and batch `PushFrontN`, `PushBackN`, `PopFrontN`, `PopBackN`.
- `set`: map-backed set with `Of`, `From`, `ContainsAll`, `ContainsAny`, in-place
  algebra (`AddSet`, `RemoveSet`, `RetainAll`), and set algebra (`Union`,
  `Intersection`, `Difference`, `SymmetricDifference`, `Equal`, `IsSubset`,
  `IsSuperset`, `IsDisjoint`).
- `binheap`: binary heap with min, max, or custom order: `Push`, `PushN`, `Pop`,
  `PopN`, `PushPop`, `Replace`, `Remove`, `Contains`, `Sorted`, and `From`
  constructors that heapify in O(n).
- `pqueue`: keyed priority queue. `Push` inserts or updates a key's priority in
  O(log n), `Remove` and `Priority` work by key, and equal priorities pop in
  insertion order.
- `orderedmap`: hash map that iterates in insertion order, with O(1) `First`,
  `Last`, `PopFirst`, `PopLast`, and `MoveToBack`, plus `Backward`, `Keys`,
  `Values`, and `RemoveFunc`.
- `orderedset`: set that iterates in insertion order. It mirrors the `set` API
  and adds `First`, `Last`, `PopFirst`, `PopLast`, `MoveToBack`, `Backward`,
  and `RemoveFunc`. Set algebra returns results in a documented order.
- `stack`: `PushN`, `PopN`, `Clone`, `Slice`, `Grow`, `Shrink`, and `Clip`.
- Every package: `Clone` and `Slice` snapshots, `Grow`/`Shrink`/`Clip` where
  capacity is observable, runnable examples, unit tests, benchmarks, and a
  README with an API reference and measured benchmark results.
- Continuous integration running `go vet`, `go test`, and the race detector.
- Public roadmap and root README with the shared API promises.

### Changed

- Shared conventions across all packages: `New` and `NewWithCap` (a negative
  capacity is treated as 0), comma-ok lookups and removals, non-consuming
  `All` iterators, `Clear` releases storage while `Reset` keeps it.
- Nil receivers behave as empty containers for reads and removals. Methods that
  store or resize data panic on a nil receiver with a uniform message of the
  form `pkg: Method on nil receiver`.
- The zero value of `Stack`, `Queue`, `Deque`, `Set`, `orderedmap.Map`, and
  `orderedset.Set` is ready to use. `Heap` and `PriorityQueue` hold a
  comparator, so their zero value reads as empty and panics on `Push`.
- `queue` and `deque` share one ring-buffer implementation; bulk pushes
  became about 1.5 to 2.5 times faster.

### Performance

- `set`: adding a value that is already present skips the map write.
- `binheap` and `pqueue` share sift primitives; position-tracking sifts move
  elements into a hole instead of swapping them.

## [0.1.0] - 2026-06-11

Initial release under the module path `github.com/Nergous/argos`.

### Added

- `stack`: generic slice-backed LIFO stack with `New`, `NewWithCap`, `Push`,
  `Pop`, `Peek`, `Len`, `Cap`, `IsEmpty`, `All`, `Clear`, and `Reset`.

[Unreleased]: https://github.com/Nergous/structures/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Nergous/structures/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Nergous/structures/releases/tag/v0.1.0
