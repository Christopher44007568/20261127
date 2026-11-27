package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) MulVec(b Vec3) Vec3 { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}
func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l > 1e-12 {
		return a.Mul(1 / l)
	}
	return Vec3{}
}
func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Mul(t)) }
func mix(a, b, t float64) float64          { return a + (b-a)*t }
func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
func saturate(x float64) float64 { return clamp(x, 0, 1) }
func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := saturate((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}
func smootherstep(edge0, edge1, x float64) float64 {
	t := saturate((x - edge0) / (edge1 - edge0))
	return t * t * t * (t*(t*6-15) + 10)
}

func hash2(x, y int32) float64 {
	n := uint32(x)*0x7feb352d ^ uint32(y)*0x846ca68b
	n ^= n >> 15
	n *= 0x2c1b3c6d
	n ^= n >> 12
	n *= 0x297a2d39
	n ^= n >> 15
	return float64(n) / float64(^uint32(0))
}

func valueNoise(x, y float64) float64 {
	ix := int32(math.Floor(x))
	iy := int32(math.Floor(y))
	fx := x - float64(ix)
	fy := y - float64(iy)
	ux := fx * fx * (3 - 2*fx)
	uy := fy * fy * (3 - 2*fy)
	a := hash2(ix, iy)
	b := hash2(ix+1, iy)
	c := hash2(ix, iy+1)
	d := hash2(ix+1, iy+1)
	return mix(mix(a, b, ux), mix(c, d, ux), uy)
}

func fbm2(x, y float64, octaves int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * valueNoise(x, y)
		norm += amp
		x = x*2.03 + 17.17
		y = y*2.01 - 9.31
		amp *= 0.5
	}
	return sum / norm
}

func triplanarWeights(n Vec3, sharpness float64) [3]float64 {
	w := [3]float64{
		math.Pow(math.Abs(n.X), sharpness),
		math.Pow(math.Abs(n.Y), sharpness),
		math.Pow(math.Abs(n.Z), sharpness),
	}
	sum := w[0] + w[1] + w[2]
	if sum < 1e-9 {
		return [3]float64{0, 1, 0}
	}
	w[0] /= sum
	w[1] /= sum
	w[2] /= sum
	return w
}

func sampleTriplanar(p, n Vec3, scale, seed float64, sample func(x, y float64) float64) float64 {
	w := triplanarWeights(n, 5)
	a := sample(p.Z*scale+seed, p.Y*scale)
	b := sample(p.X*scale+seed, p.Z*scale)
	c := sample(p.X*scale+seed, p.Y*scale)
	return a*w[0] + b*w[1] + c*w[2]
}
