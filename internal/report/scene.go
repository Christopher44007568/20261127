package report

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"

	"skinlab/internal/skin"
)

const (
	sceneWidth  = 1000
	sceneHeight = 420
	panelWidth  = 465
	sceneTop    = 52
	sceneBottom = 24
)

type projectedPoint struct {
	x, y, depth float64
}

type sceneFit struct {
	minX, minY float64
	scale      float64
}

type sceneFace struct {
	depth float64
	svg   string
}

func projectPoint(point skin.Vec3) projectedPoint {
	return projectedPoint{
		x:     0.86*point.X - 0.50*point.Z,
		y:     -0.92*point.Y + 0.20*point.Z,
		depth: 0.34*point.X + 0.52*point.Y + 0.78*point.Z,
	}
}

// RenderMeshComparisonSVG draws bind-pose ghosts and both skinned surfaces with
// one shared camera and color scale.
func RenderMeshComparisonSVG(mesh skin.Mesh, bind []skin.Vec3, comparison skin.Comparison) string {
	all := append(append(append([]skin.Vec3{}, bind...), comparison.LBS...), comparison.DQ...)
	fit := fitScene(all)

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, `<svg class="scene" viewBox="0 0 %d %d" role="img" aria-label="LBS 与双四元数蒙皮网格对比">`, sceneWidth, sceneHeight)
	buffer.WriteString(`<rect x="0" y="0" width="1000" height="420" rx="18" fill="#111820"/>`)

	left := 18.0
	right := 517.0
	drawScenePanel(&buffer, mesh, bind, comparison.LBS, comparison.Quads, "LBS", left, fit, comparison.MinLBSRatio, comparison.MaxLBSRatio, true)
	drawScenePanel(&buffer, mesh, bind, comparison.DQ, comparison.Quads, "DUAL QUATERNION", right, fit, comparison.MinDQRatio, comparison.MaxDQRatio, false)

	buffer.WriteString(`<g transform="translate(34 402)">`)
	writeHeatLegend(&buffer, 0, 0, 190, 8)
	buffer.WriteString(`<text x="0" y="-7" fill="#91a0ae" font-size="10">压缩</text><text x="95" y="-7" text-anchor="middle" fill="#c4ccd3" font-size="10">绑定面积 100%</text><text x="190" y="-7" text-anchor="end" fill="#91a0ae" font-size="10">鼓包</text></g>`)
	buffer.WriteString(`</svg>`)
	return buffer.String()
}

func fitScene(points []skin.Vec3) sceneFit {
	fit := sceneFit{minX: math.Inf(1), minY: math.Inf(1)}
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, point := range points {
		projected := projectPoint(point)
		fit.minX = math.Min(fit.minX, projected.x)
		fit.minY = math.Min(fit.minY, projected.y)
		maxX = math.Max(maxX, projected.x)
		maxY = math.Max(maxY, projected.y)
	}

	const horizontalPadding = 30.0
	const verticalPadding = 18.0
	width := panelWidth - 2*horizontalPadding
	height := sceneHeight - sceneTop - sceneBottom - 2*verticalPadding
	fit.scale = math.Min(width/math.Max(maxX-fit.minX, 0.01), height/math.Max(maxY-fit.minY, 0.01))
	fit.minX -= horizontalPadding / fit.scale
	fit.minY -= verticalPadding / fit.scale
	return fit
}

func (fit sceneFit) screen(point skin.Vec3, panelLeft float64) (float64, float64) {
	projected := projectPoint(point)
	return panelLeft + (projected.x-fit.minX)*fit.scale,
		sceneTop + (projected.y-fit.minY)*fit.scale
}

func drawScenePanel(
	buffer *bytes.Buffer,
	mesh skin.Mesh,
	bind, deformed []skin.Vec3,
	quads []skin.QuadSample,
	title string,
	left float64,
	fit sceneFit,
	minRatio, maxRatio float64,
	showBind bool,
) {
	fmt.Fprintf(buffer, `<rect x="%.1f" y="18" width="%d" height="%d" rx="14" fill="#17212b" stroke="#283646"/>`, left, panelWidth, sceneHeight-28)
	fmt.Fprintf(buffer, `<text x="%.1f" y="39" fill="#f1f4f6" font-size="13" font-weight="700" letter-spacing="1.2">%s</text>`, left+16, title)
	fmt.Fprintf(buffer, `<text x="%.1f" y="39" text-anchor="end" fill="#9caab7" font-size="11">截面 %.1f%%–%.1f%%</text>`, left+panelWidth-16, minRatio*100, maxRatio*100)

	if showBind {
		drawBindGhost(buffer, mesh, bind, left, fit)
	}

	faces := make([]sceneFace, 0, len(quads))
	for _, quad := range quads {
		visibleQuad := mesh.Quads[quad.Ring*mesh.Segments+quad.Segment]
		a := deformed[visibleQuad.A]
		b := deformed[visibleQuad.B]
		c := deformed[visibleQuad.C]
		d := deformed[visibleQuad.D]
		x1, y1 := fit.screen(a, left)
		x2, y2 := fit.screen(b, left)
		x3, y3 := fit.screen(c, left)
		x4, y4 := fit.screen(d, left)
		ratio := quad.DQRatio
		if title == "LBS" {
			ratio = quad.LBSRatio
		}
		color := heatColor(ratio)
		faces = append(faces, sceneFace{
			depth: (projectPoint(a).depth + projectPoint(b).depth + projectPoint(c).depth + projectPoint(d).depth) / 4,
			svg: fmt.Sprintf(
				`<polygon points="%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f" fill="%s" stroke="%s" stroke-width=".7" stroke-linejoin="round"/>`,
				x1, y1, x2, y2, x3, y3, x4, y4, color, color,
			),
		})
	}
	sort.SliceStable(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, face := range faces {
		buffer.WriteString(face.svg)
	}

	centerX, centerY := fit.screen(skin.Vec3{}, left)
	fmt.Fprintf(buffer, `<circle cx="%.2f" cy="%.2f" r="4" fill="#f6c85f" stroke="#111820" stroke-width="1.5"/>`, centerX, centerY)
	fmt.Fprintf(buffer, `<text x="%.2f" y="%.2f" fill="#f6c85f" font-size="10">肘部</text>`, centerX+8, centerY+3)
}

func drawBindGhost(buffer *bytes.Buffer, mesh skin.Mesh, bind []skin.Vec3, left float64, fit sceneFit) {
	buffer.WriteString(`<g fill="none" stroke="#8fa1b2" stroke-width=".75" opacity=".28">`)
	for ring := 0; ring < mesh.RingCount; ring += 7 {
		points := make([]string, 0, mesh.Segments+1)
		for segment := 0; segment <= mesh.Segments; segment++ {
			index := mesh.VertexIndex(ring, segment)
			x, y := fit.screen(bind[index], left)
			points = append(points, fmt.Sprintf("%.2f,%.2f", x, y))
		}
		fmt.Fprintf(buffer, `<polyline points="%s"/>`, strings.Join(points, " "))
	}
	for segment := 0; segment < mesh.Segments; segment += 8 {
		for ring := 0; ring < mesh.RingCount-1; ring++ {
			a := mesh.VertexIndex(ring, segment)
			b := mesh.VertexIndex(ring+1, segment)
			x1, y1 := fit.screen(bind[a], left)
			x2, y2 := fit.screen(bind[b], left)
			fmt.Fprintf(buffer, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f"/>`, x1, y1, x2, y2)
		}
	}
	buffer.WriteString(`</g>`)
}
