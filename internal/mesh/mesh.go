package mesh

import "math"

// Tri is one consistently wound triangle.
type Tri [3]int

// TriangleMesh is an indexed triangle mesh.
type TriangleMesh struct {
	Vertices []Vec3
	Faces    []Tri
}

// EdgeKey is an undirected edge. A is always the smaller vertex index.
type EdgeKey struct{ A, B int }

func Edge(a, b int) EdgeKey {
	if a > b {
		a, b = b, a
	}
	return EdgeKey{a, b}
}

// FaceNormal returns the unnormalized triangle normal.
func (m *TriangleMesh) FaceNormal(f Tri) Vec3 {
	a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
	return b.Sub(a).Cross(c.Sub(a))
}

func (m *TriangleMesh) FaceArea(f Tri) float64 {
	return 0.5 * m.FaceNormal(f).Len()
}

func (m *TriangleMesh) FaceCentroid(f Tri) Vec3 {
	a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
	return a.Add(b).Add(c).Scale(1.0 / 3.0)
}

// EdgeFace records the incident faces of an edge.
type EdgeFace struct {
	Edge  EdgeKey
	Faces []int
}

// EdgeMap returns all edges and their incident triangle indices.
func (m *TriangleMesh) EdgeMap() map[EdgeKey]*EdgeFace {
	edges := make(map[EdgeKey]*EdgeFace, len(m.Faces)*3/2)
	for fi, f := range m.Faces {
		for k := 0; k < 3; k++ {
			key := Edge(f[k], f[(k+1)%3])
			e := edges[key]
			if e == nil {
				e = &EdgeFace{Edge: key}
				edges[key] = e
			}
			e.Faces = append(e.Faces, fi)
		}
	}
	return edges
}

// FeatureOptions controls crease detection.
type FeatureOptions struct {
	CreaseAngleDeg float64
}

// FeatureInfo describes detected hard edges and non-manifold topology.
type FeatureInfo struct {
	Edges       map[EdgeKey]bool
	Boundary    int
	Crease      int
	NonManifold int
}

// DetectFeatures marks boundary edges and edges whose dihedral angle exceeds
// the crease threshold. Non-manifold edges are always features.
func (m *TriangleMesh) DetectFeatures(opt FeatureOptions) FeatureInfo {
	info := FeatureInfo{Edges: make(map[EdgeKey]bool)}
	threshold := opt.CreaseAngleDeg * math.Pi / 180.0
	for key, e := range m.EdgeMap() {
		if len(e.Faces) == 1 {
			info.Edges[key] = true
			info.Boundary++
			continue
		}
		if len(e.Faces) != 2 {
			info.Edges[key] = true
			info.NonManifold++
			continue
		}
		n1 := m.FaceNormal(m.Faces[e.Faces[0]]).Normalize()
		n2 := m.FaceNormal(m.Faces[e.Faces[1]]).Normalize()
		angle := math.Acos(math.Max(-1, math.Min(1, n1.Dot(n2))))
		if angle > threshold {
			info.Edges[key] = true
			info.Crease++
		}
	}
	return info
}

// Weld merges vertices closer than eps after placing the mesh in a unit box.
// It removes degenerate faces and duplicate faces.
func (m *TriangleMesh) Weld(eps float64) *TriangleMesh {
	if eps <= 0 || len(m.Vertices) == 0 {
		return m
	}
	lo, hi := Bounds(m.Vertices)
	diag := hi.Sub(lo).Len()
	if diag < 1e-12 {
		diag = 1
	}
	type cell struct{ X, Y, Z int64 }
	cells := make(map[cell][]int, len(m.Vertices))
	remap := make([]int, len(m.Vertices))
	vertices := make([]Vec3, 0, len(m.Vertices))
	q := eps * diag
	if q < 1e-12 {
		q = 1e-12
	}
	key := func(p Vec3) cell {
		return cell{int64(math.Round(p.X / q)), int64(math.Round(p.Y / q)), int64(math.Round(p.Z / q))}
	}
	for i, p := range m.Vertices {
		k := key(p)
		found := -1
		for dx := int64(-1); dx <= 1 && found < 0; dx++ {
			for dy := int64(-1); dy <= 1 && found < 0; dy++ {
				for dz := int64(-1); dz <= 1 && found < 0; dz++ {
					for _, j := range cells[cell{k.X + dx, k.Y + dy, k.Z + dz}] {
						if vertices[j].Sub(p).Len() <= q {
							found = j
							break
						}
					}
				}
			}
		}
		if found >= 0 {
			remap[i] = found
			continue
		}
		remap[i] = len(vertices)
		vertices = append(vertices, p)
		cells[k] = append(cells[k], len(vertices)-1)
	}
	out := &TriangleMesh{Vertices: vertices}
	seen := make(map[[3]int]bool)
	for _, f := range m.Faces {
		t := Tri{remap[f[0]], remap[f[1]], remap[f[2]]}
		if t[0] == t[1] || t[1] == t[2] || t[2] == t[0] {
			continue
		}
		ordered := t
		// Canonical form ignores winding for duplicate removal.
		if ordered[0] > ordered[1] {
			ordered[0], ordered[1] = ordered[1], ordered[0]
		}
		if ordered[1] > ordered[2] {
			ordered[1], ordered[2] = ordered[2], ordered[1]
		}
		if ordered[0] > ordered[1] {
			ordered[0], ordered[1] = ordered[1], ordered[0]
		}
		if seen[[3]int(ordered)] {
			continue
		}
		seen[[3]int(ordered)] = true
		out.Faces = append(out.Faces, t)
	}
	return out
}

// EdgeLength returns the length of an edge key.
func (m *TriangleMesh) EdgeLength(e EdgeKey) float64 {
	return m.Vertices[e.A].Sub(m.Vertices[e.B]).Len()
}

func (m *TriangleMesh) IsClosed() bool {
	for _, e := range m.EdgeMap() {
		if len(e.Faces) != 2 {
			return false
		}
	}
	return true
}

// Orient makes adjacent faces consistently wound and flips the whole mesh
// when its signed volume points inward. It assumes manifold connectivity.
func (m *TriangleMesh) Orient() {
	if len(m.Faces) == 0 {
		return
	}
	edges := m.EdgeMap()
	adj := make([][]int, len(m.Faces))
	for _, e := range edges {
		if len(e.Faces) != 2 {
			continue
		}
		f0, f1 := e.Faces[0], e.Faces[1]
		adj[f0] = append(adj[f0], f1)
		adj[f1] = append(adj[f1], f0)
	}
	flip := make([]int8, len(m.Faces)) // 0 unvisited, 1 keep, -1 flip
	for start := range m.Faces {
		if flip[start] != 0 {
			continue
		}
		flip[start] = 1
		queue := []int{start}
		for len(queue) > 0 {
			fi := queue[0]
			queue = queue[1:]
			for _, fj := range adj[fi] {
				shared := sharedEdge(m.Faces[fi], m.Faces[fj])
				if shared == nil {
					continue
				}
				di, dj := edgeDirection(m.Faces[fi], *shared), edgeDirection(m.Faces[fj], *shared)
				same := di == dj
				if flip[fi] < 0 {
					same = !same
				}
				want := int8(1)
				if same {
					want = -1
				}
				if flip[fj] == 0 {
					flip[fj] = want
					queue = append(queue, fj)
				}
			}
		}
	}
	if m.IsClosed() {
		var signedVolume float64
		for fi, f := range m.Faces {
			a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
			v := a.Dot(b.Cross(c)) / 6
			if flip[fi] < 0 {
				v = -v
			}
			signedVolume += v
		}
		if signedVolume < 0 {
			for i := range flip {
				flip[i] = -flip[i]
			}
		}
	}
	for i, f := range m.Faces {
		if flip[i] < 0 {
			m.Faces[i] = Tri{f[0], f[2], f[1]}
		}
	}
}

func sharedEdge(a, b Tri) *EdgeKey {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if (a[i] == b[j] && a[(i+1)%3] == b[(j+1)%3]) ||
				(a[i] == b[(j+1)%3] && a[(i+1)%3] == b[j]) {
				k := Edge(a[i], a[(i+1)%3])
				return &k
			}
		}
	}
	return nil
}

func edgeDirection(f Tri, key EdgeKey) int {
	for i := 0; i < 3; i++ {
		a, b := f[i], f[(i+1)%3]
		if a == key.A && b == key.B {
			return 1
		}
		if a == key.B && b == key.A {
			return -1
		}
	}
	return 0
}
