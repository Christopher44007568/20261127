package main

import "math"

// Tangents computes unit tangents for a polyline (central differences).
func Tangents(p []Vec3) []Vec3 {
	n := len(p)
	t := make([]Vec3, n)
	for i := 0; i < n; i++ {
		var d Vec3
		switch {
		case i == 0:
			d = p[1].Sub(p[0])
		case i == n-1:
			d = p[n-1].Sub(p[n-2])
		default:
			d = p[i+1].Sub(p[i-1])
		}
		t[i] = d.Norm()
	}
	return t
}

// ParallelTransport propagates a rotation-minimizing frame along the
// polyline, so consecutive cross-sections never twist around the axis.
func ParallelTransport(p []Vec3) (normal, binormal []Vec3) {
	n := len(p)
	t := Tangents(p)
	normal = make([]Vec3, n)
	binormal = make([]Vec3, n)
	a := Vec3{1, 0, 0}
	if math.Abs(t[0].Y) <= math.Abs(t[0].X) && math.Abs(t[0].Y) <= math.Abs(t[0].Z) {
		a = Vec3{0, 1, 0}
	} else if math.Abs(t[0].Z) <= math.Abs(t[0].X) && math.Abs(t[0].Z) <= math.Abs(t[0].Y) {
		a = Vec3{0, 0, 1}
	}
	normal[0] = a.Sub(t[0].Scale(a.Dot(t[0]))).Norm()
	binormal[0] = t[0].Cross(normal[0]).Norm()
	for i := 1; i < n; i++ {
		v := normal[i-1].Sub(t[i].Scale(normal[i-1].Dot(t[i])))
		if v.Len() < 1e-9 {
			v = binormal[i-1].Sub(t[i].Scale(binormal[i-1].Dot(t[i])))
		}
		normal[i] = v.Norm()
		binormal[i] = t[i].Cross(normal[i]).Norm()
	}
	return normal, binormal
}

// Curvature returns the curvature of the circumscribed circle of a,b,c.
func Curvature(a, b, c Vec3) float64 {
	ab := b.Sub(a)
	bc := c.Sub(b)
	ac := c.Sub(a)
	den := ab.Len() * bc.Len() * ac.Len()
	if den < 1e-14 {
		return 0
	}
	return 2 * ab.Cross(ac).Len() / den
}
