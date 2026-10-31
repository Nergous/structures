// Simulation of internal/ring.Buffer, shared by queue and deque, plus the
// circular view that draws it.
import { AnimatePresence, motion } from "motion/react";
import { ringCap } from "../core/gomem";
import { cell, list, okRet, parseInt1, rec, type Rec } from "../core/rec";
import type { Cell, Frame, MethodDef, Tone } from "../core/types";
import { ArrayView, Box, CapBar, Label, Pointer, Slot, Stage, Tray, toneOf, useEase } from "../ui/viz";

export interface RingS {
  data: (Cell | null)[]; // len(data) is the capacity
  head: number;
  len: number;
}

type R = Rec<RingS>;
type Marks = Record<string, Tone>;

export const ringOf = (vs: number[], cap: number, head = 0): RingS => {
  const data: (Cell | null)[] = Array(cap).fill(null);
  vs.forEach((v, i) => (data[(head + i) % cap] = cell(v)));
  return { data, head, len: vs.length };
};

export const idx = (s: RingS, off: number) => {
  const i = s.head + off;
  return i >= s.data.length ? i - s.data.length : i;
};

export const live = (s: RingS): Cell[] =>
  Array.from({ length: s.len }, (_, k) => s.data[idx(s, k)]!);

const allMarks = (s: RingS, t: Tone): Marks => {
  const m: Marks = {};
  live(s).forEach((c) => (m["c" + c.id] = t));
  return m;
};

/** growFor: reallocate so cap >= need, moving the front to index 0. */
export function growFor(r: R, need: number) {
  const s = r.s;
  if (need <= s.data.length) return;
  const old = s.data.length;
  const next = ringCap(old, need);
  const items = live(s);
  if (items.length && s.head + s.len > old) {
    r.snap(`the buffer is full and wraps around: growing will unwrap it`, allMarks(s, "hot"));
  }
  s.data = Array(next).fill(null);
  items.forEach((c, i) => (s.data[i] = c));
  s.head = 0;
  r.snap(
    old === 0
      ? `no array yet: allocate cap ${next}`
      : `full (len ${s.len} = cap ${old}): allocate cap ${next}, copy front→back to index 0, head = 0`,
    allMarks(s, "hot"),
  );
}

export function pushBack(r: R, v: number) {
  const s = r.s;
  if (s.len === s.data.length) growFor(r, s.len + 1);
  const i = idx(s, s.len);
  const c = cell(v);
  s.data[i] = c;
  s.len++;
  r.snap(
    `data[(head+len) % cap] = data[${i}] = ${v}; len = ${s.len}` +
      (i < s.head ? " — the write wrapped around to the start" : ""),
    { ["c" + c.id]: "new" },
  );
}

export function pushFront(r: R, v: number) {
  const s = r.s;
  if (s.len === s.data.length) growFor(r, s.len + 1);
  s.head = s.head - 1 < 0 ? s.data.length - 1 : s.head - 1;
  const c = cell(v);
  s.data[s.head] = c;
  s.len++;
  r.snap(`head-- (wrapping to ${s.head}); data[${s.head}] = ${v}`, { ["c" + c.id]: "new" });
}

export function pushBackN(r: R, vs: number[]) {
  const s = r.s;
  if (!vs.length) return;
  growFor(r, s.len + vs.length);
  const m: Marks = {};
  vs.forEach((v, k) => {
    const c = cell(v);
    s.data[idx(s, s.len + k)] = c;
    m["c" + c.id] = "new";
  });
  s.len += vs.length;
  r.snap(`grew at most once, then wrote ${vs.length} values after the back`, m);
}

export function pushFrontN(r: R, vs: number[]) {
  const s = r.s;
  if (!vs.length) return;
  growFor(r, s.len + vs.length);
  s.head -= vs.length;
  if (s.head < 0) s.head += s.data.length;
  const m: Marks = {};
  vs.forEach((v, k) => {
    const c = cell(v);
    s.data[idx(s, k)] = c;
    m["c" + c.id] = "new";
  });
  s.len += vs.length;
  r.snap(`head -= ${vs.length}; ${vs[0]} is the new front, argument order kept`, m);
}

export function popFront(r: R, label: string): Cell | null {
  const s = r.s;
  if (!s.len) return null;
  const c = s.data[s.head]!;
  r.snap(`take data[head] = data[${s.head}]`, { ["c" + c.id]: "del" });
  s.data[s.head] = null;
  s.head = s.head + 1 === s.data.length ? 0 : s.head + 1;
  s.len--;
  if (!s.len) s.head = 0;
  r.emit(c.v, "del", label);
  r.snap(`slot zeroed; head = ${s.head}${s.len ? "" : " (empty: head resets to 0)"}, len = ${s.len}`);
  return c;
}

export function popBack(r: R, label: string): Cell | null {
  const s = r.s;
  if (!s.len) return null;
  const i = idx(s, s.len - 1);
  const c = s.data[i]!;
  r.snap(`take data[(head+len-1) % cap] = data[${i}]`, { ["c" + c.id]: "del" });
  s.data[i] = null;
  s.len--;
  if (!s.len) s.head = 0;
  r.emit(c.v, "del", label);
  r.snap(`slot zeroed; len = ${s.len}`);
  return c;
}

export function iterate(r: R, backward: boolean) {
  const s = r.s;
  for (let k = 0; k < s.len; k++) {
    const off = backward ? s.len - 1 - k : k;
    const i = idx(s, off);
    const c = s.data[i]!;
    r.emit(c.v, undefined, "yielded");
    r.snap(`yield data[${i}] = ${c.v}`, { ["c" + c.id]: "hot" });
  }
}

/** Memory controls of the ring, identical for queue and deque. */
export function ringMemory(recv: string, unit = "elements"): MethodDef<RingS>[] {
  return [
    {
      name: "Clear",
      group: "memory",
      sig: `func (${recv}) Clear()`,
      doc: `Removes all ${unit} and releases the backing array: cap becomes 0.`,
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data = [];
        r.s.head = 0;
        r.s.len = 0;
        return r.done("", "data = nil, head = 0, len = 0");
      },
    },
    {
      name: "Reset",
      group: "memory",
      sig: `func (${recv}) Reset()`,
      doc: "Removes all elements but keeps the array. Live slots (both segments) are zeroed.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        r.snap("clear both live segments", allMarks(r.s, "del"));
        r.s.data = r.s.data.map(() => null);
        r.s.head = 0;
        r.s.len = 0;
        return r.done("", `head = 0, len = 0; cap stays ${r.s.data.length}`);
      },
    },
    {
      name: "Shrink",
      group: "memory",
      sig: `func (${recv}) Shrink()`,
      doc: "Copies the live elements into a new array of exactly Len slots starting at index 0. No-op when already right-sized at head 0.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const s = r.s;
        if (!s.len) {
          s.data = [];
          s.head = 0;
          return r.done("", "empty: drop the array");
        }
        if (s.data.length === s.len && s.head === 0) return r.done("", "already right-sized and starting at 0: no-op");
        const items = live(s);
        s.data = [...items];
        s.head = 0;
        return r.done("", `new array of cap ${s.len}; elements copied front→back`, allMarks(s, "new"));
      },
    },
    {
      name: "Clip",
      group: "memory",
      sig: `func (${recv}) Clip()`,
      doc: "Reduces cap to Len. Contiguous elements are resliced (no copy); wrapped elements must be copied into a compact array.",
      cost: "O(1) or O(n) when wrapped",
      run: (s0) => {
        const r = rec(s0);
        const s = r.s;
        if (!s.len) {
          s.data = [];
          s.head = 0;
          return r.done("", "empty: drop the array");
        }
        if (s.data.length === s.len && s.head === 0) return r.done("", "already clipped: no-op");
        const wraps = s.head + s.len > s.data.length;
        const items = live(s);
        s.data = [...items];
        s.head = 0;
        return r.done(
          "",
          wraps
            ? "elements wrap around: copied into a compact array"
            : `contiguous: data = data[head:end:end], no copy`,
          wraps ? allMarks(s, "new") : undefined,
        );
      },
    },
    {
      name: "Grow",
      group: "memory",
      sig: `func (${recv}) Grow(n int)`,
      doc: "Makes room for at least n more elements (doubling from the current cap). No-op for n <= 0 or if room exists.",
      cost: "O(n) when it reallocates",
      args: [{ name: "n", kind: "int", def: "4" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = parseInt1(a[0]);
        if (n <= 0) return r.done("", "n <= 0: no-op");
        if (r.s.len + n <= r.s.data.length) return r.done("", "capacity already suffices: no-op");
        growFor(r, r.s.len + n);
        return r.done("", `room for ${r.s.data.length - r.s.len} more without reallocating`);
      },
    },
  ];
}

export const ringRet = (c: Cell | null) => okRet(c?.v, !!c);
export const ringList = (cs: Cell[]) => (cs.length ? list(cs.map((c) => c.v)) : "nil");

/** Circle + linear backing array. */
export function RingView({ f, front = "head", back = "back" }: { f: Frame<RingS>; front?: string; back?: string }) {
  const s = f.s;
  const cap = s.data.length;
  const t = useEase();
  const cw = cap > 16 ? 34 : 42;
  const R0 = Math.max(120, (cap * (cw + 22)) / (2 * Math.PI));
  const cx = R0 + 90;
  const cy = R0 + 64;
  const W = Math.max(cx * 2 + 220, Math.min(cap, 16) * (cw + 6) + 80, 600);
  const pos = (i: number, rad = R0) => {
    const a = -Math.PI / 2 + (i / Math.max(cap, 1)) * Math.PI * 2;
    return { x: cx + Math.cos(a) * rad - cw / 2, y: cy + Math.sin(a) * rad - cw / 2, a };
  };
  const tail = s.len ? idx(s, s.len - 1) : -1;
  const lineY = cy * 2 + 40;
  const cols = 16;
  const rows = Math.max(1, Math.ceil(cap / cols));
  const H = lineY + rows * (cw + 22) + 40;

  // live window arc
  let arc: string | null = null;
  if (s.len && cap) {
    const a0 = -Math.PI / 2 + ((s.head - 0.45) / cap) * Math.PI * 2;
    const a1 = -Math.PI / 2 + ((s.head + s.len - 0.55) / cap) * Math.PI * 2;
    const rr = R0 + cw / 2 + 10;
    const p0 = { x: cx + Math.cos(a0) * rr, y: cy + Math.sin(a0) * rr };
    const p1 = { x: cx + Math.cos(a1) * rr, y: cy + Math.sin(a1) * rr };
    const large = a1 - a0 > Math.PI ? 1 : 0;
    arc = s.len === cap
      ? `M ${cx} ${cy - rr} A ${rr} ${rr} 0 1 1 ${cx - 0.01} ${cy - rr}`
      : `M ${p0.x} ${p0.y} A ${rr} ${rr} 0 ${large} 1 ${p1.x} ${p1.y}`;
  }

  return (
    <div className="view">
      <Stage w={W} h={H}>
        <svg className="ring-svg" width={W} height={cy * 2 + 20}>
          <circle cx={cx} cy={cy} r={R0} className="ring-guide" />
          {arc && (
            <motion.path
              d={arc}
              className="ring-arc"
              initial={false}
              animate={{ d: arc }}
              transition={t}
            />
          )}
        </svg>
        <AnimatePresence>
          {s.data.map((_, i) => {
            const p = pos(i);
            const inside = (i - s.head + cap) % cap < s.len;
            return (
              <Slot key={"rs" + i} x={p.x} y={p.y} w={cw} h={cw} kind={inside ? "used" : "spare"} tone={f.marks?.["s" + i]} />
            );
          })}
        </AnimatePresence>
        {s.data.map((_, i) => {
          const p = pos(i, R0 - cw / 2 - 14);
          return (
            <Label key={"ri" + i} x={p.x + cw / 2 - 6} y={p.y + cw / 2 - 7} className="axis">
              {i}
            </Label>
          );
        })}
        <AnimatePresence>
          {s.data.map((c, i) => {
            if (!c) return null;
            const p = pos(i);
            return (
              <Box key={"r" + c.id} x={p.x + 3} y={p.y + 3} w={cw - 6} h={cw - 6} tone={toneOf(f.marks, "c" + c.id)}>
                {c.v}
              </Box>
            );
          })}
        </AnimatePresence>
        {cap > 0 && (
          <Pointer
            id="head"
            x={pos(s.head, R0 + cw / 2 + 26).x + cw / 2 - 18}
            y={pos(s.head, R0 + cw / 2 + 26).y + cw / 2 - 10}
            label={front}
            dir="down"
          />
        )}
        {s.len > 1 && (
          <Pointer
            id="tail"
            x={pos(tail, R0 + cw / 2 + 26).x + cw / 2 - 18}
            y={pos(tail, R0 + cw / 2 + 26).y + cw / 2 - 10}
            label={back}
            dir="down"
            tone="cyan"
          />
        )}
        <Label x={cx - 52} y={cy - 22} className="ring-center">
          <span>
            head <b>{s.head}</b>
          </span>
          <span>
            len <b>{s.len}</b>
          </span>
          <span>
            cap <b>{cap}</b>
          </span>
        </Label>

        <Label x={30} y={lineY - 24} className="title">
          backing array data[]
        </Label>
        {cap === 0 ? (
          <Label x={30} y={lineY + 4} className="note-spare">
            data == nil — nothing allocated
          </Label>
        ) : (
          <ArrayView
            x={30}
            y={lineY}
            cells={s.data}
            cap={cap}
            cols={cols}
            cw={cw}
            ns="lin"
            marks={f.marks}
            live={(i) => (i - s.head + cap) % cap < s.len}
          />
        )}
      </Stage>
      <CapBar len={s.len} cap={cap} extra={<> · head <b>{s.head}</b></>} />
      <Tray f={f} />
    </div>
  );
}
