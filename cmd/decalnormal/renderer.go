package main

import "math"

// Scene accumulates every view-dependent output in a single ray pass.
type Scene struct {
	Camera       Camera
	Decals       []Decal
	Base         *FloatImage
	DecalLayer   *FloatImage
	Final        *FloatImage
	Naive        *FloatImage
	MaxDevAll    float64
	MaxDevOver   float64
	OverlapCount int
	MinX         []int
	MaxX         []int
}

func newScene(cam Camera, decals []Decal) *Scene {
	return &Scene{
		Camera:     cam,
		Decals:     decals,
		Base:       newFloatImage(cam.WidthPx, cam.HeightPx),
		DecalLayer: newFloatImage(cam.WidthPx, cam.HeightPx),
		Final:      newFloatImage(cam.WidthPx, cam.HeightPx),
		Naive:      newFloatImage(cam.WidthPx, cam.HeightPx),
		MinX:       make([]int, len(decals)),
		MaxX:       make([]int, len(decals)),
	}
}

// render traces every pixel once and fills all output images plus audits.
func (s *Scene) render() {
	for i := range s.MinX {
		s.MinX[i] = s.Camera.WidthPx
	}
	bg := v3(0.035, 0.045, 0.07)
	neutral := normalColor(v3(0, 0, 1))
	for py := 0; py < s.Camera.HeightPx; py++ {
		for px := 0; px < s.Camera.WidthPx; px++ {
			row := s.Camera.HeightPx - 1 - py
			origin, dir := s.Camera.ray(px, py)
			p, ok := castRay(origin, dir)
			if !ok {
				s.Base.set(px, row, bg)
				s.DecalLayer.set(px, row, neutral)
				s.Final.set(px, row, bg)
				s.Naive.set(px, row, bg)
				continue
			}
			frame := frameAt(p.X, p.Y)
			res := evaluate(p, frame, s.Decals)

			s.Base.set(px, row, shade(frame.N, baseAlbedo))

			layerColor := neutral
			for idx, d := range s.Decals {
				pr := d.project(p)
				if pr.Mask <= 0 {
					continue
				}
				tn := d.detailNormal(pr.U, pr.V)
				layerColor = Lerp(layerColor, normalColor(tn), pr.Mask)
				if px < s.MinX[idx] {
					s.MinX[idx] = px
				}
				if px > s.MaxX[idx] {
					s.MaxX[idx] = px
				}
			}
			s.DecalLayer.set(px, row, layerColor)
			s.Final.set(px, row, shade(res.rnmN, res.albedo))
			s.Naive.set(px, row, shade(res.linN, res.albedo))

			dev := math.Abs(res.rnmN.Length() - 1)
			if dev > s.MaxDevAll {
				s.MaxDevAll = dev
			}
			if res.overlap {
				s.OverlapCount++
				if dev > s.MaxDevOver {
					s.MaxDevOver = dev
				}
			}
		}
	}
}

// footprintWidths returns the rendered mask width in screen pixels.
func (s *Scene) footprintWidths() []float64 {
	out := make([]float64, len(s.Decals))
	for i := range s.Decals {
		if s.MinX[i] <= s.MaxX[i] {
			out[i] = float64(s.MaxX[i] - s.MinX[i] + 1)
		}
	}
	return out
}

// writeComparison builds a side-by-side image: naive linear average on the
// left, reoriented normal mapping on the right, separated by a dark seam.
func (s *Scene) writeComparison(path string) error {
	w, h := s.Camera.WidthPx, s.Camera.HeightPx
	out := newFloatImage(w, h)
	seam := v3(0.02, 0.02, 0.02)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if x == w/2 {
				out.set(x, y, seam)
				continue
			}
			if x < w/2 {
				out.R[i], out.G[i], out.B[i] = s.Naive.R[i], s.Naive.G[i], s.Naive.B[i]
			} else {
				out.R[i], out.G[i], out.B[i] = s.Final.R[i], s.Final.G[i], s.Final.B[i]
			}
		}
	}
	out.drawText("LINEAR AVERAGE", 0, w/2, 12, v3(0.95, 0.95, 0.95))
	out.drawText("REORIENTED (RNM)", w/2, w, 12, v3(0.95, 0.95, 0.95))
	return out.writePNG(path)
}

// surfaceResult collects everything needed for shading one surface sample.
type surfaceResult struct {
	rnmN    Vec3
	linN    Vec3
	albedo  Vec3
	overlap bool
}

// evaluate runs all decal projectors against one surface point.
func evaluate(p Vec3, frame Frame, decals []Decal) surfaceResult {
	res := surfaceResult{rnmN: frame.N, albedo: baseAlbedo}

	var linSum Vec3
	linCount := 0.0
	for _, d := range decals {
		pr := d.project(p)
		if pr.Mask <= 0 {
			continue
		}
		// Reoriented normal mapping, accumulated per decal.
		tn := d.detailNormal(pr.U, pr.V)
		res.rnmN = blendDetail(res.rnmN, d, pr)
		// Naive path for the comparison image: average world-space normals.
		linSum = linSum.Add(reorient(frame.N, tn))
		linCount++

		tint := Lerp(baseAlbedo, d.Tint, pr.Mask*0.85)
		res.albedo = Lerp(res.albedo, tint, pr.Mask)
	}
	res.overlap = linCount >= 2
	if linCount > 0 {
		linSum = linSum.Add(frame.N)
		res.linN = linSum.Scale(1 / (linCount + 1)).Normalized()
	} else {
		res.linN = frame.N
	}
	return res
}

var baseAlbedo = v3(0.62, 0.64, 0.68)
