// Simulation of orderedmap.Map internals, shared by orderedmap and orderedset
// (an orderedset is an orderedmap with struct{} values).
import { appendCap } from "../core/gomem";
import { nid, parseInt1, rec, type Rec } from "../core/rec";
import type { Cell, MethodDef, Tone } from "../core/types";

export const MIN_COMPACT = 16;

export interface OE extends Cell {
  k: string;
}

export interface OMS {
  entries: (OE | null)[]; // null = hole
  cap: number;
  head: number;
  holes: number;
  elem: number; // entry size in bytes, for growth
  b?: string[]; // orderedset operand
  res?: string[] | null;
  resLabel?: string;
}

type M = Rec<OMS>;
type Marks = Record<string, Tone>;

export const oe = (k: string, v = 0): OE => ({ id: nid(), k, v });
export const liveEntries = (s: OMS) => s.entries.filter((e): e is OE => !!e);
export const posOf = (s: OMS, k: string) => s.entries.findIndex((e) => e?.k === k);
export const km = (e: OE, t: Tone): Marks => ({ ["c" + e.id]: t, ["k" + e.k]: t });

export function omOf(keys: [string, number][], cap: number, elem: number): OMS {
  return { entries: keys.map(([k, v]) => oe(k, v)), cap: Math.max(cap, keys.length), head: 0, holes: 0, elem };
}

export function append(r: M, e: OE, why: string) {
  const s = r.s;
  if (s.entries.length + 1 > s.cap) {
    const old = s.cap;
    s.cap = appendCap(old, s.entries.length + 1, s.elem);
    r.snap(old ? `entries is full: append grows cap ${old} → ${s.cap} (holes are copied too)` : `append allocates entries with cap ${s.cap}`);
  }
  s.entries.push(e);
  r.snap(`${why}: append at position ${s.entries.length - 1}; index["${e.k}"] = ${s.entries.length - 1}`, km(e, "new"));
}

/** Map.Set: update in place or append. Returns whether the key was new. */
export function omSet(r: M, k: string, v: number, showValue: boolean): boolean {
  const i = posOf(r.s, k);
  if (i >= 0) {
    const e = r.s.entries[i]!;
    if (showValue) {
      e.v = v;
      r.snap(`index["${k}"] = ${i}: replace the value in place — the key keeps its position`, km(e, "hot"));
    } else r.snap(`"${k}" already present at ${i}: unchanged`, km(e, "dim"));
    return false;
  }
  append(r, oe(k, v), `new key "${k}"`);
  return true;
}

function compact(r: M) {
  const s = r.s;
  r.snap(`holes (${s.holes}) > len(entries)/2 (${s.entries.length}/2) with len ≥ ${MIN_COMPACT}: compact in place`, { compact: "hot" });
  s.entries = liveEntries(s);
  s.head = 0;
  s.holes = 0;
  const m: Marks = {};
  s.entries.forEach((e) => Object.assign(m, km(e!, "cmp")));
  r.snap("live entries slid to the front, index positions rewritten, tail zeroed; capacity kept", m);
}

/** Map.removeAt: hole, trim tail, advance head, maybe compact. */
export function removeAt(r: M, i: number, label: string): OE {
  const s = r.s;
  const e = s.entries[i]!;
  r.snap(`delete index["${e.k}"]; position ${i} becomes a hole (zeroed)`, km(e, "del"));
  s.entries[i] = null;
  s.holes++;
  r.emit(label === "kv" ? `${e.k}:${e.v}` : e.k, "del");
  r.snap(`holes = ${s.holes} — nothing shifted`, { ["s" + i]: "del" });
  let n = s.entries.length;
  while (n > 0 && !s.entries[n - 1]) {
    n--;
    s.holes--;
  }
  if (n !== s.entries.length) {
    s.entries.length = n;
    if (n === 0) s.head = 0;
    r.snap("trailing holes trimmed: the last entry is always live");
  }
  if (s.entries.length > 0 && i === s.head) {
    while (!s.entries[s.head]) s.head++;
    r.snap(`head advances to the next live entry: ${s.head}`, { ["s" + s.head]: "hot" });
  }
  if (s.entries.length >= MIN_COMPACT && s.holes > s.entries.length / 2) compact(r);
  return e;
}

export function moveToBack(r: M, k: string): boolean {
  const s = r.s;
  const i = posOf(s, k);
  if (i < 0) {
    r.snap(`"${k}" is absent: not added`);
    return false;
  }
  if (i === s.entries.length - 1) {
    r.snap(`"${k}" is already the newest: no-op`, km(s.entries[i]!, "ok"));
    return true;
  }
  const e = removeAt(r, i, "silent");
  r.out.pop();
  append(r, e, `re-append "${k}"`);
  return true;
}

export function removeFunc(r: M, del: (e: OE) => boolean): number {
  const s = r.s;
  const before = liveEntries(s).length;
  let w = 0;
  const kept: OE[] = [];
  for (const e of s.entries) {
    if (!e) continue;
    if (del(e)) {
      r.snap(`del(${e.k}) = true → drop`, km(e, "del"));
      r.emit(e.k, "del", "deleted");
      continue;
    }
    r.snap(`keep ${e.k} → write position ${w}`, km(e, "ok"));
    kept.push(e);
    w++;
  }
  s.entries = kept;
  s.head = 0;
  s.holes = 0;
  r.snap("one pass: survivors packed in order, no holes left, tail zeroed");
  return before - kept.length;
}

/** Clear / Reset / Shrink / Clip / Grow, as in orderedmap. */
export function omMemory(recv: string, unit: string, cleanup?: (s: OMS) => void): MethodDef<OMS>[] {
  return [
    {
      name: "Clear",
      group: "memory",
      sig: `func (${recv}) Clear()`,
      doc: "Releases the entry slice and the index: Cap becomes 0.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        Object.assign(r.s, { entries: [], cap: 0, head: 0, holes: 0 });
        cleanup?.(r.s);
        return r.done("", "entries = nil, index = nil");
      },
    },
    {
      name: "Reset",
      group: "memory",
      sig: `func (${recv}) Reset()`,
      doc: `Removes all ${unit} but keeps the entry slice's capacity; entries are zeroed and the index cleared.`,
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const m: Marks = {};
        liveEntries(r.s).forEach((e) => Object.assign(m, km(e, "del")));
        r.snap("clear(entries), clear(index)", m);
        Object.assign(r.s, { entries: [], head: 0, holes: 0 });
        cleanup?.(r.s);
        return r.done("", `cap stays ${r.s.cap}`);
      },
    },
    {
      name: "Shrink",
      group: "memory",
      sig: `func (${recv}) Shrink()`,
      doc: "Packs out holes into a right-sized slice (copy only if there are holes or spare cap) and always rebuilds the index — Go maps never shrink in place.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const s = r.s;
        const copy = s.holes > 0 || s.cap !== s.entries.length;
        if (copy) {
          s.entries = liveEntries(s);
          s.cap = s.entries.length;
          s.head = 0;
          s.holes = 0;
        }
        const m: Marks = {};
        liveEntries(s).forEach((e) => Object.assign(m, km(e, copy ? "new" : "cmp")));
        return r.done("", copy ? `packed into a cap-${s.cap} slice; index rebuilt` : "slice already tight; index rebuilt anyway", m);
      },
    },
    {
      name: "Clip",
      group: "memory",
      sig: `func (${recv}) Clip()`,
      doc: "Reslices entries to cap == len without copying. Holes and the index stay as they are.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (r.s.cap === r.s.entries.length) return r.done("", "already clipped: no-op");
        r.s.cap = r.s.entries.length;
        return r.done("", `slices.Clip: cap ${r.s.cap}${r.s.holes ? ` — the ${r.s.holes} holes remain` : ""}`);
      },
    },
    {
      name: "Grow",
      group: "memory",
      sig: `func (${recv}) Grow(n int)`,
      doc: "Ensures entries has room for n more appends. The index map is not preallocated.",
      cost: "O(n) when it reallocates",
      args: [{ name: "n", kind: "int", def: "4" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = parseInt1(a[0]);
        const len = r.s.entries.length;
        if (n <= 0 || r.s.cap - len >= n) return r.done("", "room already: no-op");
        r.s.cap = appendCap(r.s.cap, len + n, r.s.elem);
        return r.done("", `cap ${r.s.cap}`);
      },
    },
  ];
}

/** Entry strip + index panel (+ operand and result strips for orderedset). */
