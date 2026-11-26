package mesh

import "math"

// Box returns a closed, outward-facing axis-aligned box.
func Box(min, max Vec3) Mesh {
	v := [8]Vec3{
		{min.X, min.Y, min.Z},
		{max.X, min.Y, min.Z},
		{max.X, max.Y, min.Z},
		{min.X, max.Y, min.Z},
		{min.X, min.Y, max.Z},
		{max.X, min.Y, max.Z},
		{max.X, max.Y, max.Z},
		{min.X, max.Y, max.Z},
	}
	quads := [][4]int{
		{0, 3, 2, 1}, // -Z
		{4, 5, 6, 7}, // +Z
		{0, 1, 5, 4}, // -Y
		{3, 7, 6, 2}, // +Y
		{0, 4, 7, 3}, // -X
		{1, 2, 6, 5}, // +X
	}
	out := Mesh{Triangles: make([]Triangle, 0, 12)}
	for _, q := range quads {
		out.Triangles = append(out.Triangles,
			Triangle{v[q[0]], v[q[1]], v[q[2]]},
			Triangle{v[q[0]], v[q[2]], v[q[3]]},
		)
	}
	return out
}

// Sphere creates a UV sphere. The seam vertices are intentionally unique;
// the normal unification pass welds them before boolean clipping.
func Sphere(center Vec3, radius float64, segments, rings int) Mesh {
	if segments < 3 {
		segments = 3
	}
	if rings < 2 {
		rings = 2
	}
	point := func(ring, segment int) Vec3 {
		theta := math.Pi * float64(ring) / float64(rings)
		phi := 2 * math.Pi * float64(segment) / float64(segments)
		return Vec3{
			center.X + radius*math.Sin(theta)*math.Cos(phi),
			center.Y + radius*math.Cos(theta),
			center.Z + radius*math.Sin(theta)*math.Sin(phi),
		}
	}
	out := Mesh{Triangles: make([]Triangle, 0, segments*rings*2)}
	for ring := 0; ring < rings; ring++ {
		for segment := 0; segment < segments; segment++ {
			a := point(ring, segment)
			b := point(ring, segment+1)
			c := point(ring+1, segment)
			d := point(ring+1, segment+1)
			if ring != 0 {
				out.Triangles = append(out.Triangles, Triangle{a, b, c})
			}
			if ring != rings-1 {
				out.Triangles = append(out.Triangles, Triangle{b, d, c})
			}
		}
	}
	return out.OrientOutward()
}
