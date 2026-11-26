package main

import "math"

// faceGrid controls the resolution of the procedural face. The same grid is
// used for the base and every morph target, so indices always correspond.
const (
	gridU  = 49 // vertices along U
	gridV  = 49 // vertices along V
	uMin   = -1.0
	uMax   = 1.0
	vMin   = -1.0
	vMax   = 1.0
	derivH = 1e-4 // finite-difference step for tangent/normal extraction
)

type faceParams struct {
	gridU, gridV int
}

// basePoint maps a parameter point (u, v) on the face patch to 3D space.
// U runs ear-to-ear, V runs chin-to-forehead, and +Z points outward.
func basePoint(u, v float64) Vec3 {
	r2 := u*u + v*v
	dome := 0.4 * math.Sqrt(math.Max(0, 1.0-r2))

	// Nose: narrow vertical ridge with a rounded tip.
	nose := 0.22 * math.Exp(-(math.Pow(u/0.16, 2) + math.Pow((v-0.02)/0.34, 2)))
	tip := 0.07 * math.Exp(-(math.Pow(u/0.13, 2) + math.Pow((v-0.16)/0.12, 2)))

	// Brow ridge and a softer chin keep the patch from looking spherical.
	brow := 0.05 * math.Exp(-math.Pow((v-0.46)/0.08, 2)) *
		(0.75 + 0.25*math.Cos(math.Pi*u))
	chin := 0.04 * math.Exp(-(math.Pow(u/0.35, 2) + math.Pow((v+0.72)/0.16, 2)))

	return Vec3{X: u, Y: v, Z: dome + nose + tip + brow + chin}
}

// surfaceFrame derives the U tangent, V tangent and outward normal of any
// parameterized point function via central finite differences.
func surfaceFrame(point func(u, v float64) Vec3, u, v float64) (tangent, bitangent, normal Vec3) {
	h := derivH
	du := point(u+h, v).Sub(point(u-h, v)).Scale(1 / (2 * h))
	dv := point(u, v+h).Sub(point(u, v-h)).Scale(1 / (2 * h))
	tangent = du.Normalized()
	bitangent = dv.Normalized()
	normal = du.Cross(dv).Normalized()
	return tangent, bitangent, normal
}

// BuildBaseFace constructs the neutral face mesh.
func BuildBaseFace() *Mesh {
	count := gridU * gridV
	mesh := &Mesh{
		Positions: make([]Vec3, count),
		Normals:   make([]Vec3, count),
		Tangents:  make([]Vec3, count),
		UVs:       make([]UV, count),
		Tris:      make([][3]int, 0, 2*(gridU-1)*(gridV-1)),
	}

	for i := 0; i < gridU; i++ {
		u := uMin + (uMax-uMin)*float64(i)/float64(gridU-1)
		for j := 0; j < gridV; j++ {
			v := vMin + (vMax-vMin)*float64(j)/float64(gridV-1)
			idx := i*gridV + j

			tangent, _, normal := surfaceFrame(basePoint, u, v)
			mesh.Positions[idx] = basePoint(u, v)
			mesh.Tangents[idx] = tangent
			mesh.Normals[idx] = normal
			mesh.UVs[idx] = UV{
				U: (u - uMin) / (uMax - uMin),
				V: (v - vMin) / (vMax - vMin),
			}
		}
	}

	for i := 0; i < gridU-1; i++ {
		for j := 0; j < gridV-1; j++ {
			a := i*gridV + j
			b := (i+1)*gridV + j
			c := (i+1)*gridV + (j + 1)
			d := i*gridV + (j + 1)
			// Both triangles wind counter-clockwise seen from +Z.
			mesh.Tris = append(mesh.Tris, [3]int{a, b, d}, [3]int{b, c, d})
		}
	}
	return mesh
}
