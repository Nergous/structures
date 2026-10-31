// Per-structure presentation data for the memory views.
import type { FC } from "react";
import type { BitS } from "../structures/bitset";
import type { HeapS } from "../structures/binheap";
import type { OMS } from "../structures/omcore";
import type { PQS } from "../structures/pqueue";
import type { RingS } from "../structures/ring";
import type { SetS } from "../structures/set";
import type { StackS } from "../structures/stack";
import type { ViewProps } from "./kit";
import { BitsetView, HeapView, PQView, SetView, StackView, omView, ringView } from "./views";

export interface Meta<S> {
  /** Receiver variable name in the console. */
  v: string;
  /** Heading, e.g. Stack[T]. */
  type: string;
  /** Console line describing the starting state. */
  init: string;
  /** Capacity of the main backing array (0 when nil); a change means a new array. */
  capOf: (s: S) => number;
  /** Cell texts of the main backing array, for the "old array" ghost. */
  ghost: (s: S) => string[];
  View: FC<ViewProps<S>>;
  /** Methods shown as chips before "all methods". */
  quick: string[];
}

const pad = (xs: string[], n: number) => [...xs, ...Array(Math.max(n - xs.length, 0)).fill("0")];
const ring = (s: RingS) => s.data.map((c) => (c ? String(c.v) : "0"));
const om = (s: OMS) => pad(s.entries.map((e) => (e ? e.k : "hole")), s.cap).map((x, i) => (i < s.entries.length ? x : ""));

const stack: Meta<StackS> = {
  v: "s",
  type: "Stack[T]",
  init: "s := stack.NewWithCap[int](4); s.PushN(12, 7, 31)",
  capOf: (s) => s.cap,
  ghost: (s) => pad(s.data.map((c) => String(c.v)), s.cap),
  View: StackView,
  quick: ["Push", "Pop", "Peek", "PushN", "PopN"],
};

const queue: Meta<RingS> = {
  v: "q",
  type: "Queue[T]",
  init: "// q holds 14 3 27: cap 8, head 5 after earlier pops",
  capOf: (s) => s.data.length,
  ghost: ring,
  View: ringView("q", "*Queue[int]", "head", "back"),
  quick: ["Push", "Pop", "Peek", "PushN", "PopN"],
};

const deque: Meta<RingS> = {
  v: "d",
  type: "Deque[T]",
  init: "// d holds 8 21 5 40: cap 8, front at index 6",
  capOf: (s) => s.data.length,
  ghost: ring,
  View: ringView("d", "*Deque[int]", "front", "back"),
  quick: ["PushFront", "PushBack", "PopFront", "PopBack", "Front", "Back"],
};

const set: Meta<SetS> = {
  v: "s",
  type: "Set[T]",
  init: "s := set.Of(4, 8, 15, 16, 23); other := set.Of(8, 16, 42, 7)",
  capOf: () => 0,
  ghost: () => [],
  View: SetView,
  quick: ["Add", "Remove", "Contains", "Union", "Intersection", "Difference"],
};

const binheap: Meta<HeapS> = {
  v: "h",
  type: "Heap[T]",
  init: "h := binheap.NewWithCap[int](8); h.PushN(3, 9, 5, 14, 11, 8, 21)",
  capOf: (s) => s.cap,
  ghost: (s) => pad(s.data.map((c) => String(c.v)), s.cap),
  View: HeapView,
  quick: ["Push", "Pop", "Peek", "PushPop", "Replace", "Remove"],
};

const pqueue: Meta<PQS> = {
  v: "q",
  type: "Queue[K, P]",
  init: 'q := pqueue.From([]pqueue.Item{{"io", 4}, {"db", 9}, {"ui", 7}, {"gc", 15}, {"net", 12}, {"log", 20}})',
  capOf: (s) => s.cap,
  ghost: (s) => pad(s.data.map((e) => (e ? `${e.k}:${e.v}` : "hole")), s.cap),
  View: PQView,
  quick: ["Push", "Pop", "Peek", "Priority", "Remove"],
};

const orderedmap: Meta<OMS> = {
  v: "m",
  type: "Map[K, V]",
  init: 'm := orderedmap.NewWithCap[string, int](8) // ant:3 bee:14 cat:7 dog:21 eel:9',
  capOf: (s) => s.cap,
  ghost: om,
  View: omView("m", "*Map[string, int]", true),
  quick: ["Set", "Get", "Remove", "PopFirst", "MoveToBack"],
};

const orderedset: Meta<OMS> = {
  v: "s",
  type: "Set[T]",
  init: "s := orderedset.Of(5, 12, 3, 27, 9); other := orderedset.Of(27, 4, 12, 30)",
  capOf: (s) => s.cap,
  ghost: om,
  View: omView("s", "*orderedset.Set[int]", false),
  quick: ["Add", "Remove", "PopFirst", "MoveToBack", "Union", "Intersection"],
};

const bitset: Meta<BitS> = {
  v: "s",
  type: "bitset.Set",
  init: "s := bitset.Of(1, 3, 6, 10, 15, 64, 70, 130)",
  capOf: (s) => s.data.length,
  ghost: (s) => s.data.map((w, i) => (i < s.len ? "…" + w.toString(16).slice(-4) : "")),
  View: BitsetView,
  quick: ["Add", "Remove", "Contains", "Min", "Max", "Union"],
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const META: Record<string, Meta<any>> = { stack, queue, deque, set, binheap, pqueue, orderedmap, orderedset, bitset };
