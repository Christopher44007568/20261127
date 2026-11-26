package main

import "math"

// Blender evaluates morph targets against a fixed base face.
type Blender struct {
	Base    *Mesh
	Targets []*Target
}

// ResolveWeights turns raw UI weights into the weights actually used by the
// blend, in two stages:
//
//  1. Clamping: NaN is treated as 0, then every weight is clamped into the
//     target's [Min, Max] range so extreme expressions cannot fold the face.
//  2. Normalization: when the clamped weights of several targets sum to more
//     than 1, they are divided by that sum. A single active target (sum <= 1)
//     keeps its full weight, so single-target poses are unchanged.
//
// The returned denominator is 0 when every target is off.
func (b *Blender) ResolveWeights(raw []float64) (clamped, effective []float64, denom float64) {
	clamped = make([]float64, len(b.Targets))
	for i, target := range b.Targets {
		w := 0.0
		if i < len(raw) && !math.IsNaN(raw[i]) {
			w = raw[i]
		}
		clamped[i] = clampScalar(w, target.Min, target.Max)
		if clamped[i] != 0 {
			denom += clamped[i]
		}
	}

	effective = append([]float64(nil), clamped...)
	if denom > 1 {
		scale := 1 / denom
		for i := range effective {
			effective[i] *= scale
		}
	}
	return clamped, effective, denom
}

// Blend evaluates one expression. With all targets off it returns a bit-for-bit
// copy of the base mesh: the position, normal and tangent of every vertex match
// the base exactly, satisfying the per-vertex identity requirement without a
// floating-point round trip.
func (b *Blender) Blend(raw []float64) *BlendResult {
	if len(raw) != len(b.Targets) {
		panic("facemorph: one weight per morph target is required")
	}

	clamped, effective, denom := b.ResolveWeights(raw)
	out := b.Base.Clone()
	if denom == 0 {
		return &BlendResult{
			Mesh:             out,
			ClampedWeights:   clamped,
			EffectiveWeights: effective,
			Denominator:      0,
		}
	}

	base := b.Base
	for vertex := range base.Positions {
		var posDelta, normalDelta, tanDelta Vec3
		for i, target := range b.Targets {
			w := effective[i]
			if w == 0 {
				continue
			}
			posDelta = posDelta.Add(target.PosD[vertex].Scale(w))
			normalDelta = normalDelta.Add(target.NormalD[vertex].Scale(w))
			tanDelta = tanDelta.Add(target.TanD[vertex].Scale(w))
		}

		out.Positions[vertex] = base.Positions[vertex].Add(posDelta)

		// Re-normalize the blended normal instead of merely translating the
		// base normal, which is what left smile-line shading wrong before.
		blendedNormal := base.Normals[vertex].Add(normalDelta)
		if blendedNormal.LengthSquared() == 0 {
			blendedNormal = base.Normals[vertex]
		}
		out.Normals[vertex] = blendedNormal.Normalized()

		// Blend the tangent and re-orthogonalize it against the new normal so
		// the tangent frame stays orthonormal under combined targets.
		blendedTangent := base.Tangents[vertex].Add(tanDelta)
		blendedTangent = blendedTangent.Sub(out.Normals[vertex].Scale(blendedTangent.Dot(out.Normals[vertex])))
		if blendedTangent.LengthSquared() == 0 {
			blendedTangent = base.Tangents[vertex]
		}
		out.Tangents[vertex] = blendedTangent.Normalized()
	}

	return &BlendResult{
		Mesh:             out,
		ClampedWeights:   clamped,
		EffectiveWeights: effective,
		Denominator:      denom,
	}
}
