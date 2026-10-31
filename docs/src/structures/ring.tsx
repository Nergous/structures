// Simulation of internal/ring.Buffer, shared by queue and deque, plus the
// circular view that draws it.
import { ringCap } from "../core/gomem";
import { cell, list, okRet, parseInt1, rec, type Rec } from "../core/rec";
import type { Cell, MethodDef, Tone } from "../core/types";

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
