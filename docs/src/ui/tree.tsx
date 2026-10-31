import { AnimatePresence, motion } from "motion/react";
import type { ReactNode } from "react";
import type { Tone } from "../core/types";
import { Box, Label, useEase } from "./viz";

export interface TreeNode {
  key: number;
  label: ReactNode;
  tone?: Tone;
  title?: string;
}

/** Position of heap index i in a complete binary tree. */
export function treePos(i: number, o: { x: number; y: number; w: number; lh: number; d: number }) {
  const lvl = Math.floor(Math.log2(i + 1));
  const first = 2 ** lvl - 1;
  const slots = 2 ** lvl;
  return {
    x: o.x + ((i - first + 0.5) * o.w) / slots - o.d / 2,
    y: o.y + lvl * o.lh,
  };
}

export const treeHeight = (n: number, lh: number, d: number) =>
  n <= 0 ? d : Math.floor(Math.log2(n)) * lh + d;

/**
 * Binary heap drawn as a tree. nodes[i] is the element at heap index i; a
 * null entry is a hole (hole-shift sifts). `held` floats above index `at`.
 */
export function TreeView(p: {
  x: number;
  y: number;
  w: number;
  nodes: (TreeNode | null)[];
  held?: { node: TreeNode; at: number } | null;
  d?: number;
  lh?: number;
  wide?: number;
  slotTone?: Record<number, Tone | undefined>;
}) {
  const t = useEase();
  const d = p.d ?? 42;
  const lh = p.lh ?? 70;
  const bw = p.wide ?? d;
  const geo = { x: p.x, y: p.y, w: p.w, lh, d: bw };
  const n = p.nodes.length;
  const edges = [];
  for (let i = 1; i < n; i++) {
    const a = treePos(Math.floor((i - 1) / 2), geo);
    const b = treePos(i, geo);
    edges.push(
      <motion.line
        key={"e" + i}
        x1={a.x + bw / 2}
        y1={a.y + d / 2}
        x2={b.x + bw / 2}
        y2={b.y + d / 2}
        className={"edge" + (p.slotTone?.[i] ? " on" : "")}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={t}
      />,
    );
  }
  const H = treeHeight(n, lh, d) + 40;
  return (
    <>
      <svg className="tree-svg" width={p.x * 2 + p.w} height={p.y + H} style={{ left: 0, top: 0 }}>
        <AnimatePresence>{edges}</AnimatePresence>
      </svg>
      {p.nodes.map((_, i) => {
        const pos = treePos(i, geo);
        return (
          <Label key={"ti" + i} x={pos.x + bw / 2 - 12} y={pos.y + d + 1} className="tree-i">
            {i}
          </Label>
        );
      })}
      {p.nodes.map((nd, i) => {
        if (nd) return null;
        const pos = treePos(i, geo);
        return <div key={"th" + i} className="tree-hole" style={{ transform: `translate(${pos.x}px, ${pos.y}px)`, width: bw, height: d }} />;
      })}
      <AnimatePresence>
        {p.nodes.map((nd, i) => {
          if (!nd) return null;
          const pos = treePos(i, geo);
          return (
            <Box key={"t" + nd.key} x={pos.x} y={pos.y} w={bw} h={d} round={bw === d} tone={nd.tone} title={nd.title} small={bw !== d}>
              {nd.label}
            </Box>
          );
        })}
        {p.held &&
          (() => {
            const pos = treePos(p.held.at, geo);
            return (
              <Box key={"t" + p.held.node.key} x={pos.x + 14} y={pos.y - 24} w={bw} h={d} round={bw === d} tone="held" small={bw !== d}>
                {p.held.node.label}
              </Box>
            );
          })()}
      </AnimatePresence>
    </>
  );
}
