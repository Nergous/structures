// Drawing kit for the memory views: a variable card in the stack frame, a
// pointer arrow, and strips of cells on the heap. Everything is absolutely
// positioned inside a <Region>, in pixels.
import { createContext, useContext, type ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import type { Frame, Tone } from "../core/types";

const CW = 64; // default cell width
const CH = 60; // default cell height
const COLS = 12;
const HEAD_H = 38;
const ROW_H = 36;

export interface ViewProps<S> {
  f: Frame<S>;
  /** Address of the current backing array. */
  base: number;
}

export interface MemCtx {
  addrs: boolean;
  still: boolean;
  /** Width of the memory region in px. */
  w: number;
}
export const Mem = createContext<MemCtx>({ addrs: true, still: false, w: 1100 });
export const useMem = () => useContext(Mem);

/** Below this width the struct card moves above the heap (phones). */
export const NARROW = 820;

/**
 * Geometry of a memory view for the current width. `fields` is the number of
 * rows in the struct card: on narrow screens the card sits on top, so the
 * heap starts under it.
 */
export interface Lay {
  m: boolean;
  W: number;
  HX: number;
  CW: number;
  CH: number;
  COLS: number;
  ROW: number;
  VAR_W: number;
  /** Top of the first cell row. */
  Y: number;
  /** Top of the struct card. */
  VY: number;
  cardBottom: number;
  /** Tree level height and node radius. */
  LVL: number;
  NR: number;
}

export const cardH = (fields: number, narrow: boolean) => HEAD_H + (narrow ? 1 + Math.ceil((fields - 1) / 2) : fields) * ROW_H + 28;

export function useLay(fields: number): Lay {
  const { w } = useMem();
  if (w >= NARROW) {
    const HX = 300;
    const COLS = Math.max(8, Math.min(12, Math.floor((w - HX) / 64)));
    return { m: false, W: w, HX, CW: 64, CH: 60, COLS, ROW: 96, VAR_W: 262, Y: 92, VY: 54, cardBottom: 54 + cardH(fields, false), LVL: 64, NR: 21 };
  }
  const HX = 22;
  const COLS = 8;
  const CW = Math.max(32, Math.min(56, Math.floor((w - HX - 2) / COLS)));
  const CH = Math.min(52, CW + 4);
  const VY = 34;
  const cb = VY + cardH(fields, true);
  return { m: true, W: w, HX, CW, CH, COLS, ROW: CH + 56, VAR_W: w, Y: cb + 92, VY, cardBottom: cb, LVL: 52, NR: 16 };
}

/** Start of the pointer arrow: the ptr field (wide) or the card's bottom-left (narrow). */
export const ptrFrom = (L: Lay, vy = L.VY): [number, number] => (L.m ? [10, L.cardBottom] : [L.VAR_W, fieldY(vy, 0)]);

export function useMotion() {
  const { still } = useMem();
  return still ? { duration: 0 } : { type: "spring" as const, stiffness: 320, damping: 30 };
}

export const hex = (n: number) => "0x" + n.toString(16);
export const shortAddr = (n: number) => "…" + n.toString(16).slice(-3);
export type Marks = Record<string, Tone> | undefined;
export const tone = (m: Marks, ...keys: string[]): Tone | undefined => {
  for (const k of keys) if (m?.[k]) return m[k];
  return undefined;
};

/** The dotted canvas. Children are positioned in its coordinates. */
export function Region({ h, children, heapLabel = "heap", card }: { h: number; children: ReactNode; heapLabel?: string; card?: [top: number, fields: number] }) {
  const L = useLay(card?.[1] ?? 0);
  const min = card && !L.m ? card[0] + cardH(card[1], false) + 30 : 0;
  return (
    <section className={"region" + (L.m ? " narrow" : "")} style={{ height: Math.max(h, min) }}>
      <div className="col-l">stack frame</div>
      <div className="col-l" style={L.m ? { left: L.HX, top: L.cardBottom + 10 } : { left: L.HX }}>
        {heapLabel}
      </div>
      {children}
    </section>
  );
}

/* ---------- variable card ---------- */

export interface Field {
  k: string;
  v: ReactNode;
  kind?: "ptr" | "nil" | "num" | "text";
  hot?: boolean;
}

export const fieldY = (y: number, i: number) => y + HEAD_H + i * ROW_H + ROW_H / 2;

export function Var({ y, name, type, fields, foot }: { y: number; name: string; type: string; fields: Field[]; foot?: string }) {
  const L = useLay(fields.length);
  return (
    <div className={"var" + (L.m ? " m" : "")} style={{ top: y, width: L.VAR_W }}>
      <div className="var-h">
        <b>{name}</b>
        <span>{type}</span>
      </div>
      {fields.map((f, i) => {
        // narrow: the pointer row and an unpaired last field span both columns
        const wide = L.m && (i === 0 || (i === fields.length - 1 && (fields.length - 1) % 2 === 1));
        const left = L.m && !wide && (i - 1) % 2 === 0;
        return (
        <div key={f.k} className={"field" + (f.hot ? " hot" : "") + (wide ? " wide" : "") + (left ? " l" : "")}>
          <span>{f.k}</span>
          <motion.code key={String(f.v)} className={"fv-" + (f.kind ?? "num")} initial={{ y: -5, opacity: 0 }} animate={{ y: 0, opacity: 1 }}>
            {f.v}
          </motion.code>
        </div>
        );
      })}
      {foot && <div className="var-f">{foot}</div>}
    </div>
  );
}

/** Pointer value for a field: the address, or a dot when addresses are off. */
export function Ptr({ to }: { to: number | null }) {
  const { addrs } = useMem();
  if (to === null) return <>nil</>;
  return <>{addrs ? hex(to) : "●"}</>;
}

/* ---------- arrow ---------- */

export function Arrow({ from, to, id }: { from: [number, number]; to: [number, number]; id: string | number }) {
  const { still } = useMem();
  const [x0, y0] = from;
  const [x1, y1] = to;
  const mx = (x0 + x1) / 2;
  const d = x1 - x0 < 40 ? `M ${x0} ${y0} C ${x0} ${y1}, ${x0} ${y1}, ${x1 - 3} ${y1}` : `M ${x0} ${y0} C ${mx} ${y0}, ${mx} ${y1}, ${x1 - 3} ${y1}`;
  return (
    <svg className="arrow">
      <defs>
        <marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M0 0 10 5 0 10z" fill="currentColor" />
        </marker>
      </defs>
      <motion.path
        key={id}
        d={d}
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        markerEnd="url(#ah)"
        initial={{ pathLength: 0 }}
        animate={{ pathLength: 1 }}
        transition={still ? { duration: 0 } : { duration: 0.5 }}
      />
    </svg>
  );
}

/* ---------- strips ---------- */

export type SlotKind = "live" | "spare" | "hole" | "stale";

export interface Slot {
  kind: SlotKind;
  /** Text shown in an empty slot (zero value). */
  zero?: ReactNode;
  tone?: Tone;
}

export interface Item {
  id: string | number;
  at: number;
  content: ReactNode;
  tone?: Tone;
  /** Draw lifted above its slot (heap sifts that hold an element). */
  lifted?: boolean;
}

export const cellX = (x: number, i: number, cols = COLS, cw = CW) => x + (i % cols) * cw;
export const cellY = (y: number, i: number, cols = COLS, rowGap = 96) => y + Math.floor(i / cols) * rowGap;

export function Strip(p: {
  x: number;
  y: number;
  slots: Slot[];
  items: Item[];
  base?: number;
  elem?: number;
  cw?: number;
  ch?: number;
  cols?: number;
  rowGap?: number;
  ns?: string;
}) {
  const { addrs } = useMem();
  const t = useMotion();
  const { still } = useMem();
  const cw = p.cw ?? CW;
  const ch = p.ch ?? CH;
  const cols = p.cols ?? COLS;
  const rg = p.rowGap ?? 96;
  const ns = p.ns ?? "";
  const pos = (i: number) => ({ x: cellX(p.x, i, cols, cw), y: cellY(p.y, i, cols, rg) });
  return (
    <>
      {p.slots.map((s, i) => {
        const { x, y } = pos(i);
        return (
          <div key={ns + "s" + i} className={"slot k-" + s.kind + (s.tone ? " st-" + s.tone : "")} style={{ left: x, top: y, width: cw, height: ch }}>
            {addrs && p.base !== undefined && <span className="addr">{shortAddr(p.base + i * (p.elem ?? 8))}</span>}
            <span className="idx">{i}</span>
            {s.kind !== "live" && s.zero !== undefined && <span className="zero">{s.zero}</span>}
          </div>
        );
      })}
      <AnimatePresence>
        {p.items.map((it) => {
          const { x, y } = pos(it.at);
          const yy = it.lifted ? y - ch - 18 : y;
          const cls = "val" + (it.tone ? " " + it.tone : "") + (it.lifted ? " lifted" : "");
          if (still)
            return (
              <div key={ns + it.id} className={cls} style={{ width: cw, height: ch, transform: `translate(${x}px, ${yy}px)` }}>
                {it.content}
              </div>
            );
          return (
            <motion.div
              key={ns + it.id}
              className={"val" + (it.tone ? " " + it.tone : "") + (it.lifted ? " lifted" : "")}
              style={{ width: cw, height: ch }}
              initial={{ opacity: 0, x, y: yy - 30 }}
              animate={{ opacity: 1, x, y: yy }}
              exit={{ opacity: 0, y: yy - 30 }}
              transition={t}
            >
              {it.content}
            </motion.div>
          );
        })}
      </AnimatePresence>
    </>
  );
}

/** A small black tag with a caret, pointing down at a cell. */
export function Tag({ x, y, label, tone: t }: { x: number; y: number; label: string; tone?: "accent" | "blue" }) {
  return (
    <div className={"tag" + (t ? " " + t : "")} style={{ left: x, top: y }}>
      {label}
    </div>
  );
}

export function Brace({ x, y, w, label, strong }: { x: number; y: number; w: number; label: string; strong?: boolean }) {
  const { still } = useMem();
  return (
    <motion.div className={"brace" + (strong ? " strong" : "")} style={{ left: x, top: y }} initial={false} animate={{ width: Math.max(w, 0) }} transition={still ? { duration: 0 } : { duration: 0.3 }}>
      <span>{label}</span>
    </motion.div>
  );
}

export function Caption({ x, y, children, w }: { x: number; y: number; children: ReactNode; w?: number }) {
  return (
    <div className="caption" style={{ left: x, top: y, width: w }}>
      {children}
    </div>
  );
}

export function Note({ x, y, children }: { x: number; y: number; children: ReactNode }) {
  return (
    <div className="nil-note" style={{ left: x, top: y }}>
      {children}
    </div>
  );
}

/* ---------- map card (index maps, Go maps) ---------- */

export function MapCard({ x, y, title, rows, w = 200, empty = "empty" }: { x: number; y: number; title: string; rows: { k: string; v: ReactNode; tone?: Tone }[]; w?: number; empty?: string }) {
  return (
    <div className="mapcard" style={{ left: x, top: y, width: w }}>
      <div className="mapcard-h">{title}</div>
      {rows.map((r) => (
        <div key={r.k} className={"mapcard-r" + (r.tone ? " st-" + r.tone : "")}>
          <code>{r.k}</code>
          <span>→</span>
          <code>{r.v}</code>
        </div>
      ))}
      {!rows.length && <div className="mapcard-e">{empty}</div>}
    </div>
  );
}

/* ---------- chip rows (operands, results) ---------- */

export function Row({ x, y, label, items }: { x: number; y: number; label: string; items: { key: string; text: ReactNode; tone?: Tone; dim?: boolean }[] }) {
  const t = useMotion();
  return (
    <div className="row" style={{ left: x, top: y }}>
      <div className="row-l">{label}</div>
      <div className="row-c">
        <AnimatePresence>
          {items.map((it) => (
            <motion.span key={it.key} layout transition={t} initial={{ opacity: 0, y: -6 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }} className={"chip" + (it.tone ? " " + it.tone : "") + (it.dim ? " dim" : "")}>
              {it.text}
            </motion.span>
          ))}
        </AnimatePresence>
        {!items.length && <span className="row-e">empty</span>}
      </div>
    </div>
  );
}

/* ---------- logical tree (heaps) ---------- */

export interface Node {
  id: string | number;
  at: number;
  label: ReactNode;
  tone?: Tone;
}

export const treeH = (n: number, lvl = 64) => (n ? Math.floor(Math.log2(n)) + 1 : 1) * lvl;

export function Tree({ x, y, w, nodes, slots, holes = [], r = 21, lvlH = 64 }: { x: number; y: number; w: number; nodes: Node[]; slots: number; holes?: number[]; r?: number; lvlH?: number }) {
  const t = useMotion();
  const { still } = useMem();
  const R = r;
  const pos = (i: number) => {
    const lvl = Math.floor(Math.log2(i + 1));
    const first = 2 ** lvl - 1;
    const span = w / 2 ** lvl;
    return { x: x + span * (i - first) + span / 2, y: y + lvl * lvlH + R };
  };
  const filled = new Set([...nodes.map((n) => n.at), ...holes]);
  const edges = [];
  for (let i = 1; i < slots; i++) {
    if (!filled.has(i)) continue;
    const a = pos((i - 1) >> 1);
    const b = pos(i);
    edges.push(<line key={i} x1={a.x} y1={a.y} x2={b.x} y2={b.y} />);
  }
  return (
    <>
      <svg className="edges">{edges}</svg>
      {holes.map((i) => {
        const p = pos(i);
        return <div key={"h" + i} className="node hole" style={{ left: p.x - R, top: p.y - R, width: 2 * R, height: 2 * R }} />;
      })}
      <AnimatePresence>
        {nodes.map((n) => {
          const p = pos(n.at);
          if (still)
            return (
              <div key={"n" + n.id} className={"node" + (n.tone ? " " + n.tone : "")} style={{ width: 2 * R, height: 2 * R, transform: `translate(${p.x - R}px, ${p.y - R}px)` }}>
                {n.label}
              </div>
            );
          return (
            <motion.div
              key={"n" + n.id}
              className={"node" + (n.tone ? " " + n.tone : "")}
              style={{ width: 2 * R, height: 2 * R }}
              initial={{ opacity: 0, x: p.x - R, y: p.y - R - 20 }}
              animate={{ opacity: 1, x: p.x - R, y: p.y - R }}
              exit={{ opacity: 0 }}
              transition={t}
            >
              {n.label}
            </motion.div>
          );
        })}
      </AnimatePresence>
    </>
  );
}
