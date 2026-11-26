package cover

import "math"

// RenderBlur is the deliberate "cover it with blur" baseline: aliased
// center-sample coverage is composited into premultiplied RGBA, then blurred
// with a truncated Gaussian. Unlike real coverage the blur spreads ink
// outside the geometry, which lets disjoint contours stain each other.
// If exclude >= 0, that layer is omitted (used by stain measurements).
func RenderBlur(scene Scene, sigma float64, bg RGB, exclude int) (rgb []RGB, alpha []float64) {
	n := scene.W * scene.H
	pr := make([]float64, n)
	pg := make([]float64, n)
	pb := make([]float64, n)
	pa := make([]float64, n)

	for k, layer := range scene.Layers {
		if k == exclude {
			continue
		}
		cov := SampledLayer(layer, scene.W, scene.H, 1)
		for i, c := range cov {
			if c <= 0 {
				continue
			}
			// Premultiplied source-over.
			inv := 1 - c
			pr[i] = c*layer.Color.R + pr[i]*inv
			pg[i] = c*layer.Color.G + pg[i]*inv
			pb[i] = c*layer.Color.B + pb[i]*inv
			pa[i] = c + pa[i]*inv
		}
	}

	radius := int(math.Ceil(3 * sigma))
	if radius < 1 {
		radius = 1
	}
	kernel := gaussianKernel(sigma, radius)
	pr = blurSeparable(pr, scene.W, scene.H, kernel, radius)
	pg = blurSeparable(pg, scene.W, scene.H, kernel, radius)
	pb = blurSeparable(pb, scene.W, scene.H, kernel, radius)
	pa = blurSeparable(pa, scene.W, scene.H, kernel, radius)

	alpha = pa
	rgb = make([]RGB, n)
	for i := range rgb {
		a := clamp01(pa[i])
		// pr/pg/pb are already premultiplied source terms; composite over
		// the opaque background directly.
		rgb[i] = RGB{
			R: clamp01(pr[i]+bg.R*(1-a)) * 255,
			G: clamp01(pg[i]+bg.G*(1-a)) * 255,
			B: clamp01(pb[i]+bg.B*(1-a)) * 255,
		}
		alpha[i] = a
	}
	return rgb, alpha
}

func gaussianKernel(sigma float64, radius int) []float64 {
	k := make([]float64, 2*radius+1)
	sum := 0.0
	twoS2 := 2 * sigma * sigma
	for i := -radius; i <= radius; i++ {
		v := math.Exp(-float64(i*i) / twoS2)
		k[i+radius] = v
		sum += v
	}
	for i := range k {
		k[i] /= sum
	}
	return k
}

// blurSeparable applies the same 1-D kernel horizontally then vertically
// with clamp-to-edge extension.
func blurSeparable(src []float64, w, h int, k []float64, radius int) []float64 {
	tmp := make([]float64, len(src))
	dst := make([]float64, len(src))
	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			s := 0.0
			for o := -radius; o <= radius; o++ {
				xx := clampInt(x+o, 0, w-1)
				s += src[row+xx] * k[o+radius]
			}
			tmp[row+x] = s
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := 0.0
			for o := -radius; o <= radius; o++ {
				yy := clampInt(y+o, 0, h-1)
				s += tmp[yy*w+x] * k[o+radius]
			}
			dst[y*w+x] = s
		}
	}
	return dst
}
