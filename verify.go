package qem

import "math"

type VerificationReport struct {
	ActiveTriangles           int      `json:"active_triangles"`
	ActiveVertices            int      `json:"active_vertices"`
	DegenerateFaces           []int    `json:"degenerate_faces"`
	FlippedFaces              []int    `json:"flipped_faces"`
	DuplicateTopologyKeys     []string `json:"duplicate_topology_keys,omitempty"`
	MissingProtectedEdges     []Edge   `json:"missing_protected_edges"`
	InactiveProtectedVertices []int    `json:"inactive_protected_vertices"`
	FrozenPositionViolations  []Edge   `json:"frozen_position_violations"`
}

func (r VerificationReport) OK() bool {
	return len(r.DegenerateFaces) == 0 &&
		len(r.FlippedFaces) == 0 &&
		len(r.DuplicateTopologyKeys) == 0 &&
		len(r.MissingProtectedEdges) == 0 &&
		len(r.InactiveProtectedVertices) == 0 &&
		len(r.FrozenPositionViolations) == 0
}

func (s *Simplifier) ProtectedEdgeCounts() map[string]int {
	out := make(map[string]int)
	for _, reason := range s.protected {
		out[reason]++
	}
	return out
}

// Verify checks hard guarantees against the untouched original mesh clone.
func (s *Simplifier) Verify(original *Mesh) VerificationReport {
	report := VerificationReport{
		DegenerateFaces:           []int{},
		FlippedFaces:              []int{},
		DuplicateTopologyKeys:     []string{},
		MissingProtectedEdges:     []Edge{},
		InactiveProtectedVertices: []int{},
		FrozenPositionViolations:  []Edge{},
	}
	m := s.Mesh
	report.ActiveTriangles = m.ActiveTriangleCount()
	report.ActiveVertices = m.ActiveVertexCount()
	scale2 := s.scale * s.scale

	for fid, tri := range m.Triangles {
		if !tri.Active {
			continue
		}
		n, area2 := m.FaceGeometry(fid)
		if area2 <= 1e-12*scale2 {
			report.DegenerateFaces = append(report.DegenerateFaces, fid)
			continue
		}
		if original != nil && fid < len(original.Triangles) && original.Triangles[fid].Active {
			oldN, oldArea2 := original.FaceGeometry(fid)
			if oldArea2 > 1e-12*scale2 {
				dot := oldN.Dot(n) / (oldArea2 * area2)
				if math.IsNaN(dot) || dot <= 0 {
					report.FlippedFaces = append(report.FlippedFaces, fid)
				}
			}
		}
	}

	keys := make(map[[3]int]int)
	for _, tri := range m.Triangles {
		if !tri.Active {
			continue
		}
		key := canonicalTriKey(tri.Vertices())
		keys[key]++
	}
	for key, count := range keys {
		if count > 1 {
			report.DuplicateTopologyKeys = append(report.DuplicateTopologyKeys, triKeyString(key))
		}
	}

	for e := range s.protected {
		aOK := e.A >= 0 && e.A < len(m.Vertices) && m.Vertices[e.A].Active
		bOK := e.B >= 0 && e.B < len(m.Vertices) && m.Vertices[e.B].Active
		if !aOK || !bOK {
			if !aOK {
				report.InactiveProtectedVertices = append(report.InactiveProtectedVertices, e.A)
			}
			if !bOK {
				report.InactiveProtectedVertices = append(report.InactiveProtectedVertices, e.B)
			}
			continue
		}
		if _, exists := s.Topo.EdgeFaces[e]; !exists {
			report.MissingProtectedEdges = append(report.MissingProtectedEdges, e)
		}
		if s.Opt.FreezeLockedPositions && original != nil {
			if m.Vertices[e.A].Pos.Distance(original.Vertices[e.A].Pos) > 1e-10*s.scale ||
				m.Vertices[e.B].Pos.Distance(original.Vertices[e.B].Pos) > 1e-10*s.scale {
				report.FrozenPositionViolations = append(report.FrozenPositionViolations, e)
			}
		}
	}
	return report
}

func triKeyString(k [3]int) string {
	return "{" + itoa(k[0]) + "," + itoa(k[1]) + "," + itoa(k[2]) + "}"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
