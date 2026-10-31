// Memory views: each draws one structure the way the Go package lays it out
// (struct fields in the stack frame, backing arrays and maps on the heap).
// Geometry comes from useLay: wide screens put the struct card left of the
// heap, narrow ones put it on top and wrap the heap to the screen width.
import type { FC } from "react";
import { AnimatePresence, motion } from "motion/react";
import type { Cell, Tone } from "../core/types";
import type { BitS } from "../structures/bitset";
import type { HeapS } from "../structures/binheap";
import type { OMS } from "../structures/omcore";
import type { PQS } from "../structures/pqueue";
import type { RingS } from "../structures/ring";
import { grp, type SetS } from "../structures/set";
import type { StackS } from "../structures/stack";
import {
  Arrow,
  Brace,
  Caption,
  MapCard,
  Note,
  Ptr,
  Region,
  Row,
  Strip,
  Tag,
  Tree,
  Var,
  cellX,
  cellY,
  ptrFrom,
  shortAddr,
  tone,
  treeH,
  useLay,
  useMem,
  useMotion,
  type Item,
  type Lay,
  type Slot,
  type ViewProps,
} from "./kit";

const rowsOf = (L: Lay, cap: number) => Math.max(1, Math.ceil(cap / L.COLS));
const arrayBottom = (L: Lay, cap: number, y = L.Y) => y + (rowsOf(L, cap) - 1) * L.ROW + L.CH;
const tagAt = (L: Lay, i: number, label: string, t?: "accent" | "blue", y = L.Y) => (
  <Tag
    key={label + i}
    x={Math.max(0, cellX(L.HX, i, L.COLS, L.CW) + L.CW / 2 - 7 - label.length * 3.4)}
    y={cellY(y, i, L.COLS, L.ROW) - 50}
    label={label}
    tone={t}
  />
);
/** Space under the array taken by braces or the summary line. */
const UNDER = 64;

/** len / cap braces under a one-row array, or a summary line when it wraps. */
function Braces({ L, len, cap, elem, lenLabel, y = L.Y }: { L: Lay; len: number; cap: number; elem: number; lenLabel?: string; y?: number }) {
  if (!cap) return null;
  const capLabel = `cap ${cap} · ${cap * elem} bytes`;
  const lenText = lenLabel ?? `len ${len}`;
  if (cap > L.COLS) {
    return (
      <Caption x={L.HX} y={arrayBottom(L, cap, y) + 10}>
        {lenText} · {capLabel}
      </Caption>
    );
  }
  return (
    <>
      {len > 0 && <Brace x={L.HX} y={y + L.CH + 8} w={len * L.CW} label={lenText} strong />}
      <Brace x={L.HX} y={y + L.CH + 30} w={cap * L.CW} label={capLabel} />
    </>
  );
}

function PtrArrow({ L, cap, gen, vy, y = L.Y }: { L: Lay; cap: number; gen: number; vy?: number; y?: number }) {
  if (!cap) return null;
  return <Arrow id={gen} from={ptrFrom(L, vy)} to={[L.HX, y + L.CH / 2]} />;
}

const strip = (L: Lay) => ({ cw: L.CW, ch: L.CH, cols: L.COLS, rowGap: L.ROW });

const cellItems = (cells: (Cell | null)[], marks: Record<string, Tone> | undefined): Item[] =>
  cells.flatMap((c, i) => (c ? [{ id: c.id, at: i, content: c.v, tone: tone(marks, "c" + c.id) }] : []));

/* ---------- stack ---------- */

export const StackView: FC<ViewProps<StackS>> = ({ f, base }) => {
  const L = useLay(3);
  const s = f.s;
  const n = s.data.length;
  const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({ kind: i < n ? "live" : "spare", zero: "0" }));
  return (
    <Region h={arrayBottom(L, s.cap) + UNDER + 6} card={[L.VY, 3]}>
      <Var
        y={L.VY}
        name="s"
        type="*Stack[int]"
        fields={[
          { k: "data.ptr", v: <Ptr to={s.cap ? base : null} />, kind: s.cap ? "ptr" : "nil" },
          { k: "data.len", v: n },
          { k: "data.cap", v: s.cap },
        ]}
        foot="slice header · 24 bytes"
      />
      <PtrArrow L={L} cap={s.cap} gen={base} />
      {!s.cap && <Note x={L.HX} y={L.Y}>no backing array — the first Push allocates one</Note>}
      <Strip x={L.HX} y={L.Y} slots={slots} items={cellItems(s.data, f.marks)} base={base} {...strip(L)} />
      {n > 0 && tagAt(L, n - 1, "top")}
      <Braces L={L} len={n} cap={s.cap} elem={8} />
    </Region>
  );
};

/* ---------- queue / deque ---------- */

export function ringView(name: string, type: string, front: string, back: string): FC<ViewProps<RingS>> {
  return function RingView({ f, base }) {
    const L = useLay(4);
    const s = f.s;
    const cap = s.data.length;
    const slots: Slot[] = Array.from({ length: cap }, (_, i) => ({ kind: s.data[i] ? "live" : "spare", zero: "0" }));
    const backI = (s.head + s.len - 1) % Math.max(cap, 1);
    const wraps = s.head + s.len > cap;
    const first = wraps ? cap - s.head : s.len;
    const lenLabel = `len ${s.len}${wraps ? " — wraps →" : ""}`;
    return (
      <Region h={arrayBottom(L, cap) + UNDER + 30} card={[L.VY, 4]}>
        <Var
          y={L.VY}
          name={name}
          type={type}
          fields={[
            { k: "buf.data.ptr", v: <Ptr to={cap ? base : null} />, kind: cap ? "ptr" : "nil" },
            { k: "buf.data.len", v: cap },
            { k: "buf.head", v: s.head },
            { k: "buf.len", v: s.len },
          ]}
          foot="ring.Buffer · len(data) is the capacity"
        />
        <PtrArrow L={L} cap={cap} gen={base} />
        {!cap && <Note x={L.HX} y={L.Y}>no backing array yet — the first push allocates one</Note>}
        <Strip x={L.HX} y={L.Y} slots={slots} items={cellItems(s.data, f.marks)} base={base} {...strip(L)} />
        {s.len > 0 && (backI === s.head ? tagAt(L, s.head, `${front} · ${back}`) : [tagAt(L, s.head, front), tagAt(L, backI, back, "blue")])}
        {cap > 0 && cap <= L.COLS ? (
          <>
            {s.len > 0 && <Brace x={L.HX + s.head * L.CW} y={L.Y + L.CH + 8} w={first * L.CW} label={lenLabel} strong />}
            {wraps && <Brace x={L.HX} y={L.Y + L.CH + 8} w={(s.len - first) * L.CW} label="" strong />}
            <Brace x={L.HX} y={L.Y + L.CH + 30} w={cap * L.CW} label={`cap ${cap} · ${cap * 8} bytes`} />
          </>
        ) : (
          <Braces L={L} len={s.len} cap={cap} elem={8} lenLabel={lenLabel} />
        )}
        {cap > 0 && (
          <Caption x={L.HX} y={arrayBottom(L, cap) + (cap > L.COLS ? 32 : 58)}>
            element i lives at data[(head + i) % cap]
          </Caption>
        )}
      </Region>
    );
  };
}

/* ---------- set (Go map) ---------- */

const G = 8;

export const SetView: FC<ViewProps<SetS>> = ({ f, base }) => {
  const L = useLay(2);
  const s = f.s;
  const t = useMotion();
  const room = L.W - L.HX;
  const gcols = room >= 8 * 96 ? 8 : 4;
  const GW = Math.min(96, Math.floor(room / gcols));
  const groups: number[][] = Array.from({ length: G }, () => []);
  for (const v of [...s.a].sort((a, b) => a - b)) groups[grp(v)].push(v);
  const grows = G / gcols;
  const maxIn = (r: number) => Math.max(2, ...groups.slice(r * gcols, (r + 1) * gcols).map((g) => g.length));
  const ghs = Array.from({ length: grows }, (_, r) => 34 + maxIn(r) * 30);
  const gy = (r: number) => L.Y + ghs.slice(0, r).reduce((a, b) => a + b + 8, 0);
  const gridH = ghs.reduce((a, b) => a + b, 0) + (grows - 1) * 8;
  const capY = L.Y + gridH + 8;
  const rowsY = capY + (L.m ? 60 : 32);
  const nil = s.nilMap;
  return (
    <Region h={rowsY + (s.res ? 130 : 70) + (L.m ? 40 : 0)} card={[L.VY, 2]}>
      <Var
        y={L.VY}
        name="s"
        type="*Set[int]"
        fields={[
          { k: "m", v: <Ptr to={nil ? null : base} />, kind: nil ? "nil" : "ptr" },
          { k: "len(m)", v: s.a.length },
        ]}
        foot="map[int]struct{} · values only"
      />
      {!nil && <Arrow id={base} from={ptrFrom(L)} to={[L.HX, L.Y + 16]} />}
      {nil ? (
        <Note x={L.HX} y={L.Y}>m == nil — no map yet; reads still work, Add allocates</Note>
      ) : (
        <>
          {groups.map((_, i) => {
            const r = Math.floor(i / gcols);
            return (
              <div
                key={i}
                className={"group" + (f.marks?.["g" + i] ? " st-" + f.marks["g" + i] : "")}
                style={{ left: L.HX + (i % gcols) * GW, top: gy(r), width: GW - 6, height: ghs[r] }}
              >
                <span>group {i}</span>
              </div>
            );
          })}
          <AnimatePresence>
            {groups.flatMap((g, i) =>
              g.map((v, k) => {
                const x = L.HX + (i % gcols) * GW + 8;
                const y = gy(Math.floor(i / gcols)) + 26 + k * 30;
                return (
                  <motion.div
                    key={v}
                    className={"mval" + (tone(f.marks, "v" + v) ? " " + tone(f.marks, "v" + v) : "")}
                    initial={{ opacity: 0, x, y: y - 12 }}
                    animate={{ opacity: 1, x, y }}
                    exit={{ opacity: 0 }}
                    transition={t}
                    style={{ width: GW - 22 }}
                  >
                    {v}
                  </motion.div>
                );
              }),
            )}
          </AnimatePresence>
          <Caption x={L.HX} y={capY} w={L.m ? room : undefined}>
            hash(v) picks a group; groups here are illustrative — Go seeds its hash per map, so iteration order is random
          </Caption>
        </>
      )}
      <Row
        x={L.HX}
        y={rowsY}
        label="other"
        items={s.b.map((v) => ({ key: "b" + v, text: v, tone: tone(f.marks, "v" + v), dim: !s.a.includes(v) && !tone(f.marks, "v" + v) }))}
      />
      {s.res && (
        <Row x={L.HX} y={rowsY + 60} label={s.resLabel} items={[...s.res].sort((a, b) => a - b).map((v) => ({ key: "r" + v, text: v, tone: tone(f.marks, "r" + v) ?? "ok" }))} />
      )}
    </Region>
  );
};

/* ---------- binheap ---------- */

const ORDER: Record<string, string> = { min: "cmp.Compare (min)", max: "Greater (max)", func: "|a-50| vs |b-50|" };

/** Tree width: as wide as the last level needs, within the free room. */
const treeW = (L: Lay, slots: number, per: number, max: number) =>
  Math.min(L.W - L.HX, max, Math.max(L.m ? 0 : 320, 2 ** Math.floor(Math.log2(Math.max(slots, 1))) * per));

export const HeapView: FC<ViewProps<HeapS>> = ({ f, base }) => {
  const L = useLay(4);
  const s = f.s;
  const n = s.data.length;
  const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({ kind: i < n ? "live" : "spare", zero: "0" }));
  const ty = arrayBottom(L, s.cap) + UNDER + 28;
  const tw = treeW(L, Math.max(s.cap, n), 70, 760);
  return (
    <Region h={ty + treeH(n, L.LVL) + 20} card={[L.VY, 4]}>
      <Var
        y={L.VY}
        name="h"
        type="*Heap[int]"
        fields={[
          { k: "data.ptr", v: <Ptr to={s.cap ? base : null} />, kind: s.cap ? "ptr" : "nil" },
          { k: "data.len", v: n },
          { k: "data.cap", v: s.cap },
          { k: "compare", v: ORDER[s.mode], kind: "text" },
        ]}
        foot="data[0] is always the top"
      />
      <PtrArrow L={L} cap={s.cap} gen={base} />
      {!s.cap && <Note x={L.HX} y={L.Y}>no backing array yet</Note>}
      <Strip x={L.HX} y={L.Y} slots={slots} items={cellItems(s.data, f.marks)} base={base} {...strip(L)} />
      {n > 0 && tagAt(L, 0, "top")}
      <Braces L={L} len={n} cap={s.cap} elem={8} />
      <Caption x={L.HX} y={ty - 30}>
        {L.m ? "the slice as a tree: children of i are 2i+1, 2i+2" : "the same slice read as a tree: children of i are 2i+1 and 2i+2"}
      </Caption>
      <Tree x={L.HX} y={ty} w={tw} r={L.NR} lvlH={L.LVL} slots={n} nodes={s.data.map((c, i) => ({ id: c.id, at: i, label: c.v, tone: tone(f.marks, "c" + c.id) }))} />
    </Region>
  );
};

/* ---------- pqueue ---------- */

const PQ_ELEM = 32;

export const PQView: FC<ViewProps<PQS>> = ({ f, base }) => {
  const L = useLay(5);
  const s = f.s;
  const n = s.data.length;
  // room above the array for an entry lifted out during a sift
  const PY = L.m ? L.Y + L.CH - 20 : L.Y + 70;
  const vy = L.m ? L.VY : PY - 38;
  const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({
    kind: i < n ? (s.data[i] ? "live" : "hole") : "spare",
    zero: i < n ? "hole" : "",
  }));
  const items: Item[] = s.data.flatMap((e, i) =>
    e ? [{ id: e.id, at: i, content: <KP k={e.k} p={e.v} small={L.m} />, tone: tone(f.marks, "c" + e.id) }] : [],
  );
  if (s.held) items.push({ id: s.held.e.id, at: s.held.at, content: <KP k={s.held.e.k} p={s.held.e.v} small={L.m} />, tone: "held", lifted: true });
  const ty = arrayBottom(L, s.cap, PY) + UNDER + 28;
  const holes = s.data.flatMap((e, i) => (e ? [] : [i]));
  const keys = Object.keys(s.index).sort();
  const tw = treeW(L, Math.max(s.cap, n), 66, 520);
  const side = L.W - L.HX >= tw + 220;
  const th = treeH(n, L.LVL);
  const my = side ? ty - 6 : ty + th + 16;
  const mapH = 34 + Math.max(keys.length, 1) * 28;
  return (
    <Region h={Math.max(ty + th, my + mapH) + 30} card={[vy, 5]}>
      <Var
        y={vy}
        name="q"
        type="*Queue[string, int]"
        fields={[
          { k: "data.ptr", v: <Ptr to={s.cap ? base : null} />, kind: s.cap ? "ptr" : "nil" },
          { k: "data.len", v: n },
          { k: "data.cap", v: s.cap },
          { k: "index", v: keys.length || s.cap ? "map[K]int" : "nil", kind: "text" },
          { k: "nextSeq", v: s.nextSeq },
        ]}
        foot={`entry{key, priority, seq} · ${PQ_ELEM} bytes`}
      />
      <PtrArrow L={L} cap={s.cap} gen={base} vy={vy} y={PY} />
      {!s.cap && <Note x={L.HX} y={PY}>no backing array yet</Note>}
      <Strip x={L.HX} y={PY} slots={slots} items={items} base={base} elem={PQ_ELEM} {...strip(L)} />
      <Braces L={L} len={n} cap={s.cap} elem={PQ_ELEM} y={PY} />
      <Caption x={L.HX} y={ty - 30}>
        heap order by priority, ties by seq (push order)
      </Caption>
      <Tree
        x={L.HX}
        y={ty}
        w={tw}
        r={L.NR}
        lvlH={L.LVL}
        slots={n}
        holes={holes}
        nodes={s.data.flatMap((e, i) => (e ? [{ id: e.id, at: i, label: <KP k={e.k} p={e.v} small />, tone: tone(f.marks, "c" + e.id) }] : []))}
      />
      <MapCard
        x={side ? L.HX + tw + 40 : L.HX}
        y={my}
        w={180}
        title="index map[K]int"
        rows={keys.map((k) => ({ k: JSON.stringify(k), v: s.index[k], tone: tone(f.marks, "k" + k) }))}
      />
    </Region>
  );
};

function KP({ k, p, small }: { k: string; p: number; small?: boolean }) {
  return (
    <span className={"kp" + (small ? " sm" : "")}>
      <b>{k}</b>
      <i>{p}</i>
    </span>
  );
}

/* ---------- orderedmap / orderedset ---------- */

export function omView(name: string, type: string, values: boolean): FC<ViewProps<OMS>> {
  return function OMView({ f, base }) {
    const L = useLay(6);
    const s = f.s;
    const n = s.entries.length;
    const live = s.entries.filter(Boolean).length;
    const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({
      kind: i < n ? (s.entries[i] ? "live" : "hole") : "spare",
      zero: i < n ? "hole" : "",
      tone: f.marks?.["s" + i],
    }));
    const items: Item[] = s.entries.flatMap((e, i) =>
      e ? [{ id: e.id, at: i, content: values ? <KP k={e.k} p={e.v} small={L.m} /> : e.k, tone: tone(f.marks, "c" + e.id) }] : [],
    );
    const elem = s.elem;
    const iy = arrayBottom(L, s.cap) + UNDER + 12;
    const keys = s.entries.flatMap((e, i) => (e ? [[e.k, i] as const] : []));
    const mapH = 34 + Math.max(keys.length, 1) * 28;
    const ops = !values;
    const side = L.W - L.HX >= 560;
    const rx = side ? L.HX + 240 : L.HX;
    const ry = side ? iy : iy + mapH + 20;
    return (
      <Region h={Math.max(iy + mapH, ops ? ry + (s.res ? 130 : 70) : 0) + 20} card={[L.VY, 6]}>
        <Var
          y={L.VY}
          name={name}
          type={type}
          fields={[
            { k: "entries.ptr", v: <Ptr to={s.cap ? base : null} />, kind: s.cap ? "ptr" : "nil" },
            { k: "entries.len", v: n },
            { k: "entries.cap", v: s.cap },
            { k: "index", v: s.cap ? "map[K]int" : "nil", kind: "text" },
            { k: "head", v: s.head },
            { k: "holes", v: s.holes, hot: s.holes > 0 },
          ]}
          foot={`entry{key${values ? ", value" : ""}, alive} · ${elem} bytes`}
        />
        <PtrArrow L={L} cap={s.cap} gen={base} />
        {!s.cap && <Note x={L.HX} y={L.Y}>entries == nil</Note>}
        <Strip x={L.HX} y={L.Y} slots={slots} items={items} base={base} elem={elem} {...strip(L)} />
        {live > 0 && tagAt(L, s.head, "head")}
        <Braces L={L} len={n} cap={s.cap} elem={elem} lenLabel={L.m ? `len ${n} · ${s.holes} holes` : `len(entries) ${n} · ${live} live, ${s.holes} holes`} />
        <MapCard
          x={L.HX}
          y={iy}
          w={200}
          title="index map[K]int"
          rows={keys.map(([k, i]) => ({ k: values ? JSON.stringify(k) : k, v: i, tone: tone(f.marks, "k" + k) }))}
        />
        {ops && s.b && (
          <Row x={rx} y={ry} label="other" items={s.b.map((k) => ({ key: "b" + k, text: k, tone: tone(f.marks, "b" + k), dim: !tone(f.marks, "b" + k) }))} />
        )}
        {ops && s.res && (
          <Row x={rx} y={ry + 62} label={s.resLabel ?? "result"} items={s.res.map((k) => ({ key: "r" + k, text: k, tone: tone(f.marks, "r" + k) ?? "ok" }))} />
        )}
      </Region>
    );
  };
}

/* ---------- bitset ---------- */

const ONE = 1n;

function Words({ L, ws, len, marks, prefix, y, addrBase }: { L: Lay; ws: bigint[]; len: number; marks?: Record<string, Tone>; prefix: string; y: number; addrBase?: number }) {
  const { addrs } = useMem();
  // narrow screens: each word wraps into two lines of 32 bits
  const per = L.m ? 32 : 64;
  return (
    <div className={"words" + (L.m ? " m" : "")} style={{ left: L.HX, top: y }}>
      {ws.map((w, i) => {
        const spare = i >= len;
        const wt = prefix === "b" ? marks?.["w" + i] : undefined;
        const lines = [];
        for (let from = 0; from < 64; from += per) {
          const bits = [];
          for (let b = from; b < from + per; b++) {
            const v = i * 64 + b;
            const on = ((w >> BigInt(b)) & ONE) === ONE;
            const t = marks?.[prefix + v];
            bits.push(<span key={b} className={"bit" + (on ? (spare ? " stale" : " on") : "") + (t ? " t-" + t : "") + (b % 8 === 0 && b > from ? " oct" : "")} title={String(v)} />);
          }
          lines.push(
            <span key={from} className="bits">
              {bits}
            </span>,
          );
        }
        return (
          <div key={i} className={"word" + (spare ? " spare" : "") + (wt ? " wt-" + wt : "")}>
            {!L.m && <span className="word-a">{addrs && addrBase !== undefined ? shortAddr(addrBase + i * 8) : ""}</span>}
            <span className="word-i">
              [{i}]<i>{i * 64}–{i * 64 + 63}</i>
            </span>
            {L.m ? <span className="bits2">{lines}</span> : lines}
            {L.m ? (
              spare && <code className="word-h">{w ? "stale" : "spare"}</code>
            ) : (
              <code className="word-h">{spare ? (w ? "stale bits" : "spare") : "0x" + w.toString(16).padStart(16, "0")}</code>
            )}
          </div>
        );
      })}
      {!ws.length && <div className="words-e">no words</div>}
    </div>
  );
}

const wordsOf = (vs: number[]) => {
  const ws: bigint[] = [];
  for (const v of vs) {
    while (ws.length <= v >> 6) ws.push(0n);
    ws[v >> 6] |= ONE << BigInt(v & 63);
  }
  return ws;
};

export const BitsetView: FC<ViewProps<BitS>> = ({ f, base }) => {
  const L = useLay(4);
  const s = f.s;
  const cap = s.data.length;
  const WH = L.m ? 40 : 32;
  const Y = L.m ? L.Y - 50 : L.Y;
  const oy = Y + Math.max(cap, 1) * WH + 40;
  const bw = wordsOf(s.b);
  const ry = oy + 34 + Math.max(bw.length, 1) * WH + 20;
  return (
    <Region h={(s.res ? ry + 34 + Math.max(s.res.length, 1) * WH : ry) + 20} card={[L.VY, 4]}>
      <Var
        y={L.VY}
        name="s"
        type="*Set"
        fields={[
          { k: "data.ptr", v: <Ptr to={cap ? base : null} />, kind: cap ? "ptr" : "nil" },
          { k: "data.len", v: s.len },
          { k: "data.cap", v: cap },
          { k: "n", v: s.n },
        ]}
        foot="[]uint64 · v is bit v%64 of word v/64"
      />
      {cap > 0 && <Arrow id={base} from={ptrFrom(L)} to={[L.HX, Y + 14]} />}
      {!cap && <Note x={L.HX} y={Y}>data == nil</Note>}
      <Words L={L} ws={s.data} len={s.len} marks={f.marks} prefix="b" y={Y} addrBase={base} />
      <div className="words-t" style={{ left: L.HX, top: oy }}>
        other
      </div>
      <Words L={L} ws={bw} len={bw.length} marks={f.marks} prefix="o" y={oy + 22} />
      {s.res && (
        <>
          <div className="words-t" style={{ left: L.HX, top: ry }}>
            {s.resLabel}
          </div>
          <Words L={L} ws={s.res} len={s.res.length} marks={f.marks} prefix="r" y={ry + 22} />
        </>
      )}
    </Region>
  );
};
