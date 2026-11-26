package main

import (
	"image"
	"image/color"
	"math"
)

var (
	colCamPath = color.RGBA{110, 170, 255, 255}
	colOOB     = color.RGBA{255, 80, 80, 255}
	colKey     = color.RGBA{255, 214, 90, 255}
	colFrustum = color.RGBA{235, 235, 235, 200}
	colCharMap = color.RGBA{90, 230, 120, 255}
)

// PathMap renders a top-down overview of the whole world, independent of the
// animated camera. Camera spline portions outside the playable rectangle are
// drawn in red so the out-of-bounds path is visible at a glance.
type PathMap struct {
	Width     int
	Height    int
	Margin    int
	Bounds    Rect
	Obstacles []Obstacle
}

func (m PathMap) scale() float64 {
	sx := float64(m.Width-2*m.Margin) / m.Bounds.Width()
	sy := float64(m.Height-2*m.Margin) / m.Bounds.Height()
	return math.Min(sx, sy)
}

func (m PathMap) project(p Vec2) (float64, float64) {
	s := m.scale()
	x := float64(m.Margin) + (p.X-m.Bounds.Min.X)*s
	y := float64(m.Height-m.Margin) - (p.Y-m.Bounds.Min.Y)*s
	return x, y
}

// Build samples the camera densely and paints in-bounds vs out-of-bounds
// segments, character tracks, keyframes, frustums and a legend.
func (m PathMap) Build(cam CameraPath, chars []CharacterTrack, r Renderer, samples int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	fillAll(img, color.RGBA{18, 21, 27, 255})

	m.drawRect(img, m.Bounds, color.RGBA{30, 36, 47, 255}, colBounds)
	for _, ob := range m.Obstacles {
		m.drawRect(img, ob.Rect, colObstacle, colObstEdge)
	}

	// Camera spline, colored by inside/outside the playable bounds.
	var prev Vec2
	duration := cam.Keys[len(cam.Keys)-1].Time
	for i := 0; i <= samples; i++ {
		t := duration * float64(i) / float64(samples)
		pos := cam.At(t).Pos
		if i > 0 {
			x0, y0 := m.project(prev)
			x1, y1 := m.project(pos)
			if !strictContains(m.Bounds, pos) || !strictContains(m.Bounds, prev) {
				// Out-of-bounds stretches are drawn thicker and red so the
				// conflict is unmistakable against the normal spline.
				plotLine(img, x0, y0, x1, y1, colOOB)
				plotLine(img, x0, y0+1, x1, y1+1, colOOB)
				plotLine(img, x0+1, y0, x1+1, y1, colOOB)
				plotLine(img, x0, y0-1, x1, y1-1, colOOB)
				plotLine(img, x0-1, y0, x1-1, y1, colOOB)
			} else {
				plotLine(img, x0, y0, x1, y1, colCamPath)
			}
		}
		prev = pos
	}
	for _, k := range cam.Keys {
		x, y := m.project(k.Pos)
		plotSquare(img, int(x), int(y), 3, colKey)
	}

	for _, tr := range chars {
		for i, k := range tr.Keys {
			if i > 0 {
				x0, y0 := m.project(tr.Keys[i-1].Pos)
				x1, y1 := m.project(k.Pos)
				plotLine(img, x0, y0, x1, y1, colCharMap)
			}
			x, y := m.project(k.Pos)
			fillCircle(img, int(x), int(y), 3, colCharMap)
		}
	}
	m.drawFrustums(img, cam, r)

	m.drawLegend(img)
	return img
}

// strictContains reports points genuinely inside the playable rectangle,
// without the boundary tolerance used by collision math.
func strictContains(r Rect, p Vec2) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X && p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

func (m PathMap) drawFrustums(img *image.RGBA, cam CameraPath, r Renderer) {
	if len(cam.Keys) < 2 {
		return
	}
	for _, k := range cam.Keys {
		halfW := (r.ViewWidth / k.Zoom) / 2
		halfH := halfW * float64(r.Height) / float64(r.Width)
		cos, sin := math.Cos(k.Heading), math.Sin(k.Heading)
		local := [4]Vec2{{-halfW, -halfH}, {halfW, -halfH}, {halfW, halfH}, {-halfW, halfH}}
		sx := [4]float64{}
		sy := [4]float64{}
		for j, v := range local {
			world := Vec2{k.Pos.X + cos*v.X - sin*v.Y, k.Pos.Y + sin*v.X + cos*v.Y}
			sx[j], sy[j] = m.project(world)
		}
		for j := 0; j < 4; j++ {
			plotLine(img, sx[j], sy[j], sx[(j+1)%4], sy[(j+1)%4], colFrustum)
		}
	}
}

func (m PathMap) drawRect(img *image.RGBA, rect Rect, fill, edge color.Color) {
	x0, y0 := m.project(rect.Min)
	x1, y1 := m.project(rect.Max)
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := int(y0); y <= int(y1); y++ {
		for x := int(x0); x <= int(x1); x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.Set(x, y, fill)
			}
		}
	}
	for x := int(x0); x <= int(x1); x++ {
		img.Set(x, int(y0), edge)
		img.Set(x, int(y1), edge)
	}
	for y := int(y0); y <= int(y1); y++ {
		img.Set(int(x0), y, edge)
		img.Set(int(x1), y, edge)
	}
}

func (m PathMap) drawLegend(img *image.RGBA) {
	lines := []string{
		"CAMERA PATH", "OOB SEGMENT", "CHARACTER", "OBSTACLE", "CAM KEY",
	}
	cols := []color.Color{colCamPath, colOOB, colCharMap, colObstacle, colKey}
	y := 6
	for i, line := range lines {
		fillSquare(img, 6, y+1, 5, cols[i])
		DrawText(img, 16, y, 1, line, colText)
		y += 11
	}
}

func fillAll(img *image.RGBA, col color.Color) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, col)
		}
	}
}

func fillSquare(img *image.RGBA, x, y, size int, col color.Color) {
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			img.Set(x+dx, y+dy, col)
		}
	}
}

func plotSquare(img *image.RGBA, cx, cy, half int, col color.Color) {
	for dy := -half; dy <= half; dy++ {
		for dx := -half; dx <= half; dx++ {
			if image.Pt(cx+dx, cy+dy).In(img.Bounds()) {
				img.Set(cx+dx, cy+dy, col)
			}
		}
	}
}
