package mesh

import (
	"math"
	"sort"
)

// Triangle is the public, non-indexed representation used by export and render.
type Triangle struct {
	A, B, C Vec3
}

func (t Triangle) Normal() Vec3 {
	return t.B.Sub(t.A).Cross(t.C.Sub(t.A)).Normalize()
}

func (t Triangle) Area() float64 {
	return t.B.Sub(t.A).Cross(t.C.Sub(t.A)).Length() * 0.5
}

func (t Triangle) Degenerate() bool {
	return t.Area() <= 1e-12
}

// Mesh stores a triangle soup. Boolean operations produce the same type.
type Mesh struct {
	Triangles []Triangle
}

type Topology struct {
	Vertices         int
	Triangles        int
	BoundaryEdges    int
	NonManifoldEdges int
	DuplicateFaces   int
	OpenEdgeSegments []Segment
}

type Segment struct {
	A, B Vec3
}

// AnalyzeTopology reports edge incidence and duplicate vertex triples after
// tolerance welding. A closed, consistently split result has no boundary or
// non-manifold edges and no duplicate faces.
func (m Mesh) AnalyzeTopology() Topology {
	welded, remap := weldMesh(m)
	stats := Topology{Vertices: len(uniqueVertices(welded)), Triangles: len(welded.Triangles)}
	edges := make(map[edgeKey]int)
	faces := make(map[[3]int]int)
	points := make(map[int]Vec3)
	for triangleIndex, ids := range remap {
		vertices := [3]Vec3{
			welded.Triangles[triangleIndex].A,
			welded.Triangles[triangleIndex].B,
			welded.Triangles[triangleIndex].C,
		}
		for i, id := range ids {
			points[id] = vertices[i]
		}
		for i := 0; i < 3; i++ {
			a, b := ids[i], ids[(i+1)%3]
			edges[edgeKey{min(a, b), max(a, b)}]++
		}
		key := [3]int{ids[0], ids[1], ids[2]}
		sort.Ints(key[:])
		faces[key]++
	}
	for key, count := range edges {
		if count == 1 {
			stats.BoundaryEdges++
			stats.OpenEdgeSegments = append(stats.OpenEdgeSegments,
				Segment{A: points[key.A], B: points[key.B]})
		} else if count > 2 {
			stats.NonManifoldEdges++
		}
	}
	for _, count := range faces {
		if count > 1 {
			stats.DuplicateFaces += count - 1
		}
	}
	return stats
}

func uniqueVertices(m Mesh) map[Vec3]struct{} {
	points := make(map[Vec3]struct{})
	for _, t := range m.Triangles {
		points[t.A] = struct{}{}
		points[t.B] = struct{}{}
		points[t.C] = struct{}{}
	}
	return points
}

func (m Mesh) Clone() Mesh {
	out := Mesh{Triangles: make([]Triangle, len(m.Triangles))}
	copy(out.Triangles, m.Triangles)
	return out
}

func (m Mesh) Bounds() (min, max Vec3) {
	if len(m.Triangles) == 0 {
		return Vec3{}, Vec3{}
	}
	min = m.Triangles[0].A
	max = min
	for _, t := range m.Triangles {
		for _, p := range []Vec3{t.A, t.B, t.C} {
			min.X = math.Min(min.X, p.X)
			min.Y = math.Min(min.Y, p.Y)
			min.Z = math.Min(min.Z, p.Z)
			max.X = math.Max(max.X, p.X)
			max.Y = math.Max(max.Y, p.Y)
			max.Z = math.Max(max.Z, p.Z)
		}
	}
	return min, max
}

func (m Mesh) Volume() float64 {
	return math.Abs(m.SignedVolume())
}

func (m Mesh) SignedVolume() float64 {
	var volume float64
	for _, t := range m.Triangles {
		volume += t.A.Dot(t.B.Cross(t.C)) / 6
	}
	return volume
}

func (m Mesh) SurfaceArea() float64 {
	var area float64
	for _, t := range m.Triangles {
		area += t.Area()
	}
	return area
}

// Components returns edge-connected shells. It is used by diagnostics and the
// report to identify cavities and disjoint result bodies.
func (m Mesh) Components() []Mesh {
	welded, remap := weldMesh(m)
	adj := buildEdgeAdjacency(welded, remap)
	visited := make([]bool, len(welded.Triangles))
	out := make([]Mesh, 0)
	for seed := range welded.Triangles {
		if visited[seed] {
			continue
		}
		visited[seed] = true
		queue := []int{seed}
		component := Mesh{}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			component.Triangles = append(component.Triangles, welded.Triangles[current])
			ids := remap[current]
			for j := 0; j < 3; j++ {
				a, b := ids[j], ids[(j+1)%3]
				for _, use := range adj[edgeKey{min(a, b), max(a, b)}] {
					if !visited[use.tri] {
						visited[use.tri] = true
						queue = append(queue, use.tri)
					}
				}
			}
		}
		out = append(out, component)
	}
	return out
}

func (m Mesh) Translate(d Vec3) Mesh {
	out := m.Clone()
	for i := range out.Triangles {
		out.Triangles[i].A = out.Triangles[i].A.Add(d)
		out.Triangles[i].B = out.Triangles[i].B.Add(d)
		out.Triangles[i].C = out.Triangles[i].C.Add(d)
	}
	return out
}

func (m Mesh) Scale(s Vec3) Mesh {
	out := m.Clone()
	for i := range out.Triangles {
		t := &out.Triangles[i]
		t.A = Vec3{t.A.X * s.X, t.A.Y * s.Y, t.A.Z * s.Z}
		t.B = Vec3{t.B.X * s.X, t.B.Y * s.Y, t.B.Z * s.Z}
		t.C = Vec3{t.C.X * s.X, t.C.Y * s.Y, t.C.Z * s.Z}
	}
	return out
}

// HealTJunctions inserts existing vertices that lie on another triangle's
// edge, then fan-triangulates the subdivided triangle. This closes the visual
// gaps that BSP splitting can leave between differently split neighbors.
func (m Mesh) HealTJunctions() Mesh {
	current := m.UnifyNormals()
	for pass := 0; pass < 3; pass++ {
		points := uniqueVertexList(current)
		changed := false
		next := Mesh{Triangles: make([]Triangle, 0, len(current.Triangles))}
		for _, triangle := range current.Triangles {
			polygon := make([]Vec3, 0, 8)
			for _, edge := range [][2]Vec3{{triangle.A, triangle.B}, {triangle.B, triangle.C}, {triangle.C, triangle.A}} {
				polygon = append(polygon, edge[0])
				for _, point := range pointsOnSegment(edge[0], edge[1], points) {
					polygon = append(polygon, point)
					changed = true
				}
			}
			if len(polygon) == 3 {
				next.Triangles = append(next.Triangles, triangle)
				continue
			}
			for i := 1; i+1 < len(polygon); i++ {
				part := Triangle{A: polygon[0], B: polygon[i], C: polygon[i+1]}
				if !part.Degenerate() {
					next.Triangles = append(next.Triangles, part)
				}
			}
		}
		if !changed {
			return current
		}
		current = next.UnifyNormals()
	}
	return current
}

func uniqueVertexList(m Mesh) []Vec3 {
	seen := make(map[Vec3]struct{}, len(m.Triangles))
	points := make([]Vec3, 0, len(m.Triangles))
	for _, triangle := range m.Triangles {
		for _, point := range []Vec3{triangle.A, triangle.B, triangle.C} {
			if _, ok := seen[point]; ok {
				continue
			}
			seen[point] = struct{}{}
			points = append(points, point)
		}
	}
	return points
}

type edgePoint struct {
	point Vec3
	t     float64
}

func pointsOnSegment(a, b Vec3, candidates []Vec3) []Vec3 {
	ab := b.Sub(a)
	lengthSquared := ab.Dot(ab)
	if lengthSquared < 1e-16 {
		return nil
	}
	var hits []edgePoint
	for _, point := range candidates {
		ap := point.Sub(a)
		t := ap.Dot(ab) / lengthSquared
		if t <= 1e-7 || t >= 1-1e-7 {
			continue
		}
		projected := a.Add(ab.Mul(t))
		if projected.Sub(point).Length() > 1e-5 {
			continue
		}
		hits = append(hits, edgePoint{point: point, t: t})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].t < hits[j].t })
	out := make([]Vec3, len(hits))
	for i := range hits {
		out[i] = hits[i].point
	}
	return out
}

// UnifyNormals welds vertices and makes edge-adjacent triangles agree, while
// preserving the supplied side of each shell. CSG output already has correct
// inside/outside orientation, so this repairs seams without turning cavities
// outward.
func (m Mesh) UnifyNormals() Mesh {
	welded, _, flipped := propagateOrientation(m)
	for i := range welded.Triangles {
		if flipped[i] {
			welded.Triangles[i].B, welded.Triangles[i].C =
				welded.Triangles[i].C, welded.Triangles[i].B
		}
	}
	return welded
}

// OrientOutward is used for closed primitive inputs. In addition to seam
// repair, nested shells are oriented by containment depth: outer surfaces face
// out and cavity surfaces face into the cavity.
func (m Mesh) OrientOutward() Mesh {
	welded, remap, flipped := propagateOrientation(m)
	adj := buildEdgeAdjacency(welded, remap)
	components := connectedTriangleSets(welded, remap, adj)

	for index, component := range components {
		depth := 0
		probe := componentProbePoint(welded.Triangles, component, flipped)
		for otherIndex, other := range components {
			if index == otherIndex {
				continue
			}
			if pointInTriangles(probe, trianglesAt(welded.Triangles, other)) {
				depth++
			}
		}
		var volume float64
		for _, i := range component {
			t := welded.Triangles[i]
			if flipped[i] {
				t.B, t.C = t.C, t.B
			}
			volume += t.A.Dot(t.B.Cross(t.C)) / 6
		}
		wantPositive := depth%2 == 0
		if (volume > 0) != wantPositive {
			for _, i := range component {
				flipped[i] = !flipped[i]
			}
		}
	}

	for i := range welded.Triangles {
		if flipped[i] {
			welded.Triangles[i].B, welded.Triangles[i].C =
				welded.Triangles[i].C, welded.Triangles[i].B
		}
	}
	return welded
}

func propagateOrientation(m Mesh) (Mesh, [][3]int, []bool) {
	welded, remap := weldMesh(m)
	adj := buildEdgeAdjacency(welded, remap)
	flipped := make([]bool, len(welded.Triangles))
	visited := make([]bool, len(welded.Triangles))

	for seed := range welded.Triangles {
		if visited[seed] {
			continue
		}
		queue := []int{seed}
		visited[seed] = true
		for len(queue) > 0 {
			i := queue[0]
			queue = queue[1:]
			ids := remap[i]
			for j := 0; j < 3; j++ {
				a, b := ids[j], ids[(j+1)%3]
				edge := edgeKey{min(a, b), max(a, b)}
				forward := a < b
				for _, other := range adj[edge] {
					if other.tri == i || visited[other.tri] {
						continue
					}
					if other.forward == forward {
						flipped[other.tri] = !flipped[i]
					} else {
						flipped[other.tri] = flipped[i]
					}
					visited[other.tri] = true
					queue = append(queue, other.tri)
				}
			}
		}
	}
	return welded, remap, flipped
}

func connectedTriangleSets(welded Mesh, remap [][3]int, adj map[edgeKey][]edgeUse) [][]int {
	visited := make([]bool, len(welded.Triangles))
	components := make([][]int, 0)
	for seed := range welded.Triangles {
		if visited[seed] {
			continue
		}
		visited[seed] = true
		queue := []int{seed}
		component := []int{seed}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			ids := remap[current]
			for j := 0; j < 3; j++ {
				a, b := ids[j], ids[(j+1)%3]
				for _, use := range adj[edgeKey{min(a, b), max(a, b)}] {
					if visited[use.tri] {
						continue
					}
					visited[use.tri] = true
					queue = append(queue, use.tri)
					component = append(component, use.tri)
				}
			}
		}
		components = append(components, component)
	}
	return components
}

func trianglesAt(triangles []Triangle, indices []int) []Triangle {
	out := make([]Triangle, len(indices))
	for i, index := range indices {
		out[i] = triangles[index]
	}
	return out
}

func componentProbePoint(triangles []Triangle, component []int, flipped []bool) Vec3 {
	if len(component) == 0 {
		return Vec3{}
	}
	t := triangles[component[0]]
	if flipped[component[0]] {
		t.B, t.C = t.C, t.B
	}
	normal := t.Normal()
	return t.A.Add(t.B).Add(t.C).Mul(1.0 / 3.0).Add(normal.Mul(1e-5))
}

// pointInTriangles uses a fixed generic ray and parity counting. Shell
// components are triangle-closed, so odd crossings means containment.
func pointInTriangles(point Vec3, triangles []Triangle) bool {
	direction := Vec3{X: 1, Y: 0.3713906763541037, Z: 0.3172091371375824}.Normalize()
	crossings := 0
	for _, t := range triangles {
		if rayHitsTriangle(point, direction, t) {
			crossings++
		}
	}
	return crossings%2 == 1
}

func rayHitsTriangle(origin, direction Vec3, triangle Triangle) bool {
	edge1 := triangle.B.Sub(triangle.A)
	edge2 := triangle.C.Sub(triangle.A)
	h := direction.Cross(edge2)
	a := edge1.Dot(h)
	if math.Abs(a) < 1e-10 {
		return false
	}
	f := 1 / a
	s := origin.Sub(triangle.A)
	u := f * s.Dot(h)
	if u < -1e-9 || u > 1+1e-9 {
		return false
	}
	q := s.Cross(edge1)
	v := f * direction.Dot(q)
	if v < -1e-9 || u+v > 1+1e-9 {
		return false
	}
	t := f * edge2.Dot(q)
	return t > 1e-8
}

type edgeKey struct {
	A, B int
}

type edgeUse struct {
	tri     int
	forward bool
}

func weldMesh(m Mesh) (Mesh, [][3]int) {
	type pointKey struct{ X, Y, Z int64 }
	const grid = 1e6
	index := make(map[pointKey]int)
	points := make([]Vec3, 0, len(m.Triangles)*3)
	rawRemap := make([][3]int, len(m.Triangles))
	add := func(p Vec3) int {
		key := pointKey{
			int64(math.Round(p.X * grid)),
			int64(math.Round(p.Y * grid)),
			int64(math.Round(p.Z * grid)),
		}
		if i, ok := index[key]; ok {
			return i
		}
		i := len(points)
		index[key] = i
		points = append(points, p)
		return i
	}
	for i, t := range m.Triangles {
		rawRemap[i] = [3]int{add(t.A), add(t.B), add(t.C)}
	}
	out := Mesh{Triangles: make([]Triangle, 0, len(m.Triangles))}
	remap := make([][3]int, 0, len(m.Triangles))
	for _, ids := range rawRemap {
		if ids[0] == ids[1] || ids[1] == ids[2] || ids[2] == ids[0] {
			continue
		}
		t := Triangle{points[ids[0]], points[ids[1]], points[ids[2]]}
		if !t.Degenerate() {
			out.Triangles = append(out.Triangles, t)
			remap = append(remap, ids)
		}
	}
	return out, remap
}

func buildEdgeAdjacency(m Mesh, remap [][3]int) map[edgeKey][]edgeUse {
	adj := make(map[edgeKey][]edgeUse)
	for i, t := range m.Triangles {
		_ = t
		ids := remap[i]
		for j := 0; j < 3; j++ {
			a, b := ids[j], ids[(j+1)%3]
			key := edgeKey{min(a, b), max(a, b)}
			adj[key] = append(adj[key], edgeUse{tri: i, forward: a < b})
		}
	}
	return adj
}

// SortTriangles gives deterministic output for OBJ files and reports.
func (m *Mesh) SortTriangles() {
	key := func(t Triangle) [9]float64 {
		values := [9]float64{
			t.A.X, t.A.Y, t.A.Z, t.B.X, t.B.Y, t.B.Z, t.C.X, t.C.Y, t.C.Z,
		}
		if comparePoint(t.B, t.A) < 0 {
			values[0], values[1], values[2], values[3], values[4], values[5] =
				values[3], values[4], values[5], values[0], values[1], values[2]
		}
		if comparePoint(t.C, t.B) < 0 {
			values[3], values[4], values[5], values[6], values[7], values[8] =
				values[6], values[7], values[8], values[3], values[4], values[5]
		}
		return values
	}
	sort.SliceStable(m.Triangles, func(i, j int) bool {
		a, b := key(m.Triangles[i]), key(m.Triangles[j])
		for k := range a {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return false
	})
}

func comparePoint(a, b Vec3) int {
	if a.X != b.X {
		if a.X < b.X {
			return -1
		}
		return 1
	}
	if a.Y != b.Y {
		if a.Y < b.Y {
			return -1
		}
		return 1
	}
	if a.Z < b.Z {
		return -1
	}
	if a.Z > b.Z {
		return 1
	}
	return 0
}
