package main

import "math"

// MeshVertex is one bind-pose vertex with up to two bone weights.
type MeshVertex struct {
	Pos     Vec3
	Weights []Weight
	Color   Vec3
}

// Mesh stores triangles as index triplets into Vertices.
type Mesh struct {
	Vertices []MeshVertex
	Indices  []int
}

func newMesh() *Mesh { return &Mesh{} }

func (m *Mesh) addVertex(v MeshVertex) int {
	m.Vertices = append(m.Vertices, v)
	return len(m.Vertices) - 1
}

func (m *Mesh) addTri(a, b, c int) { m.Indices = append(m.Indices, a, b, c) }

func weight1(bone int) []Weight {
	return []Weight{{Bone: bone, Weight: 1}}
}

func weight2(a, b int, wa float64) []Weight {
	return []Weight{{Bone: a, Weight: wa}, {Bone: b, Weight: 1 - wa}}
}

// basisFrame returns two unit vectors perpendicular to d, forming an
// orthonormal frame with d itself.
func basisFrame(d Vec3) (Vec3, Vec3) {
	ref := Vec3{0, 0, 1}
	if d.Cross(ref).Len() < 1e-6 {
		ref = Vec3{1, 0, 0}
	}
	u := d.Cross(ref).Norm()
	v := d.Cross(u).Norm()
	return u, v
}

// addCapsule builds a three-ring tapered cylinder between two bind-pose
// joint positions. The center ring is shared 50/50 between the bones so
// the limb bends without cracks or rigid seams.
func (m *Mesh) addCapsule(p0, p1 Vec3, r0, r1 float64, boneA, boneB int, color Vec3) {
	axis := p1.Sub(p0)
	d := axis.Norm()
	u, v := basisFrame(d)
	const radial = 10
	rings := [3]struct {
		t      float64
		r      float64
		weight []Weight
	}{
		{0, r0, weight1(boneA)},
		{0.5, (r0+r1)*0.5 + (r0+r1)*0.08, weight2(boneA, boneB, 0.5)},
		{1, r1, weight1(boneB)},
	}
	var ids [3][radial]int
	for ri, ring := range rings {
		center := p0.Add(axis.Scale(ring.t))
		for k := 0; k < radial; k++ {
			ang := float64(k) / float64(radial) * 2 * math.Pi
			off := u.Scale(math.Cos(ang) * ring.r).Add(v.Scale(math.Sin(ang) * ring.r))
			ids[ri][k] = m.addVertex(MeshVertex{
				Pos:     center.Add(off),
				Weights: ring.weight,
				Color:   color,
			})
		}
	}
	for ri := 0; ri < 2; ri++ {
		for k := 0; k < radial; k++ {
			k2 := (k + 1) % radial
			a := ids[ri][k]
			b := ids[ri][k2]
			c := ids[ri+1][k2]
			d := ids[ri+1][k]
			m.addTri(a, b, c)
			m.addTri(a, c, d)
		}
	}
}

// addSphere creates a low-poly UV sphere centered at center with the
// given per-axis scale, fully weighted to one bone.
func (m *Mesh) addSphere(center Vec3, scale Vec3, bone int, color Vec3) {
	const stacks = 4
	const slices = 10
	var grid [stacks + 1][slices]int
	for i := 0; i <= stacks; i++ {
		phi := float64(i)/float64(stacks)*math.Pi - math.Pi/2
		cp, sp := math.Cos(phi), math.Sin(phi)
		for j := 0; j < slices; j++ {
			theta := float64(j) / float64(slices) * 2 * math.Pi
			dir := Vec3{cp * math.Cos(theta), sp, cp * math.Sin(theta)}
			pos := center.Add(Vec3{dir.X * scale.X, dir.Y * scale.Y, dir.Z * scale.Z})
			grid[i][j] = m.addVertex(MeshVertex{
				Pos:     pos,
				Weights: weight1(bone),
				Color:   color,
			})
		}
	}
	for i := 0; i < stacks; i++ {
		for j := 0; j < slices; j++ {
			j2 := (j + 1) % slices
			m.addTri(grid[i][j], grid[i][j2], grid[i+1][j2])
			m.addTri(grid[i][j], grid[i+1][j2], grid[i+1][j])
		}
	}
}
