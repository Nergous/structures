import { sizeReads } from "../core/common";
import { PREDICATES, panic, parseInt1, rand, rec } from "../core/rec";
import type { MethodDef, StructureDef } from "../core/types";
import {
  liveEntries,
  km,
  moveToBack,
  omMemory,
  omOf,
  omSet,
  posOf,
  removeAt,
  removeFunc,
  type OMS,
} from "./omcore";

const R = "m *Map[K, V]";
const ELEM = 32; // string key + int value + alive flag
const KEYS = ["ant", "bee", "cat", "dog", "eel", "fox", "gnu", "hen", "jay", "koi", "owl", "pig", "rat", "yak", "emu", "cow", "elk", "ape", "bat", "cod", "asp", "boa"];
const freeKey = (s: OMS) => KEYS.find((k) => posOf(s, k) < 0) ?? "k" + rand(10, 99);
const anyKey = (s: OMS, n = 1) => liveEntries(s)[n]?.k ?? liveEntries(s)[0]?.k ?? "ant";
const kv = (s: OMS, first: boolean) => {
  const l = liveEntries(s);
  const e = first ? l[0] : l[l.length - 1];
  return e ? `(${e.k}, ${e.v}, true)` : '("", 0, false)';
};


const iter = (name: string, sig: string, doc: string, back: boolean, show: (k: string, v: number) => string): MethodDef<OMS> => ({
  name,
  group: "iterate",
  sig,
  doc,
  cost: "O(n)",
  run: (s0) => {
    const r = rec(s0);
    const s = r.s;
    const order = back ? [...s.entries.keys()].reverse() : [...s.entries.keys()];
    for (const i of order) {
      const e = s.entries[i];
      if (!e) {
        r.snap(`position ${i} is a hole: skip`, { ["s" + i]: "dim" });
        continue;
      }
      r.emit(show(e.k, e.v), undefined, "yielded");
      r.snap(`yield ${show(e.k, e.v)}`, km(e, "hot"));
    }
    return r.done("", back ? "newest to oldest" : "oldest to newest");
  },
});

export const orderedmap: StructureDef<OMS> = {
  id: "orderedmap",
  title: "orderedmap",
  kind: "insertion-ordered hash map",
  tagline: "A hash map that remembers insertion order — O(1) lookups, O(1) deletes, and First / Last / MoveToBack for LRU caches.",
  about: [
    "A Map[K, V] keeps entries in a slice in insertion order, plus index map[K]int pointing from each key to its slot. Get is one map lookup; iteration walks the slice.",
    "Deleting does not shift anything: the slot becomes a zeroed hole, head skips leading holes, and trailing holes are trimmed so the last entry is always live.",
    "Once holes are more than half of a slice of 16+ entries, the slice is compacted in place — O(n), but only after O(n) deletions, so deletes stay amortized O(1). Try deleting many keys from a big map to watch it happen.",
  ],
  cost: [
    ["Get / Contains", "O(1) avg"],
    ["Set / Remove", "amortized O(1)"],
    ["PopFirst / PopLast / MoveToBack", "amortized O(1)"],
    ["First / Last", "O(1)"],
    ["All / Keys / Values / RemoveFunc", "O(n)"],
  ],
  seed: () =>
    omOf(
      [
        ["ant", 3],
        ["bee", 14],
        ["cat", 7],
        ["dog", 21],
        ["eel", 9],
      ],
      8,
      ELEM,
    ),
  stats: (s) => {
    const l = liveEntries(s);
    return [
      ["len", l.length],
      ["cap", s.cap],
      ["holes", s.holes],
      ["first", l[0]?.k ?? "—"],
      ["last", l[l.length - 1]?.k ?? "—"],
    ];
  },
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New[K comparable, V any]() *Map[K, V]",
      doc: "Returns an empty map with no preallocated capacity.",
      cost: "O(1)",
      run: () => rec<OMS>(omOf([], 0, ELEM)).done("&Map{}", "empty map"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap[K comparable, V any](n int) *Map[K, V]",
      doc: "Preallocates the entry slice and the index for n keys.",
      cost: "O(n)",
      args: [{ name: "n", kind: "int", def: "16" }],
      run: (_, a) => {
        const n = Math.max(parseInt1(a[0]), 0);
        return rec<OMS>(omOf([], n, ELEM)).done("&Map{}", `entries cap ${n}, index sized for ${n}`);
      },
    },
    ...sizeReads<OMS>({
      recv: R,
      len: (s) => liveEntries(s).length,
      cap: (s) => s.cap,
      unit: "keys",
      capDoc: "Capacity of the entry slice — live entries and holes alike. Not the index size (Go hides it).",
    }),
    {
      name: "Get",
      group: "read",
      sig: "func (m *Map[K, V]) Get(key K) (V, bool)",
      doc: "index[key] gives the slot; the value is read from entries.",
      cost: "O(1) avg",
      args: [{ name: "key", kind: "key", def: (s) => anyKey(s, 2) }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        const i = posOf(r.s, k);
        if (i < 0) return r.done("(0, false)", `index has no "${k}"`);
        const e = r.s.entries[i]!;
        r.snap(`index["${k}"] = ${i}`, { ["k" + k]: "hot" });
        return r.done(`(${e.v}, true)`, `entries[${i}].value = ${e.v}`, km(e, "ok"));
      },
    },
    {
      name: "Contains",
      group: "read",
      sig: "func (m *Map[K, V]) Contains(key K) bool",
      doc: "Reports whether key is present — one index lookup.",
      cost: "O(1) avg",
      args: [{ name: "key", kind: "key", def: (s) => anyKey(s, 1) }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        const i = posOf(r.s, k);
        return r.done(String(i >= 0), i >= 0 ? `index["${k}"] = ${i}` : `no "${k}" in the index`, i >= 0 ? km(r.s.entries[i]!, "ok") : undefined);
      },
    },
    {
      name: "First",
      group: "read",
      sig: "func (m *Map[K, V]) First() (K, V, bool)",
      doc: "The oldest pair: entries[head].",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const e = r.s.entries[r.s.head];
        return r.done(kv(r.s, true), e ? `entries[head] = entries[${r.s.head}]` : "empty map", e ? km(e, "ok") : undefined);
      },
    },
    {
      name: "Last",
      group: "read",
      sig: "func (m *Map[K, V]) Last() (K, V, bool)",
      doc: "The newest pair: the last entry (never a hole).",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const e = r.s.entries[r.s.entries.length - 1];
        return r.done(kv(r.s, false), e ? "entries[len-1] is always live" : "empty map", e ? km(e, "ok") : undefined);
      },
    },
    iter("All", "func (m *Map[K, V]) All() iter.Seq2[K, V]", "Pairs from oldest to newest; holes are skipped.", false, (k, v) => `${k}:${v}`),
    iter("Backward", "func (m *Map[K, V]) Backward() iter.Seq2[K, V]", "Pairs from newest to oldest.", true, (k, v) => `${k}:${v}`),
    iter("Keys", "func (m *Map[K, V]) Keys() iter.Seq[K]", "Keys from oldest to newest.", false, (k) => k),
    iter("Values", "func (m *Map[K, V]) Values() iter.Seq[V]", "Values in key insertion order.", false, (_, v) => String(v)),
    {
      name: "Clone",
      group: "copy",
      sig: "func (m *Map[K, V]) Clone() *Map[K, V]",
      doc: "Independent copy with the same order, no holes, cap == len, fresh index.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        liveEntries(r.s).forEach((e) => r.emit(`${e.k}:${e.v}`, "new", "clone (packed)"));
        return r.done(`&Map{len ${liveEntries(r.s).length}}`, "packArray + rebuildIndex");
      },
    },
    {
      name: "Set",
      group: "mutate",
      sig: "func (m *Map[K, V]) Set(key K, value V) bool",
      doc: "Stores value under key. An existing key keeps its position; a new key is appended as the newest. Returns whether key was new.",
      cost: "amortized O(1)",
      args: [
        { name: "key", kind: "key", def: freeKey },
        { name: "value", kind: "int", def: () => String(rand(1, 99)) },
      ],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        if (!k) panic("empty key");
        return r.done(String(omSet(r, k, parseInt1(a[1]), true)));
      },
    },
    {
      name: "Remove",
      group: "mutate",
      sig: "func (m *Map[K, V]) Remove(key K) (V, bool)",
      doc: "Deletes key, leaving a hole; may trim, advance head, or compact.",
      cost: "amortized O(1)",
      args: [{ name: "key", kind: "key", def: (s) => anyKey(s, 1) }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = a[0].trim();
        const i = posOf(r.s, k);
        if (i < 0) return r.done("(0, false)", `"${k}" is absent`);
        r.outLabel = "removed";
        const e = removeAt(r, i, "kv");
        return r.done(`(${e.v}, true)`);
      },
    },
    {
      name: "RemoveFunc",
      group: "mutate",
      sig: "func (m *Map[K, V]) RemoveFunc(del func(K, V) bool) int",
      doc: "Deletes every pair where del(key, value) is true, oldest to newest, in one packing pass — no holes remain.",
      cost: "O(n)",
      args: [{ name: "del", kind: "choice", options: Object.keys(PREDICATES).map((p) => p.replace("v", "value")) }],
      run: (s0, a) => {
        const r = rec(s0);
        const p = PREDICATES[a[0].replace("value", "v")];
        return r.done(String(removeFunc(r, (e) => p(e.v))));
      },
    },
    {
      name: "PopFirst",
      group: "mutate",
      sig: "func (m *Map[K, V]) PopFirst() (K, V, bool)",
      doc: "Removes the oldest pair (at head). The usual LRU eviction.",
      cost: "amortized O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!liveEntries(r.s).length) return r.done('("", 0, false)', "empty map");
        r.outLabel = "popped";
        const e = removeAt(r, r.s.head, "kv");
        return r.done(`(${e.k}, ${e.v}, true)`);
      },
    },
    {
      name: "PopLast",
      group: "mutate",
      sig: "func (m *Map[K, V]) PopLast() (K, V, bool)",
      doc: "Removes the newest pair.",
      cost: "amortized O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!liveEntries(r.s).length) return r.done('("", 0, false)', "empty map");
        r.outLabel = "popped";
        const e = removeAt(r, r.s.entries.length - 1, "kv");
        return r.done(`(${e.k}, ${e.v}, true)`);
      },
    },
    {
      name: "MoveToBack",
      group: "mutate",
      sig: "func (m *Map[K, V]) MoveToBack(key K) bool",
      doc: "Makes key the newest: remove (leaving a hole) and re-append. An LRU cache calls this on every hit.",
      cost: "amortized O(1)",
      args: [{ name: "key", kind: "key", def: (s) => anyKey(s, 0) }],
      run: (s0, a) => {
        const r = rec(s0);
        return r.done(String(moveToBack(r, a[0].trim())));
      },
    },
    ...omMemory(R, "pairs"),
  ],
};
