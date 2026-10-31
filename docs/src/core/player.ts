import { useCallback, useEffect, useRef, useState } from "react";
import { GoPanic } from "./rec";
import type { Frame, MethodDef, StructureDef } from "./types";

const BASE_MS = 950;

export interface Call {
  method: string;
  args: string[];
}

/**
 * Owns the committed state of the selected structure and plays the frames of
 * the last call. The committed state jumps to the end immediately, so a new
 * call always starts from the post-call state even mid-animation.
 */
export function usePlayer<S>(def: StructureDef<S>) {
  const [state, setState] = useState<S>(def.seed);
  const [frames, setFrames] = useState<Frame<S>[]>(() => [
    { s: state, note: "Pick a method on the right to see it run." },
  ]);
  const [idx, setIdx] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [speed, setSpeed] = useState(1);
  const [call, setCall] = useState<Call | null>(null);
  const history = useRef<S[]>([]);

  useEffect(() => {
    if (!playing) return;
    if (idx >= frames.length - 1) {
      setPlaying(false);
      return;
    }
    const t = setTimeout(() => setIdx((i) => i + 1), BASE_MS / speed);
    return () => clearTimeout(t);
  }, [playing, idx, frames.length, speed]);

  const runFrom = useCallback((from: S, m: MethodDef<S>, args: string[]) => {
    let fr: Frame<S>[];
    try {
      fr = m.run(from, args);
    } catch (e) {
      const msg = e instanceof GoPanic ? e.message : String(e);
      fr = [{ s: from, note: `panic: ${msg}`, panic: true }];
    }
    if (!fr.length) return;
    setCall({ method: m.name, args });
    const last = fr[fr.length - 1];
    if (!last.panic) {
      history.current.push(from);
      if (history.current.length > 50) history.current.shift();
      setState(last.s);
    } else {
      setState(from);
    }
    setFrames(fr);
    setIdx(0);
    setPlaying(fr.length > 1);
  }, []);

  const run = useCallback(
    (m: MethodDef<S>, args: string[]) => runFrom(state, m, args),
    [state, runFrom],
  );

  const undo = useCallback(() => {
    const prev = history.current.pop();
    if (prev === undefined) return;
    setState(prev);
    setFrames([{ s: prev, note: "Undid the last call." }]);
    setIdx(0);
    setPlaying(false);
    setCall(null);
  }, []);

  const step = useCallback(
    (d: number) => {
      setPlaying(false);
      setIdx((i) => Math.min(Math.max(i + d, 0), frames.length - 1));
    },
    [frames.length],
  );

  const toggle = useCallback(() => {
    if (idx >= frames.length - 1) {
      setIdx(0);
      setPlaying(frames.length > 1);
      return;
    }
    setPlaying((p) => !p);
  }, [idx, frames.length]);

  const cur = Math.min(idx, frames.length - 1);
  return {
    state,
    frame: frames[cur],
    frames,
    idx: cur,
    playing,
    speed,
    setSpeed,
    call,
    run,
    runFrom,
    undo,
    canUndo: history.current.length > 0,
    /** States before each recent call, oldest first. */
    history: history.current,
    step,
    toggle,
    jump: (i: number) => {
      setPlaying(false);
      setIdx(Math.max(0, i));
    },
  };
}
