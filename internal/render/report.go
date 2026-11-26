package render

import (
	"fmt"
	"strings"

	"bipedblend/internal/biped"
)

func HTMLReport(result *biped.Result, framesSVG, weightsSVG, slipSVG string) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	b.WriteString(`<title>二足动画相位同步混合报告</title><style>`)
	b.WriteString(`:root{color-scheme:light;font-family:"Noto Sans CJK SC","Microsoft YaHei",system-ui,sans-serif}body{margin:0;background:#eaf0f7;color:#0f172a}main{max-width:1360px;margin:0 auto;padding:28px 20px 60px}.hero{background:linear-gradient(135deg,#0f172a,#1e3a8a);color:#fff;border-radius:24px;padding:30px 34px;box-shadow:0 18px 50px #0f172a22}h1{margin:0 0 10px;font-size:30px}.hero p{margin:6px 0;color:#cbd5e1;line-height:1.7}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin:20px 0}.card{background:#fff;border:1px solid #dbe4ef;border-radius:18px;padding:16px 18px;box-shadow:0 8px 24px #0f172a0d}.metric{font-size:26px;font-weight:800;margin-top:6px}.label{color:#64748b;font-size:13px}.panel{background:#fff;border:1px solid #dbe4ef;border-radius:22px;padding:18px;margin:18px 0;box-shadow:0 10px 30px #0f172a12}.panel h2{margin:4px 6px 14px;font-size:21px}.svg-wrap{overflow:auto;border-radius:14px}.svg-wrap svg{display:block;width:100%;height:auto;min-width:980px}table{width:100%;border-collapse:collapse;font-size:14px}th,td{padding:9px 10px;border-bottom:1px solid #e2e8f0;text-align:left}th{color:#475569;background:#f8fafc;position:sticky;top:0}.table-wrap{max-height:430px;overflow:auto;border:1px solid #e2e8f0;border-radius:14px}code{background:#f1f5f9;padding:2px 6px;border-radius:6px}.ok{color:#15803d;font-weight:800}.warn{color:#c2410c;font-weight:800}@media(max-width:900px){.grid{grid-template-columns:repeat(2,1fr)}}`)
	b.WriteString(`</style></head><body><main>`)
	b.WriteString(`<section class="hero"><h1>二足角色待机 / 行走 / 跑步混合</h1>`)
	b.WriteString(`<p>同一归一化相位驱动左右脚；速度 BlendSpace 权重逐帧归一；多 Clip 的同相落地/离地事件按帧邻近与交叠范围合并，并通过边沿门只触发一次。</p>`)
	b.WriteString(`<p>速度设定在 0.8s、2.8s、5.2s 突变；实际速度使用帧率无关的指数平滑，权重由平滑速度重新求解。</p></section>`)

	b.WriteString(`<section class="grid">`)
	metricCard("帧数", fmt.Sprintf("%d", result.Summary.FrameCount), fmt.Sprintf("%.1f 秒 @%dfps", result.Summary.DurationS, biped.ScenarioFPS))
	metricCard("权重误差", fmt.Sprintf("%.1e", result.Summary.WeightSumMaxError), "Σw 始终为 1")
	metricCard("锚定峰值滑移", fmt.Sprintf("%0.3f m/s", result.Summary.MaxLockedSlipSpeed), "接触帧世界 X 不变")
	metricCard("合并/触发事件", fmt.Sprintf("%d / %d", result.Summary.MergedEventCount, result.Summary.TriggeredEventCount), fmt.Sprintf("原始候选 %d", result.Summary.RawEventCount))
	b.WriteString(`</section>`)

	writePanel(&b, "三种速度连续帧", framesSVG)
	writePanel(&b, "权重曲线", weightsSVG)
	writePanel(&b, "脚底滑移", slipSVG)
	writeEventTable(&b, result)

	b.WriteString(`</main></body></html>`)
	return b.String()
}

func metricCard(label, value, hint string) string {
	return `<div class="card"><div class="label">` + label + `</div><div class="metric">` + value + `</div><div class="label">` + hint + `</div></div>`
}

func writePanel(b *strings.Builder, title, svg string) {
	b.WriteString(`<section class="panel"><h2>` + title + `</h2><div class="svg-wrap">`)
	b.WriteString(svg)
	b.WriteString(`</div></section>`)
}

func writeEventTable(b *strings.Builder, result *biped.Result) {
	b.WriteString(`<section class="panel"><h2>合并事件帧明细</h2><div class="table-wrap"><table>`)
	b.WriteString(`<thead><tr><th>ID</th><th>时间</th><th>类型</th><th>帧范围</th><th>原始帧</th><th>来源 Clip</th><th>候选数</th></tr></thead><tbody>`)
	for _, event := range result.Events {
		time := float64(event.Frame) / float64(biped.ScenarioFPS)
		frameRange := fmt.Sprintf("%d", event.Frame)
		if event.EndFrame != event.Frame {
			frameRange = fmt.Sprintf("%d–%d", event.Frame, event.EndFrame)
		}
		rawFrames := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(event.RawFrames)), ","), "[]")
		sources := strings.Join(event.SourceIDs, ", ")
		if sources == "" {
			sources = "—"
		}
		fmt.Fprintf(b, `<tr><td><code>%s</code></td><td>%.3fs</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="ok">%d → 1</td></tr>`,
			event.ID, time, event.Kind.Label(), frameRange, rawFrames, sources, event.Multiplicity)
	}
	b.WriteString(`</tbody></table></div></section>`)
}
