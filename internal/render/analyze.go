package render

import (
	"image"
	"image/color"
	"math"

	"glyphsdf/internal/field"
	"glyphsdf/internal/geom"
)

type CornerMetric struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	AngleDeg   float64 `json:"angleDeg"`
	Convex     bool    `json:"convex"`
	SDFLossPx  float64 `json:"sdfLossPx"`
	MSDFLossPx float64 `json:"msdfLossPx"`
	SDFBroken  bool    `json:"sdfBroken"`
	MSDFBroken bool    `json:"msdfBroken"`
}

type Report struct {
	Corners         []CornerMetric `json:"corners"`
	SDFBrokenCount  int            `json:"sdfBrokenCount"`
	MSDFBrokenCount int            `json:"msdfBrokenCount"`
	SDFRedPixels    int            `json:"sdfRedPixels"`
	MSDFRedPixels   int            `json:"msdfRedPixels"`
}

type cornerGeom struct {
	p       geom.V
	tipDir  geom.V
	fillDir geom.V
	angle   float64
	convex  bool
}

func Analyze(f *field.Field, scale float64) Report {
	var rep Report
	corners := collectCornerGeometry(f)
	for _, c := range corners {
		m := CornerMetric{
			X: c.p.X * scale, Y: c.p.Y * scale,
			AngleDeg: c.angle * 180 / math.Pi, Convex: c.convex,
		}
		m.SDFLossPx = tipLoss(f, MethodSDF, c, scale)
		m.MSDFLossPx = tipLoss(f, MethodMSDF, c, scale)
		m.SDFBroken = m.SDFLossPx >= 4.0
		m.MSDFBroken = m.MSDFLossPx >= 4.0
		if m.SDFBroken {
			rep.SDFBrokenCount++
		}
		if m.MSDFBroken {
			rep.MSDFBrokenCount++
		}
		rep.Corners = append(rep.Corners, m)
	}
	return rep
}

func collectCornerGeometry(f *field.Field) []cornerGeom {
	var out []cornerGeom
	for _, c := range f.Shape.Contours {
		pts := c.Points
		n := len(pts)
		if n < 3 {
			continue
		}
		for i := 0; i < n; i++ {
			p := pts[i]
			prev := pts[(i-1+n)%n]
			next := pts[(i+1)%n]
			in := p.Sub(prev)
			outv := next.Sub(p)
			if in.Len() < 1e-8 || outv.Len() < 1e-8 {
				continue
			}
			inN := in.Norm()
			outN := outv.Norm()
			dot := math.Max(-1, math.Min(1, inN.Dot(outN)))
			turn := math.Acos(dot)
			if turn < 0.42 {
				continue
			}
			toPrev := prev.Sub(p).Norm()
			toNext := next.Sub(p).Norm()
			bisect := toPrev.Add(toNext)
			if bisect.Len() < 1e-6 {
				bisect = inN.Mul(-1)
			}
			bisect = bisect.Norm()
			convex := in.Cross(outv) > 0
			fillDir := bisect
			if !convex {
				fillDir = bisect.Mul(-1)
			}
			out = append(out, cornerGeom{p: p, tipDir: bisect, fillDir: fillDir, angle: turn, convex: convex})
		}
	}
	return out
}

func tipLoss(f *field.Field, method Method, c cornerGeom, scale float64) float64 {
	// Search outward from the tip in output pixels. A correct sharp field
	// becomes positive almost immediately; a rounded SDF remains negative
	// through a measurable tip cap.
	lastBad := 0.0
	goodStreak := 0
	probe := c.p.Add(c.tipDir.Mul(0.45))
	wantInside := geom.Inside(probe, f.Shape.Contours)
	for rPx := 0.75; rPx <= 14; rPx += 0.5 {
		rTex := rPx / scale
		q := c.p.Add(c.tipDir.Mul(rTex))
		inside := geom.Inside(q, f.Shape.Contours)
		if inside != wantInside {
			goodStreak++
			if goodStreak >= 4 {
				break
			}
			continue
		}
		var d float64
		if method == MethodMSDF {
			d = f.SampleMSDF(q.X, q.Y)
		} else {
			d = f.SampleSDF(q.X, q.Y)
		}
		aa := 0.45 / scale
		bad := d <= aa
		if !wantInside {
			bad = d >= -aa
		}
		if bad {
			lastBad = rPx
			goodStreak = 0
		} else {
			goodStreak++
			if goodStreak >= 4 {
				break
			}
		}
	}
	return lastBad
}

var breakRed = color.RGBA{R: 235, G: 32, B: 42, A: 255}

// PaintBreaks tints reference-inside pixels whose reconstructed coverage is
// missing near sharp corners. It returns the number of red pixels.
func PaintBreaks(dst *image.RGBA, cov *image.Gray, f *field.Field, scale float64, rep *Report, method Method) int {
	corners := collectCornerGeometry(f)
	radiusPx := int(math.Max(13, scale*1.35+8))
	count := 0
	b := cov.Bounds()
	for _, c := range corners {
		cx := int(math.Round(c.p.X * scale))
		cy := int(math.Round(c.p.Y * scale))
		broken := false
		for _, m := range rep.Corners {
			if math.Hypot(m.X-float64(cx), m.Y-float64(cy)) < 2 && ((method == MethodSDF && m.SDFBroken) || (method == MethodMSDF && m.MSDFBroken)) {
				broken = true
			}
		}
		if !broken {
			StrokeCircle(dst, cx, cy, 5, 1, color.RGBA{R: 30, G: 170, B: 90, A: 255})
			continue
		}
		for y := cy - radiusPx; y <= cy+radiusPx; y++ {
			for x := cx - radiusPx; x <= cx+radiusPx; x++ {
				if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
					continue
				}
				dx := float64(x) + 0.5 - c.p.X*scale
				dy := float64(y) + 0.5 - c.p.Y*scale
				if dx*dx+dy*dy > float64(radiusPx*radiusPx) {
					continue
				}
				tx := (float64(x) + 0.5) / scale
				ty := (float64(y) + 0.5) / scale
				q := geom.V{X: tx, Y: ty}
				referenceInside := geom.Inside(q, f.Shape.Contours)
				a := cov.GrayAt(x, y).Y
				if referenceInside {
					if a >= 76 {
						continue
					}
					strong := 1.0 - float64(a)/76.0
					blendRedPixel(dst, x, y, 0.42+0.5*strong)
					count++
				} else {
					if a <= 214 {
						continue
					}
					strong := (float64(a) - 214.0) / 41.0
					blendRedPixel(dst, x, y, 0.45+0.45*strong)
					count++
				}
			}
		}
		StrokeCircle(dst, cx, cy, int(math.Max(7, scale*0.9)), 3, breakRed)
		DrawCross(dst, cx, cy, int(math.Max(4, scale*0.45)), 2, breakRed)
	}
	if method == MethodSDF {
		rep.SDFRedPixels = count
	} else {
		rep.MSDFRedPixels = count
	}
	return count
}

func blendRedPixel(dst *image.RGBA, x, y int, alpha float64) {
	old := dst.RGBAAt(x, y)
	c := blend(old, breakRed, alpha)
	dst.SetRGBA(x, y, c)
}
