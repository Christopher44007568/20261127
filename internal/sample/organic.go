package sample

import (
	"math"

	"meshdiagnostics/internal/mesh"
)

// Organic returns a procedurally generated organic surface. An icosphere
// keeps the base triangles well-shaped; radial waves give it an organic
// silhouette, while a steep dorsal groove creates real feature edges. A few
// deliberately warped vertices near the groove produce local angle collapse
// for the diagnostic views.
func Organic() *mesh.TriangleMesh {
	m := icosphere(4)
	for i, p := range m.Vertices {
		m.Vertices[i] = deformOrganic(p)
	}
	m.Orient()
	return m
}

func icosphere(level int) *mesh.TriangleMesh {
	const t = 1.6180339887498948
	verts := []mesh.Vec3{
		mesh.V(-1, t, 0), mesh.V(1, t, 0), mesh.V(-1, -t, 0), mesh.V(1, -t, 0),
		mesh.V(0, -1, t), mesh.V(0, 1, t), mesh.V(0, -1, -t), mesh.V(0, 1, -t),
		mesh.V(t, 0, -1), mesh.V(t, 0, 1), mesh.V(-t, 0, -1), mesh.V(-t, 0, 1),
	}
	for i := range verts {
		verts[i] = verts[i].Normalize()
	}
	faces := []mesh.Tri{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	m := &mesh.TriangleMesh{Vertices: verts, Faces: faces}
	for l := 0; l < level; l++ {
		next := &mesh.TriangleMesh{Vertices: m.Vertices}
		mid := make(map[mesh.EdgeKey]int, len(m.Faces)*2)
		midpoint := func(a, b int) int {
			key := mesh.Edge(a, b)
			if v, ok := mid[key]; ok {
				return v
			}
			p := m.Vertices[a].Add(m.Vertices[b]).Normalize()
			v := len(next.Vertices)
			next.Vertices = append(next.Vertices, p)
			mid[key] = v
			return v
		}
		for _, f := range m.Faces {
			ab, bc, ca := midpoint(f[0], f[1]), midpoint(f[1], f[2]), midpoint(f[2], f[0])
			next.Faces = append(next.Faces,
				mesh.Tri{f[0], ab, ca},
				mesh.Tri{f[1], bc, ab},
				mesh.Tri{f[2], ca, bc},
				mesh.Tri{ab, bc, ca},
			)
		}
		m = next
	}
	return m
}

func deformOrganic(p mesh.Vec3) mesh.Vec3 {
	lat := math.Acos(math.Max(-1, math.Min(1, p.Y)))
	lon := math.Atan2(p.Z, p.X)
	sinLat := math.Sin(lat)
	du := wrapPi(lon - math.Atan2(0.15, 0.9))
	ridge := 0.17 * math.Tanh(du/0.035) * math.Pow(sinLat, 2)
	r := 1 +
		0.075*math.Sin(3*lon+0.3)*math.Pow(sinLat, 2) +
		0.035*math.Sin(6*lon-0.8)*math.Sin(4*lat+0.4) +
		0.022*math.Cos(9*lat+0.9*math.Sin(3*lon))
	// A narrow warp near the groove makes a handful of triangles highly
	// anisotropic without degrading the rest of the surface.
	kink := 0.045 * math.Exp(-math.Pow(du/0.075, 2)) * math.Sin(lat*7+lon*3)
	x := p.X * (1.08*r + ridge)
	z := p.Z * (0.86*r + ridge*0.85)
	y := p.Y * 1.28 * (1 + 0.025*math.Sin(4*lon+0.5)*sinLat)
	return mesh.V(x+kink, y, z-kink*0.7)
}

func wrapPi(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
