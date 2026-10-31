import { sizeReads } from "../core/common";
import { freshInt, list, okRet, parseInt1, parseInts, randInts, rec } from "../core/rec";
import type { StructureDef } from "../core/types";
import {
  iterate,
  live,
  popFront,
  pushBack,
  pushBackN,
  ringList,
  ringMemory,
  ringOf,
  type RingS,
} from "./ring";

const R = "q *Queue[T]";


export const queue: StructureDef<RingS> = {
  id: "queue",
  title: "queue",
  kind: "FIFO · ring buffer",
  tagline: "First in, first out. Push at the back, Pop from the front — neither end ever shifts the others.",
  about: [
    "A Queue[T] wraps internal/ring.Buffer: a slice used as a circle, plus head (index of the front) and len. The back is (head+len) % cap.",
    "Pop just zeroes data[head] and advances head; Push writes after the back and wraps to index 0 when it runs off the end. No element ever moves, so both ends are O(1).",
    "When the circle is full, the buffer doubles (by a quarter past 1024) and copies the elements front→back into the new array, unwrapping them so head becomes 0.",
  ],
  cost: [
    ["Push", "amortized O(1)"],
    ["Pop / Peek", "O(1)"],
    ["PushN / PopN", "O(k)"],
    ["All / Slice / Clone", "O(n)"],
  ],
  seed: () => ringOf([14, 3, 27], 8, 5),
  stats: (s) => [
    ["len", s.len],
    ["cap", s.data.length],
    ["head", s.head],
    ["front", s.len ? s.data[s.head]!.v : "—"],
  ],
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New[T any]() *Queue[T]",
      doc: "Returns an empty queue with no preallocated capacity.",
      cost: "O(1)",
      run: () => rec<RingS>({ data: [], head: 0, len: 0 }).done("&Queue{}", "empty ring, no array"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap[T any](n int) *Queue[T]",
      doc: "Returns an empty queue with a ring of exactly n slots. Negative n is treated as 0.",
      cost: "O(n)",
      args: [{ name: "n", kind: "int", def: "6" }],
      run: (_, a) => {
        const n = Math.max(parseInt1(a[0]), 0);
        return rec<RingS>({ data: Array(n).fill(null), head: 0, len: 0 }).done("&Queue{}", `ring of ${n} slots, head 0`);
      },
    },
    ...sizeReads<RingS>({ recv: R, len: (s) => s.len, cap: (s) => s.data.length }),
    {
      name: "Peek",
      group: "read",
      sig: "func (q *Queue[T]) Peek() (T, bool)",
      doc: "Returns the front element (the next to Pop) without removing it.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.len) return r.done(okRet(0, false), "empty queue");
        const c = r.s.data[r.s.head]!;
        return r.done(okRet(c.v, true), `data[head] = data[${r.s.head}] = ${c.v}`, { ["c" + c.id]: "ok" });
      },
    },
    {
      name: "All",
      group: "iterate",
      sig: "func (q *Queue[T]) All() iter.Seq[T]",
      doc: "Iterates front to back (FIFO order), following the wrap, without consuming.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        iterate(r, false);
        return r.done("", "front to back, nothing removed");
      },
    },
    {
      name: "Clone",
      group: "copy",
      sig: "func (q *Queue[T]) Clone() *Queue[T]",
      doc: "Returns an independent copy, unwrapped to index 0, with cap == len.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        live(r.s).forEach((c) => r.emit(c.v, "new", "clone (front → back)"));
        return r.done(`&Queue{len ${r.s.len}, cap ${r.s.len}}`, "both segments copied in order into a fresh array");
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (q *Queue[T]) Slice() []T",
      doc: "Returns a copy of the elements front to back. Empty queue returns nil.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const cs = live(r.s);
        cs.forEach((c) => r.emit(c.v, undefined, "[]T (front first)"));
        return r.done(ringList(cs), "CopyTo: first segment, then the wrapped one");
      },
    },
    {
      name: "Push",
      group: "mutate",
      sig: "func (q *Queue[T]) Push(v T)",
      doc: "Adds v at the back. Grows (and unwraps) the ring when it is full.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: (s) => String(freshInt(live(s).map((c) => c.v))) }],
      run: (s0, a) => {
        const r = rec(s0);
        pushBack(r, parseInt1(a[0]));
        return r.done("", `${a[0]} waits at the back`);
      },
    },
    {
      name: "PushN",
      group: "mutate",
      sig: "func (q *Queue[T]) PushN(vs ...T)",
      doc: "Pushes vs in order (vs[0] leaves first among them). Grows at most once.",
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
      name: "Pop",
      group: "mutate",
      sig: "func (q *Queue[T]) Pop() (T, bool)",
      doc: "Removes and returns the front element; (zero, false) when empty.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const c = popFront(r, "popped");
        return r.done(okRet(c?.v, !!c), c ? undefined : "empty queue: nothing to pop");
      },
    },
    {
      name: "PopN",
      group: "mutate",
      sig: "func (q *Queue[T]) PopN(n int) []T",
      doc: "Removes up to n elements from the front, returned in FIFO order.",
      cost: "O(k)",
      args: [{ name: "n", kind: "int", def: "2" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = Math.min(parseInt1(a[0]), r.s.len);
        if (n <= 0) return r.done("nil", "nothing to pop: nil");
        const out: number[] = [];
        for (let i = 0; i < n; i++) out.push(popFront(r, "popped (FIFO)")!.v);
        return r.done(list(out));
      },
    },
    ...ringMemory(R),
  ],
};
