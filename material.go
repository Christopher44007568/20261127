package main

import "math"

type MaterialSample struct {
	Albedo   Vec3
	Height   float64
	Rough    float64
	Specular float64
}

func sampleTP(p Vec3, weights [3]float64, scale, seed float64, sample func(x, y float64) float64) float64 {
	a := sample(p.Z*scale+seed, p.Y*scale-seed*0.31)
	b := sample(p.X*scale-seed*0.17, p.Z*scale+seed)
	c := sample(p.X*scale+seed*1.7, p.Y*scale+seed*0.43)
	return a*weights[0] + b*weights[1] + c*weights[2]
}

func sampleMaterial(mat int, p Vec3, n Vec3) MaterialSample {
	tw := triplanarWeights(n, 5)
	switch mat {
	case Mud:
		macro := sampleTP(p, tw, 5.4, 2.3, func(x, y float64) float64 {
			return fbm2(x, y, 3)
		})
		fine := sampleTP(p, tw, 28.0, 7.1, func(x, y float64) float64 {
			return valueNoise(x, y)
		})
		crack := 1 - smoothstep(0.012, 0.070, math.Abs(macro-0.50))
		t := smoothstep(0.22, 0.82, macro)
		albedo := Vec3{0.105, 0.068, 0.039}.Lerp(Vec3{0.270, 0.190, 0.105}, t)
		albedo = albedo.Mul(0.78 + 0.30*fine)
		albedo = albedo.Mul(1 - 0.32*crack)
		return MaterialSample{
			Albedo: albedo, Height: 0.42 + 0.40*macro + 0.14*fine - 0.24*crack,
			Rough: 0.82, Specular: 0.10,
		}
	case Moss:
		macro := sampleTP(p, tw, 11.0, 13.7, func(x, y float64) float64 {
			return fbm2(x, y, 3)
		})
		fine := sampleTP(p, tw, 54.0, 19.2, func(x, y float64) float64 {
			return valueNoise(x, y)
		})
		tips := smoothstep(0.76, 0.94, fine)
		t := smoothstep(0.18, 0.84, macro)
		albedo := Vec3{0.022, 0.092, 0.016}.Lerp(Vec3{0.115, 0.270, 0.043}, t)
		albedo = albedo.Add(Vec3{0.100, 0.140, 0.018}.Mul(tips * 0.48))
		return MaterialSample{
			Albedo: albedo, Height: 0.46 + 0.34*macro + 0.26*fine,
			Rough: 0.93, Specular: 0.045,
		}
	default:
		strata := sampleTP(p, tw, 3.2, 23.6, func(x, y float64) float64 {
			return fbm2(x, y, 3)
		})
		fine := sampleTP(p, tw, 31.0, 31.9, func(x, y float64) float64 {
			return valueNoise(x, y)
		})
		bands := 0.5 + 0.5*math.Sin(p.Y*19.0+strata*7.0+p.X*1.7)
		crack := 1 - smoothstep(0.014, 0.085, math.Abs(strata-0.50))
		t := smoothstep(0.20, 0.82, strata)
		albedo := Vec3{0.075, 0.105, 0.125}.Lerp(Vec3{0.255, 0.315, 0.350}, t)
		albedo = albedo.Mul(0.82 + 0.12*bands + 0.12*fine)
		albedo = albedo.Mul(1 - 0.42*crack)
		return MaterialSample{
			Albedo: albedo, Height: 0.44 + 0.34*strata + 0.20*fine + 0.08*bands - 0.32*crack,
			Rough: 0.34, Specular: 0.58,
		}
	}
}

func normalizeWeights(w [numMaterials]float64) [numMaterials]float64 {
	sum := w[0] + w[1] + w[2]
	if sum < 1e-12 {
		return [numMaterials]float64{1.0 / 3, 1.0 / 3, 1.0 / 3}
	}
	for i := range w {
		w[i] /= sum
		if w[i] < 0 {
			w[i] = 0
		}
	}
	return w
}

func naiveWeights(coverage [numMaterials]float64) [numMaterials]float64 {
	return normalizeWeights(coverage)
}

func heightWeights(coverage [numMaterials]float64, samples [numMaterials]MaterialSample) [numMaterials]float64 {
	// The last term approximates physical layer order: stone > moss > mud.
	// Color is taken from these weights, but the geometric height is never used
	// as an output of this function.
	bias := [numMaterials]float64{0.00, 0.018, 0.042}
	layer := [numMaterials]float64{}
	for i := range layer {
		layer[i] = 0.39 + 0.42*samples[i].Height + 0.19*coverage[i] + bias[i]
	}
	maxLayer := layer[0]
	for i := 1; i < numMaterials; i++ {
		if layer[i] > maxLayer {
			maxLayer = layer[i]
		}
	}
	var w [numMaterials]float64
	for i := range w {
		w[i] = coverage[i] * math.Exp(18.0*(layer[i]-maxLayer))
	}
	return normalizeWeights(w)
}
