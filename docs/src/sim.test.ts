// Invariant and behaviour tests for the simulations: they must stay faithful
// to the Go packages they visualize.
import { describe, expect, it } from "vitest";
import { GoPanic } from "./core/rec";
import type { MethodDef, StructureDef } from "./core/types";
import { STRUCTURES } from "./structures";
import type { StackS } from "./structures/stack";
import type { RingS } from "./structures/ring";
import type { HeapS } from "./structures/binheap";
import type { PQS } from "./structures/pqueue";
import type { OMS } from "./structures/omcore";
import type { BitS } from "./structures/bitset";
import type { SetS } from "./structures/set";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const byId = (id: string): StructureDef<any> => STRUCTURES.find((s) => s.id === id)!;

function call<S>(def: StructureDef<S>, s: S, name: string, ...args: string[]) {
  const m = def.methods.find((x) => x.name === name);
  if (!m) throw new Error(`${def.id}.${name} missing`);
  const fr = m.run(s, args);
  return { s: fr[fr.length - 1].s, ret: fr[fr.length - 1].ret, frames: fr };
}

function defaults<S>(m: MethodDef<S>, s: S): string[] {
  return (m.args ?? []).map((a) => {
    if (a.kind === "choice") return a.options![Math.floor(Math.random() * a.options!.length)];
    return typeof a.def === "function" ? a.def(s) : (a.def ?? "");
  });
}

const CMP = {
  min: (a: number, b: number) => a - b,
  max: (a: number, b: number) => b - a,
  func: (a: number, b: number) => Math.abs(a - 50) - Math.abs(b - 50),
};

const pop = (w: bigint) => {
  let c = 0;
  for (let x = w; x; x &= x - 1n) c++;
  return c;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const INVARIANTS: Record<string, (s: any) => void> = {
  stack: (s: StackS) => expect(s.data.length).toBeLessThanOrEqual(s.cap),
  queue: (s: RingS) => ringInv(s),
  deque: (s: RingS) => ringInv(s),
  set: (s: SetS) => expect(new Set(s.a).size).toBe(s.a.length),
  binheap: (s: HeapS) => {
    expect(s.data.length).toBeLessThanOrEqual(s.cap);
    for (let i = 1; i < s.data.length; i++)
      expect(CMP[s.mode](s.data[(i - 1) >> 1].v, s.data[i].v)).toBeLessThanOrEqual(0);
  },
  pqueue: (s: PQS) => {
    expect(s.held).toBeNull();
    expect(s.data.length).toBeLessThanOrEqual(s.cap);
    expect(Object.keys(s.index).length).toBe(s.data.length);
    s.data.forEach((e, i) => {
      expect(e).not.toBeNull();
      expect(s.index[e!.k]).toBe(i);
      if (i) {
        const p = s.data[(i - 1) >> 1]!;
        const c = CMP[s.mode](p.v, e!.v) || p.seq - e!.seq;
        expect(c).toBeLessThan(0);
      }
    });
  },
  orderedmap: (s: OMS) => omInv(s),
  orderedset: (s: OMS) => omInv(s),
  bitset: (s: BitS) => {
    expect(s.len).toBeLessThanOrEqual(s.data.length);
    if (s.len) expect(s.data[s.len - 1]).not.toBe(0n);
    expect(s.data.slice(0, s.len).reduce((a, w) => a + pop(w), 0)).toBe(s.n);
  },
};

function ringInv(s: RingS) {
  const cap = s.data.length;
  expect(s.len).toBeLessThanOrEqual(cap);
  if (!s.len) expect(s.head).toBe(0);
  for (let i = 0; i < cap; i++) {
    const inside = (i - s.head + cap) % cap < s.len;
    expect(s.data[i] != null).toBe(inside);
  }
}

function omInv(s: OMS) {
  expect(s.entries.length).toBeLessThanOrEqual(s.cap);
  expect(s.entries.filter((e) => !e).length).toBe(s.holes);
  if (s.entries.length) {
    expect(s.entries[s.entries.length - 1]).not.toBeNull();
    expect(s.entries[s.head]).not.toBeNull();
    for (let i = 0; i < s.head; i++) expect(s.entries[i]).toBeNull();
  } else {
    expect(s.head).toBe(0);
    expect(s.holes).toBe(0);
  }
  const keys = s.entries.filter(Boolean).map((e) => e!.k);
  expect(new Set(keys).size).toBe(keys.length);
}

describe("random operation sequences keep invariants", () => {
  for (const def of STRUCTURES) {
    it(def.id, () => {
      for (let round = 0; round < 6; round++) {
        let s = def.seed();
        for (let step = 0; step < 150; step++) {
          const m = def.methods[Math.floor(Math.random() * def.methods.length)];
          const args = defaults(m, s);
          let fr;
          try {
            fr = m.run(s, args);
          } catch (e) {
            if (e instanceof GoPanic) continue;
            throw new Error(`${def.id}.${m.name}(${args.join(", ")}): ${e}`);
          }
          expect(fr.length, `${m.name} produced no frames`).toBeGreaterThan(0);
          s = fr[fr.length - 1].s;
          try {
            INVARIANTS[def.id](s);
          } catch (e) {
            throw new Error(`after ${def.id}.${m.name}(${args.join(", ")}): ${e}`);
          }
        }
      }
    });
  }
});

describe("behaviour matches the Go packages", () => {
  it("stack: append growth 1, 2, 4, 8 and LIFO pops", () => {
    const d = byId("stack");
    let s = call(d, d.seed(), "New").s;
    const caps: number[] = [];
    for (let i = 1; i <= 5; i++) {
      s = call(d, s, "Push", String(i)).s;
      caps.push(s.cap);
    }
    expect(caps).toEqual([1, 2, 4, 4, 8]);
    expect(call(d, s, "PopN", "3").ret).toBe("[5 4 3]");
    expect(call(d, s, "Slice").ret).toBe("[5 4 3 2 1]");
  });

  it("queue: wraps, then unwraps to head 0 on growth", () => {
    const d = byId("queue");
    let s: RingS = call(d, d.seed(), "NewWithCap", "4").s;
    s = call(d, s, "PushN", "1, 2, 3").s;
    s = call(d, s, "PopN", "2").s;
    s = call(d, s, "PushN", "4, 5, 6").s; // wraps around
    expect(s.head).toBe(2);
    expect(call(d, s, "Slice").ret).toBe("[3 4 5 6]");
    s = call(d, s, "Push", "7").s; // full: grows to 8 and unwraps
    expect(s.data.length).toBe(8);
    expect(s.head).toBe(0);
    expect(call(d, s, "PopN", "9").ret).toBe("[3 4 5 6 7]");
  });

  it("deque: PushFrontN keeps argument order", () => {
    const d = byId("deque");
    let s = call(d, d.seed(), "New").s;
    s = call(d, s, "PushBackN", "4, 5").s;
    s = call(d, s, "PushFrontN", "1, 2, 3").s;
    expect(call(d, s, "Slice").ret).toBe("[1 2 3 4 5]");
    expect(call(d, s, "PopBackN", "2").ret).toBe("[5 4]");
  });

  it("binheap: pops come out sorted for every order", () => {
    const d = byId("binheap");
    const input = "31, 4, 15, 9, 26, 5, 35, 8, 97, 93, 23, 84, 62, 64";
    for (const [ctor, sorted] of [
      ["From", [4, 5, 8, 9, 15, 23, 26, 31, 35, 62, 64, 84, 93, 97]],
      ["FromMax", [97, 93, 84, 64, 62, 35, 31, 26, 23, 15, 9, 8, 5, 4]],
    ] as const) {
      const s = call(d, d.seed(), ctor, input).s;
      expect(call(d, s, "PopN", "20").ret).toBe(`[${sorted.join(" ")}]`);
    }
    let s = call(d, d.seed(), "From", "5, 9, 7").s;
    expect(call(d, s, "PushPop", "1").ret).toBe("1");
    const r = call(d, s, "PushPop", "8");
    expect(r.ret).toBe("5");
    s = r.s;
    expect(call(d, s, "Replace", "1").ret).toBe("(7, true)");
  });

  it("pqueue: stable ties, priority update, Remove by key", () => {
    const d = byId("pqueue");
    let s: PQS = call(d, d.seed(), "From", "a:5, b:3, c:5, d:1, e:3").s;
    expect(call(d, s, "Sorted").ret).toBe("[{d 1} {b 3} {e 3} {a 5} {c 5}]");
    s = call(d, s, "Push", "c", "0").s;
    expect(call(d, s, "Push", "c", "0").ret).toBe("false");
    s = call(d, s, "Remove", "b").s;
    expect(call(d, s, "PopN", "9").ret).toBe("[{c 0} {d 1} {e 3} {a 5}]");
  });

  it("orderedmap: holes, head, trim and compaction at 16", () => {
    const d = byId("orderedmap");
    let s: OMS = call(d, d.seed(), "New").s;
    const keys = Array.from({ length: 16 }, (_, i) => "k" + String(i).padStart(2, "0"));
    for (const k of keys) s = call(d, s, "Set", k, "1").s;
    s = call(d, s, "Remove", "k00").s;
    expect(s.head).toBe(1);
    expect(s.holes).toBe(1);
    s = call(d, s, "Remove", "k15").s; // trailing hole is trimmed
    expect(s.entries.length).toBe(15);
    expect(s.holes).toBe(1);
    // 15 entries < 16: no compaction yet even with many holes
    for (let i = 1; i <= 8; i++) s = call(d, s, "Remove", keys[i * 1 + 1]).s;
    expect(s.entries.length).toBe(15);
    s = call(d, s, "Set", "k99", "1").s; // 16 entries, insertions never compact
    expect(s.holes).toBe(9);
    s = call(d, s, "Remove", "k12").s; // 10 holes > 16/2: compact
    expect(s.holes).toBe(0);
    expect(s.head).toBe(0);
    expect(call(d, s, "Keys").frames.at(-1)!.out!.map((o) => o.t)).toEqual(["k01", "k10", "k11", "k13", "k14", "k99"]);
  });

  it("orderedset: algebra results are ordered", () => {
    const d = byId("orderedset");
    const s = call(d, d.seed(), "Of", "5, 12, 3, 27, 9").s;
    expect(call(d, s, "Union", "27, 4, 12, 30").ret).toBe("{5 12 3 27 9 4 30}");
    expect(call(d, s, "Intersection", "27, 4, 12, 30").ret).toBe("{12 27}");
    expect(call(d, s, "SymmetricDifference", "27, 4, 12, 30").ret).toBe("{5 3 9 4 30}");
  });

  it("bitset: Reset leaves stale words that grow clears", () => {
    const d = byId("bitset");
    let s: BitS = call(d, d.seed(), "Of", "1, 70, 130").s;
    expect(s.len).toBe(3);
    s = call(d, s, "Reset").s;
    expect(s.len).toBe(0);
    expect(s.data[1]).not.toBe(0n); // stale
    s = call(d, s, "Add", "64").s;
    expect(s.len).toBe(2);
    expect(s.data[1]).toBe(1n); // bit 70 was cleared by grow
    expect(call(d, s, "Slice").ret).toBe("[64]");
    s = call(d, s, "Remove", "64").s;
    expect(s.len).toBe(0);
    expect(() => call(d, s, "Add", "-1")).toThrow(GoPanic);
  });
});
