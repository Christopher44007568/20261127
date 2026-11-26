package main

import (
	"math"
	"math/rand"
)

type Hole struct {
	X      float64
	Y      float64
	R      float64
	Lobes  int
	Amount float64
	Phase  float64
}

func (h Hole) Inside(x, y float64) bool {
	dx := x - h.X
	dy := y - h.Y
	r := h.R
	if h.Lobes > 0 && h.Amount > 0 {
		angle := math.Atan2(dy, dx)
		r *= 1 + h.Amount*math.Sin(float64(h.Lobes)*angle+h.Phase)
	}
	return dx*dx+dy*dy <= r*r
}

type DemoScene struct {
	Name         string
	Cloud        *Cloud
	Holes        []Hole
	XMin         float64
	XMax         float64
	YMin         float64
	YMax         float64
	NoiseSigma   float64
	OutlierCount int
	OutlierStart int
}

func demoHeight(x, y float64) float64 {
	return 0.17*math.Sin(1.15*x+0.25)*math.Cos(1.28*y-0.35) +
		0.045*math.Sin(2.05*x+0.75*y) +
		0.025*math.Cos(1.7*y-0.4*x)
}

func demoNormal(x, y float64) V3 {
	e := 0.0015
	dx := (demoHeight(x+e, y) - demoHeight(x-e, y)) / (2 * e)
	dy := (demoHeight(x, y+e) - demoHeight(x, y-e)) / (2 * e)
	return Normalize(V3{-dx, -dy, 1})
}

func demoCurvature(x, y float64) float64 {
	e := 0.004
	fx := (demoHeight(x+e, y) - demoHeight(x-e, y)) / (2 * e)
	fy := (demoHeight(x, y+e) - demoHeight(x, y-e)) / (2 * e)
	fxx := (demoHeight(x+e, y) - 2*demoHeight(x, y) + demoHeight(x-e, y)) / (e * e)
	fyy := (demoHeight(x, y+e) - 2*demoHeight(x, y) + demoHeight(x, y-e)) / (e * e)
	fxy := (demoHeight(x+e, y+e) - demoHeight(x+e, y-e) -
		demoHeight(x-e, y+e) + demoHeight(x-e, y-e)) / (4 * e * e)
	den := 1 + fx*fx + fy*fy
	return ((1+fy*fy)*fxx - 2*fx*fy*fxy + (1+fx*fx)*fyy) /
		(2 * math.Pow(den, 1.5))
}

func NewDemoScene(seed int64) *DemoScene {
	rng := rand.New(rand.NewSource(seed))
	scene := &DemoScene{
		Name:       "noisy_holed_shell",
		XMin:       -1.75,
		XMax:       1.75,
		YMin:       -1.30,
		YMax:       1.30,
		NoiseSigma: 0.0065,
		Holes: []Hole{
			{X: -0.82, Y: 0.43, R: 0.285, Lobes: 3, Amount: 0.13, Phase: 0.7},
			{X: 0.78, Y: -0.47, R: 0.225, Lobes: 0},
			{X: 0.55, Y: 0.72, R: 0.165, Lobes: 5, Amount: 0.10, Phase: 1.8},
		},
	}

	const nx = 232
	const ny = 174
	dx := (scene.XMax - scene.XMin) / nx
	dy := (scene.YMax - scene.YMin) / ny
	cloud := &Cloud{HasN: true}

	for iy := 0; iy < ny; iy++ {
		for ix := 0; ix < nx; ix++ {
			x := scene.XMin + (float64(ix)+0.15+0.7*rng.Float64())*dx
			y := scene.YMin + (float64(iy)+0.15+0.7*rng.Float64())*dy
			if scene.InHole(x, y) {
				continue
			}
			z := demoHeight(x, y)
			n := demoNormal(x, y)
			noise := rng.NormFloat64() * scene.NoiseSigma
			p := V3{x, y, z}.Add(n.Mul(noise))

			perturbed := perturbNormal(n, rng, 0.11)
			if rng.Float64() < 0.43 {
				perturbed = perturbed.Mul(-1)
			}
			cloud.P = append(cloud.P, p)
			cloud.N = append(cloud.N, perturbed)
		}
	}

	zMin, zMax := math.Inf(1), math.Inf(-1)
	for ix := 0; ix <= 80; ix++ {
		x := lerpFloat(scene.XMin, scene.XMax, float64(ix)/80)
		for iy := 0; iy <= 60; iy++ {
			y := lerpFloat(scene.YMin, scene.YMax, float64(iy)/60)
			z := demoHeight(x, y)
			zMin = math.Min(zMin, z)
			zMax = math.Max(zMax, z)
		}
	}

	scene.OutlierCount = 280
	scene.OutlierStart = len(cloud.P)
	for i := 0; i < scene.OutlierCount; i++ {
		x := lerpFloat(scene.XMin-0.25, scene.XMax+0.25, rng.Float64())
		y := lerpFloat(scene.YMin-0.22, scene.YMax+0.22, rng.Float64())
		z := lerpFloat(zMin-0.42, zMax+0.42, rng.Float64())
		if rng.Float64() < 0.35 {
			// Tighter ghost points just away from the sheet test density filtering.
			x = lerpFloat(-1.4, 1.4, rng.Float64())
			y = lerpFloat(-1.0, 1.0, rng.Float64())
			z = demoHeight(x, y) + (rng.Float64()-0.5)*0.55 + signRandom(rng)*0.18
		}
		cloud.P = append(cloud.P, V3{x, y, z})
		cloud.N = append(cloud.N, V3{0, 0, 1})
	}
	scene.Cloud = cloud
	return scene
}

func signRandom(rng *rand.Rand) float64 {
	if rng.Float64() < 0.5 {
		return -1
	}
	return 1
}

func perturbNormal(n V3, rng *rand.Rand, amount float64) V3 {
	ref := V3{0, 0, 1}
	if math.Abs(n.Z) > 0.9 {
		ref = V3{1, 0, 0}
	}
	u := Normalize(Cross(ref, n))
	v := Cross(n, u)
	a := rng.NormFloat64() * amount
	b := rng.NormFloat64() * amount
	return Normalize(n.Add(u.Mul(a)).Add(v.Mul(b)))
}

func (s *DemoScene) InHole(x, y float64) bool {
	for _, h := range s.Holes {
		if h.Inside(x, y) {
			return true
		}
	}
	return false
}
