import { AnimatePresence } from "motion/react";
import { sizeReads } from "../core/common";
import { distinctInts, freshInt, list, parseInts, PREDICATES, rec, parseInt1, type Rec } from "../core/rec";
import type { MethodDef, StructureDef, Tone, ViewProps } from "../core/types";
import { Box, Label, Stage, Tray, toneOf } from "../ui/viz";

export interface SetS {
  a: number[]; // receiver s (kept sorted only for stable drawing)
  b: number[]; // operand "other"
  res: number[] | null; // result of a non-mutating algebra call
  resLabel: string;
  nilMap: boolean;
}

const R = "s *Set[T]";
const G = 8; // illustrative groups
type S = Rec<SetS>;
type Marks = Record<string, Tone>;

/** Illustrative hash → group (Go's real map uses a seeded hash and a Swiss table). */
export const grp = (v: number) => (Math.imul(v ^ 0x5bd1e995, 0x9e3779b1) >>> 0) % G;
const has = (xs: number[], v: number) => xs.includes(v);
const sortNum = (xs: number[]) => [...xs].sort((x, y) => x - y);
const shuffle = (xs: number[]) => {
  const o = [...xs];
  for (let i = o.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [o[i], o[j]] = [o[j], o[i]];
  }
  return o;
};
const vm = (v: number, t: Tone, g = true): Marks => (g ? { ["v" + v]: t, ["g" + grp(v)]: t } : { ["v" + v]: t });
const setRet = (xs: number[]) => (xs.length ? `{${sortNum(xs).join(" ")}}` : "{}");

function lookup(r: S, v: number): boolean {
  const g = grp(v);
  r.snap(`hash(${v}) → group ${g}: probe only that group`, { ["g" + g]: "cmp" });
  return has(r.s.a, v);
}

function setOther(r: S, raw: string) {
  r.s.b = [...new Set(parseInts(raw))];
  r.s.res = null;
}

function View({ f }: ViewProps<SetS>) {
  const s = f.s;
  const W = 680;
  const cy = 150;
  const ax = 245;
  const bx = 435;
  const rad = 135;
  const onlyA = sortNum(s.a.filter((v) => !has(s.b, v)));
  const both = sortNum(s.a.filter((v) => has(s.b, v)));
  const onlyB = sortNum(s.b.filter((v) => !has(s.a, v)));
  const place = (xs: number[], cx: number, cols: number) =>
    xs.map((v, k) => {
      const rows = Math.ceil(xs.length / cols);
      const row = Math.floor(k / cols);
      const inRow = Math.min(cols, xs.length - row * cols);
      const col = k % cols;
      return { v, x: cx + (col - (inRow - 1) / 2) * 46 - 20, y: cy + (row - (rows - 1) / 2) * 46 - 20 };
    });
  const pts = [...place(onlyA, 175, 3), ...place(both, 340, 2), ...place(onlyB, 505, 3)];
  const gy = 330;
  const gw = (W - 40) / G;
  const groups: number[][] = Array.from({ length: G }, () => []);
  for (const v of sortNum(s.a)) groups[grp(v)].push(v);
  const maxG = Math.max(1, ...groups.map((g) => g.length));
  const ry = gy + 40 + maxG * 30 + 40;
  const H = s.res ? ry + 70 : ry;
  return (
    <div className="view">
      <Stage w={W} h={H}>
        <svg className="venn" width={W} height={cy * 2 + 20}>
          <circle cx={ax} cy={cy} r={rad} className={"venn-a" + (f.marks?.A ? " on" : "")} />
          <circle cx={bx} cy={cy} r={rad} className={"venn-b" + (f.marks?.B ? " on" : "")} />
        </svg>
        <Label x={ax - rad + 10} y={8} className="venn-l">
          s
        </Label>
        <Label x={bx + rad - 50} y={8} className="venn-l b">
          other
        </Label>
        <AnimatePresence>
          {pts.map((p) => {
            const inA = has(s.a, p.v);
            return (
              <Box key={"v" + p.v} x={p.x} y={p.y} w={40} h={40} round tone={toneOf(f.marks, "v" + p.v) ?? (inA ? undefined : "dim")}>
                {p.v}
              </Box>
            );
          })}
        </AnimatePresence>
        <Label x={20} y={gy - 4} className="title">
          inside s: map[T]struct{} — values grouped by hash (illustrative)
        </Label>
        {s.nilMap && (
          <Label x={20} y={gy + 20} className="note-spare">
            m == nil — no map allocated (reads still work)
          </Label>
        )}
        {groups.map((_, i) => (
          <div
            key={"g" + i}
            className={"bucket" + (f.marks?.["g" + i] ? " st-" + f.marks["g" + i] : "")}
            style={{ transform: `translate(${20 + i * gw}px, ${gy + 20}px)`, width: gw - 6, height: maxG * 30 + 22 }}
          >
            <span className="bucket-i">g{i}</span>
          </div>
        ))}
        <AnimatePresence>
          {groups.flatMap((g, i) =>
            g.map((v, k) => (
              <Box key={"m" + v} x={20 + i * gw + (gw - 6) / 2 - 18} y={gy + 38 + k * 30} w={36} h={24} small tone={toneOf(f.marks, "v" + v)}>
                {v}
              </Box>
            )),
          )}
        </AnimatePresence>
        {s.res && (
          <>
            <Label x={20} y={ry - 4} className="title">
              {s.resLabel}
            </Label>
            <AnimatePresence>
              {sortNum(s.res).map((v, k) => (
                <Box key={"r" + v} x={20 + k * 46} y={ry + 18} w={40} h={36} tone={toneOf(f.marks, "r" + v) ?? "ok"}>
                  {v}
                </Box>
              ))}
            </AnimatePresence>
            {!s.res.length && (
              <Label x={20} y={ry + 26} className="note-spare">
                empty set
              </Label>
            )}
          </>
        )}
      </Stage>
      <Tray f={f} />
    </div>
  );
}

const otherArg = { name: "other", kind: "ints" as const, def: (s: SetS) => s.b.join(", ") };
const valsArg = (n = 3) => ({ name: "vs", kind: "ints" as const, def: (s: SetS) => [...s.a.slice(0, n - 1), freshInt(s.a)].join(", ") });

function algebra(
  name: string,
  doc: string,
  cost: string,
  body: (r: S) => void,
  variadic = true,
): MethodDef<SetS> {
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
      r.s.res = [];
      r.s.resLabel = `${name} → a new set (s and other unchanged)`;
      body(r);
      return r.done(setRet(r.s.res));
    },
  };
}

function predicate(name: string, doc: string, test: (s: SetS) => boolean, scan: (r: S) => void): MethodDef<SetS> {
  return {
    name,
    group: "algebra",
    sig: `func (s *Set[T]) ${name}(other *Set[T]) bool`,
    doc,
    cost: "O(n) average",
    args: [otherArg],
    run: (s0, a) => {
      const r = rec(s0);
      setOther(r, a[0]);
      scan(r);
      return r.done(String(test(r.s)));
    },
  };
}

export const set: StructureDef<SetS> = {
  id: "set",
  title: "set",
  kind: "unordered · map-backed",
  tagline: "Membership in O(1) and set algebra — union, intersection, difference — on a plain Go map.",
  about: [
    "A Set[T] is a map[T]struct{}: the empty struct takes no space, so the map stores only keys. Add, Remove, and Contains hash the value and look in one place — O(1) on average.",
    "The groups below the diagram are illustrative: Go's runtime map is a Swiss table with a per-map random hash seed. That is also why iteration order is unspecified — run All or Slice twice and the order changes.",
    "Union, Intersection, Difference, and SymmetricDifference build a new set; AddSet, RetainSet, and RemoveSet are their in-place forms. Go maps expose no capacity, so there is no Cap/Clip/Grow — Shrink rebuilds the map to reclaim memory.",
  ],
  cost: [
    ["Add / Remove / Contains", "O(1) avg"],
    ["Union / SymmetricDifference", "O(n+m)"],
    ["Intersection / Difference", "O(n·k)"],
    ["IsDisjoint", "O(min(n, m))"],
    ["All / Slice / Clone / Shrink", "O(n)"],
  ],
  seed: () => ({ a: [4, 8, 15, 16, 23], b: [8, 16, 42, 7], res: null, resLabel: "", nilMap: false }),
  stats: (s) => [
    ["len", s.a.length],
    ["other", s.b.length],
  ],
  View,
  methods: [
    {
      name: "New",
      group: "construct",
      sig: "func New[T comparable]() *Set[T]",
      doc: "Returns an empty set. The map is allocated lazily on first Add.",
      cost: "O(1)",
      run: (s0) => rec<SetS>({ ...s0, a: [], res: null, nilMap: false }).done("&Set{}", "empty set"),
    },
    {
      name: "NewWithCap",
      group: "construct",
      sig: "func NewWithCap[T comparable](n int) *Set[T]",
      doc: "Returns an empty set whose map is sized for n values (make(map, n)).",
      cost: "O(n)",
      args: [{ name: "n", kind: "int", def: "16" }],
      run: (s0, a) => rec<SetS>({ ...s0, a: [], res: null, nilMap: false }).done("&Set{}", `make(map[T]struct{}, ${Math.max(parseInt1(a[0]), 0)})`),
    },
    {
      name: "Of",
      group: "construct",
      sig: "func Of[T comparable](vs ...T) *Set[T]",
      doc: "Returns a set of vs; duplicates are stored once.",
      cost: "O(k)",
      args: [{ name: "vs", kind: "ints", def: () => distinctInts(5, 1, 40).join(", ") + ", 3, 3" }],
      run: (s0, a) => {
        const vs = parseInts(a[0]);
        const r = rec<SetS>({ ...s0, a: [], res: null, nilMap: false });
        for (const v of vs) {
          if (has(r.s.a, v)) {
            r.snap(`${v} is already present: stored once`, vm(v, "dim"));
            continue;
          }
          r.s.a.push(v);
          r.snap(`m[${v}] = struct{}{}`, vm(v, "new"));
        }
        return r.done("&Set{}");
      },
    },
    {
      name: "From",
      group: "construct",
      sig: "func From[T comparable](s []T) *Set[T]",
      doc: "Returns a set of the values in slice s; never aliases s.",
      cost: "O(k)",
      args: [{ name: "s", kind: "ints", def: () => distinctInts(6, 1, 40).join(", ") }],
      run: (s0, a) => {
        const vs = [...new Set(parseInts(a[0]))];
        const r = rec<SetS>({ ...s0, a: vs, res: null, nilMap: false });
        return r.done("&Set{}", `${vs.length} distinct values copied into a new map`);
      },
    },
    ...sizeReads<SetS>({ recv: R, len: (s) => s.a.length, unit: "values" }),
    {
      name: "Contains",
      group: "read",
      sig: "func (s *Set[T]) Contains(v T) bool",
      doc: "Reports whether v is in the set — one hash lookup.",
      cost: "O(1) avg",
      args: [{ name: "v", kind: "int", def: (s) => String(s.a[1] ?? 4) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        const ok = lookup(r, v);
        return r.done(String(ok), ok ? `${v} found in group ${grp(v)}` : `${v} is not in group ${grp(v)}: absent`, ok ? vm(v, "ok") : { ["g" + grp(v)]: "del" });
      },
    },
    {
      name: "ContainsAll",
      group: "read",
      sig: "func (s *Set[T]) ContainsAll(vs ...T) bool",
      doc: "Reports whether every value of vs is present; stops at the first miss. True for no arguments.",
      cost: "O(k)",
      args: [valsArg()],
      run: (s0, a) => {
        const r = rec(s0);
        for (const v of parseInts(a[0])) {
          if (!lookup(r, v)) return r.done("false", `${v} is missing: stop`, { ["g" + grp(v)]: "del" });
          r.snap(`${v} present`, vm(v, "ok"));
        }
        return r.done("true", "all present");
      },
    },
    {
      name: "ContainsAny",
      group: "read",
      sig: "func (s *Set[T]) ContainsAny(vs ...T) bool",
      doc: "Reports whether at least one value of vs is present; stops at the first hit. False for no arguments.",
      cost: "O(k)",
      args: [{ name: "vs", kind: "ints", def: (s) => [freshInt(s.a), s.a[0] ?? 1].join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        for (const v of parseInts(a[0])) {
          if (lookup(r, v)) return r.done("true", `${v} is present: stop`, vm(v, "ok"));
          r.snap(`${v} absent`, { ["g" + grp(v)]: "del" });
        }
        return r.done("false", "none present");
      },
    },
    {
      name: "All",
      group: "iterate",
      sig: "func (s *Set[T]) All() iter.Seq[T]",
      doc: "Iterates in unspecified order — Go randomizes map iteration. Run it twice!",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        for (const v of shuffle(r.s.a)) {
          r.emit(v, undefined, "yielded (random order)");
          r.snap(`yield ${v}`, vm(v, "hot"));
        }
        return r.done("", "map order is randomized on every range");
      },
    },
    {
      name: "Clone",
      group: "copy",
      sig: "func (s *Set[T]) Clone() *Set[T]",
      doc: "Returns an independent copy with a map sized for its length.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        shuffle(r.s.a).forEach((v) => r.emit(v, "new", "clone"));
        return r.done(`&Set{len ${r.s.a.length}}`);
      },
    },
    {
      name: "Slice",
      group: "copy",
      sig: "func (s *Set[T]) Slice() []T",
      doc: "Returns the values in unspecified order. Empty set returns nil.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const vs = shuffle(r.s.a);
        vs.forEach((v) => r.emit(v, undefined, "[]T (random order)"));
        return r.done(vs.length ? list(vs) : "nil");
      },
    },
    {
      name: "Add",
      group: "mutate",
      sig: "func (s *Set[T]) Add(v T) bool",
      doc: "Inserts v and reports whether it was new.",
      cost: "O(1) avg",
      args: [{ name: "v", kind: "int", def: (s) => String(freshInt(s.a, 1, 50)) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        r.s.res = null;
        if (lookup(r, v)) return r.done("false", `${v} is already present: unchanged`, vm(v, "dim"));
        if (r.s.nilMap) {
          r.s.nilMap = false;
          r.snap("m == nil: allocate the map on first insertion");
        }
        r.s.a.push(v);
        return r.done("true", `m[${v}] = struct{}{} in group ${grp(v)}`, vm(v, "new"));
      },
    },
    {
      name: "AddN",
      group: "mutate",
      sig: "func (s *Set[T]) AddN(vs ...T)",
      doc: "Inserts every value of vs; duplicates are stored once.",
      cost: "O(k) avg",
      args: [{ name: "vs", kind: "ints", def: (s) => [freshInt(s.a, 1, 50), freshInt(s.a, 1, 50), s.a[0] ?? 1].join(", ") }],
      run: (s0, a) => {
        const r = rec(s0);
        r.s.res = null;
        r.s.nilMap = false;
        for (const v of parseInts(a[0])) {
          if (has(r.s.a, v)) {
            r.snap(`${v} already present`, vm(v, "dim"));
            continue;
          }
          r.s.a.push(v);
          r.snap(`add ${v}`, vm(v, "new"));
        }
        return r.done("");
      },
    },
    {
      name: "Remove",
      group: "mutate",
      sig: "func (s *Set[T]) Remove(v T) bool",
      doc: "Deletes v and reports whether it was present.",
      cost: "O(1) avg",
      args: [{ name: "v", kind: "int", def: (s) => String(s.a[s.a.length - 1] ?? 1) }],
      run: (s0, a) => {
        const r = rec(s0);
        const v = parseInt1(a[0]);
        r.s.res = null;
        if (!lookup(r, v)) return r.done("false", `${v} absent: nothing to delete`);
        r.snap(`delete(m, ${v})`, vm(v, "del"));
        r.s.a = r.s.a.filter((x) => x !== v);
        r.emit(v, "del", "removed");
        return r.done("true");
      },
    },
    {
      name: "RemoveFunc",
      group: "mutate",
      sig: "func (s *Set[T]) RemoveFunc(del func(T) bool) int",
      doc: "Deletes every value for which del returns true, visiting values in unspecified order.",
      cost: "O(n)",
      args: [{ name: "del", kind: "choice", options: Object.keys(PREDICATES) }],
      run: (s0, a) => {
        const r = rec(s0);
        const p = PREDICATES[a[0]];
        r.s.res = null;
        let n = 0;
        for (const v of shuffle(r.s.a)) {
          if (p(v)) {
            r.snap(`del(${v}) = true → delete`, vm(v, "del"));
            r.s.a = r.s.a.filter((x) => x !== v);
            r.emit(v, "del", "deleted");
            n++;
          } else r.snap(`del(${v}) = false → keep`, vm(v, "ok"));
        }
        return r.done(String(n));
      },
    },
    {
      name: "AddSet",
      group: "algebra",
      sig: "func (s *Set[T]) AddSet(other *Set[T])",
      doc: "In-place union: inserts every value of other into s.",
      cost: "O(m) avg",
      args: [otherArg],
      run: (s0, a) => {
        const r = rec(s0);
        setOther(r, a[0]);
        r.snap("for v := range other.m", { B: "hot" });
        for (const v of shuffle(r.s.b)) {
          if (has(r.s.a, v)) {
            r.snap(`${v} already in s`, vm(v, "dim"));
            continue;
          }
          r.s.a.push(v);
          r.snap(`s.m[${v}] = struct{}{}`, vm(v, "new"));
        }
        return r.done("", "s now holds s ∪ other");
      },
    },
    {
      name: "RemoveSet",
      group: "algebra",
      sig: "func (s *Set[T]) RemoveSet(other *Set[T])",
      doc: "In-place difference: deletes every value of other from s.",
      cost: "O(m) avg",
      args: [otherArg],
      run: (s0, a) => {
        const r = rec(s0);
        setOther(r, a[0]);
        for (const v of shuffle(r.s.b)) {
          if (has(r.s.a, v)) {
            r.snap(`delete(s.m, ${v})`, vm(v, "del"));
            r.s.a = r.s.a.filter((x) => x !== v);
          } else r.snap(`${v} not in s: delete is a no-op`, vm(v, "dim", false));
        }
        return r.done("", "s now holds s \\ other");
      },
    },
    {
      name: "RetainSet",
      group: "algebra",
      sig: "func (s *Set[T]) RetainSet(other *Set[T])",
      doc: "In-place intersection: keeps only values of s that other contains.",
      cost: "O(n) avg",
      args: [otherArg],
      run: (s0, a) => {
        const r = rec(s0);
        setOther(r, a[0]);
        for (const v of shuffle(r.s.a)) {
          if (has(r.s.b, v)) r.snap(`${v} is in other: keep`, vm(v, "ok"));
          else {
            r.snap(`${v} not in other: delete`, vm(v, "del"));
            r.s.a = r.s.a.filter((x) => x !== v);
          }
        }
        return r.done("", "s now holds s ∩ other");
      },
    },
    algebra("Union", "New set with the values of s and every other set.", "O(n+m)", (r) => {
      r.s.res = [...r.s.a];
      r.snap("out := s.Clone()", { A: "hot" });
      for (const v of shuffle(r.s.b)) {
        if (has(r.s.res, v)) continue;
        r.s.res.push(v);
        r.snap(`add ${v} from other`, { ["v" + v]: "new", ["r" + v]: "new" });
      }
    }),
    algebra("Intersection", "New set with the values of s present in every other set.", "O(n·k)", (r) => {
      r.s.res = [...r.s.a];
      r.snap("out := s.Clone(), then delete what other lacks", { A: "hot" });
      for (const v of shuffle(r.s.a)) {
        if (has(r.s.b, v)) r.snap(`${v} is in other: keep`, { ["v" + v]: "ok" });
        else {
          r.snap(`${v} not in other: delete from out`, { ["v" + v]: "del", ["r" + v]: "del" });
          r.s.res = r.s.res.filter((x) => x !== v);
        }
      }
    }),
    algebra("Difference", "New set with the values of s absent from every other set.", "O(n·k)", (r) => {
      r.s.res = [...r.s.a];
      r.snap("out := s.Clone(), then delete what other has", { A: "hot" });
      for (const v of shuffle(r.s.a)) {
        if (!has(r.s.b, v)) r.snap(`${v} not in other: keep`, { ["v" + v]: "ok" });
        else {
          r.snap(`${v} is in other: delete from out`, { ["v" + v]: "del", ["r" + v]: "del" });
          r.s.res = r.s.res.filter((x) => x !== v);
        }
      }
    }),
    algebra(
      "SymmetricDifference",
      "New set with the values in exactly one of s and other.",
      "O(n+m)",
      (r) => {
        r.s.res = [...r.s.a];
        r.snap("out := s.Clone(); toggle every value of other", { A: "hot" });
        for (const v of shuffle(r.s.b)) {
          if (has(r.s.res, v)) {
            r.snap(`${v} in both: delete from out`, { ["v" + v]: "del", ["r" + v]: "del" });
            r.s.res = r.s.res.filter((x) => x !== v);
          } else {
            r.s.res.push(v);
            r.snap(`${v} only in other: add to out`, { ["v" + v]: "new", ["r" + v]: "new" });
          }
        }
      },
      false,
    ),
    predicate(
      "Equal",
      "Same length and s ⊆ other.",
      (s) => s.a.length === s.b.length && s.a.every((v) => has(s.b, v)),
      (r) => {
        if (r.s.a.length !== r.s.b.length) return r.snap(`len ${r.s.a.length} ≠ ${r.s.b.length}: not equal`, { A: "del", B: "del" });
        for (const v of r.s.a) {
          if (!has(r.s.b, v)) return r.snap(`${v} missing from other`, { ["v" + v]: "del" });
          r.snap(`${v} ✓`, { ["v" + v]: "ok" });
        }
      },
    ),
    predicate(
      "IsSubset",
      "Every value of s is in other (fails fast when s is larger).",
      (s) => s.a.length <= s.b.length && s.a.every((v) => has(s.b, v)),
      (r) => {
        if (r.s.a.length > r.s.b.length) return r.snap("s is larger than other: cannot be a subset", { A: "del" });
        for (const v of r.s.a) {
          if (!has(r.s.b, v)) return r.snap(`${v} not in other: stop`, { ["v" + v]: "del" });
          r.snap(`${v} in other ✓`, { ["v" + v]: "ok" });
        }
      },
    ),
    predicate(
      "IsSuperset",
      "Every value of other is in s — other.IsSubset(s).",
      (s) => s.b.length <= s.a.length && s.b.every((v) => has(s.a, v)),
      (r) => {
        if (r.s.b.length > r.s.a.length) return r.snap("other is larger: s cannot contain it", { B: "del" });
        for (const v of r.s.b) {
          if (!has(r.s.a, v)) return r.snap(`${v} not in s: stop`, { ["v" + v]: "del" });
          r.snap(`${v} in s ✓`, { ["v" + v]: "ok" });
        }
      },
    ),
    predicate(
      "IsDisjoint",
      "No common values. Iterates the smaller set and probes the larger.",
      (s) => !s.a.some((v) => has(s.b, v)),
      (r) => {
        const small = r.s.a.length <= r.s.b.length ? r.s.a : r.s.b;
        const large = small === r.s.a ? r.s.b : r.s.a;
        r.snap(`iterate the smaller set (${small === r.s.a ? "s" : "other"})`, { [small === r.s.a ? "A" : "B"]: "hot" });
        for (const v of small) {
          if (has(large, v)) return r.snap(`${v} is in both: not disjoint`, { ["v" + v]: "del" });
          r.snap(`${v} only on one side`, { ["v" + v]: "ok" });
        }
      },
    ),
    {
      name: "Clear",
      group: "memory",
      sig: "func (s *Set[T]) Clear()",
      doc: "Removes all values and releases the map (m = nil).",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        r.s.a = [];
        r.s.res = null;
        r.s.nilMap = true;
        return r.done("", "m = nil: the map is garbage now");
      },
    },
    {
      name: "Reset",
      group: "memory",
      sig: "func (s *Set[T]) Reset()",
      doc: "Removes all values but keeps the map's allocated space (clear(m)).",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const m: Marks = {};
        r.s.a.forEach((v) => Object.assign(m, vm(v, "del")));
        r.snap("clear(m)", m);
        r.s.a = [];
        r.s.res = null;
        return r.done("", "empty, but the map keeps its buckets for reuse");
      },
    },
    {
      name: "Shrink",
      group: "memory",
      sig: "func (s *Set[T]) Shrink()",
      doc: "Rebuilds the map sized for Len, since Go maps never shrink in place. An empty set drops its map.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        if (!r.s.a.length) {
          r.s.nilMap = true;
          return r.done("", "empty: m = nil");
        }
        const m: Marks = {};
        r.s.a.forEach((v) => Object.assign(m, vm(v, "new")));
        return r.done("", `copied ${r.s.a.length} values into a right-sized map`, m);
      },
    },
  ],
};
