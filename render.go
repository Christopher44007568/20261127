package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type Vec3 struct{ X, Y, Z float64 }

func (v Vec3) Norm() Vec3 {
	l := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	return Vec3{v.X / l, v.Y / l, v.Z / l}
}

func (v Vec3) Dot(w Vec3) float64 { return v.X*w.X + v.Y*w.Y + v.Z*w.Z }
func (v Vec3) Add(w Vec3) Vec3    { return Vec3{v.X + w.X, v.Y + w.Y, v.Z + w.Z} }

var viewDir = Vec3{0, 0, 1}

// shadeNormal computes the iridescent reflectance spectrum for a surface
// normal.
// The thin-film angle uses the half-vector between view and light; the
// diffuse base uses the light incidence cosine.
func shadeNormal(n, light Vec3, lut *FilmLUT) Spectrum {
	half := viewDir.Add(light).Norm()
	cosFilm := clamp01(n.Dot(half))
	cosLight := clamp01(n.Dot(light))
	return lut.Shade(cosFilm, cosLight)
}

// renderSphere draws one antialiased sphere of the given radius into img,
// centered at (cx, cy), lit from lightDir (pointing toward the scene).
func renderSphere(img *image.RGBA, cx, cy, radius float64, light Vec3, lut *FilmLUT, exposure float64) {
	const ss = 3 // supersampling per axis
	x0 := int(cx - radius - 1)
	x1 := int(cx + radius + 1)
	y0 := int(cy - radius - 1)
	y1 := int(cy + radius + 1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			var acc Spectrum
			hits := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					fx := float64(x) + (float64(sx)+0.5)/ss
					fy := float64(y) + (float64(sy)+0.5)/ss
					nx := (fx - cx) / radius
					ny := (cy - fy) / radius
					d2 := nx*nx + ny*ny
					if d2 > 1 {
						continue
					}
					n := Vec3{nx, ny, math.Sqrt(1 - d2)}
					s := shadeNormal(n, light, lut)
					for i := 0; i < Samples; i++ {
						acc[i] += s[i]
					}
					hits++
				}
			}
			if hits == 0 {
				continue
			}
			f := 1 / float64(hits)
			for i := 0; i < Samples; i++ {
				acc[i] *= f
			}
			r, g, b := acc.ToSRGB(exposure)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(255*r + 0.5),
				G: uint8(255*g + 0.5),
				B: uint8(255*b + 0.5),
				A: 255,
			})
		}
	}
}

func newCanvas(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func savePNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// RenderLightDirections renders the beetle stack sphere under three
// different light directions side by side.
func RenderLightDirections(path string, lut *FilmLUT) error {
	const (
		radius = 150.0
		w, h   = 1080, 420
	)
	img := newCanvas(w, h, color.RGBA{18, 18, 22, 255})
	lights := []Vec3{
		(Vec3{-0.7, 0.5, 0.5}).Norm(), // from upper left
		(Vec3{0.0, 0.9, 0.44}).Norm(), // from top
		(Vec3{0.7, 0.5, 0.5}).Norm(),  // from upper right
	}
	for i, l := range lights {
		renderSphere(img, 180+float64(i)*360, 210, radius, l, lut, 1.6)
	}
	return savePNG(path, img)
}

// RenderCurvatureSphere renders a large sphere where hue is driven purely
// by surface curvature: the film angle equals the normal-view angle, so
// concentric iridescent bands map the foreshortening.
func RenderCurvatureSphere(path string, lut *FilmLUT) error {
	const (
		radius = 330.0
		w, h   = 760, 760
	)
	img := newCanvas(w, h, color.RGBA{18, 18, 22, 255})
	light := (Vec3{0.35, 0.55, 0.76}).Norm()
	renderSphere(img, w/2, h/2, radius, light, lut, 1.6)
	return savePNG(path, img)
}

// ellipsoidNormal returns the world-space normal of an ellipsoid
// (semi-axes a, b, c) rotated by alpha around the vertical axis,
// at orthographic screen position (x, y). ok is false on a miss.
func ellipsoidNormal(x, y, a, b, c, alpha float64) (n Vec3, ok bool) {
	sn, cs := math.Sin(alpha), math.Cos(alpha)
	// Object-frame ray point p = (cs*x - sn*zs, y, sn*x + cs*zs); solve
	// (px/a)^2 + (py/b)^2 + (pz/c)^2 = 1 for the front-most zs.
	A := sn*sn/(a*a) + cs*cs/(c*c)
	B := 2 * sn * cs * x * (1/(c*c) - 1/(a*a))
	C := cs*cs*x*x/(a*a) + y*y/(b*b) + sn*sn*x*x/(c*c) - 1
	disc := B*B - 4*A*C
	if disc < 0 {
		return Vec3{}, false
	}
	zs := (-B + math.Sqrt(disc)) / (2 * A)
	px := cs*x - sn*zs
	py := y
	pz := sn*x + cs*zs
	no := Vec3{px / (a * a), py / (b * b), pz / (c * c)}.Norm()
	// Rotate the normal back into world space.
	return Vec3{
		X: cs*no.X + sn*no.Z,
		Y: no.Y,
		Z: -sn*no.X + cs*no.Z,
	}, true
}

// RenderRotation renders the same ellipsoid at four rotation angles to
// show the iridescent sweep: colors change because the film's spectral
// peak moves along wavelength, not because RGB channels are re-phased.
func RenderRotation(path string, lut *FilmLUT) error {
	const (
		w, h     = 1120, 360
		a, b, c  = 130.0, 95.0, 45.0
		ss       = 3
		exposure = 1.6
	)
	img := newCanvas(w, h, color.RGBA{18, 18, 22, 255})
	light := (Vec3{0.3, 0.6, 0.74}).Norm()
	angles := []float64{0, 20, 40, 60}
	for k, deg := range angles {
		alpha := deg * math.Pi / 180
		cx := 140 + float64(k)*280
		cy := float64(h) / 2
		for y := 0; y < h; y++ {
			for x := int(cx - a - 1); x <= int(cx+a+1); x++ {
				var acc Spectrum
				hits := 0
				for sy := 0; sy < ss; sy++ {
					for sx := 0; sx < ss; sx++ {
						fx := float64(x) + (float64(sx)+0.5)/ss - cx
						fy := cy - (float64(y) + (float64(sy)+0.5)/ss)
						n, ok := ellipsoidNormal(fx, fy, a, b, c, alpha)
						if !ok {
							continue
						}
						s := shadeNormal(n, light, lut)
						for i := 0; i < Samples; i++ {
							acc[i] += s[i]
						}
						hits++
					}
				}
				if hits == 0 {
					continue
				}
				f := 1 / float64(hits)
				for i := 0; i < Samples; i++ {
					acc[i] *= f
				}
				r, g, bl := acc.ToSRGB(exposure)
				img.SetRGBA(x, y, color.RGBA{
					R: uint8(255*r + 0.5),
					G: uint8(255*g + 0.5),
					B: uint8(255*bl + 0.5),
					A: 255,
				})
			}
		}
	}
	return savePNG(path, img)
}
