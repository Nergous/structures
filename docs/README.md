# structures — visual field guide

An animated, step-by-step visualizer for every method of the containers in
this module: `stack`, `queue`, `deque`, `set`, `binheap`, `pqueue`,
`orderedmap`, `orderedset`, and `bitset`.

Each method is simulated in TypeScript following the Go source step by step:
append growth (`runtime.growslice` size classes), the `internal/ring` growth
policy and wrap-around, `heapcore` sifts (hole-shift variants for `pqueue`),
`orderedmap` holes / head / trim / compaction, and `bitset` word layout with
stale words. `src/sim.test.ts` checks the invariants on random operation
sequences and pins the behaviour against known Go results.

## Develop

```sh
cd docs
npm install
npm run dev        # http://localhost:5173
npm test           # simulation tests (vitest)
npm run build      # typecheck + static build into docs/dist
```

The build uses a relative base, so `dist/` can be served from any path,
for example GitHub Pages via an Actions workflow that uploads `docs/dist`.

## Deep links

`?call=Push&args=42#/stack` runs a call on load; `pre=Push:5;Push:9` runs
setup calls first (arguments separated by `|`), `frame=N` jumps to a step,
and `still` makes animations instant (for screenshots).

## Layout

- `src/core` — frame recorder, Go memory models, player, shared method sets.
- `src/ui` — animated primitives (boxes, slots, pointers, tree, tray).
- `src/structures` — one simulation + view per package; `ring.tsx` and
  `omcore.tsx` mirror `internal/ring` and `orderedmap` internals shared by
  their siblings.
