import { sizeReads } from "../core/common";
import { PREDICATES, distinctInts, freshInt, list, parseInt1, parseInts, rec, type Rec } from "../core/rec";
import type { MethodDef, StructureDef, Tone } from "../core/types";
import {
  append,
  km,
  liveEntries,
  moveToBack,
  oe,
  omMemory,
  omOf,
  omSet,
  posOf,
  removeAt,
  removeFunc,
  type OMS,
} from "./omcore";

const R = "s *Set[T]";
const ELEM = 16; // int key + struct{} + alive flag
type S = Rec<OMS>;
type Marks = Record<string, Tone>;

const keys = (s: OMS) => liveEntries(s).map((e) => e.k);
const nums = (s: OMS) => keys(s).map(Number);
const fresh = (s: OMS) => String(freshInt(nums(s), 1, 60));
const has = (s: OMS, k: string) => posOf(s, k) >= 0;
const setRet = (ks: string[]) => (ks.length ? `{${ks.join(" ")}}` : "{}");
const bm = (k: string, t: Tone): Marks => ({ ["b" + k]: t, ["k" + k]: t });

function setOther(r: S, raw: string) {
  r.s.b = [...new Set(parseInts(raw).map(String))];
  r.s.res = null;
}


const otherArg = { name: "other", kind: "ints" as const, def: (s: OMS) => (s.b ?? []).join(", ") };
const markEntry = (r: S, k: string, t: Tone): Marks => {
  const i = posOf(r.s, k);
  return i >= 0 ? { ...km(r.s.entries[i]!, t), ["b" + k]: t } : bm(k, t);
};

function inPlace(name: string, doc: string, cost: string, body: (r: S) => void): MethodDef<OMS> {
  return {
    name,
    group: "algebra",
    sig: `func (s *Set[T]) ${name}(other *Set[T])`,
    doc,
    cost,
    args: [otherArg],
    run: (s0, a) => {
      const r = rec(s0);
      setOther(r, a[0]);
      body(r);
      return r.done("");
    },
  };
}

function newSet(name: string, doc: string, cost: string, body: (r: S, res: string[]) => void, variadic = true): MethodDef<OMS> {
  return {
    name,
    group: "algebra",
    sig: variadic ? `func (s *Set[T]) ${name}(others ...*Set[T]) *Set[T]` : `func (s *Set[T]) ${name}(other *Set[T]) *Set[T]`,
    doc,
    cost,
    args: [otherArg],
    run: (s0, a) => {
      const r = rec(s0);
      setOther(r, a[0]);
      const res: string[] = [];
      r.s.res = res;
      r.s.resLabel = `${name} → a new ordered set`;
      body(r, res);
      return r.done(setRet(r.s.res));
    },
  };
}

function pred(name: string, doc: string, body: (r: S) => boolean): MethodDef<OMS> {
  return {
    name,
    group: "algebra",
    sig: `func (s *Set[T]) ${name}(other *Set[T]) bool`,
    doc,
    cost: "O(n) avg",
    args: [otherArg],
    run: (s0, a) => {
      const r = rec(s0);
      setOther(r, a[0]);
      return r.done(String(body(r)));
    },
  };
}

function subset(r: S, a: string[], b: string[], aName: string): boolean {
  if (a.length > b.length) {
    r.snap(`${aName} has more values: cannot be a subset`);
    return false;
  }
  for (const k of a) {
    if (!b.includes(k)) {
      r.snap(`${k} is missing: stop`, markEntry(r, k, "del"));
      return false;
    }
    r.snap(`${k} ✓`, markEntry(r, k, "ok"));
  }
  return true;
}

export const orderedset: StructureDef<OMS> = {
  id: "orderedset",
  title: "orderedset",
  kind: "insertion-ordered set",
  tagline: "A set that remembers the order values arrived in — set algebra whose results come out in a predictable order.",
  about: [
    "A Set[T] is a thin wrapper over orderedmap.Map[T, struct{}]: the same entry slice with holes, the same index, the same compaction rule. Only the values are stored — struct{} takes no space.",
    "Every result has a defined order: Union is s's values then the new ones from each other set; Intersection and Difference keep s's order; SymmetricDifference lists s-only values, then other-only values.",
    "Adding a value that is already present keeps its original position. MoveToBack, PopFirst, and PopLast work exactly like the map's.",
  ],
  cost: [
    ["Add / Remove / Contains", "amortized O(1)"],
    ["First / Last", "O(1)"],
    ["Union / SymmetricDifference", "O(n+m)"],
    ["Intersection / Difference", "O(n·k)"],
    ["RetainSet / RemoveFunc", "O(n)"],
  ],
  seed: () => ({
    ...omOf(
      [5, 12, 3, 27, 9].map((v) => [String(v), 0] as [string, number]),
      8,
      ELEM,
    ),
    b: ["27", "4", "12", "30"],
    res: null,
    resLabel: "",
  }),
  stats: (s) => {
    const k = keys(s);
    return [
      ["len", k.length],
      ["cap", s.cap],
      ["holes", s.holes],
      ["first", k[0] ?? "—"],
      ["last", k[k.length - 1] ?? "—"],
    ];
  },
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New[T comparable]() *Set[T]",
      doc: "Returns an empty ordered set.",
      cost: "O(1)",
      run: (s0) => rec<OMS>({ ...omOf([], 0, ELEM), b: s0.b, res: null }).done("&Set{}", "empty set"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap[T comparable](n int) *Set[T]",
      doc: "Preallocates room for n values.",
      cost: "O(n)",
      args: [{ name: "n", kind: "int", def: "12" }],
      run: (s0, a) => rec<OMS>({ ...omOf([], Math.max(parseInt1(a[0]), 0), ELEM), b: s0.b, res: null }).done("&Set{}"),
    },
    {
      name: "Of",
      group: "construct",
      sig: "func Of[T comparable](vs ...T) *Set[T]",
      doc: "Values in order of first occurrence; duplicates are stored once, where they first appear.",
      cost: "O(k)",
      args: [{ name: "vs", kind: "ints", def: () => { const d = distinctInts(5, 1, 40); return [...d, d[1]].join(", "); } }],
      run: (s0, a) => {
        const vs = parseInts(a[0]);
        const r = rec<OMS>({ ...omOf([], vs.length, ELEM), b: s0.b, res: null });
        for (const v of vs) omSet(r, String(v), 0, false);
        return r.done("&Set{}");
      },
    },
    {
      name: "From",
      group: "construct",
      sig: "func From[T comparable](s []T) *Set[T]",
      doc: "Same as Of(s...): first occurrence wins; s is never aliased.",
      cost: "O(k)",
      args: [{ name: "s", kind: "ints", def: () => distinctInts(6, 1, 40).join(", ") }],
      run: (s0, a) => {
        const vs = parseInts(a[0]);
        const r = rec<OMS>({ ...omOf([], vs.length, ELEM), b: s0.b, res: null });
        for (const v of vs) omSet(r, String(v), 0, false);
        return r.done("&Set{}");
      },
    },
    ...sizeReads<OMS>({ recv: R, len: (s) => keys(s).length, cap: (s) => s.cap, unit: "values", capDoc: "Capacity of the entry slice, holes included." }),
    {
      name: "Contains",
      group: "read",
      sig: "func (s *Set[T]) Contains(v T) bool",
      doc: "One index lookup.",
      cost: "O(1) avg",
      args: [{ name: "v", kind: "int", def: (s) => keys(s)[1] ?? "5" }],
      run: (s0, a) => {
        const r = rec(s0);
        const k = String(parseInt1(a[0]));
        const ok = has(r.s, k);
        return r.done(String(ok), ok ? `index["${k}"] = ${posOf(r.s, k)}` : `${k} not in the index`, ok ? markEntry(r, k, "ok") : undefined);
      },
    },
    {
      name: "ContainsAll",
      group: "read",
      sig: "func (s *Set[T]) ContainsAll(vs ...T) bool",
      doc: "True when every value is present; stops at the first miss.",
      cost: "O(k)",
      args: [{ name: "vs", kind: "ints", def: (s) => [keys(s)[0] ?? 1, fresh(s)].join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        for (const v of parseInts(a[0])) {
          const k = String(v);
          if (!has(r.s, k)) return r.done("false", `${k} missing: stop`);
          r.snap(`${k} present`, markEntry(r, k, "ok"));
        }
        return r.done("true");
      },
    },
    {
      name: "ContainsAny",
      group: "read",
      sig: "func (s *Set[T]) ContainsAny(vs ...T) bool",
      doc: "True when at least one value is present; stops at the first hit.",
      cost: "O(k)",
      args: [{ name: "vs", kind: "ints", def: (s) => [fresh(s), keys(s)[2] ?? 1].join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        for (const v of parseInts(a[0])) {
          const k = String(v);
          if (has(r.s, k)) return r.done("true", `${k} present: stop`, markEntry(r, k, "ok"));
          r.snap(`${k} absent`);
        }
        return r.done("false");
      },
    },
    {
      name: "First",
      group: "read",
      sig: "func (s *Set[T]) First() (T, bool)",
      doc: "The oldest value, at head.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const e = r.s.entries[r.s.head];
        return r.done(e ? `(${e.k}, true)` : "(0, false)", e ? `entries[head] = entries[${r.s.head}]` : "empty", e ? km(e, "ok") : undefined);
      },
    },
    {
      name: "Last",
      group: "read",
      sig: "func (s *Set[T]) Last() (T, bool)",
      doc: "The newest value — the last entry.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        const e = r.s.entries[r.s.entries.length - 1];
        return r.done(e ? `(${e.k}, true)` : "(0, false)", e ? "the last entry is always live" : "empty", e ? km(e, "ok") : undefined);
      },
    },
    ...(["All", "Backward"] as const).map(
      (name): MethodDef<OMS> => ({
        name,
        group: "iterate",
        sig: `func (s *Set[T]) ${name}() iter.Seq[T]`,
        doc: name === "All" ? "Oldest to newest." : "Newest to oldest.",
        cost: "O(n)",
        run: (s0) => {
          const r = rec(s0);
          const idx = [...r.s.entries.keys()];
          if (name === "Backward") idx.reverse();
          for (const i of idx) {
            const e = r.s.entries[i];
            if (!e) {
              r.snap(`hole at ${i}: skip`, { ["s" + i]: "dim" });
              continue;
            }
            r.emit(e.k, undefined, "yielded");
            r.snap(`yield ${e.k}`, km(e, "hot"));
          }
          return r.done("");
        },
      }),
    ),
    {
      name: "Clone",
      group: "copy",
      sig: "func (s *Set[T]) Clone() *Set[T]",
      doc: "Independent copy, same order, packed (no holes).",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        keys(r.s).forEach((k) => r.emit(k, "new", "clone"));
        return r.done(`&Set{len ${keys(r.s).length}}`);
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (s *Set[T]) Slice() []T",
      doc: "Values in insertion order. Empty set returns nil.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const ks = keys(r.s);
        ks.forEach((k) => r.emit(k, undefined, "[]T (insertion order)"));
        return r.done(ks.length ? list(ks) : "nil");
      },
    },
    {
      name: "Add",
      group: "mutate",
      sig: "func (s *Set[T]) Add(v T) bool",
      doc: "Appends v as the newest value; an existing value keeps its position.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: fresh }],
      run: (s0, a) => {
        const r = rec(s0);
        r.s.res = null;
        return r.done(String(omSet(r, String(parseInt1(a[0])), 0, false)));
      },
    },
    {
      name: "AddN",
      group: "mutate",
      sig: "func (s *Set[T]) AddN(vs ...T)",
      doc: "Adds vs in argument order; present values keep their position.",
      cost: "amortized O(k)",
      args: [{ name: "vs", kind: "ints", def: (s) => [fresh(s), keys(s)[0] ?? 1, fresh(s)].join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        r.s.res = null;
        for (const v of parseInts(a[0])) omSet(r, String(v), 0, false);
        return r.done("");
      },
    },
    inPlace("AddSet", "In-place union: appends other's new values in other's order.", "amortized O(m)", (r) => {
      for (const k of r.s.b!) {
        if (has(r.s, k)) r.snap(`${k} already in s: keeps its place`, markEntry(r, k, "dim"));
        else append(r, oe(k), `${k} from other`);
      }
    }),
    {
      name: "Remove",
      group: "mutate",
      sig: "func (s *Set[T]) Remove(v T) bool",
      doc: "Deletes v, leaving a hole; the rest keep their order.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: (s) => keys(s)[1] ?? "5" }],
      run: (s0, a) => {
        const r = rec(s0);
        r.s.res = null;
        const k = String(parseInt1(a[0]));
        const i = posOf(r.s, k);
        if (i < 0) return r.done("false", `${k} absent`);
        r.outLabel = "removed";
        removeAt(r, i, "k");
        return r.done("true");
      },
    },
    inPlace("RemoveSet", "In-place difference: removes each value of other (holes appear, maybe compaction).", "amortized O(m)", (r) => {
      for (const k of r.s.b!) {
        const i = posOf(r.s, k);
        if (i < 0) {
          r.snap(`${k} not in s`, bm(k, "dim"));
          continue;
        }
        r.outLabel = "removed";
        removeAt(r, i, "k");
      }
    }),
    inPlace("RetainSet", "In-place intersection: keeps values of s found in other, via one RemoveFunc pass.", "O(n)", (r) => {
      const b = r.s.b!;
      removeFunc(r, (e) => !b.includes(e.k));
    }),
    {
      name: "RemoveFunc",
      group: "mutate",
      sig: "func (s *Set[T]) RemoveFunc(del func(T) bool) int",
      doc: "Deletes values where del is true, oldest to newest, packing survivors in one pass.",
      cost: "O(n)",
      args: [{ name: "del", kind: "choice", options: Object.keys(PREDICATES) }],
      run: (s0, a) => {
        const r = rec(s0);
        r.s.res = null;
        const p = PREDICATES[a[0]];
        return r.done(String(removeFunc(r, (e) => p(Number(e.k)))));
      },
    },
    {
      name: "PopFirst",
      group: "mutate",
      sig: "func (s *Set[T]) PopFirst() (T, bool)",
      doc: "Removes and returns the oldest value.",
      cost: "amortized O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!keys(r.s).length) return r.done("(0, false)", "empty");
        r.outLabel = "popped";
        const e = removeAt(r, r.s.head, "k");
        return r.done(`(${e.k}, true)`);
      },
    },
    {
      name: "PopLast",
      group: "mutate",
      sig: "func (s *Set[T]) PopLast() (T, bool)",
      doc: "Removes and returns the newest value.",
      cost: "amortized O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (!keys(r.s).length) return r.done("(0, false)", "empty");
        r.outLabel = "popped";
        const e = removeAt(r, r.s.entries.length - 1, "k");
        return r.done(`(${e.k}, true)`);
      },
    },
    {
      name: "MoveToBack",
      group: "mutate",
      sig: "func (s *Set[T]) MoveToBack(v T) bool",
      doc: "Makes v the newest value (hole + re-append). Absent values are not added.",
      cost: "amortized O(1)",
      args: [{ name: "v", kind: "int", def: (s) => keys(s)[0] ?? "5" }],
      run: (s0, a) => {
        const r = rec(s0);
        return r.done(String(moveToBack(r, String(parseInt1(a[0])))));
      },
    },
    newSet("Union", "s's values in order, then other's new values in other's order.", "O(n+m)", (r, res) => {
      res.push(...keys(r.s));
      r.snap("out := s.Clone() — keeps s's order");
      for (const k of r.s.b!) {
        if (res.includes(k)) {
          r.snap(`${k} already in out`, bm(k, "dim"));
          continue;
        }
        res.push(k);
        r.snap(`append ${k}`, { ["b" + k]: "new", ["r" + k]: "new" });
      }
    }),
    newSet("Intersection", "Values of s present in other, in s's order.", "O(n·k)", (r, res) => {
      for (const k of keys(r.s)) {
        if (r.s.b!.includes(k)) {
          res.push(k);
          r.snap(`${k} is in other: append`, { ...markEntry(r, k, "ok"), ["r" + k]: "new" });
        } else r.snap(`${k} not in other: skip`, markEntry(r, k, "dim"));
      }
    }),
    newSet("Difference", "Values of s absent from other, in s's order.", "O(n·k)", (r, res) => {
      for (const k of keys(r.s)) {
        if (!r.s.b!.includes(k)) {
          res.push(k);
          r.snap(`${k} not in other: append`, { ...markEntry(r, k, "ok"), ["r" + k]: "new" });
        } else r.snap(`${k} is in other: skip`, markEntry(r, k, "del"));
      }
    }),
    newSet(
      "SymmetricDifference",
      "s-only values in s's order, then other-only values in other's order.",
      "O(n+m)",
      (r, res) => {
        res.push(...keys(r.s));
        r.snap("out := s.Clone(); for each value of other: remove if present, else append");
        for (const k of r.s.b!) {
          const i = res.indexOf(k);
          if (i >= 0) {
            r.snap(`${k} in both: remove from out`, { ["b" + k]: "del", ["r" + k]: "del" });
            res.splice(i, 1);
          } else {
            res.push(k);
            r.snap(`${k} only in other: append`, { ["b" + k]: "new", ["r" + k]: "new" });
          }
        }
      },
      false,
    ),
    pred("Equal", "Same values regardless of order (use slices.Equal on Slice to compare order too).", (r) => {
      const a = keys(r.s);
      const b = r.s.b!;
      if (a.length !== b.length) {
        r.snap(`len ${a.length} ≠ ${b.length}`);
        return false;
      }
      return subset(r, a, b, "s");
    }),
    pred("IsSubset", "Every value of s is in other.", (r) => subset(r, keys(r.s), r.s.b!, "s")),
    pred("IsSuperset", "Every value of other is in s.", (r) => subset(r, r.s.b!, keys(r.s), "other")),
    pred("IsDisjoint", "No common values; iterates the smaller set.", (r) => {
      const a = keys(r.s);
      const b = r.s.b!;
      const [small, large] = a.length <= b.length ? [a, b] : [b, a];
      for (const k of small) {
        if (large.includes(k)) {
          r.snap(`${k} is in both: not disjoint`, markEntry(r, k, "del"));
          return false;
        }
        r.snap(`${k} only on one side`, markEntry(r, k, "ok"));
      }
      return true;
    }),
    ...omMemory(R, "values", (s) => (s.res = null)),
  ],
};
