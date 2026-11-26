package main

import "math"

// gaussian is a normalized 2D bell around (cu, cv) with the given widths.
func gaussian(u, v, cu, cv, su, sv float64) float64 {
	return math.Exp(-(math.Pow((u-cu)/su, 2) + math.Pow((v-cv)/sv, 2)))
}

// distanceToSegment returns the planar distance from (u, v) to the segment
// between p0 and p1, used to carve the nasolabial crease lines.
func distanceToSegment(u, v float64, p0, p1 [2]float64) float64 {
	dx := p1[0] - p0[0]
	dy := p1[1] - p0[1]
	length2 := dx*dx + dy*dy
	t := ((u-p0[0])*dx + (v-p0[1])*dy) / length2
	t = clampScalar(t, 0, 1)
	qx := p0[0] + t*dx
	qy := p0[1] + t*dy
	return math.Hypot(u-qx, v-qy)
}

// foldLine is the nasolabial line on one side: from the nose wing down to the
// mouth corner. sign is +1 for the right side and -1 for the left.
func foldLine(u, v float64, sign float64) float64 {
	return distanceToSegment(u, v,
		[2]float64{sign * 0.16, 0.18},
		[2]float64{sign * 0.44, -0.34})
}

// smileDisplacement lifts the mouth corners and raises the nasolabial ridge.
func smileDisplacement(u, v float64) Vec3 {
	right := gaussian(u, v, 0.42, -0.38, 0.16, 0.12)
	left := gaussian(u, v, -0.42, -0.38, 0.16, 0.12)
	corner := right + left

	var disp Vec3
	disp.X += (right - left) * 0.10 // corners move outward
	disp.Y += corner * 0.14         // corners move up
	disp.Z += corner * 0.06

	// Folded skin along each nasolabial line: the feature whose shading breaks
	// when only positions are blended.
	for _, sign := range []float64{-1, 1} {
		d := foldLine(u, v, sign)
		ridge := math.Exp(-math.Pow(d/0.035, 2))
		disp.Z += 0.05 * ridge
		disp.X += sign * 0.02 * ridge // tissue bunches toward the crease
	}
	return disp
}

// foldDisplacement deepens the nasolabial fold with a ridge and an adjacent
// groove, producing a sharp normal change around the smile line.
func foldDisplacement(u, v float64) Vec3 {
	var disp Vec3
	for _, sign := range []float64{-1, 1} {
		d := foldLine(u, v, sign)
		ridge := math.Exp(-math.Pow(d/0.03, 2))
		groove := math.Exp(-math.Pow((d-0.07)/0.035, 2))
		disp.Z += 0.08*ridge - 0.03*groove
	}
	return disp
}

// browRaiseDisplacement raises the forehead and brow region.
func browRaiseDisplacement(u, v float64) Vec3 {
	g := math.Exp(-math.Pow((v-0.52)/0.16, 2)) * (0.65 + 0.35*math.Cos(math.Pi*u))
	return Vec3{Y: 0.12 * g, Z: 0.05 * g}
}

// jawOpenDisplacement drops the jaw, with zero motion at the mask boundary.
func jawOpenDisplacement(u, v float64) Vec3 {
	t := clampScalar((-0.15-v)/0.3, 0, 1)
	g := t * t * (3 - 2*t) // smoothstep so the mask edge stays seamless
	return Vec3{X: u * 0.05 * g, Y: -0.28 * g, Z: -0.08 * g}
}

// BuildTarget evaluates a displacement function over the base grid and stores
// the resulting per-vertex deltas including the tangent frame.
func BuildTarget(base *Mesh, name string, min, max float64, disp func(u, v float64) Vec3) *Target {
	target := &Target{
		Name:    name,
		Min:     min,
		Max:     max,
		PosD:    make([]Vec3, base.VertexCount()),
		NormalD: make([]Vec3, base.VertexCount()),
		TanD:    make([]Vec3, base.VertexCount()),
	}

	for i := 0; i < gridU; i++ {
		u := uMin + (uMax-uMin)*float64(i)/float64(gridU-1)
		for j := 0; j < gridV; j++ {
			v := vMin + (vMax-vMin)*float64(j)/float64(gridV-1)
			idx := i*gridV + j

			point := func(pu, pv float64) Vec3 {
				return basePoint(pu, pv).Add(disp(pu, pv))
			}
			tangent, _, normal := surfaceFrame(point, u, v)
			target.PosD[idx] = point(u, v).Sub(base.Positions[idx])
			target.TanD[idx] = tangent.Sub(base.Tangents[idx])
			target.NormalD[idx] = normal.Sub(base.Normals[idx])
		}
	}
	return target
}

// BuildTargets returns the full morph target set for the face.
func BuildTargets(base *Mesh) []*Target {
	return []*Target{
		BuildTarget(base, "smile", 0, 1, smileDisplacement),
		BuildTarget(base, "nasolabial_fold", 0, 1, foldDisplacement),
		BuildTarget(base, "brow_raise", 0, 1, browRaiseDisplacement),
		BuildTarget(base, "jaw_open", 0, 1, jawOpenDisplacement),
	}
}
