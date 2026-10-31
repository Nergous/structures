/** Visual emphasis applied to a cell or slot in one frame. */
export type Tone =
  | "new" // just written
  | "hot" // the element the operation is working on
  | "cmp" // being compared
  | "del" // being removed
  | "ok" // match / kept / result
  | "dim" // skipped / irrelevant
  | "held"; // lifted out of the array (hole-shift sifts)

/** One stored value with a stable identity, so moves can be animated. */
export interface Cell {
  id: number;
  v: number;
}

/** An item in the "returned / yielded" tray under the stage. */
export interface OutItem {
  id: number;
  t: string;
  tone?: Tone;
}

/** A snapshot of the simulated structure plus its narration. */
export interface Frame<S> {
  s: S;
  note: string;
  /** Keys: "c<id>" for a cell, "s<i>" for a slot index, free-form otherwise. */
  marks?: Record<string, Tone>;
  out?: OutItem[];
  outLabel?: string;
  ret?: string;
  panic?: boolean;
}

export type ArgKind = "int" | "ints" | "key" | "choice";

export interface ArgDef<S> {
  name: string;
  kind: ArgKind;
  hint?: string;
  options?: string[];
  def?: string | ((s: S) => string);
}

export type Group =
  | "construct"
  | "read"
  | "iterate"
  | "copy"
  | "mutate"
  | "algebra"
  | "memory";

export interface MethodDef<S> {
  name: string;
  group: Group;
  sig: string;
  doc: string;
  cost: string;
  args?: ArgDef<S>[];
  run: (s: S, args: string[]) => Frame<S>[];
}

export interface ViewProps<S> {
  f: Frame<S>;
}

export interface StructureDef<S> {
  id: string;
  title: string;
  kind: string;
  tagline: string;
  about: string[];
  cost: [string, string][];
  seed: () => S;
  stats: (s: S) => [string, string | number][];
  methods: MethodDef<S>[];
}

export const GROUP_LABEL: Record<Group, string> = {
  construct: "Constructors",
  read: "Reads",
  iterate: "Iterators",
  copy: "Copies",
  mutate: "Mutators",
  algebra: "Set algebra",
  memory: "Memory",
};
