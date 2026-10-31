import { sizeReads } from "../core/common";
import { freshInt, list, okRet, parseInt1, parseInts, randInts, rec } from "../core/rec";
import type { MethodDef, StructureDef, ViewProps } from "../core/types";
import {
  idx,
  iterate,
  live,
  popBack,
  popFront,
  pushBack,
  pushBackN,
  pushFront,
  pushFrontN,
  ringList,
  ringMemory,
  ringOf,
  RingView,
  type RingS,
} from "./ring";

const R = "d *Deque[T]";
const fresh = (s: RingS) => String(freshInt(live(s).map((c) => c.v)));

function View({ f }: ViewProps<RingS>) {
  return <RingView f={f} front="front" back="back" />;
}

function popNs(front: boolean): MethodDef<RingS> {
  const name = front ? "PopFrontN" : "PopBackN";
  return {
    name,
    group: "mutate",
    sig: `func (d *Deque[T]) ${name}(n int) []T`,
    doc: front
      ? "Removes up to n elements from the front, returned front to back."
      : "Removes up to n elements from the back, returned in removal (back to front) order.",
    cost: "O(k)",
    args: [{ name: "n", kind: "int", def: "2" }],
    run: (s0, a) => {
      const r = rec(s0);
      const n = Math.min(parseInt1(a[0]), r.s.len);
      if (n <= 0) return r.done("nil", "nothing to pop: nil");
      const out: number[] = [];
      for (let i = 0; i < n; i++) out.push((front ? popFront : popBack)(r, "popped")!.v);
      return r.done(list(out));
    },
  };
}

export const deque: StructureDef<RingS> = {
  id: "deque",
  title: "deque",
  kind: "double-ended · ring buffer",
  tagline: "Push and pop at both ends in O(1). The front can walk backwards around the circle.",
  about: [
    "A Deque[T] uses the same ring buffer as queue. PushFront moves head one slot counter-clockwise (wrapping from 0 to cap-1) and writes there; PushBack writes at (head+len) % cap.",
    "Because both ends just move an index, nothing is ever shifted. The live window can wrap around the end of the array — the highlighted arc on the circle shows it.",
    "Growth copies the window front→back into a larger array, so after a grow the front is at index 0 again.",
  ],
  cost: [
    ["PushFront / PushBack", "amortized O(1)"],
    ["PopFront / PopBack", "O(1)"],
    ["Front / Back", "O(1)"],
    ["*N variants", "O(k)"],
    ["All / Backward / Slice", "O(n)"],
  ],
  seed: () => ringOf([8, 21, 5, 40], 8, 6),
  stats: (s) => [
    ["len", s.len],
    ["cap", s.data.length],
    ["head", s.head],
    ["front", s.len ? s.data[s.head]!.v : "—"],
    ["back", s.len ? s.data[idx(s, s.len - 1)]!.v : "—"],
  ],
  View,
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New[T any]() *Deque[T]",
      doc: "Returns an empty deque with no preallocated capacity.",
      cost: "O(1)",
      run: () => rec<RingS>({ data: [], head: 0, len: 0 }).done("&Deque{}", "empty ring, no array"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap[T any](n int) *Deque[T]",
      doc: "Returns an empty deque with a ring of n slots. Negative n is treated as 0.",
      cost: "O(n)",
      args: [{ name: "n", kind: "int", def: "6" }],
      run: (_, a) => {
        const n = Math.max(parseInt1(a[0]), 0);
        return rec<RingS>({ data: Array(n).fill(null), head: 0, len: 0 }).done("&Deque{}", `ring of ${n} slots`);
      },
    },
    ...sizeReads<RingS>({ recv: R, len: (s) => s.len, cap: (s) => s.data.length }),
    {
      name: "Front",
      group: "read",
      sig: "func (d *Deque[T]) Front() (T, bool)",
      doc: "Returns the front element without removing it.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.len) return r.done(okRet(0, false), "empty deque");
        const c = r.s.data[r.s.head]!;
        return r.done(okRet(c.v, true), `data[head] = ${c.v}`, { ["c" + c.id]: "ok" });
      },
    },
    {
      name: "Back",
      group: "read",
      sig: "func (d *Deque[T]) Back() (T, bool)",
      doc: "Returns the back element without removing it.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.len) return r.done(okRet(0, false), "empty deque");
        const i = idx(r.s, r.s.len - 1);
        const c = r.s.data[i]!;
        return r.done(okRet(c.v, true), `data[(head+len-1) % cap] = data[${i}] = ${c.v}`, { ["c" + c.id]: "ok" });
      },
    },
    {
      name: "All",
      group: "iterate",
      sig: "func (d *Deque[T]) All() iter.Seq[T]",
      doc: "Iterates front to back without consuming.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        iterate(r, false);
        return r.done("", "front to back");
      },
    },
    {
      name: "Backward",
      group: "iterate",
      sig: "func (d *Deque[T]) Backward() iter.Seq[T]",
      doc: "Iterates back to front without consuming.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        iterate(r, true);
        return r.done("", "back to front");
      },
    },
    {
      name: "Clone",
      group: "copy",
      sig: "func (d *Deque[T]) Clone() *Deque[T]",
      doc: "Returns an independent copy, unwrapped to index 0, with cap == len.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        live(r.s).forEach((c) => r.emit(c.v, "new", "clone (front → back)"));
        return r.done(`&Deque{len ${r.s.len}, cap ${r.s.len}}`);
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (d *Deque[T]) Slice() []T",
      doc: "Returns a copy of the elements front to back. Empty deque returns nil.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const cs = live(r.s);
        cs.forEach((c) => r.emit(c.v, undefined, "[]T (front first)"));
        return r.done(ringList(cs));
      },
    },
    {
      name: "PushFront",
      group: "mutate",
      sig: "func (d *Deque[T]) PushFront(v T)",
      doc: "Prepends v: head steps back one slot (wrapping) and v is written there.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: fresh }],
      run: (s0, a) => {
        const r = rec(s0);
        pushFront(r, parseInt1(a[0]));
        return r.done("");
      },
    },
    {
      name: "PushBack",
      group: "mutate",
      sig: "func (d *Deque[T]) PushBack(v T)",
      doc: "Appends v after the back, wrapping to index 0 if needed.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: fresh }],
      run: (s0, a) => {
        const r = rec(s0);
        pushBack(r, parseInt1(a[0]));
        return r.done("");
      },
    },
    {
      name: "PushFrontN",
      group: "mutate",
      sig: "func (d *Deque[T]) PushFrontN(vs ...T)",
      doc: "Prepends vs keeping argument order: vs[0] becomes the new front. Grows at most once.",
      cost: "amortized O(k)",
      args: [{ name: "vs", kind: "ints", def: () => randInts(3).join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        const vs = parseInts(a[0]);
        if (!vs.length) return r.done("", "no arguments: no-op");
        pushFrontN(r, vs);
        return r.done("");
      },
    },
    {
      name: "PushBackN",
      group: "mutate",
      sig: "func (d *Deque[T]) PushBackN(vs ...T)",
      doc: "Appends vs in order: the last argument becomes the new back. Grows at most once.",
      cost: "amortized O(k)",
      args: [{ name: "vs", kind: "ints", def: () => randInts(3).join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        const vs = parseInts(a[0]);
        if (!vs.length) return r.done("", "no arguments: no-op");
        pushBackN(r, vs);
        return r.done("");
      },
    },
    {
      name: "PopFront",
      group: "mutate",
      sig: "func (d *Deque[T]) PopFront() (T, bool)",
      doc: "Removes and returns the front element.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const c = popFront(r, "popped");
        return r.done(okRet(c?.v, !!c), c ? undefined : "empty deque");
      },
    },
    {
      name: "PopBack",
      group: "mutate",
      sig: "func (d *Deque[T]) PopBack() (T, bool)",
      doc: "Removes and returns the back element.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const c = popBack(r, "popped");
        return r.done(okRet(c?.v, !!c), c ? undefined : "empty deque");
      },
    },
    popNs(true),
    popNs(false),
    ...ringMemory(R),
  ],
};
