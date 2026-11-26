package main

type Segment3 struct {
	A Vec3
	B Vec3
}

func interpolateZero(a, b Vec3, va, vb float32) Vec3 {
	t := float64(-va / (vb - va))
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return a.Lerp(b, t)
}

// ExtractSliceContour uses two marching triangles per XY cell. It avoids a
// large case table while still returning sub-voxel zero crossings.
func ExtractSliceContour(g *VoxelGrid, iz int, minWeight float32) []Segment3 {
	segments := make([]Segment3, 0, 4096)
	if iz < 0 || iz >= g.NZ {
		return segments
	}

	addTriangle := func(ids [3]int, ps [3]Vec3, vs [3]float32) {
		cross := make([]Vec3, 0, 2)
		edges := [3][2]int{{0, 1}, {1, 2}, {2, 0}}
		for _, e := range edges {
			a, b := e[0], e[1]
			if (vs[a] < 0) != (vs[b] < 0) {
				cross = append(cross, interpolateZero(ps[a], ps[b], vs[a], vs[b]))
			}
		}
		if len(cross) == 2 {
			segments = append(segments, Segment3{A: cross[0], B: cross[1]})
		}
	}

	for iy := 0; iy < g.NY-1; iy++ {
		for ix := 0; ix < g.NX-1; ix++ {
			ids := [4]int{
				g.Index(ix, iy, iz),
				g.Index(ix+1, iy, iz),
				g.Index(ix+1, iy+1, iz),
				g.Index(ix, iy+1, iz),
			}
			valid := true
			for _, id := range ids {
				if g.Weight[id] < minWeight {
					valid = false
					break
				}
			}
			if !valid {
				continue
			}
			ps := [4]Vec3{
				g.Center(ix, iy, iz),
				g.Center(ix+1, iy, iz),
				g.Center(ix+1, iy+1, iz),
				g.Center(ix, iy+1, iz),
			}
			vs := [4]float32{g.Value[ids[0]], g.Value[ids[1]], g.Value[ids[2]], g.Value[ids[3]]}
			addTriangle([3]int{ids[0], ids[1], ids[2]}, [3]Vec3{ps[0], ps[1], ps[2]}, [3]float32{vs[0], vs[1], vs[2]})
			addTriangle([3]int{ids[0], ids[2], ids[3]}, [3]Vec3{ps[0], ps[2], ps[3]}, [3]float32{vs[0], vs[2], vs[3]})
		}
	}
	return segments
}

func SliceIndexForZ(g *VoxelGrid, z float64) int {
	iz := int((z - g.Min.Z) / g.Voxel)
	return clampInt(iz, 0, g.NZ-1)
}
