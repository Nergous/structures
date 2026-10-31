// Simulation of orderedmap.Map internals, shared by orderedmap and orderedset
// (an orderedset is an orderedmap with struct{} values).
import { AnimatePresence } from "motion/react";
import { appendCap } from "../core/gomem";
import { nid, parseInt1, rec, type Rec } from "../core/rec";
import type { Cell, Frame, MethodDef, Tone } from "../core/types";
import { ArrayView, Box, CapBar, Label, Pointer, Stage, Tray, slotPos, toneOf } from "../ui/viz";

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
export function OMView({ f, values }: { f: Frame<OMS>; values: boolean }) {
  const s = f.s;
  const cw = values ? 54 : 46;
  const cols = values ? 9 : 11;
  const x0 = 20;
  const y0 = 44;
  const rows = Math.max(1, Math.ceil(Math.max(s.cap, 1) / cols));
  const geo = { x: x0, y: y0, cw, cols, gap: 6 };
  const lens = rows * (cw + 20);
  const live = liveEntries(s);
  const ixX = x0 + cols * (cw + 6) + 30;
  let y = y0 + lens + 36;
  const opY = y;
  if (s.b) y += 90;
  const resY = y;
  if (s.res) y += 90;
  const H = Math.max(y + 10, y0 + 30 + live.length * 26);
  const headPos = slotPos(s.head, geo);
  return (
    <div className="view">
      <Stage w={ixX + 140} h={H}>
        <Label x={x0} y={6} className="title">
          entries []entry — insertion order, holes stay until compaction
        </Label>
        {s.cap === 0 && (
          <Label x={x0} y={y0 + 10} className="note-spare">
            entries == nil
          </Label>
        )}
        <ArrayView
          {...geo}
          cells={s.entries}
          cap={s.cap}
          marks={f.marks}
          live={(i) => i < s.entries.length}
          hole={(i) => i < s.entries.length && !s.entries[i]}
          render={(c) => {
            const e = c as OE;
            return values ? (
              <span className="pq sm">
                <b>{e.k}</b>
                <i>{e.v}</i>
              </span>
            ) : (
              e.k
            );
          }}
        />
        {live.length > 0 && <Pointer id="head" x={headPos.x} y={y0 - 26} label="head" dir="down" />}
        <Label x={ixX} y={6} className="title">
          index map[K]int
        </Label>
        {[...live].sort((a, b) => (a.k < b.k ? -1 : 1)).map((e, j) => (
          <Label key={e.k} x={ixX} y={y0 + j * 26} className={"ix" + (f.marks?.["k" + e.k] ? " on" : "")}>
            {e.k} → {posOf(s, e.k)}
          </Label>
        ))}
        {s.b && (
          <>
            <Label x={x0} y={opY - 4} className="title">
              other (operand, in its own order)
            </Label>
            <AnimatePresence>
              {s.b.map((k, i) => (
                <Box key={"b" + k} x={x0 + i * 46} y={opY + 18} w={40} h={36} tone={toneOf(f.marks, "b" + k) ?? "dim"}>
                  {k}
                </Box>
              ))}
            </AnimatePresence>
          </>
        )}
        {s.res && (
          <>
            <Label x={x0} y={resY - 4} className="title">
              {s.resLabel}
            </Label>
            <AnimatePresence>
              {s.res.map((k, i) => (
                <Box key={"r" + k} x={x0 + i * 46} y={resY + 18} w={40} h={36} tone={toneOf(f.marks, "r" + k) ?? "ok"}>
                  {k}
                </Box>
              ))}
            </AnimatePresence>
            {!s.res.length && (
              <Label x={x0} y={resY + 26} className="note-spare">
                empty set
              </Label>
            )}
          </>
        )}
      </Stage>
      <CapBar
        len={live.length}
        cap={s.cap}
        extra={
          <>
            {" "}
            · len(entries) <b>{s.entries.length}</b> · holes <b>{s.holes}</b> · head <b>{s.head}</b>
          </>
        }
      />
      <Tray f={f} />
    </div>
  );
}
