package main

import (
	"image"
	"image/color"
	"math"
)

var (
	colBackground = color.RGBA{24, 28, 36, 255}
	colBounds     = color.RGBA{150, 160, 175, 255}
	colObstacle   = color.RGBA{150, 60, 60, 255}
	colObstEdge   = color.RGBA{235, 120, 120, 255}
	colCharacter  = color.RGBA{90, 230, 120, 255}
	colGhost      = color.RGBA{90, 230, 120, 70}
	colText       = color.RGBA{235, 238, 245, 255}
	colTextBg     = color.RGBA{0, 0, 0, 170}
)

// Renderer draws top-down camera views. The view transform is driven entirely
// by the spline camera on the shared timeline: position, heading and zoom.
type Renderer struct {
	Width     int
	Height    int
	Bounds    Rect
	Obstacles []Obstacle
	// ViewWidth is the world width visible at zoom 1; zoom magnifies it.
	ViewWidth float64
}

// worldToScreen maps a world point into camera-local pixel coordinates.
func (r Renderer) worldToScreen(p Vec2, cam CameraKey) (float64, float64) {
	scale := float64(r.Width) / (r.ViewWidth / cam.Zoom)
	d := p.Sub(cam.Pos)
	cos, sin := math.Cos(cam.Heading), math.Sin(cam.Heading)
	localX := cos*d.X + sin*d.Y
	localY := -sin*d.X + cos*d.Y
	return float64(r.Width)/2 + localX*scale, float64(r.Height)/2 - localY*scale
}

func (r Renderer) fill(img *image.RGBA, col color.Color) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, col)
		}
	}
}

func plotLine(img *image.RGBA, x0, y0, x1, y1 float64, col color.Color) {
	// Integer DDA keeps the renderer dependency-free and crisp at this size.
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps == 0 {
		img.Set(int(math.Round(x0)), int(math.Round(y0)), col)
		return
	}
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		x := int(math.Round(x0 + dx*u))
		y := int(math.Round(y0 + dy*u))
		if image.Pt(x, y).In(img.Bounds()) {
			img.Set(x, y, col)
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, radius int, col color.Color) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				img.Set(cx+x, cy+y, col)
			}
		}
	}
}

func (r Renderer) drawWorldRect(img *image.RGBA, rect Rect, cam CameraKey, fill, edge color.Color) {
	corners := [4]Vec2{
		{rect.Min.X, rect.Min.Y}, {rect.Max.X, rect.Min.Y},
		{rect.Max.X, rect.Max.Y}, {rect.Min.X, rect.Max.Y},
	}
	sx := make([]float64, 4)
	sy := make([]float64, 4)
	for i, c := range corners {
		sx[i], sy[i] = r.worldToScreen(c, cam)
	}
	// Per-scanline spans are sufficient for these axis-aligned world boxes.
	minY := math.MaxFloat64
	maxY := -math.MaxFloat64
	for _, y := range sy {
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}
	for y := int(minY); y <= int(maxY); y++ {
		span, ok := clipPolygonRow(corners, cam, r, y)
		if ok {
			for x := int(span[0]); x <= int(span[1]); x++ {
				if image.Pt(x, y).In(img.Bounds()) {
					img.Set(x, y, fill)
				}
			}
		}
	}
	for i := 0; i < 4; i++ {
		plotLine(img, sx[i], sy[i], sx[(i+1)%4], sy[(i+1)%4], edge)
	}
}

// clipPolygonRow returns the [minX, maxX] screen span of rect on scan row y.
func clipPolygonRow(corners [4]Vec2, cam CameraKey, r Renderer, y int) ([2]float64, bool) {
	row := float64(y) + 0.5
	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	hit := false
	for i := 0; i < 4; i++ {
		a := corners[i]
		b := corners[(i+1)%4]
		ax, ay := r.worldToScreen(a, cam)
		bx, by := r.worldToScreen(b, cam)
		if (ay <= row && by > row) || (by <= row && ay > row) {
			u := (row - ay) / (by - ay)
			x := ax + (bx-ax)*u
			minX = math.Min(minX, x)
			maxX = math.Max(maxX, x)
			hit = true
		}
	}
	if !hit {
		return [2]float64{}, false
	}
	return [2]float64{minX, maxX}, true
}

// FrameInput bundles everything one exported frame needs. Ghost positions are
// the character samples across the shutter interval; Center is the frame-time
// position, both evaluated against the same shared timeline.
type FrameInput struct {
	Index      int
	FrameCount int
	Time       float64
	Camera     CameraKey
	Ghosts     map[string][]Vec2
	Centers    map[string]Vec2
}

// RenderFrame produces one RGBA thumbnail frame with a timestamp overlay.
func (r Renderer) RenderFrame(in FrameInput) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, r.Width, r.Height))
	r.fill(img, colBackground)

	r.drawWorldRect(img, r.Bounds, in.Camera, color.RGBA{34, 40, 52, 255}, colBounds)
	for _, ob := range r.Obstacles {
		r.drawWorldRect(img, ob.Rect, in.Camera, colObstacle, colObstEdge)
	}

	// Exposure ghosts show the character sweeping through the shutter interval.
	for _, pts := range in.Ghosts {
		for _, p := range pts {
			x, y := r.worldToScreen(p, in.Camera)
			if image.Pt(int(x), int(y)).In(img.Bounds()) {
				fillCircle(img, int(x), int(y), 3, colGhost)
			}
		}
	}
	for _, p := range in.Centers {
		x, y := r.worldToScreen(p, in.Camera)
		fillCircle(img, int(x), int(y), 4, colCharacter)
	}

	r.drawTimestamp(img, in)
	return img
}

func (r Renderer) drawTimestamp(img *image.RGBA, in FrameInput) {
	label := formatStamp(in.Time, in.Index, in.FrameCount)
	width := len(label)*6 + 6
	for y := 0; y < 16; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, colTextBg)
		}
	}
	DrawText(img, 3, 4, 1, label, colText)
}

func formatStamp(t float64, idx, count int) string {
	return padFloat(t) + " F" + itoaPad(idx, 3) + "/" + itoaPad(count, 3)
}

func padFloat(t float64) string {
	scaled := int(math.Round(t * 1000))
	if scaled < 0 {
		scaled = 0
	}
	whole := scaled / 1000
	frac := scaled % 1000
	if frac == 1000 {
		whole++
		frac = 0
	}
	return "T=" + itoaPad(whole, 2) + "." + itoaPad(frac, 3)
}

func itoaPad(v, width int) string {
	s := ""
	if v == 0 {
		s = "0"
	}
	for v > 0 {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	for len(s) < width {
		s = "0" + s
	}
	return s
}
