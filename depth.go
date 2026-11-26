package main

import (
	"math"
	"math/rand"
)

type DepthFrame struct {
	Index      int
	Label      string
	Width      int
	Height     int
	True       Camera
	Fusion     Camera
	Depth      []float64
	Confidence []float32
	Normal     []Vec3
	HoleMask   []bool
	HolePixels int
}

type HoleBlob struct {
	CX, CY, RX, RY float64
}

func RenderDepthFrame(index int, label string, geom Geometry, trueCam, fusionCam Camera, seed int64) *DepthFrame {
	w, h := trueCam.Width, trueCam.Height
	f := &DepthFrame{
		Index:      index,
		Label:      label,
		Width:      w,
		Height:     h,
		True:       trueCam,
		Fusion:     fusionCam,
		Depth:      make([]float64, w*h),
		Confidence: make([]float32, w*h),
		Normal:     make([]Vec3, w*h),
		HoleMask:   make([]bool, w*h),
	}
	rng := rand.New(rand.NewSource(seed))
	blobs := make([]HoleBlob, 0, 2)
	for b := 0; b < 1+index%2; b++ {
		blobs = append(blobs, HoleBlob{
			CX: rng.Float64()*0.7 + 0.15,
			CY: rng.Float64()*0.65 + 0.18,
			RX: 0.018 + rng.Float64()*0.026,
			RY: 0.022 + rng.Float64()*0.032,
		})
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			id := y*w + x
			ray := trueCam.RayAtPixel(x, y)
			t, _, _, hit := geom.Intersect(ray)
			if !hit {
				continue
			}
			p := ray.Origin.Add(ray.Dir.Mul(t))
			z := p.Sub(trueCam.Eye).Dot(trueCam.Forward)
			sigma := 0.0012 + 0.0014*z
			z += rng.NormFloat64() * sigma
			if z <= trueCam.Near || z >= trueCam.Far {
				continue
			}

			relX := (float64(x) + 0.5) / float64(w)
			relY := (float64(y) + 0.5) / float64(h)
			hole := rng.Float64() < 0.025
			for _, blob := range blobs {
				dx := (relX - blob.CX) / blob.RX
				dy := (relY - blob.CY) / blob.RY
				if dx*dx+dy*dy < 1 {
					hole = true
					break
				}
			}
			if hole {
				f.HoleMask[id] = true
				f.HolePixels++
				continue
			}
			f.Depth[id] = z
			// Normals are recovered from depth gradients below rather
			// than supplied by the analytic renderer.
		}
	}
	f.buildConfidence()
	return f
}

func (f *DepthFrame) valid(x, y int) bool {
	return x >= 0 && x < f.Width && y >= 0 && y < f.Height && f.Depth[y*f.Width+x] > 0
}

func (f *DepthFrame) buildConfidence() {
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			id := y*f.Width + x
			z := f.Depth[id]
			if z <= 0 {
				f.Confidence[id] = 0
				continue
			}

			conf := 0.85 * math.Exp(-0.07*z)
			normal := f.Normal[id]
			if normal.Len() == 0 {
				normal = f.estimateNormal(x, y)
				f.Normal[id] = normal
			}
			rayDir := f.Fusion.RayAtPixel(x, y).Dir
			if normal.Len() > 0 {
				incidence := clampFloat(-normal.Dot(rayDir), 0, 1)
				conf *= 0.25 + 0.75*incidence*incidence
			} else {
				conf *= 0.45
			}

			edgePenalty := f.depthDiscontinuity(x, y, z)
			conf *= edgePenalty
			f.Confidence[id] = float32(clampFloat(conf, 0.05, 1.2))
		}
	}
}

func (f *DepthFrame) estimateNormal(x, y int) Vec3 {
	z := f.Depth[y*f.Width+x]
	if z <= 0 {
		return Vec3{}
	}
	var dx, dy Vec3
	const normalStep = 2
	if f.valid(x-normalStep, y) && f.valid(x+normalStep, y) {
		zL := f.Depth[y*f.Width+x-normalStep]
		zR := f.Depth[y*f.Width+x+normalStep]
		if math.Abs(zL-z) < 0.06*z && math.Abs(zR-z) < 0.06*z {
			dx = f.Fusion.UnprojectDepth(x+normalStep, y, zR).Sub(f.Fusion.UnprojectDepth(x-normalStep, y, zL))
		}
	}
	if f.valid(x, y-normalStep) && f.valid(x, y+normalStep) {
		zU := f.Depth[(y-normalStep)*f.Width+x]
		zD := f.Depth[(y+normalStep)*f.Width+x]
		if math.Abs(zU-z) < 0.06*z && math.Abs(zD-z) < 0.06*z {
			dy = f.Fusion.UnprojectDepth(x, y+normalStep, zD).Sub(f.Fusion.UnprojectDepth(x, y-normalStep, zU))
		}
	}
	if dx.Len() < 1e-8 || dy.Len() < 1e-8 {
		return Vec3{}
	}
	n := dy.Cross(dx).Norm()
	rayDir := f.Fusion.RayAtPixel(x, y).Dir
	if n.Dot(rayDir) > 0 {
		n = n.Mul(-1)
	}
	return n
}

func (f *DepthFrame) depthDiscontinuity(x, y int, z float64) float64 {
	bad := 0
	checked := 0
	for _, o := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		nx, ny := x+o[0], y+o[1]
		if !f.valid(nx, ny) {
			continue
		}
		checked++
		nz := f.Depth[ny*f.Width+nx]
		if math.Abs(nz-z) > 0.045+0.035*z {
			bad++
		}
	}
	if checked == 0 || bad >= 2 {
		return 0.35
	}
	if bad == 1 {
		return 0.7
	}
	return 1
}

func (f *DepthFrame) ConfidenceAt(x, y int) float64 {
	if x < 0 || x >= f.Width || y < 0 || y >= f.Height {
		return 0
	}
	return float64(f.Confidence[y*f.Width+x])
}
