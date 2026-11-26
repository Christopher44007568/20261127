package report

const baseCSS = `
:root {
  --bg: #060a0f;
  --panel: #0b121a;
  --panel-2: #0e1822;
  --line: rgba(151, 181, 199, .16);
  --line-strong: rgba(151, 181, 199, .28);
  --text: #eef6f7;
  --muted: #8097a5;
  --green: #46e6b4;
  --amber: #ffc85b;
  --blue: #6ba8ff;
  --red: #ff5574;
  --mono: "SFMono-Regular", "Cascadia Code", "Liberation Mono", monospace;
  --sans: "Avenir Next", "Segoe UI", "Noto Sans SC", "Microsoft YaHei", sans-serif;
}
* { box-sizing: border-box; }
html { scroll-behavior: smooth; }
body {
  margin: 0;
  color: var(--text);
  background:
    radial-gradient(circle at 84% -8%, rgba(50, 213, 164, .13), transparent 31rem),
    radial-gradient(circle at 8% 34%, rgba(91, 159, 255, .08), transparent 28rem),
    linear-gradient(180deg, #05090d 0%, #081019 58%, #05090c 100%);
  font-family: var(--sans);
  min-height: 100vh;
}
body::before {
  content: "";
  position: fixed;
  inset: 0;
  pointer-events: none;
  opacity: .2;
  background-image:
    linear-gradient(rgba(132, 167, 185, .13) 1px, transparent 1px),
    linear-gradient(90deg, rgba(132, 167, 185, .13) 1px, transparent 1px);
  background-size: 56px 56px;
  mask-image: linear-gradient(to bottom, black, transparent 82%);
}
a { color: inherit; }
.shell { width: min(1440px, 100%); margin: 0 auto; padding: 26px 48px 72px; position: relative; }
.topbar {
  display: flex; align-items: center; justify-content: space-between;
  gap: 24px; padding-bottom: 22px; border-bottom: 1px solid var(--line);
}
.brand { display: flex; align-items: center; gap: 13px; text-decoration: none; }
.brand-mark {
  width: 38px; height: 38px; display: grid; place-items: center;
  border: 1px solid rgba(70, 230, 180, .5); background: rgba(70, 230, 180, .06);
  color: var(--green); font: 600 19px var(--mono); box-shadow: inset 0 0 22px rgba(70, 230, 180, .06);
}
.brand-copy { display: flex; flex-direction: column; gap: 2px; }
.brand-copy strong { font: 600 12px var(--mono); letter-spacing: .16em; }
.brand-copy span { color: var(--muted); font: 10px var(--mono); letter-spacing: .13em; }
.status { display: flex; align-items: center; gap: 18px; color: var(--muted); font: 10px var(--mono); letter-spacing: .1em; }
.status-live { display: flex; align-items: center; gap: 7px; color: var(--green); }
.status-live::before {
  content: ""; width: 6px; height: 6px; border-radius: 50%;
  background: var(--green); box-shadow: 0 0 12px var(--green);
}
.hero {
  display: grid; grid-template-columns: minmax(0, 1fr) 380px;
  gap: 62px; align-items: end; padding: 72px 0 64px;
}
.kicker { color: var(--green); font: 600 10px var(--mono); letter-spacing: .22em; margin-bottom: 25px; }
.kicker::before { content: "// "; color: var(--muted); }
h1 {
  margin: 0; font-size: clamp(58px, 7.1vw, 106px); line-height: .88;
  letter-spacing: -.065em; font-weight: 650;
}
h1 .outline {
  color: transparent; -webkit-text-stroke: 1px rgba(238, 246, 247, .7);
  text-shadow: 0 0 46px rgba(70, 230, 180, .08);
}
.hero-copy {
  max-width: 660px; color: #9bb0ba; font-size: 15px; line-height: 1.8;
  margin: 30px 0 0;
}
.hero-copy strong { color: var(--text); font-weight: 500; }
.hero-tags { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 28px; }
.pill {
  display: inline-flex; align-items: center; gap: 8px; padding: 7px 11px;
  border: 1px solid var(--line); background: rgba(11, 18, 26, .72);
  color: #a9bdc7; font: 10px var(--mono); letter-spacing: .06em;
}
.pill::before { content: "·"; color: var(--green); font-size: 16px; line-height: 0; }
.panel {
  border: 1px solid var(--line); background: linear-gradient(150deg, rgba(15, 25, 35, .96), rgba(8, 14, 20, .92));
  box-shadow: 0 30px 80px rgba(0, 0, 0, .28);
}
.hero-panel { padding: 22px; position: relative; overflow: hidden; }
.hero-panel::after {
  content: ""; position: absolute; width: 150px; height: 150px; right: -70px; top: -70px;
  border: 1px solid rgba(70, 230, 180, .22); transform: rotate(45deg);
}
.panel-label { color: var(--muted); font: 9px var(--mono); letter-spacing: .18em; margin-bottom: 17px; }
.stat-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1px; background: var(--line); border: 1px solid var(--line); }
.stat { background: #0a1118; padding: 15px 14px 13px; }
.stat b { display: block; font: 500 25px var(--mono); letter-spacing: -.05em; }
.stat span { display: block; color: var(--muted); font: 8px var(--mono); letter-spacing: .1em; margin-top: 5px; }
.stat b.green { color: var(--green); }
.pipeline { display: grid; gap: 0; margin-top: 20px; }
.pipe-row {
  display: grid; grid-template-columns: 24px 1fr auto; align-items: center;
  gap: 9px; padding: 8px 0; border-bottom: 1px solid rgba(151, 181, 199, .1);
  font-size: 11px;
}
.pipe-row:last-child { border-bottom: 0; }
.pipe-row i { color: var(--green); font: normal 9px var(--mono); }
.pipe-row small { color: #647b89; font: 8px var(--mono); letter-spacing: .08em; }
.legend {
  display: flex; align-items: center; justify-content: space-between; gap: 20px;
  border-top: 1px solid var(--line); border-bottom: 1px solid var(--line);
  padding: 14px 0; color: var(--muted); font: 10px var(--mono); letter-spacing: .05em;
}
.legend-left { display: flex; flex-wrap: wrap; gap: 20px; }
.legend-item { display: inline-flex; align-items: center; gap: 8px; }
.swatch { width: 22px; height: 8px; display: inline-block; }
.swatch.union { background: var(--green); }
.swatch.intersection { background: var(--amber); }
.swatch.difference { background: var(--blue); }
.swatch.internal { background: repeating-linear-gradient(90deg, var(--red) 0 3px, transparent 3px 5px); }
`

const cardsCSS = `
.case { padding-top: 56px; }
.case-head {
  display: grid; grid-template-columns: 112px minmax(0, 1fr) auto;
  gap: 24px; align-items: center; padding: 22px 0 25px; border-bottom: 1px solid var(--line);
}
.case-number {
  color: transparent; -webkit-text-stroke: 1px rgba(151, 181, 199, .42);
  font: 500 56px/1 var(--mono); letter-spacing: -.08em;
}
.case-title { display: flex; align-items: baseline; gap: 15px; flex-wrap: wrap; }
.case-title h2 { margin: 0; font-size: 28px; letter-spacing: -.035em; font-weight: 580; }
.case-title span { color: var(--green); font: 10px var(--mono); letter-spacing: .15em; }
.case-desc { color: #8da3ae; margin: 8px 0 0; font-size: 13px; line-height: 1.65; }
.case-tags { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 6px; max-width: 260px; }
.tag {
  border: 1px solid var(--line); padding: 5px 8px; color: #8ba1ad;
  font: 8px var(--mono); letter-spacing: .06em;
}
.inputs {
  display: flex; flex-wrap: wrap; gap: 12px 24px; padding: 14px 0;
  border-bottom: 1px solid var(--line); color: #6f8795; font: 9px var(--mono); letter-spacing: .05em;
}
.inputs b { color: #bed0d7; font-weight: 500; }
.input-chip { display: inline-flex; align-items: center; gap: 8px; }
.input-chip::before { content: ""; width: 5px; height: 5px; border: 1px solid #63808e; }
.cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; padding-top: 18px; }
.card {
  --accent: var(--green);
  min-width: 0; border: 1px solid var(--line); overflow: hidden;
  background: linear-gradient(180deg, rgba(13, 22, 31, .98), rgba(7, 12, 17, .98));
  transition: border-color .2s ease, transform .2s ease;
}
.card:hover { border-color: rgba(151, 181, 199, .34); transform: translateY(-2px); }
.card.union { --accent: var(--green); }
.card.intersection { --accent: var(--amber); }
.card.difference { --accent: var(--blue); }
.card-head {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 14px 15px; border-bottom: 1px solid var(--line);
}
.op-name { display: flex; align-items: center; gap: 11px; min-width: 0; }
.op-symbol {
  width: 36px; height: 36px; display: grid; place-items: center; flex: 0 0 auto;
  border: 1px solid color-mix(in srgb, var(--accent) 40%, transparent);
  color: var(--accent); background: color-mix(in srgb, var(--accent) 7%, transparent);
  font: 400 24px/1 var(--mono);
}
.op-title b { display: block; font-size: 14px; font-weight: 550; }
.op-title small { display: block; color: #718895; font: 8px var(--mono); letter-spacing: .13em; margin-top: 4px; }
.op-volume { text-align: right; min-width: 74px; }
.op-volume b { display: block; color: var(--accent); font: 500 15px var(--mono); }
.op-volume small { color: #667d8b; font: 7px var(--mono); letter-spacing: .08em; }
.stage {
  position: relative; aspect-ratio: 700 / 520; overflow: hidden;
  background:
    radial-gradient(circle at 50% 46%, color-mix(in srgb, var(--accent) 8%, transparent), transparent 55%),
    linear-gradient(180deg, rgba(13, 23, 32, .65), rgba(5, 9, 13, .98));
}
.stage::before {
  content: ""; position: absolute; inset: 0; opacity: .48; pointer-events: none;
  background-image:
    linear-gradient(rgba(132, 167, 185, .08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(132, 167, 185, .08) 1px, transparent 1px);
  background-size: 28px 28px;
  mask-image: radial-gradient(circle at 50% 50%, black, transparent 76%);
}
.result-render, .obj-render { position: absolute; inset: 0; }
.obj-render { display: none; }
.mesh-svg { display: block; width: 100%; height: 100%; filter: drop-shadow(0 18px 18px rgba(0, 0, 0, .25)); }
.interior-layer { mix-blend-mode: screen; }
.card[data-view="result"] .interior-layer { display: none; }
.card[data-view="internal"] .surface-layer { display: none; }
.card[data-view="obj"] .result-render { display: none; }
.card[data-view="obj"] .obj-render { display: block; }
.internal-badge {
  position: absolute; top: 12px; right: 12px; z-index: 2;
  display: flex; align-items: center; gap: 6px; padding: 5px 7px;
  color: #ff9caf; border: 1px solid rgba(255, 85, 116, .28);
  background: rgba(28, 9, 14, .74); font: 8px var(--mono); letter-spacing: .05em;
}
.internal-badge::before { content: ""; width: 5px; height: 5px; background: var(--red); }
.stage-meta {
  position: absolute; left: 12px; bottom: 11px; z-index: 2;
  color: #5f7886; font: 8px var(--mono); letter-spacing: .08em;
}
.stage-meta b { color: #9ab0ba; font-weight: 400; }
.view-switch {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 1px;
  background: var(--line); border-top: 1px solid var(--line); border-bottom: 1px solid var(--line);
}
.view-switch button {
  appearance: none; border: 0; background: #0a1118; color: #677f8d;
  padding: 8px 4px; font: 8px var(--mono); letter-spacing: .03em; cursor: pointer;
}
.view-switch button:hover { color: #b8cbd3; background: #0d1720; }
.card[data-view="both"] button[data-view="both"],
.card[data-view="result"] button[data-view="result"],
.card[data-view="internal"] button[data-view="internal"],
.card[data-view="obj"] button[data-view="obj"] {
  color: var(--accent); background: color-mix(in srgb, var(--accent) 8%, #0a1118);
  box-shadow: inset 0 -1px 0 var(--accent);
}
.card-stats { display: grid; grid-template-columns: repeat(3, 1fr); }
.metric { padding: 10px 11px; border-right: 1px solid var(--line); }
.metric:last-child { border-right: 0; }
.metric b { display: block; font: 500 11px var(--mono); color: #d6e4e8; }
.metric span { display: block; margin-top: 3px; color: #5e7481; font: 7px var(--mono); letter-spacing: .05em; }
.card-foot {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  padding: 9px 11px; border-top: 1px solid var(--line); font: 8px var(--mono);
}
.card-foot a { color: #7993a0; text-decoration: none; }
.card-foot a:hover { color: var(--accent); }
.pass { color: var(--green); }
.pass::before { content: "● "; font-size: 6px; vertical-align: 1px; }
`

const methodCSS = `
.section-head {
  display: flex; align-items: flex-end; justify-content: space-between; gap: 30px;
  margin: 78px 0 22px; padding-bottom: 17px; border-bottom: 1px solid var(--line);
}
.section-head h2 { margin: 0; font-size: 27px; letter-spacing: -.04em; font-weight: 560; }
.section-head p { margin: 0; max-width: 560px; color: #7e96a3; font-size: 12px; line-height: 1.7; text-align: right; }
.method-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 1px; background: var(--line); border: 1px solid var(--line); }
.method {
  min-height: 178px; padding: 18px 16px; background: #091017; position: relative;
}
.method i { color: var(--green); font: normal 9px var(--mono); letter-spacing: .1em; }
.method h3 { margin: 26px 0 9px; font-size: 14px; font-weight: 540; }
.method p { margin: 0; color: #718895; font-size: 11px; line-height: 1.65; }
.method::after {
  content: ""; position: absolute; right: 14px; top: 17px;
  width: 16px; height: 16px; border-top: 1px solid rgba(70, 230, 180, .22);
  border-right: 1px solid rgba(70, 230, 180, .22);
}
.verify-wrap { border: 1px solid var(--line); overflow-x: auto; background: rgba(7, 12, 17, .62); }
table { width: 100%; border-collapse: collapse; min-width: 790px; }
th {
  padding: 12px 14px; text-align: left; border-bottom: 1px solid var(--line);
  color: #647d8a; font: 8px var(--mono); letter-spacing: .11em; font-weight: 500;
}
td { padding: 12px 14px; border-bottom: 1px solid rgba(151, 181, 199, .09); font-size: 11px; }
tr:last-child td { border-bottom: 0; }
td.n { font-family: var(--mono); color: #b9cbd2; }
td.dim { color: #6c8491; }
.op-dot { display: inline-block; width: 6px; height: 6px; margin-right: 8px; border-radius: 50%; }
.op-dot.union { background: var(--green); }
.op-dot.intersection { background: var(--amber); }
.op-dot.difference { background: var(--blue); }
.check { color: var(--green); font: 9px var(--mono); }
.footer {
  display: flex; align-items: center; justify-content: space-between; gap: 20px;
  margin-top: 66px; padding-top: 22px; border-top: 1px solid var(--line);
  color: #536a77; font: 8px var(--mono); letter-spacing: .06em;
}
.footer b { color: #9eb2bb; font-weight: 400; }
@media (max-width: 1180px) {
  .shell { padding: 24px 30px 60px; }
  .hero { grid-template-columns: 1fr; gap: 35px; }
  .hero-panel { max-width: 520px; }
  .cards { grid-template-columns: 1fr; }
  .stage { aspect-ratio: 1.55; }
  .method-grid { grid-template-columns: repeat(2, 1fr); }
  .method:last-child { grid-column: span 2; }
}
@media (max-width: 720px) {
  .shell { padding: 20px 18px 50px; }
  .status span:not(.status-live) { display: none; }
  .hero { padding: 48px 0 42px; }
  h1 { font-size: 58px; }
  .case-head { grid-template-columns: 62px minmax(0, 1fr); gap: 14px; }
  .case-number { font-size: 34px; }
  .case-tags { grid-column: 1 / -1; justify-content: flex-start; max-width: none; }
  .case-title h2 { font-size: 23px; }
  .section-head { align-items: flex-start; flex-direction: column; }
  .section-head p { text-align: left; }
  .method-grid { grid-template-columns: 1fr; }
  .method:last-child { grid-column: auto; }
  .footer { align-items: flex-start; flex-direction: column; }
}
`

func pageCSS() string {
	return baseCSS + cardsCSS + methodCSS
}
