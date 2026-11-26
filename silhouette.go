package qem

import "math"

type Mask struct {
	Width  int
	Height int
	Inside []bool
}

func NewMask(w, h int) *Mask {
	return &Mask{Width: w, Height: h, Inside: make([]bool, w*h)}
}

func (m *Mask) At(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	return m.Inside[y*m.Width+x]
}

// RenderMask rasterizes a binary orthographic silhouette using the fixed
// original-model frame.
func RenderMask(mesh *Mesh, frame ViewFrame, view View) *Mask {
	mask := NewMask(frame.Width, frame.Height)
	for _, tri := range mesh.Triangles {
		if !tri.Active {
			continue
		}
		ids := tri.Vertices()
		pp := [3]projectedPoint{
			frame.Project(mesh.Vertices[ids[0]].Pos, view),
			frame.Project(mesh.Vertices[ids[1]].Pos, view),
			frame.Project(mesh.Vertices[ids[2]].Pos, view),
		}
		minX := maxInt(0, int(math.Floor(math.Min(pp[0].x, math.Min(pp[1].x, pp[2].x)))))
		maxX := minInt(frame.Width-1, int(math.Ceil(math.Max(pp[0].x, math.Max(pp[1].x, pp[2].x)))))
		minY := maxInt(0, int(math.Floor(math.Min(pp[0].y, math.Min(pp[1].y, pp[2].y)))))
		maxY := minInt(frame.Height-1, int(math.Ceil(math.Max(pp[0].y, math.Max(pp[1].y, pp[2].y)))))
		area := edgeFunction(pp[0].x, pp[0].y, pp[1].x, pp[1].y, pp[2].x, pp[2].y)
		if math.Abs(area) < 1e-9 {
			continue
		}
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				py := float64(y) + 0.5
				w0 := edgeFunction(pp[1].x, pp[1].y, pp[2].x, pp[2].y, px, py) / area
				w1 := edgeFunction(pp[2].x, pp[2].y, pp[0].x, pp[0].y, px, py) / area
				w2 := edgeFunction(pp[0].x, pp[0].y, pp[1].x, pp[1].y, px, py) / area
				if w0 >= -1e-5 && w1 >= -1e-5 && w2 >= -1e-5 {
					mask.Inside[y*frame.Width+x] = true
				}
			}
		}
	}
	return mask
}

func (m *Mask) PixelCount() int {
	n := 0
	for _, v := range m.Inside {
		if v {
			n++
		}
	}
	return n
}

func (m *Mask) Boundary() []bool {
	out := make([]bool, m.Width*m.Height)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if !m.At(x, y) {
				continue
			}
			boundary := false
			for oy := -1; oy <= 1 && !boundary; oy++ {
				for ox := -1; ox <= 1; ox++ {
					if ox == 0 && oy == 0 {
						continue
					}
					if !m.At(x+ox, y+oy) {
						boundary = true
						break
					}
				}
			}
			out[y*m.Width+x] = boundary
		}
	}
	return out
}

func edt1D(f []float64) []float64 {
	n := len(f)
	d := make([]float64, n)
	v := make([]int, n)
	z := make([]float64, n+1)
	z[0] = math.Inf(-1)
	z[1] = math.Inf(1)
	k := 0
	for q := 1; q < n; q++ {
		s := ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) /
			(2 * float64(q-v[k]))
		for s <= z[k] {
			k--
			s = ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) /
				(2 * float64(q-v[k]))
		}
		k++
		v[k] = q
		z[k] = s
		z[k+1] = math.Inf(1)
	}
	k = 0
	for q := 0; q < n; q++ {
		for z[k+1] < float64(q) {
			k++
		}
		delta := q - v[k]
		d[q] = float64(delta*delta) + f[v[k]]
	}
	return d
}

// DistanceMap returns exact Euclidean pixel distance to boundary pixels.
func (m *Mask) DistanceMap() []float64 {
	boundary := m.Boundary()
	d := make([]float64, m.Width*m.Height)
	for i := range d {
		if boundary[i] {
			d[i] = 0
		} else {
			// A large finite sentinel is important for the 1D EDT: an
			// all-Inf row makes parabola intersections NaN.
			d[i] = 1e18
		}
	}
	for y := 0; y < m.Height; y++ {
		start := y * m.Width
		row := edt1D(d[start : start+m.Width])
		copy(d[start:start+m.Width], row)
	}
	for x := 0; x < m.Width; x++ {
		col := make([]float64, m.Height)
		for y := 0; y < m.Height; y++ {
			col[y] = d[y*m.Width+x]
		}
		col = edt1D(col)
		for y := 0; y < m.Height; y++ {
			d[y*m.Width+x] = math.Sqrt(col[y])
		}
	}
	return d
}

func meanMaxBoundaryDistance(source *Mask, targetDistance []float64) (float64, float64, int) {
	boundary := source.Boundary()
	sum := 0.0
	max := 0.0
	count := 0
	for i, isBoundary := range boundary {
		if !isBoundary || math.IsInf(targetDistance[i], 1) {
			continue
		}
		sum += targetDistance[i]
		max = math.Max(max, targetDistance[i])
		count++
	}
	if count == 0 {
		return 0, 0, 0
	}
	return sum / float64(count), max, count
}

// SilhouetteOverlay colors overlap gray, original-only blue, and simplified-only red.
func SilhouetteOverlay(original, simplified *Mask) *PPM {
	img := NewPPM(original.Width, original.Height, RGB{244, 247, 250})
	for i := range original.Inside {
		a := original.Inside[i]
		b := simplified.Inside[i]
		switch {
		case a && b:
			img.Pix[i] = RGB{196, 205, 219}
		case a && !b:
			img.Pix[i] = RGB{47, 111, 237}
		case !a && b:
			img.Pix[i] = RGB{239, 68, 68}
		}
	}
	return img
}
