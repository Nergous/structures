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
import type { Cell, MethodDef, StructureDef, Tone, ViewProps } from "../core/types";
import { TreeView, treeHeight } from "../ui/tree";
import { ArrayView, CapBar, Label, Stage, Tray } from "../ui/viz";

type Mode = "min" | "max" | "func";

export interface HeapS {
  data: Cell[];
  cap: number;
  mode: Mode;
}

const R = "h *Heap[T]";
type H = Rec<HeapS>;
type Marks = Record<string, Tone>;

const CMP: Record<Mode, (a: number, b: number) => number> = {
  min: (a, b) => a - b,
  max: (a, b) => b - a,
  func: (a, b) => Math.abs(a - 50) - Math.abs(b - 50),
};
const LESS: Record<Mode, string> = { min: "<", max: ">", func: "closer to 50 than" };
const cmp = (s: HeapS, a: Cell, b: Cell) => CMP[s.mode](a.v, b.v);
const m2 = (a: Cell, ta: Tone, b?: Cell, tb?: Tone): Marks =>
  b ? { ["c" + a.id]: ta, ["c" + b.id]: tb ?? ta } : { ["c" + a.id]: ta };

function up(r: H, i: number) {
  const s = r.s.data;
  while (i > 0) {
    const p = (i - 1) >> 1;
    const x = s[i];
    const y = s[p];
    if (cmp(r.s, x, y) >= 0) {
      r.snap(`sift up: ${x.v} is not ${LESS[r.s.mode]} parent ${y.v} (data[${p}]) — stop`, m2(x, "ok", y, "cmp"));
      return;
    }
    r.snap(`sift up: ${x.v} ${LESS[r.s.mode]} parent ${y.v} → swap data[${i}] and data[${p}]`, m2(x, "hot", y, "cmp"));
    [s[i], s[p]] = [s[p], s[i]];
    r.snap(`${x.v} moved up to index ${p}`, m2(x, "hot"));
    i = p;
  }
  if (s.length) r.snap(`${s[i].v} reached the root`, m2(s[i], "ok"));
}

function down(r: H, i: number): boolean {
  const s = r.s.data;
  const i0 = i;
  const n = s.length;
  for (;;) {
    const l = 2 * i + 1;
    if (l >= n) {
      r.snap(`sift down: data[${i}] = ${s[i].v} is a leaf — stop`, m2(s[i], "ok"));
      break;
    }
    let c = l;
    const rr = l + 1;
    if (rr < n && cmp(r.s, s[rr], s[l]) < 0) c = rr;
    const marks = m2(s[i], "hot", s[l], "cmp");
    if (rr < n) marks["c" + s[rr].id] = "cmp";
    r.snap(
      rr < n
        ? `sift down: pick the better child of ${s[i].v}: ${s[c].v} (of ${s[l].v}, ${s[rr].v})`
        : `sift down: only child of ${s[i].v} is ${s[l].v}`,
      marks,
    );
    if (cmp(r.s, s[c], s[i]) >= 0) {
      r.snap(`${s[c].v} is not ${LESS[r.s.mode]} ${s[i].v}: heap order holds — stop`, m2(s[i], "ok", s[c], "cmp"));
      break;
    }
    const x = s[i];
    [s[i], s[c]] = [s[c], s[i]];
    r.snap(`swap: ${x.v} moves down to index ${c}`, m2(x, "hot", s[i], "cmp"));
    i = c;
  }
  return i > i0;
}

function heapify(r: H) {
  const n = r.s.data.length;
  if (n < 2) return;
  r.snap(`heapify: sift down every internal node, from index ${(n >> 1) - 1} to 0 — O(n) overall`);
  for (let i = (n >> 1) - 1; i >= 0; i--) down(r, i);
}

function removeAt(r: H, i: number): Cell {
  const s = r.s.data;
  const v = s[i];
  const last = s.length - 1;
  r.snap(`remove data[${i}] = ${v.v}`, m2(v, "del"));
  if (i !== last) {
    s[i] = s[last];
    s.pop();
    r.emit(v.v, "del");
    r.snap(`move the last element ${s[i].v} into the hole at ${i}; zero the tail slot`, m2(s[i], "hot"));
    if (!down(r, i)) {
      if (i > 0) up(r, i);
    }
  } else {
    s.pop();
    r.emit(v.v, "del");
    r.snap("it was the last element: just shorten the slice");
  }
  return v;
}

function grow(r: H, need: number) {
  if (need <= r.s.cap) return;
  const old = r.s.cap;
  r.s.cap = appendCap(old, need);
  r.snap(old ? `array full: append grows cap ${old} → ${r.s.cap}` : `append allocates an array with cap ${r.s.cap}`);
}

function rebuildCheaper(n: number, k: number) {
  if (k >= n) return true;
  return 2 * (n + k) < k * (n ? Math.floor(Math.log2(n)) + 1 : 0);
}

function View({ f }: ViewProps<HeapS>) {
  const s = f.s;
  const n = s.data.length;
  const W = Math.max(560, Math.min(2 ** Math.floor(Math.log2(Math.max(n, 1))), 16) * 52);
  const th = treeHeight(Math.max(n, 1), 70, 42);
  const cw = 40;
  const cols = 16;
  const ay = 40 + th + 50;
  const rows = Math.max(1, Math.ceil(Math.max(s.cap, 1) / cols));
  const H = ay + rows * (cw + 20) + 30;
  const tone = (c: Cell) => f.marks?.["c" + c.id];
  return (
    <div className="view">
      <Stage w={Math.max(W + 40, cols * 46 + 40)} h={H}>
        <Label x={20} y={6} className="title">
          {s.mode === "min" ? "min-heap · least on top" : s.mode === "max" ? "max-heap · greatest on top" : "custom · closest to 50 on top"}
        </Label>
        {n === 0 && (
          <Label x={20} y={50} className="note-spare">
            empty heap
          </Label>
        )}
        <TreeView
          x={20}
          y={40}
          w={W}
          nodes={s.data.map((c) => ({ key: c.id, label: c.v, tone: tone(c) }))}
        />
        <Label x={20} y={ay - 26} className="title">
          the same heap as a slice: children of i are 2i+1 and 2i+2
        </Label>
        <ArrayView x={20} y={ay} cells={s.data} cap={s.cap} cw={cw} cols={cols} marks={f.marks} ns="arr" />
      </Stage>
      <CapBar len={n} cap={s.cap} />
      <Tray f={f} />
    </div>
  );
}

const ctor = (name: string, mode: Mode, kind: "new" | "cap" | "from"): MethodDef<HeapS> => {
  const T = mode === "func" ? "T any" : "T cmp.Ordered";
  const cmpArg = mode === "func" ? "compare func(a, b T) int" : "";
  const sig =
    kind === "new"
      ? `func ${name}[${T}](${cmpArg}) *Heap[T]`
      : kind === "cap"
        ? `func ${name}[${T}](n int${cmpArg ? ", " + cmpArg : ""}) *Heap[T]`
        : `func ${name}[${T}](s []T${cmpArg ? ", " + cmpArg : ""}) *Heap[T]`;
  const order =
    mode === "min" ? "the least element on top" : mode === "max" ? "the greatest element on top" : "a custom comparator (here: |v-50|, closest to 50 on top)";
  return {
    name,
    group: "construct",
    sig,
    doc:
      kind === "new"
        ? `Returns an empty heap with ${order}.`
        : kind === "cap"
          ? `Returns an empty heap with ${order} and room for n elements.`
          : `Copies s and heapifies it in O(n), with ${order}. s is never aliased.`,
    cost: kind === "from" ? "O(n)" : "O(1)",
    args:
      kind === "cap"
        ? [{ name: "n", kind: "int", def: "8" }]
        : kind === "from"
          ? [{ name: "s", kind: "ints", def: () => randInts(9).join(", ") }]
          : undefined,
    run: (_, a) => {
      if (kind === "new") return rec<HeapS>({ data: [], cap: 0, mode }).done("&Heap{}", `empty ${mode} heap`);
      if (kind === "cap") {
        const n = Math.max(parseInt1(a[0]), 0);
        return rec<HeapS>({ data: [], cap: n, mode }).done("&Heap{}", `make([]T, 0, ${n})`);
      }
      const vs = parseInts(a[0]);
      const r = rec<HeapS>({ data: vs.map(cell), cap: vs.length, mode });
      r.snap("copy s into a new array — still in input order, not yet a heap");
      heapify(r);
      return r.done("&Heap{}", "heap order holds everywhere");
    },
  };
};

const freshV = (s: HeapS) => String(freshInt(s.data.map((c) => c.v)));

export const binheap: StructureDef<HeapS> = {
  id: "binheap",
  title: "binheap",
  kind: "binary heap · slice-backed",
  tagline: "Always know the smallest (or largest) element. The tree lives inside a plain slice.",
  about: [
    "A Heap[T] is a slice in heap order: every parent compares <= its children, so the top sits at data[0]. Children of index i live at 2i+1 and 2i+2 — no pointers.",
    "Push appends at the end and sifts the new element up, swapping with its parent while it is smaller. Pop moves the last element to the root and sifts it down, swapping with the smaller child.",
    "From and large PushN batches rebuild the whole heap bottom-up in O(n) instead (the Rust BinaryHeap::append heuristic). PushPop and Replace do a push and a pop with a single sift.",
  ],
  cost: [
    ["Peek", "O(1)"],
    ["Push / Pop", "O(log n)"],
    ["PushPop / Replace", "O(log n)"],
    ["From", "O(n)"],
    ["Contains / Remove", "O(n)"],
    ["Sorted", "O(n log n)"],
  ],
  seed: () => ({ data: [3, 9, 5, 14, 11, 8, 21].map(cell), cap: 8, mode: "min" }),
  stats: (s) => [
    ["len", s.data.length],
    ["cap", s.cap],
    ["order", s.mode],
    ["top", s.data.length ? s.data[0].v : "—"],
  ],
  View,
  methods: [
    ctor("New", "min", "new"),
    ctor("NewWithCap", "min", "cap"),
    ctor("From", "min", "from"),
    ctor("NewMax", "max", "new"),
    ctor("NewMaxWithCap", "max", "cap"),
    ctor("FromMax", "max", "from"),
    ctor("NewFunc", "func", "new"),
    ctor("NewFuncWithCap", "func", "cap"),
    ctor("FromFunc", "func", "from"),
    ...sizeReads<HeapS>({ recv: R, len: (s) => s.data.length, cap: (s) => s.cap }),
    {
      name: "Peek",
      group: "read",
      sig: "func (h *Heap[T]) Peek() (T, bool)",
      doc: "Returns the top element, data[0], without removing it.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.data.length) return r.done(okRet(0, false), "empty heap");
        return r.done(okRet(r.s.data[0].v, true), "the top is always data[0]", m2(r.s.data[0], "ok"));
      },
    },
    {
      name: "Contains",
      group: "read",
      sig: "func (h *Heap[T]) Contains(v T) bool",
      doc: "Reports whether an element comparing equal to v is present. A heap is not a search tree, so this is a linear scan.",
      cost: "O(n)",
      args: [{ name: "v", kind: "int", def: (s) => String(s.data[Math.floor(s.data.length / 2)]?.v ?? 5) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        for (const c of r.s.data) {
          if (CMP[r.s.mode](c.v, v) === 0) return r.done("true", `compare(${c.v}, ${v}) == 0: found`, m2(c, "ok"));
          r.snap(`scan: ${c.v} ≠ ${v}`, m2(c, "cmp"));
        }
        return r.done("false", "scanned every element: not found");
      },
    },
    {
      name: "All",
      group: "iterate",
      sig: "func (h *Heap[T]) All() iter.Seq[T]",
      doc: "Iterates in internal layout order: the first value is the top, the rest are NOT sorted.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data.forEach((c, i) => {
          r.emit(c.v, undefined, "yielded (layout order)");
          r.snap(`yield data[${i}] = ${c.v}`, m2(c, "hot"));
        });
        return r.done("", "layout order — use Sorted for priority order");
      },
    },
    {
      name: "Clone",
      group: "copy",
      sig: "func (h *Heap[T]) Clone() *Heap[T]",
      doc: "Returns an independent copy sharing the comparator; cap == len.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data.forEach((c) => r.emit(c.v, "new", "clone (layout order)"));
        return r.done(`&Heap{len ${r.s.data.length}, cap ${r.s.data.length}}`);
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (h *Heap[T]) Slice() []T",
      doc: "Returns a copy in layout order (top first, rest unspecified).",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const vs = r.s.data.map((c) => c.v);
        vs.forEach((v) => r.emit(v, undefined, "[]T (layout order)"));
        return r.done(vs.length ? list(vs) : "nil");
      },
    },
    {
      name: "Sorted",
      group: "copy",
      sig: "func (h *Heap[T]) Sorted() []T",
      doc: "Returns a sorted copy — the order repeated Pop would produce. The heap is unchanged.",
      cost: "O(n log n)",
      run: (s0) => {
        const r = rec(s0);
        const vs = r.s.data.map((c) => c.v).sort(CMP[r.s.mode]);
        r.snap("slices.Clone + slices.SortFunc with the heap's comparator");
        vs.forEach((v) => r.emit(v, "ok", "sorted copy"));
        return r.done(vs.length ? list(vs) : "nil");
      },
    },
    {
      name: "Push",
      group: "mutate",
      sig: "func (h *Heap[T]) Push(v T)",
      doc: "Appends v at the end of the slice (the next free leaf) and sifts it up.",
      cost: "O(log n)",
      args: [{ name: "v", kind: "int", def: freshV }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        grow(r, r.s.data.length + 1);
        const c = cell(v);
        r.s.data.push(c);
        r.snap(`append ${v} as the next leaf, index ${r.s.data.length - 1}`, m2(c, "new"));
        up(r, r.s.data.length - 1);
        return r.done("");
      },
    },
    {
      name: "PushN",
      group: "mutate",
      sig: "func (h *Heap[T]) PushN(vs ...T)",
      doc: "Appends vs (growing at most once), then either sifts each up or rebuilds everything — whichever needs fewer comparisons.",
      cost: "O(min(k log(n+k), n+k))",
      args: [{ name: "vs", kind: "ints", def: () => randInts(3).join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        const vs = parseInts(a[0]);
        if (!vs.length) return r.done("", "no arguments: no-op");
        const n = r.s.data.length;
        grow(r, n + vs.length);
        const marks: Marks = {};
        vs.forEach((v) => {
          const c = cell(v);
          r.s.data.push(c);
          marks["c" + c.id] = "new";
        });
        const rebuild = rebuildCheaper(n, vs.length);
        r.snap(
          rebuild
            ? `appended ${vs.length}; rebuildCheaper(${n}, ${vs.length}) is true → heapify the whole slice`
            : `appended ${vs.length}; small batch → sift each new element up`,
          marks,
        );
        if (rebuild) heapify(r);
        else for (let i = n; i < r.s.data.length; i++) up(r, i);
        return r.done("");
      },
    },
    {
      name: "Pop",
      group: "mutate",
      sig: "func (h *Heap[T]) Pop() (T, bool)",
      doc: "Removes the top: the last element takes its place and sifts down.",
      cost: "O(log n)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.data.length) return r.done(okRet(0, false), "empty heap");
        r.outLabel = "popped";
        const c = removeAt(r, 0);
        return r.done(okRet(c.v, true));
      },
    },
    {
      name: "PopN",
      group: "mutate",
      sig: "func (h *Heap[T]) PopN(n int) []T",
      doc: "Pops up to n elements in priority order; capacity is kept.",
      cost: "O(k log n)",
      args: [{ name: "n", kind: "int", def: "3" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = Math.min(parseInt1(a[0]), r.s.data.length);
        if (n <= 0) return r.done("nil", "nothing to pop: nil");
        r.outLabel = "popped (priority order)";
        const out: number[] = [];
        for (let i = 0; i < n; i++) out.push(removeAt(r, 0).v);
        return r.done(list(out));
      },
    },
    {
      name: "PushPop",
      group: "mutate",
      sig: "func (h *Heap[T]) PushPop(v T) T",
      doc: "Push then Pop in one sift. If v would be the top anyway, it is returned and the heap is untouched. Never grows.",
      cost: "O(log n)",
      args: [{ name: "v", kind: "int", def: freshV }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        const s = r.s.data;
        r.outLabel = "returned";
        if (!s.length || CMP[r.s.mode](v, s[0].v) <= 0) {
          r.emit(v, "ok");
          return r.done(String(v), s.length ? `${v} is not worse than the top ${s[0].v}: return v, heap untouched` : "empty heap: return v");
        }
        const old = s[0];
        r.snap(`${v} is worse than the top ${old.v}: it replaces the top`, m2(old, "del"));
        const c = cell(v);
        s[0] = c;
        r.emit(old.v, "del");
        r.snap(`data[0] = ${v}`, m2(c, "new"));
        down(r, 0);
        return r.done(String(old.v));
      },
    },
    {
      name: "Replace",
      group: "mutate",
      sig: "func (h *Heap[T]) Replace(v T) (T, bool)",
      doc: "Pop then Push in one sift: the old top is always returned, even if v is smaller. On an empty heap it just stores v.",
      cost: "O(log n)",
      args: [{ name: "v", kind: "int", def: freshV }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        const s = r.s.data;
        if (!s.length) {
          grow(r, 1);
          s.push(cell(v));
          return r.done("(0, false)", "empty heap: store v, nothing popped");
        }
        const old = s[0];
        r.snap(`take the top ${old.v}`, m2(old, "del"));
        const c = cell(v);
        s[0] = c;
        r.emit(old.v, "del", "popped");
        r.snap(`data[0] = ${v}`, m2(c, "new"));
        down(r, 0);
        return r.done(okRet(old.v, true));
      },
    },
    {
      name: "Remove",
      group: "mutate",
      sig: "func (h *Heap[T]) Remove(v T) bool",
      doc: "Finds one element equal to v by linear scan, then removes it in O(log n): the last element fills the hole and is fixed up or down.",
      cost: "O(n)",
      args: [{ name: "v", kind: "int", def: (s) => String(s.data[Math.max(0, s.data.length - 2)]?.v ?? 0) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        r.outLabel = "removed";
        const s = r.s.data;
        for (let i = 0; i < s.length; i++) {
          if (CMP[r.s.mode](s[i].v, v) === 0) {
            removeAt(r, i);
            return r.done("true");
          }
          r.snap(`scan: ${s[i].v} ≠ ${v}`, m2(s[i], "cmp"));
        }
        return r.done("false", "not found");
      },
    },
    ...sliceMemory<HeapS>({ recv: R }),
  ],
};
