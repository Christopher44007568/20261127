package report

const reportStyles = `
:root {
  --ink: #202b36;
  --muted: #657382;
  --paper: #f4f0e7;
  --card: #fbfaf6;
  --line: #d9d3c8;
  --navy: #111820;
  --orange: #d65a3a;
  --teal: #227f82;
}
* { box-sizing: border-box; }
html { background: var(--paper); color: var(--ink); }
body {
  margin: 0;
  font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI",
    "Noto Sans SC", "Microsoft YaHei", sans-serif;
  line-height: 1.55;
  background:
    radial-gradient(circle at 14% 0%, rgba(214,90,58,.08), transparent 32rem),
    radial-gradient(circle at 92% 24%, rgba(34,127,130,.08), transparent 34rem),
    var(--paper);
}
main { width: min(1120px, calc(100% - 32px)); margin: 28px auto 64px; }
.hero {
  position: relative;
  overflow: hidden;
  color: #f4f7f9;
  background: var(--navy);
  border: 1px solid #2b3845;
  border-radius: 26px;
  padding: 42px 46px 38px;
  box-shadow: 0 24px 60px rgba(25, 31, 38, .16);
}
.hero::after {
  content: "";
  position: absolute;
  width: 420px;
  height: 420px;
  right: -160px;
  top: -210px;
  border-radius: 50%;
  background: repeating-radial-gradient(circle, rgba(246,200,95,.22) 0 1px, transparent 1px 13px);
  opacity: .8;
}
.eyebrow {
  color: #f6c85f;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .16em;
  text-transform: uppercase;
}
h1 {
  max-width: 820px;
  margin: 10px 0 14px;
  font-size: clamp(30px, 4.2vw, 53px);
  line-height: 1.08;
  letter-spacing: -.035em;
}
.lede { max-width: 780px; margin: 0; color: #aebac4; font-size: 15px; }
.chips { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 24px; }
.chip {
  padding: 6px 11px; border: 1px solid #344352; border-radius: 999px;
  color: #dce4ea; font-size: 11px; background: rgba(255,255,255,.03);
}
.metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin: 16px 0;
}
.metric {
  min-height: 128px;
  padding: 20px 21px;
  border: 1px solid var(--line);
  border-radius: 18px;
  background: var(--card);
}
.metric .label { color: var(--muted); font-size: 11px; font-weight: 700; letter-spacing: .08em; }
.metric .value { margin-top: 9px; font-size: 29px; line-height: 1; font-weight: 760; letter-spacing: -.035em; }
.metric .note { margin-top: 11px; color: var(--muted); font-size: 11px; }
.metric.lbs .value { color: var(--orange); }
.metric.dq .value { color: var(--teal); }
.metric.exact .value { color: #497a53; font-size: 23px; }
.panel {
  margin-top: 16px;
  padding: 24px;
  border: 1px solid var(--line);
  border-radius: 22px;
  background: rgba(251,250,246,.9);
}
.panel-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin: 0 4px 15px; }
.panel h2 { margin: 0; font-size: 19px; letter-spacing: -.02em; }
.panel p { max-width: 650px; margin: 4px 0 0; color: var(--muted); font-size: 12px; }
.tag { color: var(--muted); font-size: 10px; letter-spacing: .08em; white-space: nowrap; }
.svg-wrap { overflow-x: auto; border-radius: 16px; }
.svg-wrap svg, .chart { display: block; width: 100%; height: auto; min-width: 720px; }
.split-note {
  display: grid; grid-template-columns: 1fr 1fr; gap: 12px;
  margin-top: 14px; color: var(--muted); font-size: 12px;
}
.split-note div { padding: 14px 16px; border-left: 3px solid var(--line); background: #f6f2e9; border-radius: 0 10px 10px 0; }
.split-note strong { color: var(--ink); }
.downloads { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
.downloads a {
  color: var(--ink); text-decoration: none; font-size: 11px; font-weight: 650;
  padding: 8px 12px; border: 1px solid var(--line); border-radius: 9px; background: #fffdf8;
}
.downloads a:hover { border-color: #9d968b; background: #fff; }
.method {
  display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-top: 16px;
}
.method article { padding: 22px 24px; border-radius: 18px; background: var(--card); border: 1px solid var(--line); }
.method h3 { margin: 0 0 7px; font-size: 14px; }
.method p { margin: 0; color: var(--muted); font-size: 12px; }
footer { margin-top: 22px; padding: 0 5px; color: var(--muted); font-size: 11px; }
code { padding: 1px 5px; border-radius: 5px; background: #eae4d8; color: #3c4854; }
@media (max-width: 850px) {
  .hero { padding: 30px 24px; border-radius: 20px; }
  .metrics { grid-template-columns: 1fr 1fr; }
  .panel { padding: 16px; border-radius: 18px; }
  .method, .split-note { grid-template-columns: 1fr; }
  .panel-head { align-items: flex-start; flex-direction: column; gap: 5px; }
}
@media (max-width: 520px) {
  main { width: min(100% - 20px, 1120px); margin-top: 10px; }
  .metrics { grid-template-columns: 1fr; }
  .metric { min-height: 0; }
}
`
