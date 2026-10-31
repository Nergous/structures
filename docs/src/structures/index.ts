import type { StructureDef } from "../core/types";
import { binheap } from "./binheap";
import { bitset } from "./bitset";
import { deque } from "./deque";
import { orderedmap } from "./orderedmap";
import { orderedset } from "./orderedset";
import { pqueue } from "./pqueue";
import { queue } from "./queue";
import { set } from "./set";
import { stack } from "./stack";

// Same order as the repository README.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const STRUCTURES: StructureDef<any>[] = [
  stack,
  queue,
  deque,
  set,
  binheap,
  pqueue,
  orderedmap,
  orderedset,
  bitset,
];
