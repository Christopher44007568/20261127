package report

import (
	"fmt"
	"strings"

	"characterbounds/hierbox"
)

const (
	svgWidth  = 1120
	svgHeight = 470
)

func OverlaySVG(r *hierbox.Runner, res *hierbox.PoseResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="ui-sans-serif,system-ui,'Noto Sans CJK SC','Microsoft YaHei','Droid Sans Fallback',sans-serif">`,
		svgWidth, svgHeight, svgWidth, svgHeight)
	b.WriteString(`<rect width="100%" height="100%" fill="#f8fafc"/>`)
	fmt.Fprintf(&b, `<text x="28" y="34" font-size="22" font-weight="700" fill="#0f172a">%s pose - bone-topology hierarchy boxes / %s姿态</text>`, englishPoseTitle(res.Name), poseTitle(res.Name))
	fmt.Fprintf(&b, `<text x="28" y="57" font-size="12" fill="#475569">All %d vertices covered by owner bone and root boxes. Red dashed AABB follows root only; red dots are leaks.</text>`, res.Coverage.VertexCount)

	side := newPlot(35, 82, 500, 330, -1.65, 1.25, -0.20, 2.05)
	top := newPlot(585, 82, 500, 330, -1.65, 1.25, -0.90, 0.90)
	drawPanelBackground(&b, side, "Side X-Y / 侧视", true)
	drawPanelBackground(&b, top, "Top X-Z / 顶视", false)

	drawMesh(&b, r, res, side, true)
	drawMesh(&b, r, res, top, false)
	drawBoxes(&b, r, side, true)
	drawBoxes(&b, r, top, false)
	drawNaiveBox(&b, res, side, true)
	drawNaiveBox(&b, res, top, false)
	drawNaiveLeaks(&b, res, side, true)
	drawNaiveLeaks(&b, res, top, false)
	drawLegend(&b)
	b.WriteString(`</svg>`)
	return b.String()
}

func drawPanelBackground(b *strings.Builder, p plot, title string, ground bool) {
	fmt.Fprintf(b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="10" fill="#ffffff" stroke="#cbd5e1"/>`,
		p.x-18, p.y-34, p.w+36, p.h+58)
	fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="14" font-weight="700" fill="#334155">%s</text>`, p.x-8, p.y-14, title)
	if ground {
		x0, y := p.mapPoint(-1.65, 0)
		x1, _ := p.mapPoint(1.25, 0)
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#94a3b8" stroke-width="1.4" stroke-dasharray="6 5"/>`, x0, y, x1, y)
		fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="10" fill="#64748b">ground y=0</text>`, x0+4, y-5)
	}
}

func drawMesh(b *strings.Builder, r *hierbox.Runner, res *hierbox.PoseResult, p plot, side bool) {
	b.WriteString(`<g stroke="#64748b" stroke-width="0.75" stroke-opacity="0.58" fill="none">`)
	for _, e := range r.Mesh.Edges {
		a := res.Vertices[e[0]]
		c := res.Vertices[e[1]]
		var x1, y1, x2, y2 float64
		if side {
			x1, y1 = p.mapPoint(a.X, a.Y)
			x2, y2 = p.mapPoint(c.X, c.Y)
		} else {
			x1, y1 = p.mapPoint(a.X, a.Z)
			x2, y2 = p.mapPoint(c.X, c.Z)
		}
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, x1, y1, x2, y2)
	}
	b.WriteString(`</g>`)
}

func drawBoxes(b *strings.Builder, r *hierbox.Runner, p plot, side bool) {
	for _, id := range r.Skeleton.Order {
		n := r.Hierarchy.Nodes[id]
		box := n.World
		var x, y, w, h float64
		if side {
			x, y, w, h = p.rect(box.Min.X, box.Min.Y, box.Max.X, box.Max.Y)
		} else {
			x, y, w, h = p.rect(box.Min.X, box.Min.Z, box.Max.X, box.Max.Z)
		}
		color := depthColor(n.Depth)
		width := 1.0 + float64(4-n.Depth)*0.35
		if width < 0.8 {
			width = 0.8
		}
		fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="2" fill="%s" fill-opacity="0.035" stroke="%s" stroke-width="%.2f"><title>%s depth=%d volume=%.4f m³</title></rect>`,
			x, y, w, h, color, color, width, n.Name, n.Depth, box.Volume())
		if n.Depth <= 1 && w > 42 {
			label := n.Name
			fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="10" font-weight="700" fill="%s" stroke="#fff" stroke-width="3" paint-order="stroke">%s</text>`,
				x+4, y+12, color, label)
		}
	}
}

func drawNaiveBox(b *strings.Builder, res *hierbox.PoseResult, p plot, side bool) {
	box := res.RootOnly.Box
	var x, y, w, h float64
	if side {
		x, y, w, h = p.rect(box.Min.X, box.Min.Y, box.Max.X, box.Max.Y)
	} else {
		x, y, w, h = p.rect(box.Min.X, box.Min.Z, box.Max.X, box.Max.Z)
	}
	fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="none" stroke="#dc2626" stroke-width="2.2" stroke-dasharray="9 5"><title>root-only box; violations=%d</title></rect>`,
		x, y, w, h, res.RootOnly.Violations)
	if w > 120 {
		fmt.Fprintf(b, `<text x="%.1f" y="%.1f" font-size="11" font-weight="700" fill="#b91c1c">root-only AABB</text>`, x+8, y+h-8)
	}
}

func drawNaiveLeaks(b *strings.Builder, res *hierbox.PoseResult, p plot, side bool) {
	b.WriteString(`<g fill="#dc2626" fill-opacity="0.82">`)
	for _, vx := range res.Vertices {
		if res.RootOnly.Box.ContainsPoint(vx, 1e-7) {
			continue
		}
		var x, y float64
		if side {
			x, y = p.mapPoint(vx.X, vx.Y)
		} else {
			x, y = p.mapPoint(vx.X, vx.Z)
		}
		fmt.Fprintf(b, `<circle cx="%.1f" cy="%.1f" r="1.7"/>`, x, y)
	}
	b.WriteString(`</g>`)
}

func drawLegend(b *strings.Builder) {
	x := 42.0
	y := 448.0
	items := []struct {
		color, label string
		dash         bool
	}{
		{"#ea580c", "depth 0 / root", false},
		{"#d97706", "depth 1", false},
		{"#16a34a", "depth 2", false},
		{"#2563eb", "depth 3+", false},
		{"#dc2626", "root-only AABB", true},
	}
	for i, it := range items {
		xx := x + float64(i)*205
		dash := ""
		if it.dash {
			dash = ` stroke-dasharray="6 4"`
		}
		fmt.Fprintf(b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="3"%s/><text x="%.1f" y="%.1f" font-size="12" fill="#334155">%s</text>`,
			xx, y-4, xx+28, y-4, it.color, dash, xx+35, y, it.label)
	}
}
