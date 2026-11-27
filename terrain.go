package main

import "math"

const (
	Mud = iota
	Moss
	WetStone
	numMaterials
)

const (
	waterLevel = -0.055
	worldMinX  = -5.5
	worldMaxX  = 5.5
	worldMinZ  = -7.0
	worldMaxZ  = 3.2
)

var materialNames = [numMaterials]string{"MUD", "MOSS", "WET STONE"}
var materialColors = [numMaterials]Vec3{
	{0.43, 0.30, 0.18},
	{0.25, 0.42, 0.14},
	{0.42, 0.49, 0.52},
}

type Rock struct {
	X, Z      float64
	RX, RZ    float64
	Amp, Seed float64
}

var rocks = []Rock{
	{X: -0.610, Z: 0.130, RX: 0.475, RZ: 0.365, Amp: 0.285, Seed: 1.7},
	{X: 0.900, Z: -0.420, RX: 0.420, RZ: 0.330, Amp: 0.245, Seed: 4.2},
	{X: -1.240, Z: -0.690, RX: 0.330, RZ: 0.275, Amp: 0.225, Seed: 7.8},
	{X: 0.650, Z: 0.930, RX: 0.320, RZ: 0.250, Amp: 0.205, Seed: 9.4},
	{X: -1.350, Z: 0.970, RX: 0.410, RZ: 0.320, Amp: 0.255, Seed: 12.1},
	{X: 1.420, Z: 0.690, RX: 0.350, RZ: 0.430, Amp: 0.235, Seed: 14.8},
}

func streamCenter(z float64) float64 {
	return 0.090 + 0.195*math.Sin(1.21*z+0.18) + 0.055*math.Sin(2.63*z+0.72)
}

func rockField(x, z float64, r Rock) float64 {
	dx := (x - r.X) / r.RX
	dz := (z - r.Z) / r.RZ
	q := math.Sqrt(dx*dx + dz*dz)
	warp := (fbm2(x*3.2+r.Seed, z*3.2-r.Seed, 3) - 0.5) * 0.16
	q += warp
	dome := 1 - smoothstep(0.58, 1.08, q)
	rough := 0.90 + 0.10*fbm2(x*8+r.Seed, z*8-r.Seed, 3)
	return r.Amp * dome * rough
}

func terrainHeight(x, z float64) float64 {
	d := math.Abs(x - streamCenter(z))
	bank := smoothstep(0.10, 1.38, d)
	h := -0.46 + 0.82*bank
	h += 0.060 * fbm2(x*1.35+4.3, z*1.35-2.1, 4)
	h += 0.030 * (fbm2(x*4.7-7.2, z*4.7+1.4, 3) - 0.5)
	h += 0.035 * (1 - smoothstep(0.05, 0.62, d)) * math.Sin(z*10.3+x*2.1)
	h += 0.11 * smoothstep(1.5, 4.4, math.Abs(x)) * (0.40 + 0.60*fbm2(x*0.55, z*0.55, 3))
	for _, rock := range rocks {
		h += rockField(x, z, rock)
	}
	return h
}

func terrainNormal(x, z, eps float64) Vec3 {
	dx := terrainHeight(x+eps, z) - terrainHeight(x-eps, z)
	dz := terrainHeight(x, z+eps) - terrainHeight(x, z-eps)
	return Vec3{-dx / (2 * eps), 1, -dz / (2 * eps)}.Normalize()
}

func stoneCoverage(x, z float64) float64 {
	d := math.Abs(x - streamCenter(z))
	bedPebbles := 0.30 * (1 - smoothstep(0.20, 0.62, d))
	best := bedPebbles
	for _, rock := range rocks {
		dx := (x - rock.X) / rock.RX
		dz := (z - rock.Z) / rock.RZ
		q := math.Sqrt(dx*dx + dz*dz)
		q += (fbm2(x*4.4+rock.Seed, z*4.4-rock.Seed, 3) - 0.5) * 0.18
		mask := 1 - smoothstep(0.72, 1.19, q)
		if mask > best {
			best = mask
		}
	}
	return saturate(best)
}

func coverageAt(x, z float64) [numMaterials]float64 {
	d := math.Abs(x - streamCenter(z))
	shore := smoothstep(0.18, 0.52, d) * (1 - smoothstep(1.00, 1.34, d))
	mudNoise := fbm2(x*2.30+11.2, z*2.30-8.7, 4)
	mud := shore * (0.35 + 0.80*mudNoise)

	mossNoise := fbm2(x*2.05-3.7, z*2.05+6.1, 4)
	moss := smoothstep(0.68, 1.22, d) * smoothstep(0.28, 0.62, mossNoise) * 1.15
	moss += 0.18 * (1 - smoothstep(0.88, 1.28, d))

	stone := stoneCoverage(x, z)
	h := terrainHeight(x, z)
	highRockMoss := stone * smoothstep(0.20, 0.42, h) * smoothstep(0.43, 0.62, mossNoise)
	moss += 0.55 * highRockMoss

	var c [numMaterials]float64
	c[Mud] = mud + 0.018
	c[Moss] = moss + 0.018
	c[WetStone] = stone + 0.018
	sum := c[0] + c[1] + c[2]
	for i := range c {
		c[i] /= sum
	}
	return c
}

func wetnessAt(x, z, height float64) float64 {
	// A narrow capillary band keeps stone visibly wet above the actual waterline.
	above := height - waterLevel
	band := 1 - smoothstep(0.0, 0.135, above)
	channel := 0.78 + 0.22*fbm2(x*3.1-4, z*3.1+8, 3)
	return saturate(band * channel)
}
