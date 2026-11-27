package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func writePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func tsColor(n Vec3) color.RGBA {
	return color.RGBA{
		uint8(clamp01(n.X*0.5+0.5)*255 + 0.5),
		uint8(clamp01(n.Y*0.5+0.5)*255 + 0.5),
		uint8(clamp01(n.Z*0.5+0.5)*255 + 0.5),
		255,
	}
}

// SaveNormalMap writes the tangent-space normal map. Uncovered and zero-hit
// texels stay neutral (128,128,255).
func SaveNormalMap(r *BakeResult, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	for i := 0; i < r.W*r.H; i++ {
		img.SetRGBA(i%r.W, i/r.W, tsColor(r.Nrm[i]))
	}
	return writePNG(img, path)
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		255,
	}
}

// SaveDiag colors diagnostics: mirrored UV = red tint, self-occlusion = blue
// tint, zero-hit = green, uncovered = dark gray.
func SaveDiag(r *BakeResult, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	for i := 0; i < r.W*r.H; i++ {
		x, y := i%r.W, i/r.W
		if !r.Covered[i] {
			img.SetRGBA(x, y, color.RGBA{40, 40, 40, 255})
			continue
		}
		c := tsColor(r.Nrm[i])
		if r.Mirror[i] {
			c = mix(c, color.RGBA{255, 0, 0, 255}, 0.8)
		}
		if r.Occl[i] {
			c = mix(c, color.RGBA{0, 80, 255, 255}, 0.8)
		}
		if r.Zero[i] {
			c = color.RGBA{0, 200, 0, 255}
		}
		img.SetRGBA(x, y, c)
	}
	return writePNG(img, path)
}

// sampleMap bilinearly samples the baked tangent-space normal at (u,v).
func sampleMap(r *BakeResult, u, v float64) Vec3 {
	x := clamp01(u)*float64(r.W) - 0.5
	y := (1-clamp01(v))*float64(r.H) - 0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	at := func(xx, yy int) Vec3 {
		if xx < 0 {
			xx = 0
		}
		if xx >= r.W {
			xx = r.W - 1
		}
		if yy < 0 {
			yy = 0
		}
		if yy >= r.H {
			yy = r.H - 1
		}
		return r.Nrm[yy*r.W+xx]
	}
	n00, n10 := at(x0, y0), at(x0+1, y0)
	n01, n11 := at(x0, y0+1), at(x0+1, y0+1)
	top := n00.Mul(1 - fx).Add(n10.Mul(fx))
	bot := n01.Mul(1 - fx).Add(n11.Mul(fx))
	return top.Mul(1 - fy).Add(bot.Mul(fy)).Norm()
}

// SaveNormalSphere renders a lit material-preview sphere with the baked map
// applied in the sphere's tangent space.
func SaveNormalSphere(r *BakeResult, path string, size int) error {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	light := Vec3{0.4, 0.6, 1}.Norm()
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			nx := (2*(float64(x)+0.5)/float64(size) - 1) * 0.98
			ny := (1 - 2*(float64(y)+0.5)/float64(size)) * 0.98
			d2 := nx*nx + ny*ny
			if d2 > 1 {
				g := uint8(24 + 20*(1-float64(y)/float64(size)))
				img.SetRGBA(x, y, color.RGBA{g, g, g + 6, 255})
				continue
			}
			N := Vec3{nx, ny, math.Sqrt(1 - d2)}
			T := Vec3{0, 1, 0}.Cross(N)
			if T.Len() < 1e-6 {
				T = Vec3{1, 0, 0}
			}
			T = T.Norm()
			B := N.Cross(T)
			ts := sampleMap(r, (float64(x)+0.5)/float64(size), 1-(float64(y)+0.5)/float64(size))
			w := T.Mul(ts.X).Add(B.Mul(ts.Y)).Add(N.Mul(ts.Z)).Norm()
			diff := math.Max(0, w.Dot(light))
			h := light.Add(Vec3{0, 0, 1}).Norm()
			spec := math.Pow(math.Max(0, w.Dot(h)), 32)
			lum := 0.12 + 0.88*diff
			img.SetRGBA(x, y, color.RGBA{
				uint8(clamp01(0.72*lum+0.35*spec) * 255),
				uint8(clamp01(0.74*lum+0.35*spec) * 255),
				uint8(clamp01(0.78*lum+0.35*spec) * 255),
				255,
			})
		}
	}
	return writePNG(img, path)
}
