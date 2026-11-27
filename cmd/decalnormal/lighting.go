package main

import "math"

var (
	lightDir   = v3(-0.45, -0.55, 0.75).Normalized()
	viewDir    = v3(0, 0, 1)
	lightColor = v3(1.0, 0.97, 0.9)
	ambientCol = v3(0.16, 0.18, 0.22)
)

// shade evaluates a Blinn-Phong response for one surface point. Albedo
// already includes any decal tint at the pixel.
func shade(n, albedo Vec3) Vec3 {
	ambient := albedo.Scale(0.35).Multiply(ambientCol.Scale(2))

	diff := math.Max(0, n.Dot(lightDir))
	diffuse := albedo.Scale(diff).Multiply(lightColor)

	half := lightDir.Add(viewDir).Normalized()
	spec := math.Pow(math.Max(0, n.Dot(half)), 48)
	specular := lightColor.Scale(0.55 * spec)

	rim := math.Pow(math.Max(0, 1-n.Dot(viewDir)), 3) * 0.08
	return ambient.Add(diffuse).Add(specular).Add(v3(rim, rim, rim))
}

// Multiply is component-wise (Hadamard) color multiplication.
func (a Vec3) Multiply(b Vec3) Vec3 {
	return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}
