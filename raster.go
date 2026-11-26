package main

import "image/color"

// Frame holds every per-pixel buffer for one rendered frame.
type Frame struct {
	W, H  int
	Color []color.RGBA
	Depth []float64
	VelX  []float64
	VelY  []float64
	Cover []bool // true where the skinned character wrote a pixel
	Hud   []bool // true where static HUD UI overwrote the color
}

func NewFrame(w, h int) *Frame {
	n := w * h
	return &Frame{
		W: w, H: h,
		Color: make([]color.RGBA, n),
		Depth: make([]float64, n),
		VelX:  make([]float64, n),
		VelY:  make([]float64, n),
		Cover: make([]bool, n),
		Hud:   make([]bool, n),
	}
}

func (f *Frame) idx(x, y int) int { return y*f.W + x }

func (f *Frame) Clear(bg color.RGBA) {
	for i := range f.Color {
		f.Color[i] = bg
		f.Depth[i] = 1
		f.VelX[i] = 0
		f.VelY[i] = 0
		f.Cover[i] = false
		f.Hud[i] = false
	}
}

// VOut is a post-projection vertex with its skinned screen velocity.
type VOut struct {
	SX, SY float64
	NDCZ   float64
	InvW   float64
	Color  Vec3
	VelX   float64
	VelY   float64
	World  Vec3
}

// DrawTriangle fills one triangle with a z-buffer, perspective-correct
// interpolation and flat Lambert shading. Velocity is interpolated like
// any other vertex attribute.
func (f *Frame) DrawTriangle(a, b, c VOut) {
	area := (b.SX-a.SX)*(c.SY-a.SY) - (b.SY-a.SY)*(c.SX-a.SX)
	if area > -1e-9 && area < 1e-9 {
		return
	}
	// Back-face culling in world space against the camera position.
	normal := b.World.Sub(a.World).Cross(c.World.Sub(a.World)).Norm()
	view := CameraEye.Sub(a.World).Norm()
	if normal.Dot(view) >= 0 {
		return
	}
	light := Vec3{0.4, 0.75, 0.9}.Norm()
	shade := 0.28 + 0.72*max(0, normal.Dot(light))

	minX := max(0, int(min(min(a.SX, b.SX), c.SX)))
	maxX := min(f.W-1, int(max(max(a.SX, b.SX), c.SX))+1)
	minY := max(0, int(min(min(a.SY, b.SY), c.SY)))
	maxY := min(f.H-1, int(max(max(a.SY, b.SY), c.SY))+1)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			w0 := ((b.SX-px)*(c.SY-py) - (b.SY-py)*(c.SX-px)) / area
			w1 := ((c.SX-px)*(a.SY-py) - (c.SY-py)*(a.SX-px)) / area
			w2 := 1 - w0 - w1
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			inv := 1 / (w0*a.InvW + w1*b.InvW + w2*c.InvW)
			z := w0*a.NDCZ + w1*b.NDCZ + w2*c.NDCZ
			i := f.idx(x, y)
			if f.Cover[i] && z >= f.Depth[i] {
				continue
			}
			ca := w0 * a.InvW
			cb := w1 * b.InvW
			cc := w2 * c.InvW
			col := a.Color.Scale(ca).Add(b.Color.Scale(cb)).Add(c.Color.Scale(cc)).Scale(inv).Scale(shade)
			velX := (w0*a.VelX*a.InvW + w1*b.VelX*b.InvW + w2*c.VelX*c.InvW) * inv
			velY := (w0*a.VelY*a.InvW + w1*b.VelY*b.InvW + w2*c.VelY*c.InvW) * inv

			f.Depth[i] = z
			f.Cover[i] = true
			f.VelX[i], f.VelY[i] = velX, velY
			f.Color[i] = color.RGBA{
				R: clamp8(col.X), G: clamp8(col.Y), B: clamp8(col.Z), A: 255,
			}
		}
	}
}

func clamp8(v float64) uint8 {
	v = clamp01(v)
	return uint8(v * 255)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// DrawHudRect writes UI color only: depth, coverage and velocity buffers
// are deliberately untouched so static HUD never enters the velocity map.
func (f *Frame) DrawHudRect(x0, y0, x1, y1 int, c color.RGBA) {
	for y := max(0, y0); y < min(f.H, y1); y++ {
		for x := max(0, x0); x < min(f.W, x1); x++ {
			i := f.idx(x, y)
			f.Color[i] = c
			f.Hud[i] = true
			f.VelX[i] = 0
			f.VelY[i] = 0
		}
	}
}

// DrawHudText renders bitmap glyphs through the HUD-only path.
func (f *Frame) DrawHudText(text string, x, y, scale int, c color.RGBA) {
	for _, r := range text {
		g, ok := FontGlyphs[r]
		if !ok {
			x += 4 * scale
			continue
		}
		for row := 0; row < FontHeight; row++ {
			for col := 0; col < FontWidth; col++ {
				if g[row]&(1<<(FontWidth-1-col)) != 0 {
					f.DrawHudRect(x+col*scale, y+row*scale, x+(col+1)*scale, y+(row+1)*scale, c)
				}
			}
		}
		x += (FontWidth + 1) * scale
	}
}
