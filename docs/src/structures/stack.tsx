import { sizeReads, sliceMemory } from "../core/common";
import { appendCap } from "../core/gomem";
import {
  cell,
  freshInt,
  list,
  okRet,
  parseInt1,
  parseInts,
  randInts,
  rec,
  type Rec,
} from "../core/rec";
import type { Cell, StructureDef, Tone, ViewProps } from "../core/types";
import { ArrayView, CapBar, ColumnIndex, Label, Pointer, Stage, Tray, slotPos } from "../ui/viz";

export interface StackS {
  data: Cell[];
  cap: number;
}

const R = "s *Stack[T]";

function grow(r: Rec<StackS>, need: number) {
  if (need <= r.s.cap) return;
  const old = r.s.cap;
  const next = appendCap(old, need);
  r.s.cap = next;
  const marks: Record<string, Tone> = {};
  r.s.data.forEach((c) => (marks["c" + c.id] = "hot"));
  r.snap(
    old === 0
      ? `no backing array yet: append allocates one with cap ${next}`
      : `array is full (len ${r.s.data.length} = cap ${old}): append allocates cap ${next} and copies ${r.s.data.length} elements`,
    marks,
  );
}

const topMarks = (s: StackS, n: number, t: Tone) => {
  const m: Record<string, Tone> = {};
  for (let i = 0; i < n; i++) m["c" + s.data[s.data.length - 1 - i].id] = t;
  return m;
};

function View({ f }: ViewProps<StackS>) {
  const s = f.s;
  const rows = Math.max(s.cap, 7);
  const cw = s.cap > 12 ? 34 : 44;
  const gap = 6;
  const x0 = 120;
  const y0 = 30;
  const h = rows * (cw + gap) + 62;
  const geo = { x: x0, y: y0 + (rows - Math.max(s.cap, 1)) * (cw + gap), cw, gap, vertical: true, rows: s.cap };
  const top = s.data.length - 1;
  const topPos = top >= 0 ? slotPos(top, geo) : null;
  return (
    <div className="view">
      <Stage w={560} h={h}>
        <Label x={x0 - 6} y={h - 22} className="axis">
          data[0] · bottom
        </Label>
        <ColumnIndex x={x0 - 30} y={geo.y} cap={s.cap} cw={cw} gap={gap} />
        <ArrayView
          {...geo}
          cells={s.data}
          cap={s.cap}
          marks={f.marks}
        />
        {topPos && (
          <Pointer id="top" x={x0 + cw + 14} y={topPos.y + cw / 2 - 12} label="top = data[len-1]" dir="left" />
        )}
        {s.cap > s.data.length && (
          <Label x={x0 + cw + 18} y={slotPos(s.cap - 1, geo).y + 4} className="note-spare">
            spare capacity ({s.cap - s.data.length})
          </Label>
        )}
        {s.cap === 0 && (
          <Label x={x0} y={y0 + 40} className="note-spare">
            data == nil — no array allocated
          </Label>
        )}
      </Stage>
      <CapBar len={s.data.length} cap={s.cap} />
      <Tray f={f} />
    </div>
  );
}

export const stack: StructureDef<StackS> = {
  id: "stack",
  title: "stack",
  kind: "LIFO · slice-backed",
  tagline: "Last in, first out. Push and Pop both happen at the top — the end of a Go slice.",
  about: [
    "A Stack[T] is a single Go slice. The top of the stack is the last element, so Push is append and Pop reslices data[:len-1]: no shifting, amortized O(1).",
    "When the slice is full, append allocates a bigger array (about 2× while small) and copies the elements over — watch the capacity jump on Push.",
    "Popped slots are zeroed so the array does not keep references alive. The zero value is a ready-to-use empty stack.",
  ],
  cost: [
    ["Push", "amortized O(1)"],
    ["Pop / Peek", "O(1)"],
    ["PushN / PopN", "O(k)"],
    ["All / Slice / Clone", "O(n)"],
  ],
  seed: () => ({ data: [12, 7, 31].map(cell), cap: 4 }),
  stats: (s) => [
    ["len", s.data.length],
    ["cap", s.cap],
    ["top", s.data.length ? s.data[s.data.length - 1].v : "—"],
  ],
  View,
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New[T any]() *Stack[T]",
      doc: "Returns an empty stack with no preallocated capacity.",
      cost: "O(1)",
      run: () => rec<StackS>({ data: [], cap: 0 }).done("&Stack{}", "new empty stack: data is nil"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap[T any](n int) *Stack[T]",
      doc: "Returns an empty stack with room for n elements, so the first n pushes never reallocate. Negative n is treated as 0.",
      cost: "O(n)",
      args: [{ name: "n", kind: "int", def: "6" }],
      run: (_, a) => {
        const n = Math.max(parseInt1(a[0]), 0);
        return rec<StackS>({ data: [], cap: n }).done("&Stack{}", `make([]T, 0, ${n}): len 0, cap ${n}`);
      },
    },
    ...sizeReads<StackS>({ recv: R, len: (s) => s.data.length, cap: (s) => s.cap }),
    {
      name: "Peek",
      group: "read",
      sig: "func (s *Stack[T]) Peek() (T, bool)",
      doc: "Returns the top element without removing it; (zero, false) when empty.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.data.length) return r.done(okRet(0, false), "empty stack: nothing to peek");
        const t = r.s.data[r.s.data.length - 1];
        return r.done(okRet(t.v, true), `data[len-1] is ${t.v}; the stack is unchanged`, { ["c" + t.id]: "ok" });
      },
    },
    {
      name: "All",
      group: "iterate",
      sig: "func (s *Stack[T]) All() iter.Seq[T]",
      doc: "Iterates from top to bottom (LIFO order) without consuming the stack.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        for (let i = r.s.data.length - 1; i >= 0; i--) {
          const c = r.s.data[i];
          r.emit(c.v, undefined, "yielded");
          r.snap(`yield data[${i}] = ${c.v}`, { ["c" + c.id]: "hot" });
        }
        return r.done("", "range over All: top to bottom, nothing removed");
      },
    },
    {
      name: "Clone",
      group: "copy",
      sig: "func (s *Stack[T]) Clone() *Stack[T]",
      doc: "Returns an independent copy whose capacity equals its length.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data.forEach((c) => r.emit(c.v, "new", "clone (bottom → top)"));
        return r.done(`&Stack{len ${r.s.data.length}, cap ${r.s.data.length}}`, "make + copy: the clone shares no memory with the original");
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (s *Stack[T]) Slice() []T",
      doc: "Returns a copy of the elements from top to bottom, the order All yields. Empty stack returns nil.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const vs = r.s.data.map((c) => c.v).reverse();
        vs.forEach((v) => r.emit(v, undefined, "[]T (top first)"));
        return r.done(vs.length ? list(vs) : "nil", "slices.Clone then slices.Reverse");
      },
    },
    {
      name: "Push",
      group: "mutate",
      sig: "func (s *Stack[T]) Push(v T)",
      doc: "Adds v on top. append may reallocate the backing array when it is full.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: (s) => String(freshInt(s.data.map((c) => c.v))) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        grow(r, r.s.data.length + 1);
        const c = cell(v);
        r.s.data.push(c);
        return r.done("", `data = append(data, ${v}): ${v} is the new top`, { ["c" + c.id]: "new" });
      },
    },
    {
      name: "PushN",
      group: "mutate",
      sig: "func (s *Stack[T]) PushN(vs ...T)",
      doc: "Pushes vs in argument order, so the last argument ends on top. Grows at most once.",
      cost: "amortized O(k)",
      args: [{ name: "vs", kind: "ints", def: () => randInts(3).join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        const vs = parseInts(a[0]);
        if (!vs.length) return r.done("", "no arguments: no-op");
        grow(r, r.s.data.length + vs.length);
        const marks: Record<string, Tone> = {};
        for (const v of vs) {
          const c = cell(v);
          r.s.data.push(c);
          marks["c" + c.id] = "new";
        }
        return r.done("", `append(data, vs...): ${vs[vs.length - 1]} is the new top`, marks);
      },
    },
    {
      name: "Pop",
      group: "mutate",
      sig: "func (s *Stack[T]) Pop() (T, bool)",
      doc: "Removes and returns the top element; (zero, false) when empty. The vacated slot is zeroed.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.data.length) return r.done(okRet(0, false), "empty stack: nothing to pop");
        r.snap("take data[len-1]", topMarks(r.s, 1, "del"));
        const c = r.s.data.pop()!;
        r.emit(c.v, "del", "popped");
        return r.done(okRet(c.v, true), "zero the slot, data = data[:len-1]; cap is unchanged");
      },
    },
    {
      name: "PopN",
      group: "mutate",
      sig: "func (s *Stack[T]) PopN(n int) []T",
      doc: "Removes up to n elements from the top and returns them in LIFO order. Non-positive n or empty stack returns nil.",
      cost: "O(k)",
      args: [{ name: "n", kind: "int", def: "2" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = Math.min(parseInt1(a[0]), r.s.data.length);
        if (n <= 0) return r.done("nil", "nothing to pop: returns nil");
        r.snap(`take the top ${n}`, topMarks(r.s, n, "del"));
        const out: number[] = [];
        for (let i = 0; i < n; i++) {
          const c = r.s.data.pop()!;
          out.push(c.v);
          r.emit(c.v, "del", "popped (LIFO)");
        }
        return r.done(list(out), `zero ${n} slots, data = data[:len-${n}]`);
      },
    },
    ...sliceMemory<StackS>({ recv: R }),
  ],
};
