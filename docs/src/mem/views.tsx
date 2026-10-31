// Memory views: each draws one structure the way the Go package lays it out
// (struct fields in the stack frame, backing arrays and maps on the heap).
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
  CH,
  COLS,
  CW,
  Caption,
  HX,
  MapCard,
  Note,
  Ptr,
  Region,
  Row,
  Strip,
  Tag,
  Tree,
  Var,
  VAR_W,
  cellX,
  cellY,
  fieldY,
  shortAddr,
  tone,
  treeH,
  useMem,
  useMotion,
  type Item,
  type Slot,
  type ViewProps,
} from "./kit";

const Y = 92; // top of the first cell row
const VY = Y - 38; // top of the variable card, so data.ptr lines up
const ROW = 96;

const rowsOf = (cap: number) => Math.max(1, Math.ceil(cap / COLS));
const arrayBottom = (cap: number) => Y + (rowsOf(cap) - 1) * ROW + CH;
const center = (i: number) => cellX(HX, i) + CW / 2;
const tagAt = (i: number, label: string, t?: "accent" | "blue") => (
  <Tag key={label + i} x={center(i) - 4 - label.length * 3.4} y={cellY(Y, i) - 50} label={label} tone={t} />
);

/** len / cap braces under a one-row array. */
function Braces({ len, cap, elem, lenLabel }: { len: number; cap: number; elem: number; lenLabel?: string }) {
  if (cap > COLS) return null;
  return (
    <>
      {len > 0 && <Brace x={HX} y={Y + CH + 8} w={len * CW} label={lenLabel ?? `len ${len}`} strong />}
      <Brace x={HX} y={Y + CH + 30} w={cap * CW} label={`cap ${cap} · ${cap * elem} bytes`} />
    </>
  );
}

function PtrArrow({ cap, gen }: { cap: number; gen: number }) {
  if (!cap) return null;
  return <Arrow id={gen} from={[VAR_W, fieldY(VY, 0)]} to={[HX, Y + CH / 2]} />;
}

const cellItems = (cells: (Cell | null)[], marks: Record<string, Tone> | undefined): Item[] =>
  cells.flatMap((c, i) => (c ? [{ id: c.id, at: i, content: c.v, tone: tone(marks, "c" + c.id) }] : []));

/* ---------- stack ---------- */

export const StackView: FC<ViewProps<StackS>> = ({ f, base }) => {
  const s = f.s;
  const n = s.data.length;
  const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({ kind: i < n ? "live" : "spare", zero: "0" }));
  return (
    <Region h={arrayBottom(s.cap) + 70} card={[VY, 3]}>
      <Var
        y={VY}
        name="s"
        type="*Stack[int]"
        fields={[
          { k: "data.ptr", v: <Ptr to={s.cap ? base : null} />, kind: s.cap ? "ptr" : "nil" },
          { k: "data.len", v: n },
          { k: "data.cap", v: s.cap },
        ]}
        foot="slice header · 24 bytes"
      />
      <PtrArrow cap={s.cap} gen={base} />
      {!s.cap && <Note x={HX} y={Y}>no backing array — the first Push allocates one</Note>}
      <Strip x={HX} y={Y} slots={slots} items={cellItems(s.data, f.marks)} base={base} />
      {n > 0 && tagAt(n - 1, "top")}
      <Braces len={n} cap={s.cap} elem={8} />
    </Region>
  );
};

/* ---------- queue / deque ---------- */

export function ringView(name: string, type: string, front: string, back: string): FC<ViewProps<RingS>> {
  return function RingView({ f, base }) {
    const s = f.s;
    const cap = s.data.length;
    const slots: Slot[] = Array.from({ length: cap }, (_, i) => ({ kind: s.data[i] ? "live" : "spare", zero: "0" }));
    const backI = (s.head + s.len - 1) % Math.max(cap, 1);
    const wraps = s.head + s.len > cap;
    const first = wraps ? cap - s.head : s.len;
    return (
      <Region h={arrayBottom(cap) + 96} card={[VY, 4]}>
        <Var
          y={VY}
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
        <PtrArrow cap={cap} gen={base} />
        {!cap && <Note x={HX} y={Y}>no backing array yet — the first push allocates one</Note>}
        <Strip x={HX} y={Y} slots={slots} items={cellItems(s.data, f.marks)} base={base} />
        {s.len > 0 && (backI === s.head ? tagAt(s.head, `${front} · ${back}`) : [tagAt(s.head, front), tagAt(backI, back, "blue")])}
        {cap > 0 && cap <= COLS && (
          <>
            {s.len > 0 && <Brace x={HX + s.head * CW} y={Y + CH + 8} w={first * CW} label={`len ${s.len}${wraps ? " — wraps →" : ""}`} strong />}
            {wraps && <Brace x={HX} y={Y + CH + 8} w={(s.len - first) * CW} label="" strong />}
            <Brace x={HX} y={Y + CH + 30} w={cap * CW} label={`cap ${cap} · ${cap * 8} bytes`} />
          </>
        )}
        {cap > 0 && (
          <Caption x={HX} y={arrayBottom(cap) + 58}>
            element i lives at data[(head + i) % cap]
          </Caption>
        )}
      </Region>
    );
  };
}

/* ---------- set (Go map) ---------- */

const G = 8;
const GW = 96;

export const SetView: FC<ViewProps<SetS>> = ({ f, base }) => {
  const s = f.s;
  const t = useMotion();
  const groups: number[][] = Array.from({ length: G }, () => []);
  for (const v of [...s.a].sort((a, b) => a - b)) groups[grp(v)].push(v);
  const maxG = Math.max(2, ...groups.map((g) => g.length));
  const gh = 34 + maxG * 30;
  const rowsY = Y + gh + 40;
  const nil = s.nilMap;
  return (
    <Region h={rowsY + (s.res ? 130 : 70)} card={[VY, 2]}>
      <Var
        y={VY}
        name="s"
        type="*Set[int]"
        fields={[
          { k: "m", v: <Ptr to={nil ? null : base} />, kind: nil ? "nil" : "ptr" },
          { k: "len(m)", v: s.a.length },
        ]}
        foot="map[int]struct{} · values only"
      />
      {!nil && <Arrow id={base} from={[VAR_W, fieldY(VY, 0)]} to={[HX, Y + 16]} />}
      {nil ? (
        <Note x={HX} y={Y}>m == nil — no map yet; reads still work, Add allocates</Note>
      ) : (
        <>
          {groups.map((_, i) => (
            <div key={i} className={"group" + (f.marks?.["g" + i] ? " st-" + f.marks["g" + i] : "")} style={{ left: HX + i * GW, top: Y, width: GW - 6, height: gh }}>
              <span>group {i}</span>
            </div>
          ))}
          <AnimatePresence>
            {groups.flatMap((g, i) =>
              g.map((v, k) => (
                <motion.div
                  key={v}
                  className={"mval" + (tone(f.marks, "v" + v) ? " " + tone(f.marks, "v" + v) : "")}
                  initial={{ opacity: 0, x: HX + i * GW + 8, y: Y + 26 + k * 30 - 12 }}
                  animate={{ opacity: 1, x: HX + i * GW + 8, y: Y + 26 + k * 30 }}
                  exit={{ opacity: 0 }}
                  transition={t}
                  style={{ width: GW - 22 }}
                >
                  {v}
                </motion.div>
              )),
            )}
          </AnimatePresence>
          <Caption x={HX} y={Y + gh + 8}>
            hash(v) picks a group; groups here are illustrative — Go seeds its hash per map, so iteration order is random
          </Caption>
        </>
      )}
      <Row
        x={HX}
        y={rowsY}
        label="other"
        items={s.b.map((v) => ({ key: "b" + v, text: v, tone: tone(f.marks, "v" + v), dim: !s.a.includes(v) && !tone(f.marks, "v" + v) }))}
      />
      {s.res && (
        <Row x={HX} y={rowsY + 60} label={s.resLabel} items={[...s.res].sort((a, b) => a - b).map((v) => ({ key: "r" + v, text: v, tone: tone(f.marks, "r" + v) ?? "ok" }))} />
      )}
    </Region>
  );
};

/* ---------- binheap ---------- */

const ORDER: Record<string, string> = { min: "cmp.Compare (min)", max: "Greater (max)", func: "|a-50| vs |b-50|" };

export const HeapView: FC<ViewProps<HeapS>> = ({ f, base }) => {
  const s = f.s;
  const n = s.data.length;
  const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({ kind: i < n ? "live" : "spare", zero: "0" }));
  const ty = arrayBottom(s.cap) + 92;
  const tw = Math.min(760, Math.max(360, 2 ** Math.floor(Math.log2(Math.max(s.cap, n, 1))) * 70));
  return (
    <Region h={ty + treeH(n) + 20} card={[VY, 4]}>
      <Var
        y={VY}
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
      <PtrArrow cap={s.cap} gen={base} />
      {!s.cap && <Note x={HX} y={Y}>no backing array yet</Note>}
      <Strip x={HX} y={Y} slots={slots} items={cellItems(s.data, f.marks)} base={base} />
      {n > 0 && tagAt(0, "top")}
      <Braces len={n} cap={s.cap} elem={8} />
      <Caption x={HX} y={ty - 30}>
        the same slice read as a tree: children of i are 2i+1 and 2i+2
      </Caption>
      <Tree x={HX} y={ty} w={tw} slots={n} nodes={s.data.map((c, i) => ({ id: c.id, at: i, label: c.v, tone: tone(f.marks, "c" + c.id) }))} />
    </Region>
  );
};

/* ---------- pqueue ---------- */

const PQ_ELEM = 32;
const PY = Y + 70; // room above for a lifted entry

export const PQView: FC<ViewProps<PQS>> = ({ f, base }) => {
  const s = f.s;
  const n = s.data.length;
  const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({
    kind: i < n ? (s.data[i] ? "live" : "hole") : "spare",
    zero: i < n ? "hole" : "",
  }));
  const items: Item[] = s.data.flatMap((e, i) =>
    e ? [{ id: e.id, at: i, content: <KP k={e.k} p={e.v} />, tone: tone(f.marks, "c" + e.id) }] : [],
  );
  if (s.held) items.push({ id: s.held.e.id, at: s.held.at, content: <KP k={s.held.e.k} p={s.held.e.v} />, tone: "held", lifted: true });
  const bottom = PY + (rowsOf(s.cap) - 1) * ROW + CH;
  const ty = bottom + 92;
  const holes = s.data.flatMap((e, i) => (e ? [] : [i]));
  const keys = Object.keys(s.index).sort();
  const tw = Math.min(520, Math.max(320, 2 ** Math.floor(Math.log2(Math.max(s.cap, n, 1))) * 66));
  const vy = PY - 38;
  return (
    <Region h={Math.max(ty + treeH(n), ty + 40 + keys.length * 28) + 30} card={[PY - 38, 5]}>
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
      {s.cap > 0 && <Arrow id={base} from={[VAR_W, fieldY(vy, 0)]} to={[HX, PY + CH / 2]} />}
      {!s.cap && <Note x={HX} y={PY}>no backing array yet</Note>}
      <Strip x={HX} y={PY} slots={slots} items={items} base={base} elem={PQ_ELEM} />
      {s.cap > 0 && s.cap <= COLS && (
        <>
          {n > 0 && <Brace x={HX} y={PY + CH + 8} w={n * CW} label={`len ${n}`} strong />}
          <Brace x={HX} y={PY + CH + 30} w={s.cap * CW} label={`cap ${s.cap} · ${s.cap * PQ_ELEM} bytes`} />
        </>
      )}
      <Caption x={HX} y={ty - 30}>
        heap order by priority, ties by seq (push order)
      </Caption>
      <Tree
        x={HX}
        y={ty}
        w={tw}
        slots={n}
        holes={holes}
        nodes={s.data.flatMap((e, i) => (e ? [{ id: e.id, at: i, label: <KP k={e.k} p={e.v} small />, tone: tone(f.marks, "c" + e.id) }] : []))}
      />
      <MapCard
        x={HX + tw + 40}
        y={ty - 6}
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
    const s = f.s;
    const n = s.entries.length;
    const live = s.entries.filter(Boolean).length;
    const slots: Slot[] = Array.from({ length: s.cap }, (_, i) => ({
      kind: i < n ? (s.entries[i] ? "live" : "hole") : "spare",
      zero: i < n ? "hole" : "",
      tone: f.marks?.["s" + i],
    }));
    const items: Item[] = s.entries.flatMap((e, i) =>
      e ? [{ id: e.id, at: i, content: values ? <KP k={e.k} p={e.v} /> : e.k, tone: tone(f.marks, "c" + e.id) }] : [],
    );
    const elem = s.elem;
    const iy = arrayBottom(s.cap) + 76;
    const keys = s.entries.flatMap((e, i) => (e ? [[e.k, i] as const] : []));
    const ops = !values;
    return (
      <Region h={Math.max(iy + 46 + keys.length * 28, ops ? iy + (s.res ? 170 : 100) : 0) + 20} card={[VY, 6]}>
        <Var
          y={VY}
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
        <PtrArrow cap={s.cap} gen={base} />
        {!s.cap && <Note x={HX} y={Y}>entries == nil</Note>}
        <Strip x={HX} y={Y} slots={slots} items={items} base={base} elem={elem} />
        {live > 0 && tagAt(s.head, "head")}
        {s.cap > 0 && s.cap <= COLS && (
          <>
            {n > 0 && <Brace x={HX} y={Y + CH + 8} w={n * CW} label={`len(entries) ${n} · ${live} live, ${s.holes} holes`} strong />}
            <Brace x={HX} y={Y + CH + 30} w={s.cap * CW} label={`cap ${s.cap} · ${s.cap * elem} bytes`} />
          </>
        )}
        <MapCard
          x={HX}
          y={iy}
          w={200}
          title="index map[K]int"
          rows={keys.map(([k, i]) => ({ k: values ? JSON.stringify(k) : k, v: i, tone: tone(f.marks, "k" + k) }))}
        />
        {ops && s.b && (
          <Row
            x={HX + 240}
            y={iy}
            label="other"
            items={s.b.map((k) => ({ key: "b" + k, text: k, tone: tone(f.marks, "b" + k), dim: !tone(f.marks, "b" + k) }))}
          />
        )}
        {ops && s.res && (
          <Row x={HX + 240} y={iy + 62} label={s.resLabel ?? "result"} items={s.res.map((k) => ({ key: "r" + k, text: k, tone: tone(f.marks, "r" + k) ?? "ok" }))} />
        )}
      </Region>
    );
  };
}

/* ---------- bitset ---------- */

const ONE = 1n;

function Words({ ws, len, marks, prefix, y, addrBase }: { ws: bigint[]; len: number; marks?: Record<string, Tone>; prefix: string; y: number; addrBase?: number }) {
  const { addrs } = useMem();
  return (
    <div className="words" style={{ left: HX, top: y }}>
      {ws.map((w, i) => {
        const spare = i >= len;
        const wt = prefix === "b" ? marks?.["w" + i] : undefined;
        const bits = [];
        for (let b = 0; b < 64; b++) {
          const v = i * 64 + b;
          const on = ((w >> BigInt(b)) & ONE) === ONE;
          const t = marks?.[prefix + v];
          bits.push(<span key={b} className={"bit" + (on ? (spare ? " stale" : " on") : "") + (t ? " t-" + t : "") + (b % 8 === 0 && b ? " oct" : "")} title={String(v)} />);
        }
        return (
          <div key={i} className={"word" + (spare ? " spare" : "") + (wt ? " wt-" + wt : "")}>
            <span className="word-a">{addrs && addrBase !== undefined ? shortAddr(addrBase + i * 8) : ""}</span>
            <span className="word-i">
              [{i}]<i>{i * 64}–{i * 64 + 63}</i>
            </span>
            <span className="bits">{bits}</span>
            <code className="word-h">{spare ? (w ? "stale bits" : "spare") : "0x" + w.toString(16).padStart(16, "0")}</code>
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
  const s = f.s;
  const cap = s.data.length;
  const WH = 32;
  const oy = Y + Math.max(cap, 1) * WH + 40;
  const bw = wordsOf(s.b);
  const ry = oy + 34 + Math.max(bw.length, 1) * WH + 20;
  return (
    <Region h={(s.res ? ry + 34 + Math.max(s.res.length, 1) * WH : ry) + 20} card={[VY, 4]}>
      <Var
        y={VY}
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
      {cap > 0 && <Arrow id={base} from={[VAR_W, fieldY(VY, 0)]} to={[HX, Y + 14]} />}
      {!cap && <Note x={HX} y={Y}>data == nil</Note>}
      <Words ws={s.data} len={s.len} marks={f.marks} prefix="b" y={Y} addrBase={base} />
      <div className="words-t" style={{ left: HX, top: oy }}>
        other
      </div>
      <Words ws={bw} len={bw.length} marks={f.marks} prefix="o" y={oy + 22} />
      {s.res && (
        <>
          <div className="words-t" style={{ left: HX, top: ry }}>
            {s.resLabel}
          </div>
          <Words ws={s.res} len={s.res.length} marks={f.marks} prefix="r" y={ry + 22} />
        </>
      )}
    </Region>
  );
};
