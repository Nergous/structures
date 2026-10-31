// Drawing kit for the memory views: a variable card in the stack frame, a
// pointer arrow, and strips of cells on the heap. Everything is absolutely
// positioned inside a <Region>, in pixels.
import { createContext, useContext, type ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import type { Frame, Tone } from "../core/types";

export const VAR_W = 262;
export const HX = 300; // left edge of the heap column
export const CW = 64; // cell width
export const CH = 60; // cell height
export const COLS = 12;
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
}
export const Mem = createContext<MemCtx>({ addrs: true, still: false });
export const useMem = () => useContext(Mem);

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
  const min = card ? card[0] + HEAD_H + card[1] * ROW_H + 30 + 28 : 0;
  return (
    <section className="region" style={{ height: Math.max(h, min) }}>
      <div className="col-l">stack frame</div>
      <div className="col-l" style={{ left: HX }}>
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
  return (
    <div className="var" style={{ top: y, width: VAR_W }}>
      <div className="var-h">
        <b>{name}</b>
        <span>{type}</span>
      </div>
      {fields.map((f) => (
        <div key={f.k} className={"field" + (f.hot ? " hot" : "")}>
          <span>{f.k}</span>
          <motion.code key={String(f.v)} className={"fv-" + (f.kind ?? "num")} initial={{ y: -5, opacity: 0 }} animate={{ y: 0, opacity: 1 }}>
            {f.v}
          </motion.code>
        </div>
      ))}
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
  return (
    <svg className="arrow">
      <defs>
        <marker id="ah" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M0 0 10 5 0 10z" fill="currentColor" />
        </marker>
      </defs>
      <motion.path
        key={id}
        d={`M ${x0} ${y0} C ${mx} ${y0}, ${mx} ${y1}, ${x1 - 3} ${y1}`}
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

export const treeH = (n: number) => (n ? Math.floor(Math.log2(n)) + 1 : 1) * 64;

export function Tree({ x, y, w, nodes, slots, holes = [] }: { x: number; y: number; w: number; nodes: Node[]; slots: number; holes?: number[] }) {
  const t = useMotion();
  const { still } = useMem();
  const R = 21;
  const pos = (i: number) => {
    const lvl = Math.floor(Math.log2(i + 1));
    const first = 2 ** lvl - 1;
    const span = w / 2 ** lvl;
    return { x: x + span * (i - first) + span / 2, y: y + lvl * 64 + R };
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
        return <div key={"h" + i} className="node hole" style={{ left: p.x - R, top: p.y - R }} />;
      })}
      <AnimatePresence>
        {nodes.map((n) => {
          const p = pos(n.at);
          if (still)
            return (
              <div key={"n" + n.id} className={"node" + (n.tone ? " " + n.tone : "")} style={{ transform: `translate(${p.x - R}px, ${p.y - R}px)` }}>
                {n.label}
              </div>
            );
          return (
            <motion.div
              key={"n" + n.id}
              className={"node" + (n.tone ? " " + n.tone : "")}
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
