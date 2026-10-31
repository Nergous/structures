import { appendCap } from "../core/gomem";
import { PREDICATES, distinctInts, freshInt, list, okRet, panic, parseInt1, parseInts, rec, type Rec } from "../core/rec";
import type { MethodDef, StructureDef, Tone } from "../core/types";

export interface BitS {
  data: bigint[]; // full backing array: data.length is cap (in words)
  len: number; // words in use
  n: number;
  b: number[];
  res: bigint[] | null;
  resLabel: string;
}

type B = Rec<BitS>;
type Marks = Record<string, Tone>;
const ONE = 1n;
const ALL = (1n << 64n) - 1n;

const word = (v: number) => v >> 6;
const bit = (v: number) => ONE << BigInt(v & 63);
const pop = (w: bigint) => {
  let c = 0;
  for (let x = w; x; x &= x - ONE) c++;
  return c;
};
const count = (ws: bigint[]) => ws.reduce((a, w) => a + pop(w), 0);
const valsOf = (ws: bigint[]) => {
  const out: number[] = [];
  ws.forEach((w, i) => {
    for (let b = 0; b < 64; b++) if ((w >> BigInt(b)) & ONE) out.push(i * 64 + b);
  });
  return out;
};
const used = (s: BitS) => s.data.slice(0, s.len);
const has = (s: BitS, v: number) => v >= 0 && word(v) < s.len && (s.data[word(v)] & bit(v)) !== 0n;
const hex = (w: bigint) => "0x" + w.toString(16).padStart(16, "0");
const wordsOf = (vs: number[]) => {
  const ws: bigint[] = [];
  for (const v of vs) {
    while (ws.length <= word(v)) ws.push(0n);
    ws[word(v)] |= bit(v);
  }
  while (ws.length && ws[ws.length - 1] === 0n) ws.pop();
  return ws;
};
const bm = (v: number, t: Tone): Marks => ({ ["b" + v]: t, ["w" + word(v)]: t });

function grow(r: B, k: number) {
  const s = r.s;
  if (s.len >= k) return;
  if (k > s.data.length) {
    const old = s.data.length;
    const cap = appendCap(old, k, 8);
    while (s.data.length < cap) s.data.push(0n);
    r.snap(old ? `need ${k} words but cap is ${old}: reallocate to ${cap} words` : `allocate ${cap} word${cap > 1 ? "s" : ""}`);
  }
  const stale = s.data.slice(s.len, k).some((w) => w !== 0n);
  for (let i = s.len; i < k; i++) s.data[i] = 0n;
  const m: Marks = {};
  for (let i = s.len; i < k; i++) m["w" + i] = "new";
  s.len = k;
  r.snap(stale ? `extend to ${k} words and clear them — they held stale bits from an earlier truncation` : `extend to ${k} words (zeroed)`, m);
}

function trim(r: B, why = "trailing empty words dropped (capacity kept)") {
  const s = r.s;
  const before = s.len;
  while (s.len > 0 && s.data[s.len - 1] === 0n) s.len--;
  if (s.len !== before) r.snap(why);
}

function setOther(r: B, raw: string) {
  const vs = parseInts(raw);
  if (vs.some((v) => v < 0)) panic("negative values are never present");
  r.s.b = [...new Set(vs)].sort((a, b) => a - b);
  r.s.res = null;
}

function insert(r: B, v: number): boolean {
  const s = r.s;
  if (s.data[word(v)] & bit(v)) {
    r.snap(`bit ${v & 63} of word ${word(v)} is already set: ${v} present`, bm(v, "dim"));
    return false;
  }
  s.data[word(v)] |= bit(v);
  s.n++;
  r.snap(`data[${word(v)}] |= 1<<${v & 63}`, bm(v, "new"));
  return true;
}



const otherArg = { name: "other", kind: "ints" as const, def: (s: BitS) => s.b.join(", ") };
const freshV = (s: BitS) => String(freshInt(valsOf(used(s)), 0, 190));

function wordwise(name: string, doc: string, op: (a: bigint, b: bigint) => bigint, sym: string, len: (a: number, b: number) => number, variadic = true): MethodDef<BitS> {
  return {
    name,
    group: "algebra",
    sig: variadic ? `func (s *Set) ${name}(others ...*Set) *Set` : `func (s *Set) ${name}(other *Set) *Set`,
    doc,
    cost: "O(m/64)",
    args: [otherArg],
    run: (s0, a) => {
      const r = rec(s0);
      setOther(r, a[0]);
      const A = used(r.s);
      const Bw = wordsOf(r.s.b);
      const k = len(A.length, Bw.length);
      r.s.res = Array(k).fill(0n);
      r.s.resLabel = `${name} → new set, word by word (${sym})`;
      r.snap(`allocate the result once: ${k} word${k === 1 ? "" : "s"}`);
      for (let i = 0; i < k; i++) {
        r.s.res[i] = op(A[i] ?? 0n, Bw[i] ?? 0n) & ALL;
        r.snap(`word ${i}: ${hex(A[i] ?? 0n)} ${sym} ${hex(Bw[i] ?? 0n)} = ${hex(r.s.res[i])}`, { ["w" + i]: "hot" });
      }
      while (r.s.res.length && r.s.res[r.s.res.length - 1] === 0n) r.s.res.pop();
      const vs = valsOf(r.s.res);
      return r.done(vs.length ? `{${vs.join(" ")}}` : "{}", `n = popcount of the result = ${count(r.s.res)}`);
    },
  };
}

export const bitset: StructureDef<BitS> = {
  id: "bitset",
  title: "bitset",
  kind: "bit vector of ints",
  tagline: "A set of small non-negative ints packed one bit each into 64-bit words. Set algebra is a handful of OR / AND / XOR instructions.",
  about: [
    "Value v lives at bit v%64 of word v/64. Contains and Remove are a shift and a mask — O(1). Memory grows with the largest value, not with how many values you store: {1000000} alone costs ~122 KiB.",
    "The last word in use is never zero: removing the largest value trims empty words but keeps the capacity. Words past len(data) may still hold stale bits (Reset and RetainSet just reslice); grow clears them before reuse — watch the amber bits.",
    "Union, Intersection, Difference, and SymmetricDifference combine whole words, so 64 values are processed per step. Len is a stored counter; Max is O(1) because the last word is non-zero.",
  ],
  cost: [
    ["Contains / Remove", "O(1)"],
    ["Add", "O(1) once the word exists"],
    ["Len / Max", "O(1)"],
    ["Min", "O(m/64)"],
    ["set algebra", "O(m/64)"],
    ["All / Slice", "O(m/64 + n)"],
  ],
  seed: () => {
    const ws = wordsOf([1, 3, 6, 10, 15, 64, 70, 130]);
    return { data: [...ws, 0n], len: ws.length, n: count(ws), b: [3, 10, 64, 100, 131], res: null, resLabel: "" };
  },
  stats: (s) => {
    const vs = valsOf(used(s));
    return [
      ["len", s.n],
      ["words", `${s.len}/${s.data.length}`],
      ["min", vs[0] ?? "—"],
      ["max", vs[vs.length - 1] ?? "—"],
    ];
  },
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New() *Set",
      doc: "Returns an empty set.",
      cost: "O(1)",
      run: (s0) => rec<BitS>({ ...s0, data: [], len: 0, n: 0, res: null }).done("&Set{}"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap(n int) *Set",
      doc: "Preallocates words for values 0..n-1 (rounded up to a multiple of 64).",
      cost: "O(n/64)",
      args: [{ name: "n", kind: "int", def: "200" }],
      run: (s0, a) => {
        const n = Math.max(parseInt1(a[0]), 0);
        const w = Math.ceil(n / 64);
        return rec<BitS>({ ...s0, data: Array(w).fill(0n), len: 0, n: 0, res: null }).done("&Set{}", `${w} words reserved: Cap() = ${w * 64}`);
      },
    },
    ...(["Of", "From"] as const).map(
      (name): MethodDef<BitS> => ({
        name,
        group: "construct",
        sig: name === "Of" ? "func Of(vs ...int) *Set" : "func From(s []int) *Set",
        doc: "New set with the values; grows once to cover the largest. Panics on a negative value.",
        cost: "O(k + m/64)",
        args: [{ name: name === "Of" ? "vs" : "s", kind: "ints", def: () => distinctInts(6, 0, 150).join(", ") }],
        run: (s0, a) => {
          const vs = parseInts(a[0]);
          if (vs.some((v) => v < 0)) panic(`bitset: AddN of negative value`);
          const r = rec<BitS>({ ...s0, data: [], len: 0, n: 0, res: null });
          if (vs.length) {
            const m = Math.max(...vs);
            r.snap(`largest value ${m} → needs ${word(m) + 1} words`);
            grow(r, word(m) + 1);
            for (const v of vs) insert(r, v);
          }
          return r.done("&Set{}");
        },
      }),
    ),
    {
      name: "Len",
      group: "read",
      sig: "func (s *Set) Len() int",
      doc: "Number of values — a stored counter, not a popcount.",
      cost: "O(1)",
      run: (s0) => rec(s0).done(String(s0.n), `n = ${s0.n}`),
    },
    {
      name: "Cap",
      group: "read",
      sig: "func (s *Set) Cap() int",
      doc: "Values 0..Cap()-1 fit without reallocating: cap(data) * 64. It bounds the largest value, not the count.",
      cost: "O(1)",
      run: (s0) => rec(s0).done(String(s0.data.length * 64), `${s0.data.length} words × 64`),
    },
    {
      name: "IsEmpty",
      group: "read",
      sig: "func (s *Set) IsEmpty() bool",
      doc: "Reports whether n == 0.",
      cost: "O(1)",
      run: (s0) => rec(s0).done(String(s0.n === 0)),
    },
    {
      name: "Contains",
      group: "read",
      sig: "func (s *Set) Contains(v int) bool",
      doc: "Shift and mask: word v>>6, bit v&63. Negative values are never present.",
      cost: "O(1)",
      args: [{ name: "v", kind: "int", def: (s) => String(valsOf(used(s))[2] ?? 3) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        if (v < 0) return r.done("false", "negative: never present");
        if (word(v) >= r.s.len) return r.done("false", `word ${word(v)} is beyond the ${r.s.len} words in use`);
        r.snap(`v>>6 = ${word(v)}, v&63 = ${v & 63}`, { ["w" + word(v)]: "cmp" });
        const ok = has(r.s, v);
        return r.done(String(ok), ok ? "bit is set" : "bit is clear", bm(v, ok ? "ok" : "del"));
      },
    },
    ...(["ContainsAll", "ContainsAny"] as const).map(
      (name): MethodDef<BitS> => ({
        name,
        group: "read",
        sig: `func (s *Set) ${name}(vs ...int) bool`,
        doc: name === "ContainsAll" ? "All present? Stops at the first miss." : "Any present? Stops at the first hit.",
        cost: "O(k)",
        args: [{ name: "vs", kind: "ints", def: (s) => [valsOf(used(s))[0] ?? 1, freshV(s)].join(", ") }],
        run: (s0, a) => {
          const r = rec(s0);
          const all = name === "ContainsAll";
          for (const v of parseInts(a[0])) {
            const ok = has(r.s, v);
            if (all && !ok) return r.done("false", `${v} missing: stop`, v >= 0 ? bm(v, "del") : undefined);
            if (!all && ok) return r.done("true", `${v} present: stop`, bm(v, "ok"));
            r.snap(`${v} ${ok ? "present" : "absent"}`, v >= 0 ? bm(v, ok ? "ok" : "dim") : undefined);
          }
          return r.done(String(all));
        },
      }),
    ),
    {
      name: "Min",
      group: "read",
      sig: "func (s *Set) Min() (int, bool)",
      doc: "Scans words from the start for the first non-zero one, then counts trailing zeros.",
      cost: "O(m/64)",
      run: (s0) => {
        const r = rec(s0);
        for (let i = 0; i < r.s.len; i++) {
          if (r.s.data[i] === 0n) {
            r.snap(`word ${i} is zero: next`, { ["w" + i]: "dim" });
            continue;
          }
          const v = valsOf([r.s.data[i]])[0] + i * 64;
          return r.done(okRet(v, true), `TrailingZeros64(word ${i}) → ${v}`, bm(v, "ok"));
        }
        return r.done("(0, false)", "empty set");
      },
    },
    {
      name: "Max",
      group: "read",
      sig: "func (s *Set) Max() (int, bool)",
      doc: "The last word is never zero, so Max reads only it: 64*i + 63 - LeadingZeros64.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.len) return r.done("(0, false)", "empty set");
        const vs = valsOf([r.s.data[r.s.len - 1]]);
        const v = vs[vs.length - 1] + (r.s.len - 1) * 64;
        return r.done(okRet(v, true), `last word ${r.s.len - 1} → highest set bit is ${v}`, bm(v, "ok"));
      },
    },
    ...(["All", "Backward"] as const).map(
      (name): MethodDef<BitS> => ({
        name,
        group: "iterate",
        sig: `func (s *Set) ${name}() iter.Seq[int]`,
        doc: name === "All" ? "Ascending order, skipping zero words." : "Descending order.",
        cost: "O(m/64 + n)",
        run: (s0) => {
          const r = rec(s0);
          let vs = valsOf(used(r.s));
          if (name === "Backward") vs = vs.reverse();
          for (const v of vs) {
            r.emit(v, undefined, "yielded");
            r.snap(`yield ${v}`, bm(v, "hot"));
          }
          return r.done("");
        },
      }),
    ),
    {
      name: "Clone",
      group: "copy",
      sig: "func (s *Set) Clone() *Set",
      doc: "Copies the words in use into a right-sized array.",
      cost: "O(m/64)",
      run: (s0) => {
        const r = rec(s0);
        valsOf(used(r.s)).forEach((v) => r.emit(v, "new", "clone"));
        return r.done(`&Set{len ${r.s.n}}`, `${r.s.len} words copied`);
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (s *Set) Slice() []int",
      doc: "Values in ascending order. Empty set returns nil.",
      cost: "O(m/64 + n)",
      run: (s0) => {
        const r = rec(s0);
        const vs = valsOf(used(r.s));
        vs.forEach((v) => r.emit(v, undefined, "[]int (ascending)"));
        return r.done(vs.length ? list(vs) : "nil");
      },
    },
    {
      name: "Add",
      group: "mutate",
      sig: "func (s *Set) Add(v int) bool",
      doc: "Sets bit v, first extending the array to cover word v/64 if needed. Panics on negative v.",
      cost: "O(1), O(v/64) to extend",
      args: [{ name: "v", kind: "int", def: freshV }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        if (v < 0) panic("bitset: Add of negative value");
        r.s.res = null;
        r.snap(`${v} → word ${v} >> 6 = ${word(v)}, bit ${v} & 63 = ${v & 63}`);
        grow(r, word(v) + 1);
        return r.done(String(insert(r, v)));
      },
    },
    {
      name: "AddN",
      group: "mutate",
      sig: "func (s *Set) AddN(vs ...int)",
      doc: "Validates all values first, extends once to the largest, then sets each bit.",
      cost: "O(k + m/64)",
      args: [{ name: "vs", kind: "ints", def: (s) => [freshV(s), freshV(s), freshV(s)].join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        const vs = parseInts(a[0]);
        if (vs.some((v) => v < 0)) panic("bitset: AddN of negative value");
        if (!vs.length) return r.done("", "no arguments: no-op");
        r.s.res = null;
        grow(r, word(Math.max(...vs)) + 1);
        for (const v of vs) insert(r, v);
        return r.done("");
      },
    },
    {
      name: "AddSet",
      group: "algebra",
      sig: "func (s *Set) AddSet(other *Set)",
      doc: "In-place union: extend once, then data[i] |= other[i] for each word.",
      cost: "O(m/64)",
      args: [otherArg],
      run: (s0, a) => {
        const r = rec(s0);
        setOther(r, a[0]);
        const Bw = wordsOf(r.s.b);
        if (!Bw.length) return r.done("", "other is empty: no-op");
        grow(r, Bw.length);
        Bw.forEach((w, i) => {
          r.s.n += pop(w & ~r.s.data[i] & ALL);
          r.s.data[i] |= w;
          r.snap(`data[${i}] |= other[${i}]`, { ["w" + i]: "hot" });
        });
        return r.done("");
      },
    },
    {
      name: "Remove",
      group: "mutate",
      sig: "func (s *Set) Remove(v int) bool",
      doc: "Clears bit v. Removing the largest value trims empty trailing words (capacity kept).",
      cost: "O(1) amortized",
      args: [{ name: "v", kind: "int", def: (s) => String(valsOf(used(s)).pop() ?? 0) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        r.s.res = null;
        if (!has(r.s, v)) return r.done("false", `${v} is not present`);
        r.snap(`data[${word(v)}] &^= 1<<${v & 63}`, bm(v, "del"));
        r.s.data[word(v)] &= ~bit(v) & ALL;
        r.s.n--;
        r.emit(v, "del", "removed");
        r.snap(`n = ${r.s.n}`);
        if (word(v) === r.s.len - 1 && r.s.data[word(v)] === 0n) trim(r);
        return r.done("true");
      },
    },
    {
      name: "RemoveSet",
      group: "algebra",
      sig: "func (s *Set) RemoveSet(other *Set)",
      doc: "In-place difference: data[i] &^= other[i] over the shared words, then trim.",
      cost: "O(m/64)",
      args: [otherArg],
      run: (s0, a) => {
        const r = rec(s0);
        setOther(r, a[0]);
        const Bw = wordsOf(r.s.b);
        const k = Math.min(r.s.len, Bw.length);
        for (let i = 0; i < k; i++) {
          r.s.n -= pop(r.s.data[i] & Bw[i]);
          r.s.data[i] &= ~Bw[i] & ALL;
          r.snap(`data[${i}] &^= other[${i}]`, { ["w" + i]: "hot" });
        }
        trim(r);
        return r.done("");
      },
    },
    {
      name: "RetainSet",
      group: "algebra",
      sig: "func (s *Set) RetainSet(other *Set)",
      doc: "In-place intersection: reslice to the shorter length (cut words keep stale bits!), AND each word, trim.",
      cost: "O(m/64)",
      args: [otherArg],
      run: (s0, a) => {
        const r = rec(s0);
        setOther(r, a[0]);
        const Bw = wordsOf(r.s.b);
        if (!r.s.n) return r.done("", "s is empty: no-op");
        if (!Bw.length) {
          r.s.len = 0;
          r.s.n = 0;
          return r.done("", "other is empty: data = data[:0] — the old words stay as stale bits");
        }
        if (Bw.length < r.s.len) {
          r.s.len = Bw.length;
          r.snap(`data = data[:${Bw.length}] — words past it are not cleared (stale)`);
        }
        r.s.n = 0;
        for (let i = 0; i < r.s.len; i++) {
          r.s.data[i] &= Bw[i];
          r.s.n += pop(r.s.data[i]);
          r.snap(`data[${i}] &= other[${i}]`, { ["w" + i]: "hot" });
        }
        trim(r);
        return r.done("");
      },
    },
    {
      name: "RemoveFunc",
      group: "mutate",
      sig: "func (s *Set) RemoveFunc(del func(int) bool) int",
      doc: "Calls del for each value in ascending order and clears the selected bits word by word.",
      cost: "O(m/64 + n)",
      args: [{ name: "del", kind: "choice", options: Object.keys(PREDICATES) }],
      run: (s0, a) => {
        const r = rec(s0);
        const p = PREDICATES[a[0]];
        r.s.res = null;
        let n = 0;
        for (const v of valsOf(used(r.s))) {
          if (p(v)) {
            r.s.data[word(v)] &= ~bit(v) & ALL;
            r.s.n--;
            n++;
            r.emit(v, "del", "deleted");
            r.snap(`del(${v}) = true → clear`, bm(v, "del"));
          } else r.snap(`del(${v}) = false`, bm(v, "ok"));
        }
        if (n) trim(r);
        return r.done(String(n));
      },
    },
    wordwise("Union", "New set: OR of every word.", (x, y) => x | y, "|", (x, y) => Math.max(x, y)),
    wordwise("Intersection", "New set: Clone, then RetainSet — AND over the shared words.", (x, y) => x & y, "&", (x, y) => Math.min(x, y)),
    wordwise("Difference", "New set: Clone, then RemoveSet — AND NOT.", (x, y) => x & ~y, "&^", (x) => x),
    wordwise("SymmetricDifference", "New set: XOR of the words, longer operand copied first.", (x, y) => x ^ y, "^", (x, y) => Math.max(x, y), false),
    ...(
      [
        ["Equal", "Same count and identical words (slices.Equal).", (A: bigint[], Bw: bigint[]) => A.length === Bw.length && A.every((w, i) => w === Bw[i])],
        ["IsSubset", "Every word of s has no bits outside other: w &^ other[i] == 0.", (A: bigint[], Bw: bigint[]) => A.length <= Bw.length && A.every((w, i) => (w & ~Bw[i]) === 0n)],
        ["IsSuperset", "other.IsSubset(s).", (A: bigint[], Bw: bigint[]) => Bw.length <= A.length && Bw.every((w, i) => (w & ~A[i]) === 0n)],
        ["IsDisjoint", "No shared bit in the overlapping words: a[i] & b[i] == 0.", (A: bigint[], Bw: bigint[]) => A.every((w, i) => (w & (Bw[i] ?? 0n)) === 0n)],
      ] as const
    ).map(
      ([name, doc, test]): MethodDef<BitS> => ({
        name,
        group: "algebra",
        sig: `func (s *Set) ${name}(other *Set) bool`,
        doc,
        cost: "O(m/64)",
        args: [otherArg],
        run: (s0, a) => {
          const r = rec(s0);
          setOther(r, a[0]);
          const A = used(r.s);
          const Bw = wordsOf(r.s.b);
          const k = Math.max(A.length, Bw.length);
          for (let i = 0; i < k; i++) r.snap(`compare word ${i}: ${hex(A[i] ?? 0n)} vs ${hex(Bw[i] ?? 0n)}`, { ["w" + i]: "cmp" });
          return r.done(String(test(A, Bw)));
        },
      }),
    ),
    {
      name: "Clear",
      group: "memory",
      sig: "func (s *Set) Clear()",
      doc: "Releases the array: Cap() becomes 0.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        Object.assign(r.s, { data: [], len: 0, n: 0, res: null });
        return r.done("", "data = nil");
      },
    },
    {
      name: "Reset",
      group: "memory",
      sig: "func (s *Set) Reset()",
      doc: "data = data[:0] — O(1). Words are NOT zeroed (no references to drop); grow clears them on reuse.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        r.s.len = 0;
        r.s.n = 0;
        r.s.res = null;
        return r.done("", "len 0; the old bits linger as stale words until reused");
      },
    },
    {
      name: "Shrink",
      group: "memory",
      sig: "func (s *Set) Shrink()",
      doc: "Copies the words in use into a right-sized array (empty set: drop it).",
      cost: "O(m/64)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.len) {
          r.s.data = [];
          return r.done("", "empty: data = nil");
        }
        if (r.s.data.length === r.s.len) return r.done("", "already right-sized: no-op");
        r.s.data = r.s.data.slice(0, r.s.len);
        return r.done("", `copied ${r.s.len} words into a new array; stale words are gone`);
      },
    },
    {
      name: "Clip",
      group: "memory",
      sig: "func (s *Set) Clip()",
      doc: "Reslices cap to the words in use without copying.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (r.s.data.length === r.s.len) return r.done("", "already clipped: no-op");
        r.s.data = r.s.data.slice(0, r.s.len);
        return r.done("", "slices.Clip: no copy");
      },
    },
    {
      name: "Grow",
      group: "memory",
      sig: "func (s *Set) Grow(n int)",
      doc: "Ensures values 0..n-1 fit: n is a value bound, not a count of extra values.",
      cost: "O(n/64)",
      args: [{ name: "n", kind: "int", def: "320" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = parseInt1(a[0]);
        const need = Math.ceil(Math.max(n, 0) / 64);
        if (need <= r.s.data.length) return r.done("", `Cap() is already ${r.s.data.length * 64} ≥ ${n}: no-op`);
        const cap = appendCap(r.s.data.length, need, 8);
        while (r.s.data.length < cap) r.s.data.push(0n);
        return r.done("", `cap ${cap} words → Cap() = ${cap * 64}`);
      },
    },
  ],
};
