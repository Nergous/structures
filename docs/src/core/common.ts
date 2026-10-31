import { appendCap } from "./gomem";
import { parseInt1, rec } from "./rec";
import type { MethodDef } from "./types";

/** Len, Cap (optional), and IsEmpty for any structure. */
export function sizeReads<S>(o: {
  recv: string;
  len: (s: S) => number;
  cap?: (s: S) => number;
  capDoc?: string;
  unit?: string;
}): MethodDef<S>[] {
  const unit = o.unit ?? "elements";
  const out: MethodDef<S>[] = [
    {
      name: "Len",
      group: "read",
      sig: `func (${o.recv}) Len() int`,
      doc: `Returns the number of ${unit}. A nil receiver reports 0.`,
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        return r.done(String(o.len(r.s)), `Len reads a stored counter: ${o.len(r.s)}`);
      },
    },
  ];
  if (o.cap) {
    const cap = o.cap;
    out.push({
      name: "Cap",
      group: "read",
      sig: `func (${o.recv}) Cap() int`,
      doc:
        o.capDoc ??
        "Returns how many elements fit before the backing array must grow. Spare slots are drawn hatched.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        return r.done(String(cap(r.s)), `Cap is the backing array length: ${cap(r.s)}`);
      },
    });
  }
  out.push({
    name: "IsEmpty",
    group: "read",
    sig: `func (${o.recv}) IsEmpty() bool`,
    doc: `Reports whether there are no ${unit}. A nil receiver reports true.`,
    cost: "O(1)",
    run: (s0) => {
      const r = rec(s0);
      const e = o.len(r.s) === 0;
      return r.done(String(e), e ? "Len is 0: empty" : `Len is ${o.len(r.s)}: not empty`);
    },
  });
  return out;
}

/**
 * Clear / Reset / Shrink / Clip / Grow for structures that keep a plain Go
 * slice: `data` holds the live elements and `cap` its capacity.
 */
export function sliceMemory<S extends { data: unknown[]; cap: number }>(o: {
  recv: string;
  elem?: number;
  unit?: string;
  extraClear?: (s: S) => void;
}): MethodDef<S>[] {
  const elem = o.elem ?? 8;
  const unit = o.unit ?? "elements";
  return [
    {
      name: "Clear",
      group: "memory",
      sig: `func (${o.recv}) Clear()`,
      doc: `Removes all ${unit} and releases the backing array (cap becomes 0), so the memory can be garbage collected.`,
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        r.s.data = [];
        r.s.cap = 0;
        o.extraClear?.(r.s);
        return r.done("", "data = nil: the old array is dropped, Cap is 0");
      },
    },
    {
      name: "Reset",
      group: "memory",
      sig: `func (${o.recv}) Reset()`,
      doc: `Removes all ${unit} but keeps the backing array for reuse. Slots are zeroed so they retain no references.`,
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        const marks: Record<string, "del"> = {};
        r.s.data.forEach((_, i) => (marks["s" + i] = "del"));
        r.snap("clear(data): every live slot is zeroed", marks);
        r.s.data = [];
        o.extraClear?.(r.s);
        return r.done("", `data = data[:0]: len 0, cap stays ${r.s.cap}`);
      },
    },
    {
      name: "Shrink",
      group: "memory",
      sig: `func (${o.recv}) Shrink()`,
      doc: "Copies the elements into a new array of exactly Len slots, so the old, larger array can be collected now. No-op when cap == len.",
      cost: "O(n)",
      run: (s0) => {
        const r = rec(s0);
        if (r.s.cap === r.s.data.length) return r.done("", "cap already equals len: no-op");
        const marks: Record<string, "new"> = {};
        r.s.data.forEach((_, i) => (marks["s" + i] = "new"));
        r.s.cap = r.s.data.length;
        return r.done("", `allocated a right-sized array (cap ${r.s.cap}) and copied ${r.s.data.length} ${unit}`, marks);
      },
    },
    {
      name: "Clip",
      group: "memory",
      sig: `func (${o.recv}) Clip()`,
      doc: "Reslices to cap == len without copying (slices.Clip). Cheap, but the old array stays alive until a later growth reallocates.",
      cost: "O(1)",
      run: (s0) => {
        const r = rec(s0);
        if (r.s.cap === r.s.data.length) return r.done("", "cap already equals len: no-op");
        r.s.cap = r.s.data.length;
        return r.done("", `data = data[:len:len]: reported cap is ${r.s.cap}, no copy made`);
      },
    },
    {
      name: "Grow",
      group: "memory",
      sig: `func (${o.recv}) Grow(n int)`,
      doc: `Ensures room for at least n more ${unit} without reallocating, growing the array the way append would. The counterpart of Shrink.`,
      cost: "O(n) when it reallocates",
      args: [{ name: "n", kind: "int", def: "4" }],
      run: (s0, a) => {
        const r = rec(s0);
        const n = parseInt1(a[0]);
        const len = r.s.data.length;
        if (n <= 0) return r.done("", "n <= 0: no-op");
        if (r.s.cap - len >= n)
          return r.done("", `already ${r.s.cap - len} spare slots >= ${n}: no-op`);
        const next = appendCap(r.s.cap, len + n, elem);
        r.s.cap = next;
        return r.done("", `slices.Grow: new array with cap ${next} (room for ${next - len} more)`);
      },
    },
  ];
}
