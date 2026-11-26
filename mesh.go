package main

// UV is a texture coordinate. The procedural face maps (u, v) directly to
// image space so normal-difference maps line up with the surface.
type UV struct {
	U, V float64
}

// Mesh is a triangulated face with an explicit tangent frame at every vertex.
// All blend targets must share the same vertex count and triangle indices.
type Mesh struct {
	Positions []Vec3
	Normals   []Vec3
	Tangents  []Vec3 // unit-length surface tangent along increasing U
	UVs       []UV
	Tris      [][3]int
}

// VertexCount reports the number of vertices.
func (m *Mesh) VertexCount() int {
	return len(m.Positions)
}

// Clone returns an independent deep copy of the mesh.
func (m *Mesh) Clone() *Mesh {
	cp := &Mesh{
		Positions: append([]Vec3(nil), m.Positions...),
		Normals:   append([]Vec3(nil), m.Normals...),
		Tangents:  append([]Vec3(nil), m.Tangents...),
		UVs:       append([]UV(nil), m.UVs...),
		Tris:      make([][3]int, len(m.Tris)),
	}
	copy(cp.Tris, m.Tris)
	return cp
}

// Target stores the per-vertex deltas of one morph target relative to the
// base face. Storing deltas (not full poses) keeps the blend arithmetic
// explicit: pose = base + sum(weight * delta).
type Target struct {
	Name     string
	Min, Max float64
	PosD     []Vec3
	NormalD  []Vec3 // target normal minus base normal
	TanD     []Vec3 // target tangent minus base tangent
}

// BlendResult pairs the posed mesh with the effective per-target weights after
// clamping and normalization, which the report and tests inspect.
type BlendResult struct {
	Mesh             *Mesh
	ClampedWeights   []float64
	EffectiveWeights []float64
	Denominator      float64 // sum of clamped weights; 0 means the identity path
}
