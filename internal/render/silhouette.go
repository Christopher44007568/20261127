package render

import (
	"math"

	"meshbool/internal/mesh"
)

type Silhouette struct {
	Width, Height int
	Pixels        []bool
}

func Rasterize(camera Camera, m mesh.Mesh) Silhouette {
	out := Silhouette{
		Width:  camera.Width,
		Height: camera.Height,
		Pixels: make([]bool, camera.Width*camera.Height),
	}
	for _, triangle := range m.Triangles {
		a := camera.Project(triangle.A)
		b := camera.Project(triangle.B)
		c := camera.Project(triangle.C)
		area := edge(a, b, c)
		if math.Abs(area) < 1e-9 {
			continue
		}
		minX := clampInt(int(math.Floor(math.Min(a.X, math.Min(b.X, c.X)))), 0, camera.Width-1)
		maxX := clampInt(int(math.Ceil(math.Max(a.X, math.Max(b.X, c.X)))), 0, camera.Width-1)
		minY := clampInt(int(math.Floor(math.Min(a.Y, math.Min(b.Y, c.Y)))), 0, camera.Height-1)
		maxY := clampInt(int(math.Ceil(math.Max(a.Y, math.Max(b.Y, c.Y)))), 0, camera.Height-1)
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				w0 := edge(b, c, Projected{X: px, Y: py}) / area
				w1 := edge(c, a, Projected{X: px, Y: py}) / area
				w2 := 1 - w0 - w1
				if w0 >= -1e-7 && w1 >= -1e-7 && w2 >= -1e-7 {
					out.Pixels[y*camera.Width+x] = true
				}
			}
		}
	}
	return out
}

func edge(a, b, p Projected) float64 {
	return (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
}

func (s Silhouette) IoU(other Silhouette) float64 {
	if s.Width != other.Width || s.Height != other.Height || len(s.Pixels) != len(other.Pixels) {
		return 0
	}
	intersection, union := 0, 0
	for i := range s.Pixels {
		if s.Pixels[i] || other.Pixels[i] {
			union++
			if s.Pixels[i] && other.Pixels[i] {
				intersection++
			}
		}
	}
	if union == 0 {
		return 1
	}
	return float64(intersection) / float64(union)
}

func (s Silhouette) Coverage() float64 {
	if len(s.Pixels) == 0 {
		return 0
	}
	count := 0
	for _, pixel := range s.Pixels {
		if pixel {
			count++
		}
	}
	return float64(count) / float64(len(s.Pixels))
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
