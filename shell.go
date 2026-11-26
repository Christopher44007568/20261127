package holefill

import "math"

func Thicken(m *Mesh, thickness float64) *Mesh {
	if len(m.Norm) != len(m.V) {
		m.ComputeNormals()
	}
	out := &Mesh{}
	nv := len(m.V)
	out.V = append(out.V, m.V...)
	for i, p := range m.V {
		n := m.Norm[i]
		if n.Len() < 0.5 {
			n = Vec3{0, 0, 1}
		}
		out.V = append(out.V, p.Sub(n.Scale(thickness)))
	}

	for ti, t := range m.T {
		tag := FaceTag{Kind: KindScan}
		if ti < len(m.Tags) {
			tag = m.Tags[ti]
		}
		out.AppendTri(t, tag)
		bottom := Tri{nv + t[0], nv + t[2], nv + t[1]}
		bottomTag := tag
		bottomTag.Kind = KindBottom
		out.AppendTri(bottom, bottomTag)
	}

	for _, e := range m.BoundaryEdges() {
		a, b := e.A, e.B
		ab, bb := nv+a, nv+b
		wallTag := FaceTag{Kind: KindWall, Class: ClassOuter}
		out.AppendTri(Tri{a, bb, b}, wallTag)
		out.AppendTri(Tri{a, ab, bb}, wallTag)
	}
	out.ComputeNormals()
	return out
}

type SectionSegment struct {
	X1, Z1, X2, Z2 float64
	Tag            FaceTag
}

func BuildSection(m *Mesh, planeY float64) []SectionSegment {
	var segments []SectionSegment
	for ti, tri := range m.T {
		tag := FaceTag{Kind: KindScan}
		if ti < len(m.Tags) {
			tag = m.Tags[ti]
		}
		var hits []Vec3
		coplanar := false
		for e := 0; e < 3; e++ {
			a := m.V[tri[e]]
			b := m.V[tri[(e+1)%3]]
			da := a.Y - planeY
			db := b.Y - planeY
			eps := 1e-7
			if math.Abs(da) <= eps && math.Abs(db) <= eps {
				segments = append(segments, SectionSegment{a.X, a.Z, b.X, b.Z, tag})
				coplanar = true
				continue
			}
			if math.Abs(da) <= eps {
				hits = appendHit(hits, a)
			}
			if math.Abs(db) <= eps {
				hits = appendHit(hits, b)
			}
			if da*db < 0 {
				t := da / (da - db)
				hits = appendHit(hits, a.Add(b.Sub(a).Scale(t)))
			}
		}
		if coplanar {
			continue
		}
		if len(hits) >= 2 {
			a, b := farthestHitPair(hits)
			segments = append(segments, SectionSegment{a.X, a.Z, b.X, b.Z, tag})
		}
	}
	return segments
}

func appendHit(hits []Vec3, p Vec3) []Vec3 {
	for _, q := range hits {
		if q.Distance(p) < 1e-6 {
			return hits
		}
	}
	return append(hits, p)
}

func farthestHitPair(hits []Vec3) (Vec3, Vec3) {
	best := -1.0
	a, b := hits[0], hits[1]
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			d := hits[i].Distance(hits[j])
			if d > best {
				best = d
				a, b = hits[i], hits[j]
			}
		}
	}
	return a, b
}
