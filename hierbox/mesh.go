package hierbox

import "math"

// Vertex is a rigidly bone-attached mesh vertex in bone-local coordinates.
type Vertex struct {
	Bone  BoneID `json:"bone"`
	Local Vec3   `json:"local"`
}

// Mesh is a small wireframe/solid low-poly character mesh.
type Mesh struct {
	Vertices  []Vertex `json:"vertices"`
	Edges     [][2]int `json:"edges"`
	Triangles [][3]int `json:"triangles"`
}

func NewMesh() *Mesh { return &Mesh{} }

func (m *Mesh) addVertex(bone BoneID, p Vec3) int {
	m.Vertices = append(m.Vertices, Vertex{Bone: bone, Local: p})
	return len(m.Vertices) - 1
}

func (m *Mesh) addEdge(a, b int) {
	m.Edges = append(m.Edges, [2]int{a, b})
}

func (m *Mesh) AddBox(bone BoneID, center, size Vec3) {
	h := size.Scale(0.5)
	p := []Vec3{
		center.Add(V(-h.X, -h.Y, -h.Z)),
		center.Add(V(h.X, -h.Y, -h.Z)),
		center.Add(V(h.X, h.Y, -h.Z)),
		center.Add(V(-h.X, h.Y, -h.Z)),
		center.Add(V(-h.X, -h.Y, h.Z)),
		center.Add(V(h.X, -h.Y, h.Z)),
		center.Add(V(h.X, h.Y, h.Z)),
		center.Add(V(-h.X, h.Y, h.Z)),
	}
	base := len(m.Vertices)
	for _, q := range p {
		m.addVertex(bone, q)
	}
	for _, e := range [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	} {
		m.addEdge(base+e[0], base+e[1])
	}
	for _, t := range [][3]int{
		{0, 1, 2}, {0, 2, 3}, {4, 6, 5}, {4, 7, 6},
		{0, 4, 5}, {0, 5, 1}, {1, 5, 6}, {1, 6, 2},
		{2, 6, 7}, {2, 7, 3}, {3, 7, 4}, {3, 4, 0},
	} {
		m.Triangles = append(m.Triangles, [3]int{base + t[0], base + t[1], base + t[2]})
	}
}

func perpendicularBasis(axis Vec3) (Vec3, Vec3) {
	w := axis.Normalized()
	ref := V(0, 1, 0)
	if math.Abs(w.Dot(ref)) > 0.92 {
		ref = V(1, 0, 0)
	}
	u := ref.Cross(w).Normalized()
	v := w.Cross(u).Normalized()
	return u, v
}

// AddCylinder adds an open-ended low-poly tube. Cap centers are also emitted
// so bounds include the complete segment ends.
func (m *Mesh) AddCylinder(bone BoneID, a, b Vec3, radius float64, sides int) {
	if sides < 4 {
		sides = 4
	}
	axis := b.Sub(a)
	u, v := perpendicularBasis(axis)
	ringA := make([]int, sides)
	ringB := make([]int, sides)
	for i := 0; i < sides; i++ {
		ang := 2 * math.Pi * float64(i) / float64(sides)
		off := u.Scale(math.Cos(ang) * radius).Add(v.Scale(math.Sin(ang) * radius))
		ringA[i] = m.addVertex(bone, a.Add(off))
		ringB[i] = m.addVertex(bone, b.Add(off))
	}
	ca := m.addVertex(bone, a)
	cb := m.addVertex(bone, b)
	for i := 0; i < sides; i++ {
		j := (i + 1) % sides
		m.addEdge(ringA[i], ringA[j])
		m.addEdge(ringB[i], ringB[j])
		if i%2 == 0 {
			m.addEdge(ringA[i], ringB[i])
		}
		m.Triangles = append(m.Triangles,
			[3]int{ringA[i], ringB[i], ringB[j]},
			[3]int{ringA[i], ringB[j], ringA[j]},
			[3]int{ca, ringA[j], ringA[i]},
			[3]int{cb, ringB[i], ringB[j]},
		)
	}
}

// AddEllipsoid adds a UV ellipsoid. Low segment counts are intentional: the
// report is a clear diagnostic drawing rather than a render benchmark.
func (m *Mesh) AddEllipsoid(bone BoneID, center Vec3, radii Vec3, slices, stacks int) {
	if slices < 6 {
		slices = 6
	}
	if stacks < 4 {
		stacks = 4
	}
	grid := make([][]int, stacks+1)
	for j := 0; j <= stacks; j++ {
		phi := -math.Pi/2 + math.Pi*float64(j)/float64(stacks)
		cp, sp := math.Cos(phi), math.Sin(phi)
		grid[j] = make([]int, slices)
		for i := 0; i < slices; i++ {
			theta := 2 * math.Pi * float64(i) / float64(slices)
			p := V(
				radii.X*cp*math.Cos(theta),
				radii.Y*sp,
				radii.Z*cp*math.Sin(theta),
			).Add(center)
			grid[j][i] = m.addVertex(bone, p)
		}
	}
	for j := 0; j < stacks; j++ {
		for i := 0; i < slices; i++ {
			k := (i + 1) % slices
			a, b := grid[j][i], grid[j][k]
			c, d := grid[j+1][k], grid[j+1][i]
			m.Triangles = append(m.Triangles, [3]int{a, b, c}, [3]int{a, c, d})
			if j%2 == 0 {
				m.addEdge(a, b)
			}
			if i%3 == 0 {
				m.addEdge(a, d)
			}
		}
	}
}

func (m *Mesh) PosedVertices(world []Mat4) []Vec3 {
	out := make([]Vec3, len(m.Vertices))
	for i, v := range m.Vertices {
		out[i] = world[v.Bone].TransformPoint(v.Local)
	}
	return out
}

// LocalBoxes groups every vertex under its owner bone.
func (m *Mesh) LocalBoxes(boneCount int, padding float64) []Box3 {
	boxes := make([]Box3, boneCount)
	for i := range boxes {
		boxes[i] = EmptyBox()
	}
	for _, v := range m.Vertices {
		boxes[v.Bone] = boxes[v.Bone].AddPoint(v.Local)
	}
	if padding > 0 {
		for i := range boxes {
			if !boxes[i].Empty {
				boxes[i] = boxes[i].Expanded(padding)
			}
		}
	}
	return boxes
}
