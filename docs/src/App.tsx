import { useEffect, useMemo, useState } from "react";
import { MotionConfig } from "motion/react";
import { usePlayer } from "./core/player";
import { GROUP_LABEL, type Group, type MethodDef, type StructureDef } from "./core/types";
import { STRUCTURES } from "./structures";
import { SpeedCtx } from "./ui/viz";

const REPO = "https://github.com/Nergous/structures";
/** ?still disables motion (used for static screenshots). */
const STILL = new URLSearchParams(location.search).has("still");

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
  return (
    <div className="app">
      <aside className="rail">
        <a className="brand" href={REPO} target="_blank" rel="noreferrer">
          <span className="brand-mark" aria-hidden>
            <i />
            <i />
            <i />
            <i />
          </span>
          <span>
            <b>structures</b>
            <small>a visual field guide</small>
          </span>
        </a>
        <nav>
          {STRUCTURES.map((d, i) => (
            <button
              key={d.id}
              className={"nav-i" + (d.id === def.id ? " on" : "")}
              onClick={() => setId(d.id)}
            >
              <span className="nav-n">{String(i + 1).padStart(2, "0")}</span>
              <span className="nav-t">{d.title}</span>
              <span className="nav-k">{d.kind}</span>
            </button>
          ))}
        </nav>
        <p className="rail-foot">
          Every frame is simulated from the Go source: same growth policy, same
          sift order, same compaction rule.
          <br />
          <a href={REPO} target="_blank" rel="noreferrer">
            github.com/Nergous/structures ↗
          </a>
        </p>
      </aside>
      <Workbench key={def.id} def={def as StructureDef<unknown>} />
    </div>
  );
}

function Workbench<S>({ def }: { def: StructureDef<S> }) {
  const p = usePlayer(def);
  const [sel, setSel] = useState<MethodDef<S> | null>(null);
  const View = def.View;

  // Deep link: ?call=Push&args=42&frame=1 runs a call on load.
  const [boot, setBoot] = useState(() => new URLSearchParams(location.search));
  useEffect(() => {
    const name = boot.get("call");
    if (!name) return;
    const pre = boot.get("pre");
    let s = p.state;
    if (pre) {
      // pre=Push:5;Push:9 seeds state before the shown call
      for (const step of pre.split(";")) {
        const [n, a = ""] = step.split(":");
        const pm = def.methods.find((x) => x.name === n);
        if (!pm) continue;
        const fr = pm.run(s, a ? a.split("|") : []);
        s = fr[fr.length - 1].s;
      }
    }
    const m = def.methods.find((x) => x.name === name);
    setBoot(new URLSearchParams());
    if (!m) return;
    const args = boot.get("args")?.split("|") ?? [];
    setSel(m);
    p.runFrom(s, m, args);
    const fi = Number(boot.get("frame"));
    if (boot.has("frame")) setTimeout(() => p.jump(fi), 0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "SELECT" || tag === "TEXTAREA") return;
      if (e.key === " ") {
        e.preventDefault();
        p.toggle();
      } else if (e.key === "ArrowRight") p.step(1);
      else if (e.key === "ArrowLeft") p.step(-1);
      else if ((e.ctrlKey || e.metaKey) && e.key === "z") p.undo();
    };
    addEventListener("keydown", on);
    return () => removeEventListener("keydown", on);
  }, [p]);

  const stats = def.stats(p.frame.s);

  return (
    <main className="main">
      <header className="head">
        <div className="head-pkg">
          import "github.com/Nergous/structures/<b>{def.id}</b>"
        </div>
        <div className="head-row">
          <h1>{def.title}</h1>
          <span className="head-kind">{def.kind}</span>
        </div>
        <p className="head-tag">{def.tagline}</p>
      </header>

      <section className="work">
        <div className="work-l">
          <div className="stage-card">
            <div className="stage-top">
              <div className="stats">
                {stats.map(([k, v]) => (
                  <span key={k} className="stat">
                    <i>{k}</i>
                    <b>{v}</b>
                  </span>
                ))}
              </div>
              <div className="call">
                {p.call ? (
                  <code>
                    {varName(def.id)}.{p.call.method}(
                    <em>{p.call.args.join(", ")}</em>)
                  </code>
                ) : (
                  <code className="dim">// no call yet</code>
                )}
              </div>
            </div>
            <MotionConfig reducedMotion="user">
              <SpeedCtx.Provider value={STILL ? 1000 : p.speed}>
                <View f={p.frame} />
              </SpeedCtx.Provider>
            </MotionConfig>
          </div>
          <Narration p={p} />
        </div>
        <div className="work-r">
          <MethodPanel
            def={def}
            state={p.state}
            sel={sel}
            onSelect={setSel}
            onRun={(m, a) => p.run(m, a)}
          />
        </div>
      </section>

      <section className="about">
        <div className="about-t">
          <h2>How it works</h2>
          {def.about.map((t, i) => (
            <p key={i}>{t}</p>
          ))}
        </div>
        <div className="about-c">
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
          <a
            className="about-src"
            href={`${REPO}/tree/main/${def.id}`}
            target="_blank"
            rel="noreferrer"
          >
            Read the package source ↗
          </a>
        </div>
      </section>
    </main>
  );
}

const varName = (id: string) =>
  ({ stack: "s", queue: "q", deque: "d", set: "s", binheap: "h", pqueue: "q", orderedmap: "m", orderedset: "s", bitset: "b" })[id] ?? "x";

function Narration<S>({ p }: { p: ReturnType<typeof usePlayer<S>> }) {
  const f = p.frame;
  const last = p.idx === p.frames.length - 1;
  return (
    <div className={"narr" + (f.panic ? " panic" : "")}>
      <div className="narr-ctl">
        <button onClick={() => p.step(-1)} disabled={p.idx === 0} title="Previous step (←)">
          ‹
        </button>
        <button className="play" onClick={p.toggle} title="Play / pause (space)">
          {p.playing ? "❚❚" : last && p.frames.length > 1 ? "↺" : "▶"}
        </button>
        <button onClick={() => p.step(1)} disabled={last} title="Next step (→)">
          ›
        </button>
        <label className="speed" title="Playback speed">
          <input
            type="range"
            min={0.25}
            max={3}
            step={0.25}
            value={p.speed}
            onChange={(e) => p.setSpeed(Number(e.target.value))}
          />
          <span>{p.speed}×</span>
        </label>
        <button className="undo" onClick={p.undo} disabled={!p.canUndo} title="Undo last call (Ctrl+Z)">
          undo
        </button>
      </div>
      <div className="narr-body">
        <div className="narr-dots">
          {p.frames.map((_, i) => (
            <button
              key={i}
              className={"dot" + (i === p.idx ? " on" : i < p.idx ? " past" : "")}
              onClick={() => p.jump(i)}
              aria-label={`step ${i + 1}`}
            />
          ))}
          <span className="narr-count">
            step {p.idx + 1}/{p.frames.length}
          </span>
        </div>
        <p className="narr-note" key={p.idx + f.note}>
          {f.note}
        </p>
        {f.ret !== undefined && f.ret !== "" && (
          <div className="narr-ret">
            <span>returns</span>
            <code>{f.ret}</code>
          </div>
        )}
      </div>
    </div>
  );
}

const GROUP_ORDER: Group[] = ["construct", "read", "iterate", "copy", "mutate", "algebra", "memory"];

function MethodPanel<S>(props: {
  def: StructureDef<S>;
  state: S;
  sel: MethodDef<S> | null;
  onSelect: (m: MethodDef<S>) => void;
  onRun: (m: MethodDef<S>, args: string[]) => void;
}) {
  const { def, state, sel } = props;
  const groups = useMemo(() => {
    const g = new Map<Group, MethodDef<S>[]>();
    for (const m of def.methods) {
      if (!g.has(m.group)) g.set(m.group, []);
      g.get(m.group)!.push(m);
    }
    return GROUP_ORDER.filter((k) => g.has(k)).map((k) => [k, g.get(k)!] as const);
  }, [def]);

  const [args, setArgs] = useState<string[]>([]);
  const fillDefaults = (m: MethodDef<S>, s: S) =>
    setArgs(
      (m.args ?? []).map((a) =>
        typeof a.def === "function" ? a.def(s) : (a.def ?? a.options?.[0] ?? ""),
      ),
    );

  const pick = (m: MethodDef<S>) => {
    props.onSelect(m);
    fillDefaults(m, state);
    if (!m.args?.length) props.onRun(m, []);
  };

  const runSel = () => {
    if (!sel) return;
    props.onRun(sel, args);
  };

  // Refresh generated defaults (fresh values) after each call.
  useEffect(() => {
    if (sel) fillDefaults(sel, state);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state]);

  return (
    <div className="methods">
      {groups.map(([g, ms]) => (
        <div key={g} className="mg">
          <h3>{GROUP_LABEL[g]}</h3>
          <div className="mg-list">
            {ms.map((m) => (
              <button
                key={m.name}
                className={"chip g-" + g + (sel?.name === m.name ? " on" : "")}
                onClick={() => pick(m)}
                title={m.doc}
              >
                {m.name}
                {m.args?.length ? <i>…</i> : null}
              </button>
            ))}
          </div>
        </div>
      ))}
      <div className={"detail" + (sel ? " on" : "")}>
        {sel ? (
          <>
            <code className="sig">{sel.sig}</code>
            <p>{sel.doc}</p>
            <div className="cost">
              <span>cost</span> {sel.cost}
            </div>
            {sel.args?.length ? (
              <form
                className="args"
                onSubmit={(e) => {
                  e.preventDefault();
                  runSel();
                }}
              >
                {sel.args.map((a, i) => (
                  <label key={a.name}>
                    <span>{a.name}</span>
                    {a.kind === "choice" ? (
                      <select
                        value={args[i] ?? ""}
                        onChange={(e) => setArgs((x) => x.map((v, j) => (j === i ? e.target.value : v)))}
                      >
                        {a.options!.map((o) => (
                          <option key={o}>{o}</option>
                        ))}
                      </select>
                    ) : (
                      <input
                        value={args[i] ?? ""}
                        placeholder={a.hint}
                        spellCheck={false}
                        onChange={(e) => setArgs((x) => x.map((v, j) => (j === i ? e.target.value : v)))}
                      />
                    )}
                  </label>
                ))}
                <button type="submit" className="run">
                  Run {sel.name}
                </button>
              </form>
            ) : (
              <button className="run" onClick={runSel}>
                Run {sel.name} again
              </button>
            )}
          </>
        ) : (
          <p className="detail-empty">
            Choose a method. Methods without arguments run immediately; the
            others open a small form with sensible defaults.
          </p>
        )}
      </div>
      <p className="keys">
        <kbd>space</kbd> play · <kbd>←</kbd>
        <kbd>→</kbd> step · <kbd>ctrl z</kbd> undo
      </p>
    </div>
  );
}
