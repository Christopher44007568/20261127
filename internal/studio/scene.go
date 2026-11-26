package studio

import "math"

const (
	CanvasW = 1600
	CanvasH = 1000
)

type MaskOrder int

const (
	MaskLast MaskOrder = iota
	MaskFirst
)

func (o MaskOrder) String() string {
	if o == MaskFirst {
		return "mask-first"
	}
	return "mask-last"
}

type Scene struct {
	Font       *StrokeFont
	Group      []float32
	Linear     *FloatImage
	Radial     *FloatImage
	Alpha      []float32
	Background *FloatImage
}

func NewScene() *Scene {
	s := &Scene{Font: NewStrokeFont()}
	s.Group = s.buildGroupMask()
	s.Linear = s.buildLinearLayer()
	s.Radial = s.buildRadialLayer()
	s.Alpha = AlphaMask(CanvasW, CanvasH)
	s.Background = s.buildBackground()
	return s
}

func (s *Scene) buildGroupMask() []float32 {
	group := make([]float32, CanvasW*CanvasH)
	circles := []struct {
		x, y, r float32
	}{
		{560, 452, 205},
		{860, 416, 178},
		{744, 626, 220},
	}
	for _, c := range circles {
		UnionMask(group, CircleMask(CanvasW, CanvasH, c.x, c.y, c.r))
	}

	const text = "ALPHA"
	const textH = 180
	textW := s.Font.TextWidth(text, textH)
	x := (CanvasW - textW) / 2
	y := float32(742)
	UnionMask(group, s.Font.TextMask(CanvasW, CanvasH, text, x, y, textH, 0.74))
	return group
}

func (s *Scene) buildLinearLayer() *FloatImage {
	stops := []Stop{
		{Pos: 0.00, Color: Hex("00d7c0")},
		{Pos: 0.25, Color: Hex("2f75ff")},
		{Pos: 0.52, Color: Hex("a64cff")},
		{Pos: 0.76, Color: Hex("ff4f91")},
		{Pos: 1.00, Color: Hex("ffc35c")},
	}
	layer := LinearGradient(CanvasW, CanvasH, 255, 900, 1345, 205, stops)
	clip := NewClipStack(CanvasW, CanvasH)
	clip.PushRect(0, 0, CanvasW, CanvasH)
	clip.Push(s.Group)
	clip.Apply(layer)
	clip.Pop()
	clip.Pop()
	return layer
}

func (s *Scene) buildRadialLayer() *FloatImage {
	stops := []Stop{
		{Pos: 0.00, Color: Hex("fff6d7e8")},
		{Pos: 0.16, Color: Hex("ffd66ee6")},
		{Pos: 0.48, Color: Hex("ff6759c2")},
		{Pos: 0.78, Color: Hex("bd3fffa4")},
		{Pos: 1.00, Color: Hex("180b3a00")},
	}
	layer := RadialGradient(CanvasW, CanvasH, 790, 468, 515, stops)
	clip := NewClipStack(CanvasW, CanvasH)
	clip.PushRect(0, 0, CanvasW, CanvasH)
	clip.Push(s.Group)
	clip.Apply(layer)
	clip.Pop()
	clip.Pop()
	return layer
}

func (s *Scene) buildBackground() *FloatImage {
	bg := NewImage(CanvasW, CanvasH)
	base := Hex("070a11")
	glow := Hex("14223b")
	for y := 0; y < CanvasH; y++ {
		for x := 0; x < CanvasW; x++ {
			dx := (float32(x) - 790) / 820
			dy := (float32(y) - 470) / 700
			d := sqrt32(dx*dx + dy*dy)
			v := 0.22 * (1 - smoothstep(0, 1, d))
			bg.Set(x, y, Premultiply(lerpRGBA(base, glow, v)))
		}
	}

	grid := Premultiply(Hex("7894c208"))
	for x := 80; x < CanvasW; x += 80 {
		for y := 120; y < CanvasH-40; y++ {
			bg.OverPixel(x, y, grid)
		}
	}
	for y := 120; y < CanvasH-40; y += 80 {
		for x := 40; x < CanvasW-40; x++ {
			bg.OverPixel(x, y, grid)
		}
	}

	line := Premultiply(Hex("6c88ad1f"))
	for x := 48; x < CanvasW-48; x++ {
		bg.OverPixel(x, 116, line)
		bg.OverPixel(x, CanvasH-42, line)
	}
	for y := 116; y <= CanvasH-42; y++ {
		bg.OverPixel(48, y, line)
		bg.OverPixel(CanvasW-49, y, line)
	}
	return bg
}

// BuildArt follows: clipped linear layer -> clipped radial layer -> alpha
// mask. MaskFirst deliberately moves the destination-in pass before radial so
// the later blend restores coverage at masked edges.
func (s *Scene) BuildArt(mode BlendMode, order MaskOrder) *FloatImage {
	art := NewImage(CanvasW, CanvasH)
	art.Blend(s.Linear, Normal)
	if order == MaskFirst {
		art.ApplyMask(s.Alpha)
		art.Blend(s.Radial, mode)
		return art
	}
	art.Blend(s.Radial, mode)
	art.ApplyMask(s.Alpha)
	return art
}

func (s *Scene) BuildFinal(mode BlendMode, order MaskOrder) *FloatImage {
	return CompositeOver(s.Background, s.BuildArt(mode, order))
}

func (s *Scene) BuildPremask() *FloatImage {
	art := NewImage(CanvasW, CanvasH)
	art.Blend(s.Linear, Normal)
	art.Blend(s.Radial, Normal)
	return art
}

// EdgeCrop returns a magnified roll-up of a soft group edge. It is used by the
// order comparison sheet to make transparent-edge behaviour inspectable.
func EdgeCrop(src *FloatImage, x, y, w, h, scale int) *FloatImage {
	out := NewImage(w*scale, h*scale)
	for oy := 0; oy < out.H; oy++ {
		for ox := 0; ox < out.W; ox++ {
			sx := x + ox/scale
			sy := y + oy/scale
			if sx >= 0 && sy >= 0 && sx < src.W && sy < src.H {
				out.Set(ox, oy, src.At(sx, sy))
			}
		}
	}
	return out
}

func DrawChecker(im *FloatImage, cell int, dark, light RGBA) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			c := dark
			if ((x/cell)+(y/cell))%2 == 0 {
				c = light
			}
			im.Set(x, y, Premultiply(c))
		}
	}
}

func BlendModePanels(scene *Scene) []*FloatImage {
	modes := []BlendMode{Normal, Multiply, Screen}
	out := make([]*FloatImage, 0, len(modes))
	for _, mode := range modes {
		out = append(out, scene.BuildFinal(mode, MaskLast))
	}
	return out
}

func tileBoundaryDelta(tileW, tileH int) float64 {
	maxDelta := 0.0
	for y := 0; y < tileH; y++ {
		v := float64(y) / float64(tileH)
		a := Luminance(PeriodicTileColor(1, v))
		b := Luminance(PeriodicTileColor(0.0, v))
		maxDelta = math.Max(maxDelta, math.Abs(float64(a-b)))
	}
	for x := 0; x < tileW; x++ {
		u := float64(x) / float64(tileW)
		a := Luminance(PeriodicTileColor(u, 1))
		b := Luminance(PeriodicTileColor(u, 0.0))
		maxDelta = math.Max(maxDelta, math.Abs(float64(a-b)))
	}
	return maxDelta
}
