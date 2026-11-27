package main

import "math"

type ComparisonView struct {
	Name    string
	Width   int
	Height  int
	Radius  float64
	CenterY float64
	Centers []float64
}

type Scene struct {
	Texture *RoughnessTexture
	LUT     *EnergyLUT
	Lights  []Light
	F0      Vec3
}

type Light struct {
	Direction Vec3
	Color     Vec3
	Radiance  float64
}

func DefaultLights() []Light {
	return []Light{
		{Direction: V(-0.50, 0.38, 0.78).Normalize(), Color: V(1.00, 0.84, 0.62), Radiance: 0.95},
		{Direction: V(0.06, 0.66, 0.75).Normalize(), Color: V(1.00, 0.98, 0.94), Radiance: 1.25},
		{Direction: V(0.66, 0.08, 0.75).Normalize(), Color: V(0.42, 0.82, 1.00), Radiance: 1.05},
		{Direction: V(-0.52, -0.24, 0.82).Normalize(), Color: V(0.42, 0.58, 1.00), Radiance: 0.72},
		{Direction: V(0.05, -0.72, 0.69).Normalize(), Color: V(0.20, 0.34, 0.62), Radiance: 0.16},
	}
}

func NearView() ComparisonView {
	return ComparisonView{
		Name: "near", Width: 1500, Height: 640, Radius: 208, CenterY: 326,
		Centers: []float64{322, 750, 1178},
	}
}

func FarView() ComparisonView {
	return ComparisonView{
		Name: "far", Width: 1500, Height: 640, Radius: 148, CenterY: 326,
		Centers: []float64{322, 750, 1178},
	}
}

func PreviewView() ComparisonView {
	return ComparisonView{
		Name: "preview", Width: 1500, Height: 640, Radius: 180, CenterY: 322,
		Centers: []float64{322, 750, 1178},
	}
}

func RenderComparison(view ComparisonView, scene *Scene) *Frame {
	frame := NewFrame(view.Width, view.Height)
	drawBackground(frame)
	for i, cx := range view.Centers {
		drawContactShadow(frame, cx, view.CenterY+view.Radius*0.91, view.Radius)
		mode := []RoughnessMode{ModeOriginal, ModeNaiveMIP, ModeFootprint}[i]
		renderSphere(frame, scene, cx, view.CenterY, view.Radius, mode)
	}
	frame.Bloom(1.65, 0.30, 2)
	return frame
}

func drawBackground(f *Frame) {
	top := srgbColor(19, 33, 52)
	mid := srgbColor(18, 32, 50)
	bottom := srgbColor(9, 13, 20)
	glow := srgbColor(34, 61, 91)
	for y := 0; y < f.Height; y++ {
		t := float64(y) / float64(f.Height-1)
		var base Vec3
		if t < 0.58 {
			base = top.Mul(1 - t/0.58).Add(mid.Mul(t / 0.58))
		} else {
			k := (t - 0.58) / 0.42
			base = mid.Mul(1 - k).Add(bottom.Mul(k))
		}
		for x := 0; x < f.Width; x++ {
			dx := (float64(x) - 750) / 760
			dy := (float64(y) - 150) / 520
			d := dx*dx + dy*dy
			g := math.Exp(-d * 2.35)
			vignette := 1 - 0.28*smoothstepFloat(0.45, 1.35, math.Sqrt(d))
			c := base.Mul(vignette).Add(glow.Mul(0.16 * g))
			f.Set(x, y, c)
		}
	}
}

func srgbColor(r, g, b uint8) Vec3 {
	return V(srgbToLinear8(r), srgbToLinear8(g), srgbToLinear8(b))
}

func drawContactShadow(f *Frame, cx, cy, radius float64) {
	rx := radius * 0.88
	ry := radius * 0.17
	x0 := int(math.Floor(cx - rx*1.25))
	x1 := int(math.Ceil(cx + rx*1.25))
	y0 := int(math.Floor(cy - ry*2.2))
	y1 := int(math.Ceil(cy + ry*2.2))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := (float64(x) + 0.5 - cx) / rx
			dy := (float64(y) + 0.5 - cy) / ry
			d := dx*dx + dy*dy
			if d > 2.4 {
				continue
			}
			alpha := 0.48 * math.Exp(-d*1.35)
			i := f.At(x, y)
			f.Pix[i] = f.Pix[i].Mul(1 - alpha)
		}
	}
}

func normalAtSlope(sx, sy float64) Vec3 {
	q := sx*sx + sy*sy
	nz := math.Sqrt(math.Max(0, 1-q))
	n := V(sx, -sy, nz)
	if q > 1 {
		n = n.Normalize()
	}
	return n
}

func sphereUV(n Vec3) (float64, float64) {
	u := 0.5 + math.Atan2(n.X, n.Z)/(2*math.Pi)
	v := 0.5 - math.Asin(clampFloat(n.Y, -1, 1))/math.Pi
	const scaleU = 1.42
	const offsetU = -0.355
	u = u*scaleU + offsetU
	return u, v
}

func renderSphere(f *Frame, scene *Scene, cx, cy, radius float64, mode RoughnessMode) {
	x0 := int(math.Floor(cx - radius - 1))
	x1 := int(math.Ceil(cx + radius + 1))
	y0 := int(math.Floor(cy - radius - 1))
	y1 := int(math.Ceil(cy + radius + 1))
	step := 1 / radius

	for y := y0; y <= y1; y++ {
		py := float64(y) + 0.5
		sy := (py - cy) / radius
		for x := x0; x <= x1; x++ {
			px := float64(x) + 0.5
			sx := (px - cx) / radius
			dist := math.Hypot(px-cx, py-cy)
			coverage := clampFloat(radius-dist+0.5, 0, 1)
			if coverage <= 0 {
				continue
			}
			if x < 0 || x >= f.Width || y < 0 || y >= f.Height {
				continue
			}

			n := normalAtSlope(sx, sy)
			u, v := sphereUV(n)
			uXP, vXP := sphereUV(normalAtSlope(sx+step, sy))
			uXM, vXM := sphereUV(normalAtSlope(sx-step, sy))
			uYP, vYP := sphereUV(normalAtSlope(sx, sy+step))
			uYM, vYM := sphereUV(normalAtSlope(sx, sy-step))
			duDx := wrapDelta(uXP-uXM) / 2
			dvDx := (vXP - vXM) / 2
			duDy := wrapDelta(uYP-uYM) / 2
			dvDy := (vYP - vYM) / 2
			fp := FootprintFromDerivatives(duDx, dvDx, duDy, dvDy, float64(scene.Texture.Size))

			dNdx := normalAtSlope(sx+step, sy).Sub(normalAtSlope(sx-step, sy)).Mul(0.5)
			dNdy := normalAtSlope(sx, sy+step).Sub(normalAtSlope(sx, sy-step)).Mul(0.5)
			normalVariance := (dNdx.Dot(dNdx) + dNdy.Dot(dNdy)) / 12

			rough := scene.Texture.Resolve(mode, u, v, fp)
			if mode == ModeFootprint {
				rough.AddGeometryVariance(normalVariance * 0.35)
			}

			viewDir := V(0, 0, 1)
			color := ShadeSurface(scene, mode, n, viewDir, u, v, rough, fp)
			old := f.Pix[f.At(x, y)]
			f.Pix[f.At(x, y)] = old.Mul(1 - coverage).Add(color.Mul(coverage))
		}
	}
}

func wrapDelta(d float64) float64 {
	if d > 0.5 {
		return d - 1
	}
	if d < -0.5 {
		return d + 1
	}
	return d
}
