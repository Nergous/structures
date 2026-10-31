import { useEffect, useMemo, useRef, useState } from "react";
import { usePlayer } from "./core/player";
import { GoPanic } from "./core/rec";
import { GROUP_LABEL, type Frame, type Group, type MethodDef, type StructureDef } from "./core/types";
import { Mem, hex } from "./mem/kit";
import { META, type Meta } from "./mem/meta";
import { STRUCTURES } from "./structures";

const REPO = "https://github.com/Nergous/structures";
const params = new URLSearchParams(location.search);
/** ?still makes motion instant (static screenshots). */
const STILL = params.has("still");
const BASE = 0xc000010000;
const addrOf = (gen: number) => BASE + gen * 0x400;
const GROUPS: Group[] = ["construct", "read", "iterate", "copy", "mutate", "algebra", "memory"];

function useHashId(): [string, (id: string) => void] {
  const read = () => location.hash.replace(/^#\/?/, "") || STRUCTURES[0].id;
  const [id, setId] = useState(read);
  useEffect(() => {
    const on = () => setId(read());
    addEventListener("hashchange", on);
    return () => removeEventListener("hashchange", on);
  }, []);
  return [id, (v) => (location.hash = "/" + v)];
}

export default function App() {
  const [id, setId] = useHashId();
  const def = STRUCTURES.find((d) => d.id === id) ?? STRUCTURES[0];
  const [addrs, setAddrs] = useState(true);
  return (
    <Mem.Provider value={{ addrs, still: STILL }}>
      <div className={"app" + (STILL ? " still" : "")}>
        <header className="top">
          <a className="brand" href={REPO} target="_blank" rel="noreferrer">
            <span className="mark" />
            structures
          </a>
          <nav className="tabs">
            {STRUCTURES.map((d) => (
              <button key={d.id} className={d.id === def.id ? "on" : ""} onClick={() => setId(d.id)}>
                {d.id}
              </button>
            ))}
          </nav>
          <a className="gh" href={REPO} target="_blank" rel="noreferrer">
            GitHub ↗
          </a>
        </header>
        <Workbench key={def.id} def={def} meta={META[def.id]} addrs={addrs} setAddrs={setAddrs} />
      </div>
    </Mem.Provider>
  );
}

/* ---------- console parsing ---------- */

const norm = (x: string) => x.replace(/\s+/g, "");

function defaults<S>(m: MethodDef<S>, s: S): string[] {
  return (m.args ?? []).map((a) => (typeof a.def === "function" ? a.def(s) : (a.def ?? a.options?.[0] ?? "")));
}

function parseCall<S>(def: StructureDef<S>, src: string, s: S): { m: MethodDef<S>; args: string[] } | string {
  const t = src.trim().replace(/^\w+\./, "");
  const mt = t.match(/^(\w+)\s*(?:\(([\s\S]*)\))?\s*$/) ?? t.match(/^(\w+)\s+([\s\S]*)$/);
  if (!mt) return "could not parse that — try Name(args)";
  const m = def.methods.find((x) => x.name.toLowerCase() === mt[1].toLowerCase());
  if (!m) return `${def.id} has no method ${mt[1]}`;
  const ad = m.args ?? [];
  const inner = (mt[2] ?? "").trim();
  if (!ad.length) return { m, args: [] };
  if (!inner) return { m, args: defaults(m, s) };
  const parts = ad.length === 1 ? [inner] : inner.split(",").map((x) => x.trim());
  const args = ad.map((a, i) => {
    let v = (parts[i] ?? "").replace(/^"(.*)"$/, "$1");
    if (a.kind === "choice") v = a.options!.find((o) => norm(o) === norm(v)) ?? v;
    return v;
  });
  const bad = ad.findIndex((a, i) => a.kind === "choice" && !a.options!.includes(args[i]));
  if (bad >= 0) return `${m.name}: choose one of ${ad[bad].options!.join(" | ")}`;
  return { m, args };
}

const show = <S,>(m: MethodDef<S>, args: string[]) =>
  `${m.name}(${(m.args ?? []).map((a, i) => (a.kind === "key" ? JSON.stringify(args[i]) : args[i])).join(", ")})`;

/* ---------- array generations (for addresses and the old-array ghost) ---------- */

function gens<S>(meta: Meta<S>, m: MethodDef<S>, pre: S, frames: Frame<S>[], g0: number): number[] {
  let g = g0;
  let prev = meta.capOf(pre);
  return frames.map((f, i) => {
    const cap = meta.capOf(f.s);
    if (i === 0 && m.group === "construct" && cap > 0) g++;
    else if (cap !== prev && cap > 0 && m.name !== "Clip") g++;
    prev = cap;
    return g;
  });
}

interface Entry {
  text: string;
  ret?: string;
  err?: string;
  g0: number;
  g1: number;
}

let booted = false; // StrictMode runs effects twice in dev

function Workbench<S>({ def, meta, addrs, setAddrs }: { def: StructureDef<S>; meta: Meta<S>; addrs: boolean; setAddrs: (b: boolean) => void }) {
  const p = usePlayer(def);
  const [log, setLog] = useState<Entry[]>([]);
  const [src, setSrc] = useState("");
  const [all, setAll] = useState(false);
  const [hover, setHover] = useState<MethodDef<S> | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const logBox = useRef<HTMLDivElement>(null);
  const g = log.length ? log[log.length - 1].g1 : 0;

  const exec = (text: string, from: S, prev: Entry[]): [Entry[], S] => {
    const g0 = prev.length ? prev[prev.length - 1].g1 : 0;
    const parsed = parseCall(def, text, from);
    if (typeof parsed === "string") return [[...prev, { text: text.trim(), err: parsed, g0, g1: g0 }], from];
    const { m, args } = parsed;
    const shown = show(m, args);
    let fr: Frame<S>[];
    try {
      fr = m.run(from, args);
    } catch (e) {
      p.runFrom(from, m, args);
      const msg = e instanceof GoPanic ? e.message : String(e);
      return [[...prev, { text: shown, err: "panic: " + msg, g0, g1: g0 }], from];
    }
    const gs = gens(meta, m, from, fr, g0);
    p.runFrom(from, m, args);
    return [[...prev, { text: shown, ret: fr[fr.length - 1].ret, g0, g1: gs[gs.length - 1] }], fr[fr.length - 1].s];
  };

  const submit = (text: string) => {
    if (!text.trim()) return;
    setLog(exec(text, p.state, log)[0]);
    setSrc("");
  };

  const runMethod = (m: MethodDef<S>) => {
    const call = show(m, defaults(m, p.state));
    setLog(exec(call, p.state, log)[0]);
    setSrc(m.args?.length ? call : "");
  };

  const undo = () => {
    if (!p.canUndo) return;
    p.undo();
    setLog((l) => {
      const i = l.map((e) => !e.err).lastIndexOf(true);
      return i < 0 ? l : l.slice(0, i);
    });
  };

  // ?seq=Push:42;Pop replays calls on load (args split by |); &frame=N stops at a step.
  useEffect(() => {
    const seq = params.get("seq");
    if (!seq || booted) return;
    booted = true;
    let s = p.state;
    let l: Entry[] = [];
    for (const step of seq.split(";")) {
      const [n, a] = step.split(":");
      [l, s] = exec(a === undefined ? n : `${n}(${a.split("|").join(", ")})`, s, l);
    }
    setLog(l);
    if (params.has("all")) setAll(true);
    const fi = params.get("frame");
    if (fi) setTimeout(() => p.jump(Number(fi)), 0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const el = logBox.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [log.length]);

  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "z") {
        e.preventDefault();
        undo();
        return;
      }
      if ((e.target as HTMLElement).tagName === "INPUT") return;
      if (e.key === " ") {
        e.preventDefault();
        p.toggle();
      } else if (e.key === "ArrowRight") p.step(1);
      else if (e.key === "ArrowLeft") p.step(-1);
      else if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) input.current?.focus();
    };
    addEventListener("keydown", on);
    return () => removeEventListener("keydown", on);
  });

  const cur = log[log.length - 1];
  const pre = (p.history[p.history.length - 1] ?? p.state) as S;
  const lastM = p.call ? def.methods.find((m) => m.name === p.call!.method) : undefined;
  const frameGens = useMemo(
    () => (lastM && cur && !cur.err ? gens(meta, lastM, pre, p.frames, cur.g0) : p.frames.map(() => g)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [p.frames, cur],
  );
  const gen = frameGens[p.idx] ?? g;
  const changeAt = cur ? frameGens.findIndex((x) => x !== cur.g0) : -1;
  const ghostS = changeAt >= 0 && p.idx >= changeAt ? ((changeAt === 0 ? pre : p.frames[changeAt - 1].s) as S) : null;
  const ghost = ghostS && meta.capOf(ghostS) > 0 && meta.capOf(p.frame.s) > 0 ? meta.ghost(ghostS) : null;

  const f = p.frame;
  const View = meta.View;
  const info = hover ?? lastM ?? null;
  const groups = GROUPS.map((gr) => [gr, def.methods.filter((m) => m.group === gr)] as const).filter(([, ms]) => ms.length);
  const quick = meta.quick.map((n) => def.methods.find((m) => m.name === n)!).filter(Boolean);

  const chip = (m: MethodDef<S>) => (
    <button key={m.name} type="button" onClick={() => runMethod(m)} onMouseEnter={() => setHover(m)} onMouseLeave={() => setHover(null)} onFocus={() => setHover(m)} onBlur={() => setHover(null)}>
      {m.name}
      <i>({(m.args ?? []).map((a) => a.name).join(", ")})</i>
    </button>
  );

  return (
    <main className="main">
      <section className="head">
        <div>
          <div className="pkg">
            import "github.com/Nergous/structures/<b>{def.id}</b>"
          </div>
          <h1>{meta.type}</h1>
          <p>{def.tagline}</p>
        </div>
        <label className="switch">
          <input type="checkbox" checked={addrs} onChange={(e) => setAddrs(e.target.checked)} />
          <span />
          addresses
        </label>
      </section>

      <View f={f} base={addrOf(gen)} />

      {ghost && (
        <section className="ghost">
          <div className="ghost-l">
            old array{addrs && cur ? ` at ${hex(addrOf(cur.g0))}` : ""} — nothing points here any more; the GC will free it
          </div>
          <div className="ghost-r">
            {ghost.map((t, i) => (
              <span key={i}>{t}</span>
            ))}
          </div>
        </section>
      )}

      <section className={"steps" + (f.panic ? " panic" : "")}>
        <button className="play" onClick={p.toggle} disabled={p.frames.length < 2} aria-label={p.playing ? "pause" : "play"} title="play / pause (space)">
          {p.playing ? "❚❚" : "▶"}
        </button>
        <input type="range" min={0} max={Math.max(p.frames.length - 1, 0)} value={p.idx} onChange={(e) => p.jump(Number(e.target.value))} disabled={p.frames.length < 2} aria-label="step" title="← → step" />
        <span className="stepn">
          {p.idx + 1}/{p.frames.length}
        </span>
        <p className="note" key={p.idx + f.note}>
          {p.call ? f.note : "Type a call below or click a method. Every step of it plays here."}
        </p>
        {f.out?.length ? (
          <div className="out">
            <span>{f.outLabel ?? "returned"}</span>
            {f.out.map((o) => (
              <code key={o.id} className={o.tone ? "t-" + o.tone : ""}>
                {o.t}
              </code>
            ))}
          </div>
        ) : null}
      </section>

      <section className="console" onClick={(e) => e.target === e.currentTarget && input.current?.focus()}>
        <div className="log" ref={logBox}>
          <div className="line init">{meta.init}</div>
          {log.map((e, i) => (
            <div key={i} className={"line" + (i === log.length - 1 ? " last" : "")}>
              <span className="in">
                {meta.v}.{e.text}
              </span>
              {e.err ? <span className="err">{e.err}</span> : e.ret ? <span className="ret">// {e.ret}</span> : null}
            </div>
          ))}
        </div>
        <form
          className="prompt"
          onSubmit={(e) => {
            e.preventDefault();
            submit(src);
          }}
        >
          <span className="ps">{meta.v}.</span>
          <input ref={input} value={src} onChange={(e) => setSrc(e.target.value)} placeholder={quick[0] ? show(quick[0], defaults(quick[0], p.state)) : ""} spellCheck={false} aria-label="call" />
          <kbd>enter</kbd>
        </form>
        <div className="hint">
          {info ? (
            <>
              <code>{info.sig}</code>
              <span>{info.doc}</span>
              <em>{info.cost}</em>
            </>
          ) : (
            <span>Hover a method to see its signature. Click it to run with sample arguments, then edit the call above and press enter.</span>
          )}
        </div>
        <div className="chips">
          {quick.map(chip)}
          <button type="button" className="more" onClick={() => setAll((a) => !a)}>
            {all ? "fewer" : `all ${def.methods.length} methods`} {all ? "▴" : "▾"}
          </button>
          <button type="button" className="undo" onClick={undo} disabled={!p.canUndo} title="ctrl z">
            undo
          </button>
        </div>
        {all && (
          <div className="all">
            {groups.map(([gr, ms]) => (
              <div key={gr} className="grp">
                <h3>{GROUP_LABEL[gr]}</h3>
                <div className="chips flat">{ms.map(chip)}</div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="about">
        <div>
          <h2>How it works</h2>
          {def.about.map((t, i) => (
            <p key={i}>{t}</p>
          ))}
        </div>
        <div>
          <h2>Cost</h2>
          <table>
            <tbody>
              {def.cost.map(([k, v]) => (
                <tr key={k}>
                  <td>{k}</td>
                  <td>{v}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <a href={`${REPO}/tree/main/${def.id}`} target="_blank" rel="noreferrer">
            package source ↗
          </a>
        </div>
      </section>
      <footer className="foot">
        Every step is simulated from the Go source: the same growth policy, sift order, and compaction rule. Addresses are illustrative.
      </footer>
    </main>
  );
}
