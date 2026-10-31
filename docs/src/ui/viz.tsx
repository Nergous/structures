import { createContext, useContext, type ReactNode } from "react";
import { AnimatePresence, motion } from "motion/react";
import type { Cell, Frame, Tone } from "../core/types";

/** Playback speed multiplier, shared with every animated primitive. */
export const SpeedCtx = createContext(1);

export function useEase() {
  const speed = useContext(SpeedCtx);
  return { duration: 0.42 / speed, ease: [0.25, 0.8, 0.25, 1] as const };
}

export type Marks = Frame<unknown>["marks"];

export const toneOf = (
  marks: Marks,
  ...keys: (string | undefined)[]
): Tone | undefined => {
  if (!marks) return undefined;
  for (const k of keys) if (k && marks[k]) return marks[k];
  return undefined;
};

/** Fixed-size drawing surface; children are absolutely positioned. */
export function Stage(props: {
  w: number;
  h: number;
  children: ReactNode;
}) {
  return (
    <div className="stage-scroll">
      <div className="stage" style={{ width: props.w, height: props.h }}>
        {props.children}
      </div>
    </div>
  );
}

/** An animated value tile. Keyed by the caller with the cell identity. */
export function Box(props: {
  x: number;
  y: number;
  w?: number;
  h?: number;
  tone?: Tone;
  round?: boolean;
  small?: boolean;
  children: ReactNode;
  title?: string;
}) {
  const t = useEase();
  const w = props.w ?? 44;
  const h = props.h ?? 44;
  return (
    <motion.div
      className={
        "box" +
        (props.tone ? " t-" + props.tone : "") +
        (props.round ? " round" : "") +
        (props.small ? " small" : "")
      }
      style={{ width: w, height: h }}
      title={props.title}
      initial={{ x: props.x, y: props.y - 26, opacity: 0, scale: 0.5 }}
      animate={{ x: props.x, y: props.y, opacity: 1, scale: 1 }}
      exit={{ y: props.y - 34, opacity: 0, scale: 0.4 }}
      transition={t}
    >
      {props.children}
    </motion.div>
  );
}

/** A static slot outline (a position in a backing array). */
export function Slot(props: {
  x: number;
  y: number;
  w?: number;
  h?: number;
  kind?: "used" | "spare" | "hole" | "stale";
  tone?: Tone;
  index?: number | string;
  indexBelow?: boolean;
}) {
  const t = useEase();
  const w = props.w ?? 44;
  const h = props.h ?? 44;
  return (
    <motion.div
      className={
        "slot k-" + (props.kind ?? "used") + (props.tone ? " st-" + props.tone : "")
      }
      style={{ width: w, height: h }}
      initial={{ x: props.x, y: props.y, opacity: 0 }}
      animate={{ x: props.x, y: props.y, opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={t}
    >
      {props.index !== undefined && (
        <span className={"slot-i" + (props.indexBelow === false ? " above" : "")}>
          {props.index}
        </span>
      )}
    </motion.div>
  );
}

/** A floating label with a small arrow, e.g. "head", "top", "len". */
export function Pointer(props: {
  id: string;
  x: number;
  y: number;
  label: string;
  dir?: "down" | "up" | "left" | "right";
  tone?: "accent" | "cyan" | "ink";
}) {
  const t = useEase();
  return (
    <motion.div
      className={"ptr d-" + (props.dir ?? "down") + " p-" + (props.tone ?? "accent")}
      initial={{ x: props.x, y: props.y, opacity: 0 }}
      animate={{ x: props.x, y: props.y, opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={t}
    >
      {props.label}
    </motion.div>
  );
}

export function Label(props: {
  x: number;
  y: number;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={"lbl " + (props.className ?? "")}
      style={{ transform: `translate(${props.x}px, ${props.y}px)` }}
    >
      {props.children}
    </div>
  );
}

export interface ArrayViewProps {
  x: number;
  y: number;
  /** Slots of the backing array; null marks an empty slot. */
  cells: (Cell | null)[];
  /** Number of slots to draw; cells beyond length are spare capacity. */
  cap: number;
  /** Which slots are inside the live window (default: non-null). */
  live?: (i: number) => boolean;
  hole?: (i: number) => boolean;
  stale?: (i: number) => boolean;
  marks?: Marks;
  cw?: number;
  gap?: number;
  cols?: number;
  render?: (c: Cell) => ReactNode;
  boxW?: number;
  vertical?: boolean;
  /** Prefix for React keys, so one cell can appear in two views. */
  ns?: string;
}

/** Position of slot i in an ArrayView laid out from (x, y). */
export function slotPos(
  i: number,
  o: { x: number; y: number; cw?: number; gap?: number; cols?: number; vertical?: boolean; rows?: number },
): { x: number; y: number } {
  const cw = o.cw ?? 44;
  const gap = o.gap ?? 6;
  if (o.vertical) {
    // bottom-up column
    const rows = o.rows ?? 1;
    return { x: o.x, y: o.y + (rows - 1 - i) * (cw + gap) };
  }
  const cols = o.cols ?? 1e9;
  return {
    x: o.x + (i % cols) * (cw + gap),
    y: o.y + Math.floor(i / cols) * (cw + gap + 14),
  };
}

/** A backing array: slot outlines plus animated cells inside them. */
export function ArrayView(p: ArrayViewProps) {
  const cw = p.cw ?? 44;
  const ns = p.ns ?? "a";
  const geo = { ...p, rows: p.cap };
  const slots = [];
  for (let i = 0; i < p.cap; i++) {
    const pos = slotPos(i, geo);
    const live = p.live ? p.live(i) : p.cells[i] != null;
    const kind = p.hole?.(i)
      ? "hole"
      : p.stale?.(i)
        ? "stale"
        : live
          ? "used"
          : "spare";
    slots.push(
      <Slot
        key={ns + "s" + i}
        x={pos.x}
        y={pos.y}
        w={cw}
        h={cw}
        kind={kind}
        tone={p.marks?.["s" + i]}
        index={p.vertical ? undefined : i}
      />,
    );
  }
  const boxes = [];
  for (let i = 0; i < p.cells.length; i++) {
    const c = p.cells[i];
    if (!c) continue;
    const pos = slotPos(i, geo);
    const bw = p.boxW ?? cw - 6;
    boxes.push(
      <Box
        key={ns + c.id}
        x={pos.x + (cw - bw) / 2}
        y={pos.y + 3}
        w={bw}
        h={cw - 6}
        tone={toneOf(p.marks, "c" + c.id, "s" + i)}
      >
        {p.render ? p.render(c) : c.v}
      </Box>,
    );
  }
  return (
    <>
      <AnimatePresence>{slots}</AnimatePresence>
      <AnimatePresence>{boxes}</AnimatePresence>
    </>
  );
}

/** Vertical index labels for a bottom-up column. */
export function ColumnIndex(p: { x: number; y: number; cap: number; cw?: number; gap?: number }) {
  const cw = p.cw ?? 44;
  const out = [];
  for (let i = 0; i < p.cap; i++) {
    const pos = slotPos(i, { x: p.x, y: p.y, cw, gap: p.gap, vertical: true, rows: p.cap });
    out.push(
      <Label key={i} x={pos.x} y={pos.y + cw / 2 - 8} className="col-i">
        {i}
      </Label>,
    );
  }
  return <>{out}</>;
}

/** The tray of values returned or yielded by the current call. */
export function Tray({ f }: { f: Frame<unknown> }) {
  const t = useEase();
  const items = f.out ?? [];
  return (
    <div className={"tray" + (items.length ? " on" : "")}>
      <span className="tray-l">{f.outLabel ?? "returned"}</span>
      <div className="tray-items">
        <AnimatePresence>
          {items.map((it) => (
            <motion.span
              key={it.id}
              className={"tray-i" + (it.tone ? " t-" + it.tone : "")}
              initial={{ opacity: 0, y: -18, scale: 0.6 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0 }}
              transition={t}
            >
              {it.t}
            </motion.span>
          ))}
        </AnimatePresence>
        {!items.length && <span className="tray-empty">nothing yet</span>}
      </div>
    </div>
  );
}

/** Horizontal len / cap gauge drawn under array-backed views. */
export function CapBar(p: { len: number; cap: number; extra?: ReactNode }) {
  const t = useEase();
  const max = Math.max(p.cap, 1);
  return (
    <div className="capbar">
      <div className="capbar-track">
        <motion.div
          className="capbar-cap"
          animate={{ width: p.cap ? "100%" : "0%" }}
          transition={t}
        />
        <motion.div
          className="capbar-len"
          animate={{ width: `${(p.len / max) * 100}%` }}
          transition={t}
        />
      </div>
      <span className="capbar-t">
        len <b>{p.len}</b> · cap <b>{p.cap}</b>
        {p.extra}
      </span>
    </div>
  );
}
