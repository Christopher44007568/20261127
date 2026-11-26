package report

import (
	"image"
	"image/color"
	"math"

	"featurelines/internal/gfx"
	"featurelines/internal/pipeline"
)

// ErrorPanel renders the endpoint/error diagnostic figure for a case.
func ErrorPanel(o *pipeline.Output) *gfx.Scene {
	s := newScene(Palette.BG)
	header(s, "3 / ENDPOINT ERROR MAP", o.Case.Title, "", Palette.Red)
	switch o.Case.Kind {
	case "hole":
		holeDiagram(s, o)
	case "sharp":
		sharpDiagram(s, o)
	default:
		tangentDiagram(s, o)
	}
	return s
}

// mapper is an equal-aspect 2D world->pixel transform.
type mapper struct {
	cx, cy float64
	sx, sy float64
	wcx    float64
	wcy    float64
}

func newMapper(r image.Rectangle, xmin, xmax, ymin, ymax float64, equal bool) *mapper {
	wcx, wcy := (xmin+xmax)/2, (ymin+ymax)/2
	sx := float64(r.Dx()) / (xmax - xmin)
	sy := float64(r.Dy()) / (ymax - ymin)
	if equal {
		k := math.Min(sx, sy)
		sx, sy = k, k
	}
	return &mapper{
		cx: float64(r.Min.X) + float64(r.Dx())/2,
		cy: float64(r.Min.Y) + float64(r.Dy())/2,
		sx: sx, sy: sy, wcx: wcx, wcy: wcy,
	}
}

func (mp *mapper) X(x float64) int { return int(math.Round(mp.cx + (x-mp.wcx)*mp.sx)) }
func (mp *mapper) Y(y float64) int { return int(math.Round(mp.cy - (y-mp.wcy)*mp.sy)) }

func card(s *gfx.Scene, r image.Rectangle, title, badge string, accent color.RGBA) {
	fillRect(s.Img, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, color.RGBA{255, 255, 255, 255})
	strokeRect(s, r, Palette.Grid)
	fillRect(s.Img, r.Min.X, r.Min.Y, r.Min.X+8, r.Max.Y, accent)
	gfx.DrawText(s.Img, title, r.Min.X+22, r.Min.Y+18, 2, Palette.Ink)
	if badge != "" {
		w := gfx.TextWidth(badge, 2) + 24
		fillRect(s.Img, r.Max.X-w-16, r.Min.Y+12, r.Max.X-16, r.Min.Y+40, lighten(accent, 0.82))
		gfx.DrawText(s.Img, badge, r.Max.X-w-4, r.Min.Y+19, 2, accent)
	}
}

func strokeRect(s *gfx.Scene, r image.Rectangle, c color.RGBA) {
	fillRect(s.Img, r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+2, c)
	fillRect(s.Img, r.Min.X, r.Max.Y-2, r.Max.X, r.Max.Y, c)
	fillRect(s.Img, r.Min.X, r.Min.Y, r.Min.X+2, r.Max.Y, c)
	fillRect(s.Img, r.Max.X-2, r.Min.Y, r.Max.X, r.Max.Y, c)
}

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func mapLine(s *gfx.Scene, mp *mapper, a, b Vec3, c color.RGBA, thick int) {
	s.Line2D(mp.X(a.X), mp.Y(a.Y), mp.X(b.X), mp.Y(b.Y), c, thick)
}

func mapDashed(s *gfx.Scene, mp *mapper, a, b Vec3, c color.RGBA, thick int, dash, gap float64) {
	x0, y0 := float64(mp.X(a.X)), float64(mp.Y(a.Y))
	x1, y1 := float64(mp.X(b.X)), float64(mp.Y(b.Y))
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l < 1 {
		return
	}
	for d := 0.0; d < l; d += dash + gap {
		d2 := math.Min(d+dash, l)
		s.Line2D(int(x0+dx*d/l), int(y0+dy*d/l), int(x0+dx*d2/l), int(y0+dy*d2/l), c, thick)
	}
}

func mapCircle(s *gfx.Scene, mp *mapper, cx, cy, r float64, c color.RGBA, thick int, dashed bool) {
	const n = 160
	for i := 0; i < n; i++ {
		a0 := 2 * math.Pi * float64(i) / n
		a1 := 2 * math.Pi * float64(i+1) / n
		p0 := Vec3{X: cx + r*math.Cos(a0), Y: cy + r*math.Sin(a0), Z: 0}
		p1 := Vec3{X: cx + r*math.Cos(a1), Y: cy + r*math.Sin(a1), Z: 0}
		if dashed {
			if i%8 < 4 {
				mapLine(s, mp, p0, p1, c, thick)
			}
		} else {
			mapLine(s, mp, p0, p1, c, thick)
		}
	}
}

func mapChain(s *gfx.Scene, mp *mapper, pts []Vec3, closed bool, c color.RGBA, thick int) {
	for i := 1; i < len(pts); i++ {
		mapLine(s, mp, pts[i-1], pts[i], c, thick)
	}
	if closed && len(pts) > 2 {
		mapLine(s, mp, pts[len(pts)-1], pts[0], c, thick)
	}
}
