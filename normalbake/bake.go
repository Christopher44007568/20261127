package main

import "math"

type BakeMethod int

const (
	MethodNearest BakeMethod = iota // bidirectional nearest-surface projection
	MethodCage                      // cast from inflated cage along -N
)

type BakeResult struct {
	W, H    int
	Nrm     []Vec3 // tangent-space normal per texel, default (0,0,1)
	Covered []bool // texel covered by a UV triangle
	Mirror  []bool // texel belongs to a mirrored (negative-winding) UV triangle
	Occl    []bool // ray was blocked by other low-poly geometry (self-occlusion)
	Zero    []bool // no high-poly hit; normal stays neutral
	hasHit  []bool // a valid hit was written to this texel
}

func NewBakeResult(w, h int) *BakeResult {
	n := w * h
	r := &BakeResult{
		W: w, H: h,
		Nrm:     make([]Vec3, n),
		Covered: make([]bool, n),
		Mirror:  make([]bool, n),
		Occl:    make([]bool, n),
		Zero:    make([]bool, n),
		hasHit:  make([]bool, n),
	}
	for i := range r.Nrm {
		r.Nrm[i] = Vec3{0, 0, 1} // neutral normal
	}
	return r
}

// Bake rasterizes the low-poly UV space and casts rays against the high-poly
// mesh, storing tangent-space normals. Zero-hit texels keep the neutral
// (0,0,1) normal.
func Bake(low, high *Mesh, lowBVH, highBVH *BVH, res int, method BakeMethod, cage, maxDist float64) *BakeResult {
	r := NewBakeResult(res, res)
	for ti, tri := range low.Tris {
		i0, i1, i2 := tri.I[0], tri.I[1], tri.I[2]
		uv0, uv1, uv2 := low.UV[i0], low.UV[i1], low.UV[i2]
		p0, p1, p2 := low.Pos[i0], low.Pos[i1], low.Pos[i2]
		n0, n1, n2 := low.Nrm[i0], low.Nrm[i1], low.Nrm[i2]

		det := (uv1.X-uv0.X)*(uv2.Y-uv0.Y) - (uv2.X-uv0.X)*(uv1.Y-uv0.Y)
		if math.Abs(det) < 1e-14 {
			continue
		}
		mirrored := det < 0

		// Tangent frame from UV gradient.
		e1, e2 := p1.Sub(p0), p2.Sub(p0)
		du1, dv1 := uv1.X-uv0.X, uv1.Y-uv0.Y
		du2, dv2 := uv2.X-uv0.X, uv2.Y-uv0.Y
		inv := 1 / det
		Tt := e1.Mul(dv2 * inv).Sub(e2.Mul(dv1 * inv))
		Bt := e2.Mul(du1 * inv).Sub(e1.Mul(du2 * inv))

		minU, maxU := min3(uv0.X, uv1.X, uv2.X), max3(uv0.X, uv1.X, uv2.X)
		minV, maxV := min3(uv0.Y, uv1.Y, uv2.Y), max3(uv0.Y, uv1.Y, uv2.Y)
		x0 := int(math.Max(0, math.Floor(minU*float64(res))))
		x1 := int(math.Min(float64(res-1), math.Ceil(maxU*float64(res))))
		y0 := int(math.Max(0, math.Floor((1-maxV)*float64(res))))
		y1 := int(math.Min(float64(res-1), math.Ceil((1-minV)*float64(res))))

		for py := y0; py <= y1; py++ {
			v := 1 - (float64(py)+0.5)/float64(res)
			for px := x0; px <= x1; px++ {
				u := (float64(px) + 0.5) / float64(res)
				w0 := ((uv1.Y-uv2.Y)*(u-uv2.X) + (uv2.X-uv1.X)*(v-uv2.Y)) / det
				w1 := ((uv2.Y-uv0.Y)*(u-uv2.X) + (uv0.X-uv2.X)*(v-uv2.Y)) / det
				w2 := 1 - w0 - w1
				const eps = 1e-6
				if w0 < -eps || w1 < -eps || w2 < -eps {
					continue
				}

				P := p0.Mul(w0).Add(p1.Mul(w1)).Add(p2.Mul(w2))
				N := n0.Mul(w0).Add(n1.Mul(w1)).Add(n2.Mul(w2)).Norm()
				T := Tt.Sub(N.Mul(Tt.Dot(N))).Norm()
				hand := 1.0
				if Tt.Cross(Bt).Dot(N) < 0 {
					hand = -1
				}
				B := N.Cross(T).Mul(hand)

				var ro, rd Vec3
				var hit Hit
				var ok bool
				switch method {
				case MethodNearest:
					h1, ok1 := highBVH.Intersect(P, N, 1e-5, maxDist, -1)
					h2, ok2 := highBVH.Intersect(P, N.Mul(-1), 1e-5, maxDist, -1)
					switch {
					case ok1 && (!ok2 || h1.T <= h2.T):
						ro, rd, hit, ok = P, N, h1, true
					case ok2:
						ro, rd, hit, ok = P, N.Mul(-1), h2, true
					}
				case MethodCage:
					ro = P.Add(N.Mul(cage))
					rd = N.Mul(-1)
					hit, ok = highBVH.Intersect(ro, rd, 1e-5, cage+maxDist, -1)
				}

				idx := py*res + px
				r.Covered[idx] = true
				if mirrored {
					r.Mirror[idx] = true
				}
				if !ok {
					if !r.hasHit[idx] {
						r.Zero[idx] = true // keep neutral normal
					}
					continue
				}
				// Self-occlusion: other low-poly geometry blocks the ray
				// before it reaches the captured high-poly surface.
				if _, blocked := lowBVH.Intersect(ro, rd, 1e-4, hit.T-2e-3, ti); blocked {
					r.Occl[idx] = true
				}

				ht := high.Tris[hit.Tri]
				hn := high.Nrm[ht.I[0]].Mul(1 - hit.U - hit.V).
					Add(high.Nrm[ht.I[1]].Mul(hit.U)).
					Add(high.Nrm[ht.I[2]].Mul(hit.V)).Norm()
				r.Nrm[idx] = Vec3{T.Dot(hn), B.Dot(hn), N.Dot(hn)}.Norm()
				r.hasHit[idx] = true
				r.Zero[idx] = false
			}
		}
	}
	return r
}
