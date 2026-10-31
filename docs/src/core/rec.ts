import type { Cell, Frame, OutItem, Tone } from "./types";

let nextId = 1;

/** Returns a fresh identity for a cell or tray item. */
export const nid = (): number => nextId++;

export const cell = (v: number): Cell => ({ id: nid(), v });

/** Thrown by simulations where the Go code would panic. */
export class GoPanic extends Error {}

export const panic = (msg: string): never => {
  throw new GoPanic(msg);
};

type Marks = Record<string, Tone>;

/**
 * Records the frames of one method call. The simulation mutates `s` freely
 * and calls `snap` whenever something visible happened; each snap stores a
 * deep copy, so frames never alias each other.
 */
export class Rec<S> {
  readonly s: S;
  readonly frames: Frame<S>[] = [];
  out: OutItem[] = [];
  outLabel?: string;

  constructor(s: S) {
    this.s = structuredClone(s);
  }

  snap(note: string, marks?: Marks): void {
    this.frames.push({
      s: structuredClone(this.s),
      note,
      marks,
      out: this.out.length ? [...this.out] : undefined,
      outLabel: this.outLabel,
    });
  }

  emit(t: string | number, tone?: Tone, label?: string): void {
    if (label) this.outLabel = label;
    this.out.push({ id: nid(), t: String(t), tone });
  }

  /** Final frame: the call returns `ret`. */
  done(ret: string, note?: string, marks?: Marks): Frame<S>[] {
    this.snap(note ?? (ret ? `returns ${ret}` : "done"), marks);
    this.frames[this.frames.length - 1].ret = ret;
    return this.frames;
  }
}

export const rec = <S>(s: S): Rec<S> => new Rec(s);

/** Formats a Go-style comma-ok result. */
export const okRet = (v: string | number | undefined, ok: boolean): string =>
  ok ? `(${v}, true)` : "(0, false)";

export const list = (vs: (string | number)[]): string =>
  vs.length ? `[${vs.join(" ")}]` : "[]";

export function parseInts(raw: string): number[] {
  return raw
    .split(/[\s,]+/)
    .filter(Boolean)
    .map((t) => {
      const n = Number(t);
      if (!Number.isInteger(n)) throw new GoPanic(`not an int: ${t}`);
      return n;
    });
}

export function parseInt1(raw: string): number {
  const vs = parseInts(raw);
  if (vs.length !== 1) throw new GoPanic("expected one int");
  return vs[0];
}

export const rand = (lo: number, hi: number): number =>
  lo + Math.floor(Math.random() * (hi - lo + 1));

/** A random int in [lo, hi] that is not in `taken`, when possible. */
export function freshInt(taken: Iterable<number>, lo = 1, hi = 99): number {
  const t = new Set(taken);
  for (let i = 0; i < 50; i++) {
    const v = rand(lo, hi);
    if (!t.has(v)) return v;
  }
  return rand(lo, hi);
}

export const randInts = (n: number, lo = 1, hi = 99): number[] =>
  Array.from({ length: n }, () => rand(lo, hi));

export function distinctInts(n: number, lo = 1, hi = 99): number[] {
  const out: number[] = [];
  while (out.length < n) {
    const v = freshInt(out, lo, hi);
    if (!out.includes(v)) out.push(v);
  }
  return out;
}

/** Predicates offered to RemoveFunc. */
export const PREDICATES: Record<string, (v: number) => boolean> = {
  "v%2 == 0": (v) => v % 2 === 0,
  "v%2 == 1": (v) => v % 2 === 1,
  "v < 30": (v) => v < 30,
  "v >= 50": (v) => v >= 50,
};
