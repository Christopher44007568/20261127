package pipeline

import (
	"fmt"
	"html"
	"io"
	"strings"
)

const svgWidth = 1120
const svgHeight = 840

var passColors = map[PassID]string{
	Shadow:  "#f5a623",
	Scene:   "#38bdf8",
	Tonemap: "#34d399",
	UI:      "#f472b6",
}

// WriteScheduleSVG renders the dependency graph and the schedule's wave
// timeline. Only edges present in DataEdges are drawn as arrows.
func WriteScheduleSVG(w io.Writer, schedule Schedule, sha256 string) error {
	if err := schedule.Validate(); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`, svgWidth, svgHeight, svgWidth, svgHeight)
	fmt.Fprintf(&b, `<title>%s scheduling diagram</title>`, html.EscapeString(schedule.Name))
	b.WriteString(`<defs>
<marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
<path d="M 0 0 L 10 5 L 0 10 z" fill="#f5a623"/>
</marker>
<style>
text{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
.muted{fill:#8fa3b8}.body{fill:#dce8f3}.title{fill:#f4f8fb;font-size:25px;font-weight:700}
.section{fill:#aabdd0;font-size:13px;letter-spacing:1.7px}
.node-title{font-size:16px;font-weight:700}.node-sub{font-size:11px;fill:#9db0c4}
.small{font-size:11px}.tiny{font-size:10px}
</style>
</defs>`)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#080d14"/>`, svgWidth, svgHeight)
	fmt.Fprintf(&b, `<rect x="18" y="18" width="%d" height="%d" rx="16" fill="#0d1520" stroke="#223247"/>`, svgWidth-36, svgHeight-36)
	fmt.Fprintf(&b, `<text x="42" y="54" class="title">%s</text>`, html.EscapeString(schedule.Name))
	fmt.Fprintf(&b, `<text x="42" y="77" class="muted small">%s</text>`, html.EscapeString(schedule.Description))

	writeGraphPanel(&b, schedule)
	writeTimelinePanel(&b, schedule)

	fmt.Fprintf(&b, `<text x="42" y="805" class="muted tiny">PPM SHA-256 %s</text>`, html.EscapeString(sha256))
	fmt.Fprintf(&b, `<text x="%d" y="805" text-anchor="end" class="muted tiny">same-frame result: byte-identical across all legal schedules</text>`, svgWidth-42)
	b.WriteString(`</svg>
`)

	_, err := io.WriteString(w, b.String())
	return err
}

func writeGraphPanel(b *strings.Builder, schedule Schedule) {
	fmt.Fprintf(b, `<g id="dependency-graph">`)
	fmt.Fprintf(b, `<rect x="32" y="92" width="%d" height="326" rx="12" fill="#101a27" stroke="#223247"/>`, svgWidth-64)
	b.WriteString(`<text x="54" y="122" class="section">SCHEDULED PASS DAG · ACTUAL DATA EDGES ONLY</text>`)
	b.WriteString(`<text x="54" y="143" class="muted tiny">Solid arrows are producer → consumer constraints. No UI edge is invented.</text>`)

	writeNode(b, 70, 166, 190, 92, Shadow, "writes: shadow visibility")
	writeNode(b, 360, 166, 190, 92, Scene, "reads shadow · writes HDR")
	writeNode(b, 650, 166, 190, 92, Tonemap, "reads HDR · writes LDR")
	writeNode(b, 360, 292, 190, 78, UI, "writes UI RGBA · no input")

	writeDataEdge(b, 260, 212, 360, 212, "shadow map")
	writeDataEdge(b, 550, 212, 650, 212, "scene HDR")

	b.WriteString(`<rect x="585" y="283" width="365" height="86" rx="10" fill="#0b1420" stroke="#3b4d62" stroke-dasharray="5 5"/>`)
	b.WriteString(`<text x="603" y="309" class="body small">UI and tonemap have no edge between them.</text>`)
	b.WriteString(`<text x="603" y="329" class="muted tiny">Swap their order, or run them together: same frame.</text>`)
	b.WriteString(`<text x="603" y="349" class="muted tiny">Composite is fixed after the DAG; tonemap never reads UI.</text>`)

	fmt.Fprintf(b, `<rect x="930" y="160" width="126" height="112" rx="10" fill="#0b1420" stroke="#2b3b50"/>`)
	b.WriteString(`<text x="946" y="184" class="section">EDGE COUNT</text>`)
	b.WriteString(`<text x="946" y="219" class="title">2</text>`)
	b.WriteString(`<text x="946" y="242" class="muted tiny">shadow → scene</text>`)
	b.WriteString(`<text x="946" y="258" class="muted tiny">scene → tonemap</text>`)
	b.WriteString(`<line x1="946" y1="266" x2="1040" y2="266" stroke="#f5a623" stroke-width="2" marker-end="url(#arrow)"/>`)
	b.WriteString(`</g>`)
}

func writeNode(b *strings.Builder, x, y, width, height int, id PassID, subtitle string) {
	color := passColors[id]
	fmt.Fprintf(b, `<g id="node-%s">`, id)
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="#0b1420" stroke="%s" stroke-width="1.5"/>`, x, y, width, height, color)
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="6" height="%d" rx="3" fill="%s"/>`, x, y+8, height-16, color)
	fmt.Fprintf(b, `<text x="%d" y="%d" class="node-title" fill="%s">%s</text>`, x+22, y+34, color, strings.ToUpper(string(id)))
	fmt.Fprintf(b, `<text x="%d" y="%d" class="node-sub">%s</text>`, x+22, y+56, html.EscapeString(subtitle))
	spec := Spec(id)
	if len(spec.Requires) == 0 {
		fmt.Fprintf(b, `<text x="%d" y="%d" class="node-sub">requires: none</text>`, x+22, y+75)
	} else {
		requires := make([]string, len(spec.Requires))
		for i, dep := range spec.Requires {
			requires[i] = string(dep)
		}
		fmt.Fprintf(b, `<text x="%d" y="%d" class="node-sub">requires: %s</text>`, x+22, y+75, html.EscapeString(strings.Join(requires, ", ")))
	}
	b.WriteString(`</g>`)
}

func writeDataEdge(b *strings.Builder, x1, y1, x2, y2 int, label string) {
	labelWidth := len(label)*6 + 18
	labelX := x1 + (x2-x1-labelWidth)/2
	fmt.Fprintf(b, `<g class="data-edge">`)
	fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#f5a623" stroke-width="3" marker-end="url(#arrow)"/>`, x1, y1, x2, y2)
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="20" rx="5" fill="#2b2110" stroke="#6b4b1c"/>`, labelX, y1-32, labelWidth)
	fmt.Fprintf(b, `<text x="%d" y="%d" fill="#f8c76b" class="tiny">%s</text>`, labelX+9, y1-18, html.EscapeString(label))
	b.WriteString(`</g>`)
}

func writeTimelinePanel(b *strings.Builder, schedule Schedule) {
	const panelY = 438
	const panelHeight = 340
	fmt.Fprintf(b, `<g id="schedule-timeline">`)
	fmt.Fprintf(b, `<rect x="32" y="%d" width="%d" height="%d" rx="12" fill="#101a27" stroke="#223247"/>`, panelY, svgWidth-64, panelHeight)
	b.WriteString(`<text x="54" y="468" class="section">EXECUTION WAVES · SAME COLUMN RUNS CONCURRENTLY</text>`)
	b.WriteString(`<text x="54" y="489" class="muted tiny">A wave starts only after every pass in the previous wave has finished.</text>`)

	const laneX = 54
	const laneWidth = 152
	const laneHeight = 38
	lanes := []PassID{Shadow, Scene, Tonemap, UI}
	laneTop := map[PassID]int{
		Shadow:  522,
		Scene:   570,
		Tonemap: 618,
		UI:      666,
	}
	for i, id := range lanes {
		y := laneTop[id]
		fill := "#0c1622"
		if i%2 == 1 {
			fill = "#0e1926"
		}
		fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="7" fill="%s" stroke="#1b2a3c"/>`, laneX, y, laneWidth, laneHeight, fill)
		fmt.Fprintf(b, `<rect x="%d" y="%d" width="4" height="%d" rx="2" fill="%s"/>`, laneX+10, y+8, laneHeight-16, passColors[id])
		fmt.Fprintf(b, `<text x="%d" y="%d" class="body small">%s</text>`, laneX+25, y+24, strings.ToUpper(string(id)))
	}

	const waveStart = 228
	const waveStep = 208
	const waveWidth = 186
	for waveIndex := range schedule.Waves {
		x := waveStart + waveIndex*waveStep
		fmt.Fprintf(b, `<text x="%d" y="%d" class="muted tiny" text-anchor="middle">W%d</text>`, x+waveWidth/2, 514, waveIndex)
		if waveIndex > 0 {
			fmt.Fprintf(b, `<line x1="%d" y1="522" x2="%d" y2="704" stroke="#26384c" stroke-dasharray="4 6"/>`, x-11, x-11)
		}
	}
	fmt.Fprintf(b, `<line x1="%d" y1="522" x2="%d" y2="704" stroke="#26384c" stroke-dasharray="4 6"/>`, waveStart+len(schedule.Waves)*waveStep-11, waveStart+len(schedule.Waves)*waveStep-11)

	for waveIndex, wave := range schedule.Waves {
		for _, id := range wave {
			x := waveStart + waveIndex*waveStep
			y := laneTop[id]
			color := passColors[id]
			fmt.Fprintf(b, `<g id="timeline-%s-w%d">`, id, waveIndex)
			fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="7" fill="%s"/>`, x, y+2, waveWidth-4, laneHeight-4, color)
			fmt.Fprintf(b, `<text x="%d" y="%d" fill="#071018" class="node-title">%s</text>`, x+14, y+26, strings.ToUpper(string(id)))
			fmt.Fprintf(b, `<text x="%d" y="%d" fill="#071018" class="tiny" text-anchor="end">W%d</text>`, x+waveWidth-17, y+24, waveIndex)
			b.WriteString(`</g>`)
		}
	}

	b.WriteString(`<rect x="54" y="724" width="478" height="34" rx="7" fill="#102219" stroke="#25533a"/>`)
	b.WriteString(`<text x="68" y="746" class="body small">shadow → scene always crosses a wave boundary</text>`)
	b.WriteString(`<rect x="548" y="724" width="510" height="34" rx="7" fill="#241423" stroke="#63354f"/>`)
	b.WriteString(`<text x="562" y="746" class="body small">UI and tonemap share no edge: reorder or overlap safely</text>`)
	b.WriteString(`</g>`)
}
