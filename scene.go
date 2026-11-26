package holefill

import "math"

type SceneLayout struct {
	SectionY  float64
	Thickness float64
	XMin      float64
	XMax      float64
	YMin      float64
	YMax      float64
}

func DefaultSceneLayout() SceneLayout {
	return SceneLayout{SectionY: 0.045, Thickness: 0.34, XMin: -7, XMax: 7, YMin: -4.5, YMax: 4.5}
}

func BuildDemoMesh() (*Mesh, SceneLayout) {
	layout := DefaultSceneLayout()
	const nx, ny = 161, 105
	dx := (layout.XMax - layout.XMin) / float64(nx-1)
	dy := (layout.YMax - layout.YMin) / float64(ny-1)

	mesh := &Mesh{}
	for j := 0; j < ny; j++ {
		y := layout.YMin + float64(j)*dy
		for i := 0; i < nx; i++ {
			x := layout.XMin + float64(i)*dx
			z := scanHeight(x, y)
			mesh.V = append(mesh.V, Vec3{X: x, Y: y, Z: z})
		}
	}

	id := func(i, j int) int { return j*nx + i }
	for j := 0; j < ny-1; j++ {
		y := layout.YMin + (float64(j)+0.5)*dy
		for i := 0; i < nx-1; i++ {
			x := layout.XMin + (float64(i)+0.5)*dx
			if inMissingRegion(x, y) {
				continue
			}
			v00 := id(i, j)
			v10 := id(i+1, j)
			v11 := id(i+1, j+1)
			v01 := id(i, j+1)
			mesh.AppendTri(Tri{v00, v10, v11}, FaceTag{Kind: KindScan})
			mesh.AppendTri(Tri{v00, v11, v01}, FaceTag{Kind: KindScan})
		}
	}
	mesh.Compact()
	mesh.ComputeNormals()
	return mesh, layout
}

func scanHeight(x, y float64) float64 {
	return 0.22*math.Sin(0.55*x)*math.Cos(0.50*y) +
		0.07*math.Sin(1.15*x+0.65*y) +
		0.035*math.Cos(1.7*y-0.3*x)
}

func inMissingRegion(x, y float64) bool {
	if inSlit(x, y) || inLNotch(x, y) || inCloseHoles(x, y) ||
		inNearOuterHole(x, y) || inOrdinaryHole(x, y) {
		return true
	}
	return false
}

func inSlit(x, y float64) bool {
	const cx, cy = -4.55, 0.0
	return math.Abs(x-cx) <= 1.22 && math.Abs(y-cy) <= 0.20
}

func inLNotch(x, y float64) bool {
	lx := x + 1.25
	ly := y
	verticalLeg := lx >= -1.00 && lx <= -0.34 && ly >= -1.02 && ly <= 1.02
	horizontalFoot := lx >= -1.00 && lx <= 1.02 && ly >= -1.02 && ly <= -0.34
	return verticalLeg || horizontalFoot
}

func inCloseHoles(x, y float64) bool {
	return disk(x, y, 2.62, 0.02, 0.52) || disk(x, y, 3.98, -0.02, 0.52)
}

func inOrdinaryHole(x, y float64) bool {
	return disk(x, y, 0.72, 2.72, 0.43)
}

func inNearOuterHole(x, y float64) bool {
	return disk(x, y, 5.55, 4.02, 0.32)
}

func disk(x, y, cx, cy, r float64) bool {
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

func (m *Mesh) Compact() {
	used := make([]bool, len(m.V))
	for _, t := range m.T {
		for _, idx := range t {
			used[idx] = true
		}
	}
	remap := make([]int, len(m.V))
	var verts []Vec3
	for old, ok := range used {
		remap[old] = -1
		if ok {
			remap[old] = len(verts)
			verts = append(verts, m.V[old])
		}
	}
	tris := make([]Tri, len(m.T))
	for i, t := range m.T {
		tris[i] = Tri{remap[t[0]], remap[t[1]], remap[t[2]]}
	}
	m.V = verts
	m.T = tris
	if len(m.Norm) == len(used) {
		norm := make([]Vec3, len(verts))
		for old, new := range remap {
			if new >= 0 {
				norm[new] = m.Norm[old]
			}
		}
		m.Norm = norm
	} else {
		m.Norm = nil
	}
}
