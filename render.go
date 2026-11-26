package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type Canvas struct {
	W     int
	H     int
	Img   *image.RGBA
	Depth []float64
}

func NewCanvas(w, h int, bg RGB) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := &Canvas{W: w, H: h, Img: img, Depth: make([]float64, w*h)}
	c.Clear(bg)
	c.ClearDepth()
	return c
}

func (c *Canvas) Clear(col RGB) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			c.Img.SetRGBA(x, y, color.RGBA{col.R, col.G, col.B, 255})
		}
	}
}

func (c *Canvas) ClearDepth() {
	for i := range c.Depth {
		c.Depth[i] = math.Inf(1)
	}
}

func (c *Canvas) index(x, y int) int { return y*c.W + x }

func (c *Canvas) FillRect(x, y, w, h int, col RGB) {
	for yy := y; yy < y+h; yy++ {
		if yy < 0 || yy >= c.H {
			continue
		}
		for xx := x; xx < x+w; xx++ {
			if xx < 0 || xx >= c.W {
				continue
			}
			c.Img.SetRGBA(xx, yy, color.RGBA{col.R, col.G, col.B, 255})
		}
	}
}

func (c *Canvas) SetPixel(x, y int, col RGB) {
	if x >= 0 && x < c.W && y >= 0 && y < c.H {
		c.Img.SetRGBA(x, y, color.RGBA{col.R, col.G, col.B, 255})
	}
}

func (c *Canvas) SetDepthPixel(x, y int, depth, bias float64, col RGB) bool {
	if x < 0 || x >= c.W || y < 0 || y >= c.H {
		return false
	}
	id := c.index(x, y)
	if depth-bias >= c.Depth[id] {
		return false
	}
	c.Depth[id] = depth - bias
	c.Img.SetRGBA(x, y, color.RGBA{col.R, col.G, col.B, 255})
	return true
}

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func (c *Canvas) SavePNG(path string) error {
	return SavePNG(path, c.Img)
}

type Camera struct {
	W, H    int
	Center  V3
	EyeDir  V3
	Forward V3
	Right   V3
	Up      V3
	Scale   float64
	ShiftY  float64
}

func FitCamera(box Box, w, h int, yaw, pitch, fill float64) Camera {
	eye := Normalize(V3{
		X: math.Cos(pitch) * math.Cos(yaw),
		Y: math.Cos(pitch) * math.Sin(yaw),
		Z: math.Sin(pitch),
	})
	forward := eye.Mul(-1)
	worldUp := V3{0, 0, 1}
	right := Normalize(Cross(forward, worldUp))
	if Norm(right) < 0.5 {
		right = V3{1, 0, 0}
	}
	up := Cross(right, forward)
	cam := Camera{
		W: w, H: h,
		Center:  box.Center(),
		EyeDir:  eye,
		Forward: forward,
		Right:   right,
		Up:      Normalize(up),
		ShiftY:  float64(h) * 0.045,
	}

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for ix := 0; ix < 2; ix++ {
		for iy := 0; iy < 2; iy++ {
			for iz := 0; iz < 2; iz++ {
				p := V3{
					lerpFloat(box.Min.X, box.Max.X, float64(ix)),
					lerpFloat(box.Min.Y, box.Max.Y, float64(iy)),
					lerpFloat(box.Min.Z, box.Max.Z, float64(iz)),
				}
				r := p.Sub(cam.Center)
				sx := Dot(r, right)
				sy := Dot(r, cam.Up)
				minX = math.Min(minX, sx)
				maxX = math.Max(maxX, sx)
				minY = math.Min(minY, sy)
				maxY = math.Max(maxY, sy)
			}
		}
	}
	rx := math.Max(maxX-minX, 1e-6)
	ry := math.Max(maxY-minY, 1e-6)
	if fill <= 0 {
		fill = 0.82
	}
	cam.Scale = math.Min(float64(w)*fill/rx, float64(h)*fill/ry)
	return cam
}

type Projected struct {
	X, Y  float64
	Depth float64
}

func (cam Camera) Project(p V3) Projected {
	r := p.Sub(cam.Center)
	return Projected{
		X:     float64(cam.W)*0.5 + Dot(r, cam.Right)*cam.Scale,
		Y:     float64(cam.H)*0.5 + cam.ShiftY - Dot(r, cam.Up)*cam.Scale,
		Depth: Dot(r, cam.Forward),
	}
}
