package main

import "math"

type ViewKind int

const (
	ViewFront ViewKind = iota
	ViewGrazing
	ViewFallback
)

type Scene struct {
	Name        string
	Kind        ViewKind
	Budget      int
	MaxDistance float64
}

type RenderStats struct {
	Pixels           int
	Hit              int
	BoundaryFallback int
	BudgetFallback   int
	AverageLayers    float64
}

type TraceStatus uint8

const (
	TraceHit TraceStatus = iota
	TraceBoundary
	TraceBudget
)

type TraceResult struct {
	Status TraceStatus
	U, V   float64
	Height float32
	GradX  float32
	GradY  float32
	Color  RGB
	Layers int
	Dist   float64
}

type vec3 struct {
	x, y, z float32
}

func (a vec3) dot(b vec3) float32 {
	return a.x*b.x + a.y*b.y + a.z*b.z
}

func (a vec3) scale(s float32) vec3 {
	return vec3{a.x * s, a.y * s, a.z * s}
}

func (a vec3) add(b vec3) vec3 {
	return vec3{a.x + b.x, a.y + b.y, a.z + b.z}
}

func (a vec3) sub(b vec3) vec3 {
	return vec3{a.x - b.x, a.y - b.y, a.z - b.z}
}

func normalize(v vec3) vec3 {
	length := float32(math.Sqrt(float64(v.x*v.x + v.y*v.y + v.z*v.z)))
	if length < 1e-6 {
		return vec3{0, 0, 1}
	}
	return v.scale(1 / length)
}

func inUnitUV(u, v float64) bool {
	return u >= 0 && u <= 1 && v >= 0 && v <= 1
}

func viewForScene(kind ViewKind, u, v float64) vec3 {
	var eye vec3
	switch kind {
	case ViewFront:
		eye = vec3{float32(0.48 + (u-0.5)*0.12), float32(0.40 + (v-0.5)*0.07), 2.25}
	case ViewGrazing:
		eye = vec3{float32(2.68 - 0.16*u), float32(0.62 + 0.08*(0.5-v)), 0.54}
	default:
		eye = vec3{float32(4.40 - 0.30*u), float32(0.55 + 0.10*(0.5-v)), 0.25}
	}
	point := vec3{float32(u), float32(v), 0}
	view := normalize(eye.sub(point))
	if view.z < 0.055 {
		view.z = 0.055
		view = normalize(view)
	}
	return view
}

func sceneLight(kind ViewKind) vec3 {
	if kind == ViewGrazing || kind == ViewFallback {
		return normalize(vec3{-0.48, -0.66, 0.58})
	}
	return normalize(vec3{-0.55, -0.62, 0.56})
}

func viewVisibility(tex *StoneTexture, u, v float64, height float32, view vec3) float32 {
	if view.x*view.x+view.y*view.y < 1e-5 {
		return 1
	}
	visibility := float32(1)
	const steps = 9
	lowMask := float32(1) - 0.62*height
	for i := 1; i <= steps; i++ {
		s := float32(i) * 0.022
		qu := u + float64(view.x*s)
		qv := v + float64(view.y*s)
		if !inUnitUV(qu, qv) {
			break
		}
		rayHeight := height + view.z*s
		terrain := tex.HeightAt(qu, qv)
		over := (terrain - rayHeight - 0.012) / 0.12
		if over > 0 {
			visibility *= float32(1 - 0.36*clamp01(float64(over))*clamp01(float64(lowMask)))
		}
	}
	visibility = float32(clamp01(float64(visibility)))
	return 0.46 + 0.54*visibility
}

func parallaxScale(kind ViewKind) float32 {
	switch kind {
	case ViewGrazing:
		return 0.095
	case ViewFallback:
		return 0.105
	default:
		return 0.075
	}
}

func traceParallax(tex *StoneTexture, u0, v0 float64, view vec3, kind ViewKind, budget int, maxDistance float64) TraceResult {
	scale := parallaxScale(kind)
	depthAt := func(u, v float64) float32 {
		height := tex.HeightAt(u, v)
		depth := (0.90 - height) * scale
		if depth < 0 {
			return 0
		}
		return depth
	}
	uvAt := func(t float64) (float64, float64) {
		return u0 - t*float64(view.x), v0 - t*float64(view.y)
	}
	gapAt := func(t float64) (float32, float64, float64, float32) {
		u, v := uvAt(t)
		surfaceDepth := depthAt(u, v)
		rayDepth := float32(t * float64(view.z))
		return surfaceDepth - rayDepth, u, v, surfaceDepth
	}

	lastU, lastV, lastHeight := u0, v0, tex.HeightAt(u0, v0)
	t := 0.0
	gap, u, v, _ := gapAt(t)
	if gap <= 0 {
		return finishTrace(tex, TraceHit, u, v, lastHeight, t, 0)
	}
	prevT := t
	maxSteps := budget
	if maxSteps < 1 {
		maxSteps = 1
	}
	for layer := 1; layer <= maxSteps; layer++ {
		t = prevT
		u, v = uvAt(t)
		if !inUnitUV(u, v) {
			result := finishTrace(tex, TraceBoundary, lastU, lastV, lastHeight, prevT, layer-1)
			result.Dist = prevT
			return result
		}
		gap, _, _, _ = gapAt(t)
		if gap <= 0 {
			return finishTrace(tex, TraceHit, u, v, tex.HeightAt(u, v), t, layer-1)
		}
		lastU, lastV, lastHeight = u, v, tex.HeightAt(u, v)

		gx, gy := tex.GradientAt(u, v)
		depressionSlope := scale * (gx*view.x + gy*view.y)
		denom := float32(view.z)
		if depressionSlope > 0 {
			denom += depressionSlope
		}
		if denom < 0.08 {
			denom = 0.08
		}
		step := float64(gap/denom) * 0.80
		if step < 0.0007 {
			step = 0.0007
		}
		if step > 0.0040 {
			step = 0.0040
		}
		nextT := t + step
		if nextT > maxDistance {
			nextT = maxDistance
		}
		nextGap, _, _, _ := gapAt(nextT)
		if nextGap <= 0 {
			hitU, hitV, hitT, ok := refineParallaxHit(gapAt, uvAt, prevT, nextT)
			if !ok || !inUnitUV(hitU, hitV) || hitT > maxDistance {
				result := finishTrace(tex, TraceBoundary, lastU, lastV, lastHeight, prevT, layer)
				result.Dist = prevT
				return result
			}
			return finishTrace(tex, TraceHit, hitU, hitV, tex.HeightAt(hitU, hitV), hitT, layer)
		}
		prevT = nextT
	}
	result := finishTrace(tex, TraceBudget, lastU, lastV, lastHeight, prevT, maxSteps)
	result.Dist = prevT
	return result
}

func refineParallaxHit(gapAt func(float64) (float32, float64, float64, float32), uvAt func(float64) (float64, float64), lower, upper float64) (float64, float64, float64, bool) {
	lo, hi := lower, upper
	gLo, _, _, _ := gapAt(lo)
	gHi, _, _, _ := gapAt(hi)
	if gLo <= 0 || gHi > 0 {
		return 0, 0, 0, false
	}
	for i := 0; i < 5; i++ {
		alpha := float64(gLo / (gLo - gHi))
		if alpha < 0.04 {
			alpha = 0.04
		} else if alpha > 0.96 {
			alpha = 0.96
		}
		mid := lo + (hi-lo)*alpha
		gMid, _, _, _ := gapAt(mid)
		if gMid > 0 {
			lo, gLo = mid, gMid
		} else {
			hi, gHi = mid, gMid
		}
	}
	u, v := uvAt((lo + hi) * 0.5)
	return u, v, (lo + hi) * 0.5, true
}

func finishTrace(tex *StoneTexture, status TraceStatus, u, v float64, height float32, dist float64, layers int) TraceResult {
	gx, gy := tex.GradientAt(u, v)
	return TraceResult{
		Status: status, U: u, V: v, Height: height,
		GradX: gx, GradY: gy, Color: tex.ColorAt(u, v),
		Layers: layers, Dist: dist,
	}
}
