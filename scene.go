package main

import (
	"fmt"
	"image/color"
	"math"
)

// Stop is one gradient color stop.
type Stop struct {
	Off float64
	C   color.RGBA
}

// Gradient is an editable paint server. Coordinates are userSpaceOnUse.
type Gradient struct {
	ID     string
	Radial bool
	X1, Y1 float64
	X2, Y2 float64 // linear
	CX, CY float64 // radial center
	R      float64 // radial radius
	Stops  []Stop
}

// At evaluates the gradient paint at point q.
func (g *Gradient) At(q Pt) color.RGBA {
	var t float64
	if g.Radial {
		dx, dy := q.X-g.CX, q.Y-g.CY
		t = math.Hypot(dx, dy) / g.R
	} else {
		dx, dy := g.X2-g.X1, g.Y2-g.Y1
		l2 := dx*dx + dy*dy
		if l2 == 0 {
			t = 0
		} else {
			t = ((q.X-g.X1)*dx + (q.Y-g.Y1)*dy) / l2
		}
	}
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return sampleStops(g.Stops, t)
}

func sampleStops(stops []Stop, t float64) color.RGBA {
	if len(stops) == 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	if t <= stops[0].Off {
		return stops[0].C
	}
	for i := 0; i+1 < len(stops); i++ {
		a, b := stops[i], stops[i+1]
		if t >= a.Off && t <= b.Off {
			k := 0.0
			if b.Off > a.Off {
				k = (t - a.Off) / (b.Off - a.Off)
			}
			return lerpRGBA(a.C, b.C, k)
		}
	}
	return stops[len(stops)-1].C
}

func lerpRGBA(a, b color.RGBA, k float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*k),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*k),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*k),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*k),
	}
}

// Paint is solid color or gradient reference.
type Paint struct {
	Solid color.RGBA
	Grad  *Gradient
	Ref   string // gradient id as exported/referenced
}

func solid(hex string) *Paint {
	c := parseHex(hex)
	return &Paint{Solid: c}
}

func gradientPaint(g *Gradient) *Paint {
	return &Paint{Grad: g, Ref: g.ID}
}

func (p *Paint) at(q Pt) color.RGBA {
	if p == nil {
		return color.RGBA{}
	}
	if p.Grad != nil {
		return p.Grad.At(q)
	}
	return p.Solid
}

// Clip is an editable clipPath (userSpaceOnUse), union of Paths.
type Clip struct {
	ID    string
	Paths []*Path
}

// Obj is one editable scene object. Every object keeps its original ID through
// SVG export, reload, rasterization and error attribution.
type Obj struct {
	ID       string
	Tag      string // svg-ish tag: rect|circle|ellipse|polygon|path|g
	Note     string // semantic role for reports
	Paths    []*Path
	Fill     *Paint
	Stroke   *Paint
	StrokeW  float64
	ClipID   string
	Children []*Obj
	Parent   *Obj `json:"-"`
	Index    int
	Anchor   Pt // label anchor (top-left of bbox by default)
}

// Scene is the single vector source of truth.
type Scene struct {
	W, H  int
	BG    color.RGBA
	Grads []*Gradient
	Clips []*Clip
	Objs  []*Obj
	All   []*Obj // flattened including children, z order
}

func (s *Scene) gradByID(id string) *Gradient {
	for _, g := range s.Grads {
		if g.ID == id {
			return g
		}
	}
	return nil
}

func (s *Scene) clipByID(id string) *Clip {
	for _, c := range s.Clips {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (s *Scene) indexObjects() {
	s.All = nil
	idx := 0
	var walk func(o *Obj, parent *Obj)
	walk = func(o *Obj, parent *Obj) {
		o.Index = idx
		o.Parent = parent
		idx++
		s.All = append(s.All, o)
		for _, c := range o.Children {
			walk(c, o)
		}
	}
	for _, o := range s.Objs {
		walk(o, nil)
	}
}

// BuildScene constructs the demo vector scene used by every output.
func BuildScene() *Scene {
	s := &Scene{W: 960, H: 600, BG: parseHex("#eef1f7")}

	gLin := &Gradient{
		ID: "grad-linear-bar", X1: 300, Y1: 170, X2: 540, Y2: 240,
		Stops: []Stop{{0, parseHex("#22d3ee")}, {0.55, parseHex("#3b82f6")}, {1, parseHex("#6d28d9")}},
	}
	gRad := &Gradient{
		ID: "grad-radial-sun", Radial: true, CX: 420, CY: 372, R: 96,
		Stops: []Stop{{0, parseHex("#fff7cc")}, {0.45, parseHex("#ffb020")}, {1, parseHex("#e11d48")}},
	}
	gWave := &Gradient{
		ID: "grad-wave-stroke", X1: 648, Y1: 478, X2: 900, Y2: 510,
		Stops: []Stop{{0, parseHex("#f43f5e")}, {0.5, parseHex("#f59e0b")}, {1, parseHex("#10b981")}},
	}
	gRibbon := &Gradient{
		ID: "grad-ribbon-stroke", X1: 70, Y1: 500, X2: 540, Y2: 520,
		Stops: []Stop{{0, parseHex("#fb7185")}, {0.5, parseHex("#d946ef")}, {1, parseHex("#6366f1")}},
	}
	s.Grads = []*Gradient{gLin, gRad, gWave, gRibbon}

	clipWin := &Clip{ID: "clip-circle-window", Paths: []*Path{EllipsePath(796, 268, 118, 118)}}
	s.Clips = []*Clip{clipWin}

	bg := &Obj{ID: "obj-bg", Tag: "rect", Note: "solid background fill",
		Paths: []*Path{RectPath(0, 0, 960, 600)}, Fill: solid("#eef1f7")}

	cardL := &Obj{ID: "obj-card-left", Tag: "rect", Note: "solid fill + stroke",
		Paths: []*Path{RoundRectPath(36, 96, 540, 470, 28)}, Fill: solid("#ffffff"),
		Stroke: solid("#d3dcec"), StrokeW: 3}

	cardR := &Obj{ID: "obj-card-right", Tag: "rect", Note: "clip stage card",
		Paths: []*Path{RoundRectPath(604, 96, 320, 470, 28)}, Fill: solid("#ffffff"),
		Stroke: solid("#d3dcec"), StrokeW: 3}

	star := &Obj{ID: "obj-star-stroke", Tag: "polygon", Note: "fill + thick round stroke",
		Paths: []*Path{PolygonPath(168, 252, 92, 5, 0.42, -math.Pi/2)},
		Fill:  solid("#fbbf24"), Stroke: solid("#92400e"), StrokeW: 9}

	bar := &Obj{ID: "obj-linear-gradient", Tag: "rect", Note: "linear gradient fill + stroke",
		Paths: []*Path{RoundRectPath(296, 168, 252, 74, 18)},
		Fill:  gradientPaint(gLin), Stroke: solid("#1e3a8a"), StrokeW: 4}

	sun := &Obj{ID: "obj-radial-gradient", Tag: "circle", Note: "radial gradient fill + stroke",
		Paths: []*Path{EllipsePath(420, 380, 94, 94)},
		Fill:  gradientPaint(gRad), Stroke: solid("#7f1d1d"), StrokeW: 5}

	ell := &Obj{ID: "obj-ellipse-solid", Tag: "ellipse", Note: "solid ellipse + stroke",
		Paths: []*Path{EllipsePath(176, 452, 108, 44)},
		Fill:  solid("#a5d8ff"), Stroke: solid("#1971c2"), StrokeW: 5}

	ribbon := &Obj{ID: "obj-bezier-ribbon", Tag: "path", Note: "cubic path, gradient stroke only",
		Paths: []*Path{PolylinePath()}, Stroke: gradientPaint(gRibbon), StrokeW: 12}
	rp := &Path{}
	rp.move(70, 520)
	rp.cubic(180, 470, 260, 560, 360, 520)
	rp.cubic(440, 488, 470, 470, 540, 506)
	ribbon.Paths = []*Path{rp}

	clipGroup := &Obj{ID: "obj-clip-group", Tag: "g", Note: "group clipped by clip-circle-window",
		ClipID: "clip-circle-window"}
	// Diagonal stripes + inner disc, all clipped by the group clipPath.
	clipGroup.Children = buildClipChildren()

	ring := &Obj{ID: "obj-clip-ring", Tag: "circle", Note: "clip window outline stroke",
		Paths: []*Path{EllipsePath(796, 268, 118, 118)},
		Fill:  nil, Stroke: solid("#111827"), StrokeW: 9}

	wave := &Obj{ID: "obj-zigzag-stroke", Tag: "path", Note: "polyline stroke with gradient paint, no fill",
		Paths:  []*Path{PolylinePath(648, 510, 676, 478, 704, 510, 732, 478, 760, 510, 788, 478, 816, 510, 844, 478, 872, 510, 900, 482)},
		Stroke: gradientPaint(gWave), StrokeW: 8}

	dots := &Obj{ID: "obj-dots", Tag: "path", Note: "small filled circles group-as-path",
		Fill: solid("#0f172a")}
	dp := &Path{}
	for i := 0; i < 3; i++ {
		cx := 648 + float64(i)*26
		d := EllipsePath(cx, 128, 7, 7)
		dp.Cmds = append(dp.Cmds, d.Cmds...)
	}
	dots.Paths = []*Path{dp}

	s.Objs = []*Obj{bg, cardL, cardR, ell, star, bar, sun, ribbon, clipGroup, ring, wave, dots}
	s.indexObjects()
	return s
}

func buildClipChildren() []*Obj {
	var out []*Obj
	colors := []string{"#ef4444", "#f97316", "#facc15", "#22c55e", "#06b6d4", "#6366f1", "#a855f7"}
	// Diagonal bars crossing the circle window centered (796,268) r=118.
	for i := 0; i < 7; i++ {
		x0 := 660 + float64(i)*33
		w := 22.0
		top := 150.0
		bot := 392.0
		skew := 46.0
		p := &Path{}
		p.move(x0, bot)
		p.line(x0+w, bot)
		p.line(x0+w+skew, top)
		p.line(x0+skew, top)
		p.close()
		out = append(out, &Obj{
			ID: "clip-stripe-" + string(rune('1'+i)), Tag: "polygon",
			Note: "clipped diagonal stripe", Paths: []*Path{p}, Fill: solid(colors[i]),
		})
	}
	// A bold disc behind the stripes, still clipped.
	disc := &Obj{ID: "clip-disc", Tag: "circle", Note: "clipped inner disc",
		Paths: []*Path{EllipsePath(796, 268, 70, 70)}, Fill: solid("#1e293b")}
	return append([]*Obj{disc}, out...)
}

func parseHex(h string) color.RGBA {
	var r, g, b, a uint8 = 0, 0, 0, 255
	s := h
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	switch len(s) {
	case 3:
		var rv, gv, bv uint8
		fmt.Sscanf(s, "%1x%1x%1x", &rv, &gv, &bv)
		r, g, b = rv*17, gv*17, bv*17
	case 6:
		var rv, gv, bv uint32
		fmt.Sscanf(s, "%2x%2x%2x", &rv, &gv, &bv)
		r, g, b = uint8(rv), uint8(gv), uint8(bv)
	case 8:
		var rv, gv, bv, av uint32
		fmt.Sscanf(s, "%2x%2x%2x%2x", &rv, &gv, &bv, &av)
		r, g, b, a = uint8(rv), uint8(gv), uint8(bv), uint8(av)
	}
	return color.RGBA{r, g, b, a}
}

func hexString(c color.RGBA) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}
