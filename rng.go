package main

import "math"

// Rng is a deterministic splitmix64-based random source.
type Rng struct {
	state uint64
}

func newRng(seed uint64) *Rng {
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return &Rng{state: seed}
}

func (r *Rng) next() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (r *Rng) float() float64 {
	return float64(r.next()>>11) * 0x1.0p-53
}

// cosineSampleHemisphere draws a cosine-weighted direction around n.
func (r *Rng) cosineHemisphere(n Vec3) Vec3 {
	u1, u2 := r.float(), r.float()
	radius := math.Sqrt(u1)
	phi := 2 * math.Pi * u2
	x := radius * math.Cos(phi)
	y := radius * math.Sin(phi)
	z := math.Sqrt(math.Max(0, 1-u1))
	return basisFromNormal(n).toWorld(Vec3{x, y, z})
}

type basis struct{ t, b, n Vec3 }

func basisFromNormal(n Vec3) basis {
	var ref Vec3
	if math.Abs(n.Y) < 0.95 {
		ref = Vec3{0, 1, 0}
	} else {
		ref = Vec3{1, 0, 0}
	}
	t := cross(ref, n).Normalize()
	b := cross(n, t).Normalize()
	return basis{t: t, b: b, n: n}
}

func (bs basis) toWorld(local Vec3) Vec3 {
	return bs.t.Scale(local.X).Add(bs.b.Scale(local.Y)).Add(bs.n.Scale(local.Z))
}
