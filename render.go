package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Camera is an orthographic camera fitted to the scene.
type Camera struct {
	Center      Vec3
	Right, Up   Vec3
	Fwd         Vec3
	Scale       float64
	W, H        int
	SceneRadius float64
}

func FitCamera(center Vec3, radius float64, w, h int, azDeg, elDeg float64) *Camera {
	az := azDeg * math.Pi / 180
	el := elDeg * math.Pi / 180
	fwd := Vec3{math.Cos(el) * math.Cos(az), math.Sin(el), math.Cos(el) * math.Sin(az)}.Norm()
	up0 := Vec3{0, 1, 0}
	if math.Abs(fwd.Dot(up0)) > 0.95 {
		up0 = Vec3{1, 0, 0}
	}
	right := fwd.Cross(up0).Norm()
	up := right.Cross(fwd).Norm()
	scale := 0.5 * float64(min(w, h)) / (radius * 1.1)
	return &Camera{Center: center, Right: right, Up: up, Fwd: fwd,
		Scale: scale, W: w, H: h, SceneRadius: radius}
}

func (c *Camera) Project(p Vec3) (x, y, z float64) {
	d := p.Sub(c.Center)
	x = d.Dot(c.Right)*c.Scale + float64(c.W)/2
	y = float64(c.H)/2 - d.Dot(c.Up)*c.Scale
	z = d.Dot(c.Fwd)
	return
}

// Raster is an RGB framebuffer with a z-buffer.
type Raster struct {
	W, H int
	Z    []float64
	C    [][3]float64
}

func NewRaster(w, h int, bg [3]float64) *Raster {
	r := &Raster{W: w, H: h, Z: make([]float64, w*h), C: make([][3]float64, w*h)}
	for i := range r.Z {
		r.Z[i] = math.Inf(1)
		r.C[i] = bg
	}
	return r
}

type scrPt struct{ x, y, z float64 }

// Tri rasterizes one triangle with Gouraud-interpolated colors.
func (r *Raster) Tri(p [3]scrPt, col [3][3]float64) {
	minx := max(0, int(math.Floor(min(p[0].x, min(p[1].x, p[2].x)))))
	maxx := min(r.W-1, int(math.Ceil(max(p[0].x, max(p[1].x, p[2].x)))))
	miny := max(0, int(math.Floor(min(p[0].y, min(p[1].y, p[2].y)))))
	maxy := min(r.H-1, int(math.Ceil(max(p[0].y, max(p[1].y, p[2].y)))))
	d := (p[1].y-p[2].y)*(p[0].x-p[2].x) + (p[2].x-p[1].x)*(p[0].y-p[2].y)
	if math.Abs(d) < 1e-12 {
		return
	}
	for y := miny; y <= maxy; y++ {
		for x := minx; x <= maxx; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			w0 := ((p[1].y-p[2].y)*(px-p[2].x) + (p[2].x-p[1].x)*(py-p[2].y)) / d
			w1 := ((p[2].y-p[0].y)*(px-p[2].x) + (p[0].x-p[2].x)*(py-p[2].y)) / d
			w2 := 1 - w0 - w1
			if w0 < -1e-6 || w1 < -1e-6 || w2 < -1e-6 {
				continue
			}
			z := w0*p[0].z + w1*p[1].z + w2*p[2].z
			idx := y*r.W + x
			if z >= r.Z[idx] {
				continue
			}
			r.Z[idx] = z
			for k := 0; k < 3; k++ {
				r.C[idx][k] = w0*col[0][k] + w1*col[1][k] + w2*col[2][k]
			}
		}
	}
}

func (r *Raster) blendPixel(x, y int, c [3]float64, alpha float64) {
	if x < 0 || y < 0 || x >= r.W || y >= r.H {
		return
	}
	idx := y*r.W + x
	for k := 0; k < 3; k++ {
		r.C[idx][k] = r.C[idx][k]*(1-alpha) + c[k]*alpha
	}
}

// DrawLine3D draws a 3D segment; visible parts use cVis, occluded parts
// are blended with cOcc so the skeleton stays readable through the tube.
func (r *Raster) DrawLine3D(cam *Camera, a, b Vec3, cVis, cOcc [3]float64) {
	x0, y0, z0 := cam.Project(a)
	x1, y1, z1 := cam.Project(b)
	steps := int(math.Ceil(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))) * 2
	if steps < 1 {
		steps = 1
	}
	tol := 0.015 * cam.SceneRadius
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		z := z0 + (z1-z0)*t
		ix, iy := int(math.Round(x)), int(math.Round(y))
		if ix < 0 || iy < 0 || ix >= r.W || iy >= r.H {
			continue
		}
		idx := iy*r.W + ix
		if z <= r.Z[idx]+tol {
			for dy := 0; dy <= 1; dy++ {
				for dx := 0; dx <= 1; dx++ {
					r.blendPixel(ix+dx, iy+dy, cVis, 1)
				}
			}
		} else {
			r.blendPixel(ix, iy, cOcc, 0.45)
		}
	}
}

func (r *Raster) DrawDot(cam *Camera, p Vec3, radPx int, c [3]float64, alwaysOnTop bool) {
	x, y, z := cam.Project(p)
	ix, iy := int(math.Round(x)), int(math.Round(y))
	for dy := -radPx; dy <= radPx; dy++ {
		for dx := -radPx; dx <= radPx; dx++ {
			if dx*dx+dy*dy > radPx*radPx {
				continue
			}
			px, py := ix+dx, iy+dy
			if px < 0 || py < 0 || px >= r.W || py >= r.H {
				continue
			}
			if alwaysOnTop || z <= r.Z[py*r.W+px]+0.015*cam.SceneRadius {
				r.blendPixel(px, py, c, 1)
			}
		}
	}
}

func (r *Raster) SavePNG(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			c := r.C[y*r.W+x]
			cl := func(v float64) uint8 {
				return uint8(math.Max(0, math.Min(255, v*255+0.5)))
			}
			img.Set(x, y, color.RGBA{cl(c[0]), cl(c[1]), cl(c[2]), 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func shade(n Vec3) float64 {
	l1 := Vec3{-0.45, -0.65, -0.55}.Norm()
	l2 := Vec3{0.6, 0.3, 0.4}.Norm()
	d := 0.26 + 0.72*math.Max(0, n.Dot(l1.Scale(-1))) + 0.18*math.Max(0, n.Dot(l2.Scale(-1)))
	if d > 1.1 {
		d = 1.1
	}
	return d
}

func renderMesh(m *Mesh, cam *Camera, bg [3]float64,
	colorOf func(vi int, n Vec3) [3]float64) *Raster {
	r := NewRaster(cam.W, cam.H, bg)
	sp := make([]scrPt, len(m.V))
	cols := make([][3]float64, len(m.V))
	for i, v := range m.V {
		x, y, z := cam.Project(v)
		sp[i] = scrPt{x, y, z}
		cols[i] = colorOf(i, m.N[i])
	}
	for _, f := range m.F {
		r.Tri([3]scrPt{sp[f[0]], sp[f[1]], sp[f[2]]},
			[3][3]float64{cols[f[0]], cols[f[1]], cols[f[2]]})
	}
	return r
}

// RenderSolid shades the tube with two lights.
func RenderSolid(m *Mesh, cam *Camera) *Raster {
	base := [3]float64{0.87, 0.62, 0.55}
	return renderMesh(m, cam, [3]float64{0.96, 0.97, 0.99}, func(vi int, n Vec3) [3]float64 {
		s := shade(n)
		return [3]float64{base[0] * s, base[1] * s, base[2] * s}
	})
}

// RenderHeat colors the surface by the normal-crack value.
func RenderHeat(m *Mesh, cam *Camera, crack []float64, maxCrack float64) *Raster {
	return renderMesh(m, cam, [3]float64{0.10, 0.10, 0.12}, func(vi int, n Vec3) [3]float64 {
		c := HeatColor(crack[vi] / maxCrack)
		s := 0.55 + 0.45*shade(n)
		return [3]float64{c[0] * s, c[1] * s, c[2] * s}
	})
}

// RenderOverlay draws the faint tube with the resampled skeleton (red),
// the original noisy input nodes (yellow), and joint centers (green).
func RenderOverlay(m *Mesh, cam *Camera, sk *Skeleton, rs []*Resampled, joints []JointInfo) *Raster {
	r := renderMesh(m, cam, [3]float64{0.97, 0.97, 0.98}, func(vi int, n Vec3) [3]float64 {
		s := shade(n)
		v := 0.55 + 0.35*s
		return [3]float64{v, v, v * 1.02}
	})
	red := [3]float64{0.85, 0.08, 0.10}
	occ := [3]float64{0.55, 0.25, 0.55}
	for _, b := range rs {
		for i := 0; i+1 < len(b.P); i++ {
			r.DrawLine3D(cam, b.P[i], b.P[i+1], red, occ)
		}
	}
	for _, j := range joints {
		r.DrawDot(cam, j.Pos, 4, [3]float64{0.05, 0.65, 0.15}, true)
	}
	for _, b := range sk.Branches {
		for _, p := range b.Points {
			r.DrawDot(cam, p, 2, [3]float64{0.95, 0.75, 0.05}, true)
		}
	}
	return r
}
