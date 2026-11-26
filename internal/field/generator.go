package field

import (
	"math"

	"glyphsdf/internal/geom"
)

type Edge struct {
	Seg  geom.Segment
	Mask Mask
}

type Field struct {
	W, H    int
	Range   float64
	SDF     []float64
	MSDF    [3][]float64
	Edges   []Edge
	Corners []geom.V
	Shape   geom.Shape
	rules   []cornerRule
}

type cornerRule struct {
	p        geom.V
	prev     geom.Segment
	next     geom.Segment
	maskPrev Mask
	maskNext Mask
	convex   bool
}

type Options struct {
	W, H          int
	Range         float64
	CornerAngle   float64
	ClampDistance bool
}

func DefaultOptions(w, h int) Options {
	return Options{W: w, H: h, Range: 6, CornerAngle: 0.42, ClampDistance: true}
}

func Generate(shape geom.Shape, opt Options) *Field {
	if opt.W <= 0 || opt.H <= 0 {
		opt.W, opt.H = 64, 64
	}
	if opt.Range <= 0 {
		opt.Range = 6
	}
	if opt.CornerAngle <= 0 {
		opt.CornerAngle = 0.42
	}
	shape = OrientShape(shape)
	f := &Field{
		W: opt.W, H: opt.H, Range: opt.Range, Shape: shape,
		SDF: make([]float64, opt.W*opt.H),
	}
	for ch := range f.MSDF {
		f.MSDF[ch] = make([]float64, opt.W*opt.H)
	}

	for _, contour := range shape.Contours {
		pts := contour.Points
		if len(pts) < 3 {
			continue
		}
		masks, sharp := ColorContour(pts, opt.CornerAngle)
		n := len(pts)
		for i := 0; i < n; i++ {
			seg := geom.Segment{A: pts[i], B: pts[(i+1)%n]}
			f.Edges = append(f.Edges, Edge{Seg: seg, Mask: masks[i]})
			if sharp[i] {
				f.Corners = append(f.Corners, pts[i])
				prevIdx := (i - 1 + n) % n
				in := pts[i].Sub(pts[prevIdx])
				out := pts[(i+1)%n].Sub(pts[i])
				f.rules = append(f.rules, cornerRule{
					p:        pts[i],
					prev:     geom.Segment{A: pts[prevIdx], B: pts[i]},
					next:     seg,
					maskPrev: masks[prevIdx],
					maskNext: masks[i],
					convex:   in.Cross(out) > 0,
				})
			}
		}
	}

	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			idx := y*f.W + x
			p := geom.V{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			f.generateAt(idx, p, opt.ClampDistance)
		}
	}
	f.applyCornerRules()
	return f
}

func (f *Field) generateAt(idx int, p geom.V, clamp bool) {
	best := math.Inf(1)
	type candidate struct {
		trueDist float64
		pseudo   float64
	}
	var channelBest [3]candidate
	for ch := range channelBest {
		channelBest[ch].trueDist = math.Inf(1)
		channelBest[ch].pseudo = math.Inf(1)
	}

	for _, e := range f.Edges {
		d, _ := geom.PointSegment(p, e.Seg)
		if d < best {
			best = d
		}
		pd := geom.PseudoLineDistance(p, e.Seg)
		for ch := 0; ch < 3; ch++ {
			if e.Mask&(1<<uint(ch)) == 0 {
				continue
			}
			if d < channelBest[ch].trueDist {
				channelBest[ch] = candidate{trueDist: d, pseudo: pd}
			}
		}
	}

	inside := geom.Inside(p, f.Shape.Contours)
	if inside {
		f.SDF[idx] = best
	} else {
		f.SDF[idx] = -best
	}
	if clamp {
		f.SDF[idx] = clampDist(f.SDF[idx], f.Range)
	}

	for ch := 0; ch < 3; ch++ {
		v := -f.Range
		if !math.IsInf(channelBest[ch].trueDist, 1) {
			v = channelBest[ch].pseudo
		}
		if clamp {
			v = clampDist(v, f.Range)
		}
		f.MSDF[ch][idx] = v
	}
}

func clampDist(v, r float64) float64 {
	return math.Max(-r, math.Min(r, v))
}

func (f *Field) applyCornerRules() {
	nearest := make([]float64, f.W*f.H)
	for i := range nearest {
		nearest[i] = math.Inf(1)
	}
	const radius = 2.35
	for _, rule := range f.rules {
		x0 := int(math.Max(0, math.Floor(rule.p.X-radius)))
		x1 := math.Min(float64(f.W-1), math.Ceil(rule.p.X+radius))
		y0 := int(math.Max(0, math.Floor(rule.p.Y-radius)))
		y1 := math.Min(float64(f.H-1), math.Ceil(rule.p.Y+radius))
		common := rule.maskPrev & rule.maskNext
		commonCh := -1
		for ch := 0; ch < 3; ch++ {
			if common&(1<<uint(ch)) != 0 {
				if commonCh >= 0 {
					commonCh = -2
					break
				}
				commonCh = ch
			}
		}
		if commonCh < 0 {
			continue
		}
		for y := int(y0); y <= int(y1); y++ {
			for x := int(x0); x <= int(x1); x++ {
				idx := y*f.W + x
				p := geom.V{X: float64(x) + 0.5, Y: float64(y) + 0.5}
				dCorner := p.Dist(rule.p)
				if dCorner > radius || dCorner > nearest[idx]+0.35 {
					continue
				}
				nearest[idx] = dCorner
				pdPrev := clampDist(geom.PseudoLineDistance(p, rule.prev), f.Range)
				pdNext := clampDist(geom.PseudoLineDistance(p, rule.next), f.Range)
				for ch := 0; ch < 3; ch++ {
					bit := Mask(1 << uint(ch))
					if rule.maskPrev&bit != 0 && rule.maskNext&bit == 0 {
						f.MSDF[ch][idx] = pdPrev
					}
					if rule.maskNext&bit != 0 && rule.maskPrev&bit == 0 {
						f.MSDF[ch][idx] = pdNext
					}
				}
				shared := pdPrev
				if rule.convex {
					shared = math.Min(pdPrev, pdNext)
				} else {
					shared = math.Max(pdPrev, pdNext)
				}
				f.MSDF[commonCh][idx] = shared
			}
		}
	}
}

func Median3(a, b, c float64) float64 {
	if a > b {
		a, b = b, a
	}
	if b > c {
		b = c
	}
	if a > b {
		a, b = b, a
	}
	return b
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := math.Max(0, math.Min(1, (x-edge0)/(edge1-edge0)))
	return t * t * (3 - 2*t)
}
