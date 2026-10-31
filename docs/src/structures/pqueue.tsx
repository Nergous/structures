import { sizeReads } from "../core/common";
import { appendCap } from "../core/gomem";
import { nid, panic, parseInt1, rand, rec, type Rec } from "../core/rec";
import type { Cell, MethodDef, StructureDef, Tone } from "../core/types";

type Mode = "min" | "max" | "func";

/** A heap entry: v is the priority, k the key, seq the push order. */
export interface PE extends Cell {
  k: string;
  seq: number;
}

export interface PQS {
  data: (PE | null)[];
  held: { e: PE; at: number } | null;
  index: Record<string, number>;
  cap: number;
  mode: Mode;
  nextSeq: number;
}

const R = "q *Queue[K, P]";
type Q = Rec<PQS>;
type Marks = Record<string, Tone>;

const PCMP: Record<Mode, (a: number, b: number) => number> = {
  min: (a, b) => a - b,
  max: (a, b) => b - a,
  func: (a, b) => Math.abs(a - 50) - Math.abs(b - 50),
};
const cmp = (s: PQS, a: PE, b: PE) => PCMP[s.mode](a.v, b.v) || a.seq - b.seq;
const BETTER: Record<Mode, string> = { min: "smaller", max: "larger", func: "closer to 50" };

const KEYS = ["io", "db", "ui", "gc", "net", "log", "api", "cpu", "dns", "tls", "job", "mail", "cache", "auth"];
const pe = (k: string, p: number, seq: number): PE => ({ id: nid(), v: p, k, seq });
const mk = (e: PE, t: Tone): Marks => ({ ["c" + e.id]: t, ["k" + e.k]: t });
const lbl = (e: PE) => `${e.k}:${e.v}`;

function lift(r: Q, i: number, why: string) {
  const x = r.s.data[i]!;
  r.s.data[i] = null;
  r.s.held = { e: x, at: i };
  r.snap(`${why}: lift ${lbl(x)} out — a hole opens at ${i}`, mk(x, "held"));
  return x;
}

function drop(r: Q, x: PE, i: number) {
  r.s.data[i] = x;
  r.s.held = null;
  r.s.index[x.k] = i;
  r.snap(`store ${lbl(x)} once at index ${i}; moved() sets index["${x.k}"] = ${i}`, mk(x, "ok"));
}

function shift(r: Q, from: number, to: number, dir: string) {
  const e = r.s.data[from]!;
  r.s.data[to] = e;
  r.s.data[from] = null;
  r.s.index[e.k] = to;
  r.s.held!.at = from;
  r.snap(`${lbl(e)} shifts ${dir} into the hole; moved() sets index["${e.k}"] = ${to}`, mk(e, "cmp"));
}

function upMoved(r: Q, i: number) {
  const d = r.s.data;
  const x0 = d[i]!;
  if (i === 0 || cmp(r.s, x0, d[(i - 1) >> 1]!) >= 0) {
    r.snap(i === 0 ? `${lbl(x0)} is at the root` : `${lbl(x0)} is not ${BETTER[r.s.mode]} than its parent: stays at ${i}`, mk(x0, "ok"));
    return;
  }
  const x = lift(r, i, "sift up");
  while (i > 0) {
    const p = (i - 1) >> 1;
    const y = d[p]!;
    if (cmp(r.s, x, y) >= 0) {
      r.snap(`parent ${lbl(y)} is not worse than ${lbl(x)}: stop`, mk(y, "cmp"));
      break;
    }
    shift(r, p, i, "down");
    i = p;
  }
  drop(r, x, i);
}

function downMoved(r: Q, i: number): boolean {
  const d = r.s.data;
  const n = d.length;
  const best = (j: number) => {
    const l = 2 * j + 1;
    if (l >= n) return -1;
    const rr = l + 1;
    return rr < n && cmp(r.s, d[rr]!, d[l]!) < 0 ? rr : l;
  };
  const x0 = d[i]!;
  const c0 = best(i);
  if (c0 < 0 || cmp(r.s, d[c0]!, x0) >= 0) {
    r.snap(c0 < 0 ? `${lbl(x0)} is a leaf: no sift down` : `children are not ${BETTER[r.s.mode]}: ${lbl(x0)} stays`, mk(x0, "ok"));
    return false;
  }
  const x = lift(r, i, "sift down");
  for (;;) {
    const c = best(i);
    if (c < 0) break;
    if (cmp(r.s, d[c]!, x) >= 0) {
      r.snap(`best child ${lbl(d[c]!)} is not better than ${lbl(x)}: stop`, mk(d[c]!, "cmp"));
      break;
    }
    shift(r, c, i, "up");
    i = c;
  }
  drop(r, x, i);
  return true;
}

function fixMoved(r: Q, i: number) {
  if (!downMoved(r, i)) upMoved(r, i);
}

function removeAt(r: Q, i: number): PE {
  const d = r.s.data;
  const e = d[i]!;
  r.snap(`remove ${lbl(e)} at index ${i}; delete index["${e.k}"]`, mk(e, "del"));
  delete r.s.index[e.k];
  r.emit(lbl(e), "del");
  const last = d.length - 1;
  if (i !== last) {
    d[i] = d[last];
    d.pop();
    r.s.index[d[i]!.k] = i;
    r.snap(`the last entry ${lbl(d[i]!)} fills index ${i}; tail slot zeroed`, mk(d[i]!, "hot"));
    fixMoved(r, i);
  } else {
    d.pop();
    r.snap("it was the last entry: shorten the slice");
  }
  return e;
}

function push(r: Q, k: string, p: number): boolean {
  const s = r.s;
  if (k in s.index) {
    const i = s.index[k];
    const e = s.data[i]!;
    const old = e.v;
    e.v = p;
    r.snap(`"${k}" is already queued at ${i}: priority ${old} → ${p} in place (seq ${e.seq} kept)`, mk(e, "hot"));
    fixMoved(r, i);
    return false;
  }
  if (s.data.length + 1 > s.cap) {
    const old = s.cap;
    s.cap = appendCap(old, s.data.length + 1, 32);
    r.snap(old ? `array full: append grows cap ${old} → ${s.cap}` : `append allocates cap ${s.cap}`);
  }
  const e = pe(k, p, s.nextSeq++);
  s.index[k] = s.data.length;
  s.data.push(e);
  r.snap(`append ${k}:${p} (seq ${e.seq}) at index ${s.data.length - 1}; index["${k}"] = ${s.data.length - 1}`, mk(e, "new"));
  upMoved(r, s.data.length - 1);
  return true;
}

function parseItems(raw: string): [string, number][] {
  return raw
    .split(/[\s,]+/)
    .filter(Boolean)
    .map((t) => {
      const [k, p] = t.split(":");
      const n = Number(p);
      if (!k || !Number.isInteger(n)) panic(`expected key:priority, got ${t}`);
      return [k, n];
    });
}

const freeKey = (s: PQS) => KEYS.find((k) => !(k in s.index)) ?? "k" + rand(10, 99);

function seedFrom(items: [string, number][], mode: Mode): Q {
  const r = rec<PQS>({ data: [], held: null, index: {}, cap: items.length, mode, nextSeq: 0 });
  const s = r.s;
  for (const [k, p] of items) {
    if (k in s.index) {
      s.data[s.index[k]]!.v = p;
      continue;
    }
    s.index[k] = s.data.length;
    s.data.push(pe(k, p, s.nextSeq++));
  }
  return r;
}

function heapifyMoved(r: Q) {
  const n = r.s.data.length;
  if (n < 2) return;
  r.snap("heapify: sift down each internal node from the bottom up");
  for (let i = (n >> 1) - 1; i >= 0; i--) downMoved(r, i);
}


const ctor = (name: string, mode: Mode, kind: "new" | "cap" | "from"): MethodDef<PQS> => {
  const P = mode === "func" ? "P any" : "P cmp.Ordered";
  const c = mode === "func" ? "compare func(a, b P) int" : "";
  const params = kind === "new" ? c : kind === "cap" ? `n int${c ? ", " + c : ""}` : `items []Item[K, P]${c ? ", " + c : ""}`;
  return {
    name,
    group: "construct",
    sig: `func ${name}[K comparable, ${P}](${params}) *Queue[K, P]`,
    doc:
      (kind === "from" ? "Builds a queue from items in O(n) (duplicate keys: last priority wins, first seq kept). " : kind === "cap" ? "Empty queue with room for n keys. " : "Empty queue. ") +
      (mode === "min" ? "Least priority on top." : mode === "max" ? "Greatest priority on top." : "Custom comparator — here |p-50|."),
    cost: kind === "from" ? "O(n)" : "O(1)",
    args:
      kind === "cap"
        ? [{ name: "n", kind: "int", def: "6" }]
        : kind === "from"
          ? [{ name: "items", kind: "ints", hint: "key:prio, …", def: () => KEYS.slice(0, 7).map((k) => `${k}:${rand(1, 60)}`).join(", ") }]
          : undefined,
    run: (_, a) => {
      if (kind === "from") {
        const r = seedFrom(parseItems(a[0]), mode);
        r.snap("copied items in order with fresh seq numbers — not yet a heap");
        heapifyMoved(r);
        return r.done("&Queue{}");
      }
      const n = kind === "cap" ? Math.max(parseInt1(a[0]), 0) : 0;
      return rec<PQS>({ data: [], held: null, index: {}, cap: n, mode, nextSeq: 0 }).done("&Queue{}", "empty priority queue");
    },
  };
};

export const pqueue: StructureDef<PQS> = {
  id: "pqueue",
  title: "pqueue",
  kind: "keyed priority queue · heap + index",
  tagline: "A heap of keys with priorities — and a map that always knows where each key sits, so priorities can change.",
  about: [
    "A Queue[K, P] is a binary heap of entries {key, priority, seq} plus index map[K]int giving each key's slot. Push of an existing key changes its priority in place and re-sifts it; Remove(key) is O(log n) — no scan.",
    "To keep the index right, sifts use the hole-shift variants of heapcore: the moving entry is lifted out, neighbours slide into the hole one by one (each slide calls moved() to update the index), and the entry is stored once at its final slot.",
    "seq records push order and breaks ties between equal priorities, so the queue is stable: equal priorities pop first-pushed-first.",
  ],
  cost: [
    ["Push (new or update)", "O(log n)"],
    ["Pop / Remove(key)", "O(log n)"],
    ["Peek / Contains / Priority", "O(1)"],
    ["From", "O(n)"],
    ["Sorted", "O(n log n)"],
  ],
  seed: () => {
    const r = seedFrom([["io", 4], ["db", 9], ["ui", 7], ["gc", 15], ["net", 12], ["log", 20]], "min");
    for (let i = (r.s.data.length >> 1) - 1; i >= 0; i--) downMoved(r, i);
    return { ...r.s, cap: 8 };
  },
  stats: (s) => [
    ["len", s.data.length],
    ["cap", s.cap],
    ["order", s.mode],
    ["top", s.data[0] ? lbl(s.data[0]) : "—"],
  ],
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
    ...sizeReads<PQS>({ recv: R, len: (s) => s.data.length, cap: (s) => s.cap, unit: "keys" }),
    {
      name: "Peek",
      group: "read",
      sig: "func (q *Queue[K, P]) Peek() (K, P, bool)",
      doc: "Returns the top key and its priority without removing them.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const e = r.s.data[0];
        if (!e) return r.done('("", 0, false)', "empty queue");
        return r.done(`(${e.k}, ${e.v}, true)`, "data[0] is the top", mk(e, "ok"));
      },
    },
    {
      name: "Contains",
      group: "read",
      sig: "func (q *Queue[K, P]) Contains(key K) bool",
      doc: "Reports whether key is queued — a map lookup, not a scan.",
      cost: "O(1)",
      args: [{ name: "key", kind: "key", def: (s) => Object.keys(s.index)[1] ?? "io" }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        if (!(k in r.s.index)) return r.done("false", `index["${k}"] is missing`);
        const e = r.s.data[r.s.index[k]]!;
        return r.done("true", `index["${k}"] = ${r.s.index[k]}`, mk(e, "ok"));
      },
    },
    {
      name: "Priority",
      group: "read",
      sig: "func (q *Queue[K, P]) Priority(key K) (P, bool)",
      doc: "Returns key's current priority via the index.",
      cost: "O(1)",
      args: [{ name: "key", kind: "key", def: (s) => Object.keys(s.index)[2] ?? "io" }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        if (!(k in r.s.index)) return r.done("(0, false)", `"${k}" is not queued`);
        const e = r.s.data[r.s.index[k]]!;
        return r.done(`(${e.v}, true)`, `index["${k}"] = ${r.s.index[k]} → priority ${e.v}`, mk(e, "ok"));
      },
    },
    {
      name: "All",
      group: "iterate",
      sig: "func (q *Queue[K, P]) All() iter.Seq2[K, P]",
      doc: "Iterates key/priority pairs in heap layout order (top first, rest unsorted).",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data.forEach((e, i) => {
          r.emit(lbl(e!), undefined, "yielded (layout order)");
          r.snap(`yield data[${i}]`, mk(e!, "hot"));
        });
        return r.done("");
      },
    },
    {
      name: "Clone",
      group: "copy",
      sig: "func (q *Queue[K, P]) Clone() *Queue[K, P]",
      doc: "Independent copy with its own index and its own moved() hook bound to the clone.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data.forEach((e) => r.emit(lbl(e!), "new", "clone"));
        return r.done(`&Queue{len ${r.s.data.length}}`, "copy the entries, rebuild the index, bind a new moved hook");
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (q *Queue[K, P]) Slice() []Item[K, P]",
      doc: "Copy of the items in layout order.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data.forEach((e) => r.emit(lbl(e!), undefined, "[]Item (layout order)"));
        return r.done(r.s.data.length ? `[${r.s.data.map((e) => `{${e!.k} ${e!.v}}`).join(" ")}]` : "nil");
      },
    },
    {
      name: "Sorted",
      group: "copy",
      sig: "func (q *Queue[K, P]) Sorted() []Item[K, P]",
      doc: "Items in the exact order repeated Pop returns them, including seq tie-breaking.",
      cost: "O(n log n)",
      run: (s0) => {
        const r = rec(s0);
        const sorted = [...r.s.data].sort((a, b) => cmp(r.s, a!, b!));
        sorted.forEach((e) => r.emit(lbl(e!), "ok", "sorted (pop order)"));
        return r.done(sorted.length ? `[${sorted.map((e) => `{${e!.k} ${e!.v}}`).join(" ")}]` : "nil", "sort a copy by (priority, seq)");
      },
    },
    {
      name: "Push",
      group: "mutate",
      sig: "func (q *Queue[K, P]) Push(key K, priority P) bool",
      doc: "Adds key, or — if it is already queued — changes its priority in place and re-sifts it. Returns whether key was new.",
      cost: "O(log n)",
      args: [
        { name: "key", kind: "key", def: freeKey },
        { name: "prio", kind: "int", def: () => String(rand(1, 30)) },
      ],
      run: (s0, a) => {
        const r = rec(s0);
        const isNew = push(r, a[0].trim(), parseInt1(a[1]));
        return r.done(String(isNew));
      },
    },
    {
      name: "PushN",
      group: "mutate",
      sig: "func (q *Queue[K, P]) PushN(items ...Item[K, P])",
      doc: "Pushes items in order with Push semantics (a repeated key: last occurrence wins). Grows at most once.",
      cost: "O(k log(n+k))",
      args: [{ name: "items", kind: "ints", hint: "key:prio, …", def: (s) => `${freeKey(s)}:${rand(1, 30)}, ${Object.keys(s.index)[0] ?? "io"}:${rand(1, 30)}` }],
      run: (s0, a) => {
        const r = rec(s0);
        const items = parseItems(a[0]);
        if (!items.length) return r.done("", "no items: no-op");
        const need = r.s.data.length + items.length;
        if (need > r.s.cap) {
          r.s.cap = appendCap(r.s.cap, need, 32);
          r.snap(`slices.Grow once: cap ${r.s.cap}`);
        }
        for (const [k, p] of items) push(r, k, p);
        return r.done("");
      },
    },
    {
      name: "Pop",
      group: "mutate",
      sig: "func (q *Queue[K, P]) Pop() (K, P, bool)",
      doc: "Removes the top key and returns it with its priority.",
      cost: "O(log n)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.data.length) return r.done('("", 0, false)', "empty queue");
        r.outLabel = "popped";
        const e = removeAt(r, 0);
        return r.done(`(${e.k}, ${e.v}, true)`);
      },
    },
    {
      name: "PopN",
      group: "mutate",
      sig: "func (q *Queue[K, P]) PopN(n int) []Item[K, P]",
      doc: "Pops up to n items in priority order.",
      cost: "O(k log n)",
      args: [{ name: "n", kind: "int", def: "2" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = Math.min(parseInt1(a[0]), r.s.data.length);
        if (n <= 0) return r.done("nil", "nothing to pop");
        r.outLabel = "popped (priority order)";
        const out: PE[] = [];
        for (let i = 0; i < n; i++) out.push(removeAt(r, 0));
        return r.done(`[${out.map((e) => `{${e.k} ${e.v}}`).join(" ")}]`);
      },
    },
    {
      name: "Remove",
      group: "mutate",
      sig: "func (q *Queue[K, P]) Remove(key K) (P, bool)",
      doc: "Deletes key wherever it is: the index finds its slot in O(1), the last entry fills it and is fixed up or down.",
      cost: "O(log n)",
      args: [{ name: "key", kind: "key", def: (s) => Object.keys(s.index)[1] ?? "io" }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        if (!(k in r.s.index)) return r.done("(0, false)", `"${k}" is not queued`);
        r.outLabel = "removed";
        r.snap(`index["${k}"] = ${r.s.index[k]} — no scan needed`, mk(r.s.data[r.s.index[k]]!, "hot"));
        const e = removeAt(r, r.s.index[k]);
        return r.done(`(${e.v}, true)`);
      },
    },
    {
      name: "Clear",
      group: "memory",
      sig: "func (q *Queue[K, P]) Clear()",
      doc: "Drops the array and the index (cap 0) and restarts the seq counter. The comparator stays.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data = [];
        r.s.index = {};
        r.s.cap = 0;
        r.s.nextSeq = 0;
        return r.done("", "data = nil, index = nil (re-created lazily on the next Push), nextSeq = 0");
      },
    },
    {
      name: "Reset",
      group: "memory",
      sig: "func (q *Queue[K, P]) Reset()",
      doc: "Empties the queue but keeps the array (cap) for reuse; entries are zeroed and the index cleared.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const m: Marks = {};
        r.s.data.forEach((e) => Object.assign(m, mk(e!, "del")));
        r.snap("zero every entry, clear(index)", m);
        r.s.data = [];
        r.s.index = {};
        r.s.nextSeq = 0;
        return r.done("", `len 0, cap stays ${r.s.cap}; nextSeq = 0`);
      },
    },
    {
      name: "Shrink",
      group: "memory",
      sig: "func (q *Queue[K, P]) Shrink()",
      doc: "Copies entries into a right-sized array and rebuilds the index map (Go maps never shrink in place).",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const copied = r.s.cap !== r.s.data.length;
        r.s.cap = r.s.data.length;
        return r.done("", `${copied ? `copied into a cap-${r.s.cap} array` : "array already right-sized"}; index rebuilt from scratch`);
      },
    },
    {
      name: "Clip",
      group: "memory",
      sig: "func (q *Queue[K, P]) Clip()",
      doc: "Reslices the entry array to cap == len without copying; the index is untouched.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (r.s.cap === r.s.data.length) return r.done("", "already clipped: no-op");
        r.s.cap = r.s.data.length;
        return r.done("", "slices.Clip: no copy");
      },
    },
    {
      name: "Grow",
      group: "memory",
      sig: "func (q *Queue[K, P]) Grow(n int)",
      doc: "Ensures room for n more keys in the entry array.",
      cost: "O(n) when it reallocates",
      args: [{ name: "n", kind: "int", def: "4" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = parseInt1(a[0]);
        const len = r.s.data.length;
        if (n <= 0 || r.s.cap - len >= n) return r.done("", "enough room: no-op");
        r.s.cap = appendCap(r.s.cap, len + n, 32);
        return r.done("", `cap ${r.s.cap}`);
      },
    },
  ],
};
