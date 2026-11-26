package report

import (
	"fmt"
	"image/color"

	"featurelines/internal/feature"
	"featurelines/internal/gfx"
	"featurelines/internal/linalg"
	"featurelines/internal/pipeline"
)

// Vec3 alias.
type Vec3 = linalg.Vec3

func cameraFor(id string) (yaw, pitch float64) {
	switch id {
	case "hole":
		return 0.72, 0.58
	case "sharp":
		return 0.95, 0.5
	default:
		return 1.42, 0.24
	}
}

func materialFor(id string) color.RGBA {
	switch id {
	case "hole":
		return color.RGBA{120, 156, 196, 255}
	case "sharp":
		return color.RGBA{202, 172, 112, 255}
	default:
		return color.RGBA{112, 182, 170, 255}
	}
}

func lighten(c color.RGBA, t float64) color.RGBA {
	mix := func(v uint8) uint8 { return uint8(float64(v) + (245-float64(v))*t) }
	return color.RGBA{mix(c.R), mix(c.G), mix(c.B), 255}
}

func fitScene(o *pipeline.Output, bg color.RGBA) *gfx.Scene {
	s := newScene(bg)
	lo, hi := o.Case.Mesh.Bounds()
	yaw, pitch := cameraFor(o.Case.ID)
	s.Fit(lo, hi, yaw, pitch, 92, 52)
	return s
}

// ModelPanel renders the shaded original model.
func ModelPanel(o *pipeline.Output) *gfx.Scene {
	s := fitScene(o, Palette.BG)
	s.DrawMesh(o.Case.Mesh, materialFor(o.Case.ID))
	header(s, "1 / ORIGINAL MODEL", o.Case.Title, "", Palette.Gray)
	footer(s, fmt.Sprintf("%s   |   %d vertices  %d triangles",
		o.Case.Subtitle, len(o.Case.Mesh.V), len(o.Case.Mesh.T)),
		color.RGBA{255, 255, 255, 235})
	return s
}

func chainPoints(o *pipeline.Output, ch *feature.Chain) []Vec3 {
	pts := make([]Vec3, len(ch.Vertices))
	for i, v := range ch.Vertices {
		pts[i] = o.Case.Mesh.V[v]
	}
	return pts
}

// OverlayPanel renders the model plus robust feature lines and markers.
func OverlayPanel(o *pipeline.Output) *gfx.Scene {
	s := fitScene(o, Palette.BG)
	s.DrawMesh(o.Case.Mesh, lighten(materialFor(o.Case.ID), 0.28))

	// Topological contours in neutral gray.
	for _, ch := range o.Robust.Contours {
		if len(ch.Vertices) < 2 {
			continue
		}
		s.Polyline3D(chainPoints(o, ch), ch.Closed, Palette.Gray, 1.6, false)
	}

	// Robust crease/feature chains in strong blue.
	for _, ch := range o.Robust.Creases {
		if len(ch.Vertices) < 2 {
			continue
		}
		s.Polyline3D(chainPoints(o, ch), ch.Closed, Palette.Blue, 3.0, true)
		for _, g := range ch.Gaps {
			if len(g.Points) >= 2 {
				s.Polyline3D(g.Points, false, Palette.Green, 4.2, false)
			}
		}
	}

	// Highlight the injected single-vertex failure.
	if o.Case.FailVertex >= 0 && len(o.Case.Failed) > 0 {
		p := o.Case.Mesh.V[o.Case.FailVertex]
		s.Marker3D(p, gfx.MarkerStar, 10, Palette.Amber)
	}

	// True open-feature endpoints.
	if o.Case.Kind == "sharp" {
		for _, mk := range o.Metrics.Endpoints {
			p := Vec3{X: mk.X, Y: mk.Y, Z: mk.Z}
			s.Marker3D(p, gfx.MarkerRing, 8, Palette.Green)
		}
	}

	header(s, "2 / FEATURE-LINE OVERLAY", o.Case.Title, "", Palette.Blue)
	drawOverlayLegend(s, o)
	footer(s, overlayFooter(o), color.RGBA{255, 255, 255, 235})
	return s
}

func overlayFooter(o *pipeline.Output) string {
	m := o.Metrics
	switch o.Case.Kind {
	case "hole":
		return fmt.Sprintf("1 CLOSED LOOP  |  COVERAGE %.1f%%  |  FALSE ENDS %d  |  BRIDGES %d",
			m.Coverage*100, m.EndpointCount, m.BridgeCount)
	case "sharp":
		return fmt.Sprintf("ONE OPEN CREASE  |  ENDS %d  |  MAX ENDPOINT ERR %.2f MM  |  BRIDGED %d",
			m.EndpointCount, m.MaxEndpointError, m.BridgeCount)
	default:
		return fmt.Sprintf("NORMAL-ANGLE VETO  |  FALSE CHAINS %d  |  FALSE LEN %.2f MM",
			m.RobustChains, m.FalseFeatureLength)
	}
}

type legendItem struct {
	col   color.RGBA
	label string
	kind  int // 0 line, 1 star, 2 ring, 3 dashed
}

func drawOverlayLegend(s *gfx.Scene, o *pipeline.Output) {
	items := []legendItem{
		{Palette.Blue, "traced feature line", 0},
		{Palette.Green, "gap bridge", 3},
		{Palette.Amber, "failed vertex", 1},
	}
	if o.Case.Kind == "sharp" {
		items = append(items, legendItem{Palette.Green, "true endpoint", 2})
	}
	if len(o.Robust.Contours) > 0 {
		items = append(items, legendItem{Palette.Gray, "boundary contour", 0})
	}
	x0, y0 := PW-300, 96
	h := 30*len(items) + 18
	fillRect(s.Img, x0, y0, x0+282, y0+h, color.RGBA{255, 255, 255, 235})
	for i, it := range items {
		y := y0 + 16 + i*30
		switch it.kind {
		case 0:
			s.Line2D(x0+16, y, x0+48, y, it.col, 4)
		case 3:
			for dx := 0; dx < 36; dx += 10 {
				s.Line2D(x0+16+dx, y, x0+22+dx, y, it.col, 4)
			}
		case 1:
			s.Marker2D(x0+32, y, gfx.MarkerStar, 7, it.col)
		case 2:
			s.Marker2D(x0+32, y, gfx.MarkerRing, 6, it.col)
		}
		gfx.DrawText(s.Img, it.label, x0+58, y-8, 2, Palette.Ink)
	}
}
