package qem

import (
	"math"
	"sort"
)

type simulatedFace struct {
	id            int
	old           [3]int
	next          [3]int
	oldNormal     Vec3
	oldArea2      float64
	initialNormal Vec3
}

func (s *Simplifier) prepareCollapse(e Edge) (*CollapseRecord, bool) {
	if _, protected := s.protected[e]; protected {
		return nil, false
	}
	if _, ok := s.Topo.EdgeFaces[e]; !ok {
		return nil, false
	}
	if e.A < 0 || e.A >= len(s.Mesh.Vertices) || e.B < 0 || e.B >= len(s.Mesh.Vertices) {
		return nil, false
	}
	if !s.Mesh.Vertices[e.A].Active || !s.Mesh.Vertices[e.B].Active {
		return nil, false
	}

	remove, keep := e.A, e.B
	if s.locked[remove] != "" || s.IsVertexLocked(remove) {
		if s.IsVertexLocked(keep) {
			return nil, false
		}
		remove, keep = keep, remove
	}
	if s.IsVertexLocked(remove) {
		return nil, false
	}

	affectedIDs := make(map[int]struct{})
	for _, vid := range [2]int{remove, keep} {
		for _, fid := range s.Topo.vertexFaceList(vid) {
			tri := s.Mesh.Triangles[fid]
			if tri.Active {
				affectedIDs[fid] = struct{}{}
			}
		}
	}

	removedFaces := make([]int, 0, 2)
	changes := make([]FaceChange, 0)
	simFaces := make([]simulatedFace, 0, len(affectedIDs))
	newKeys := make(map[[3]int]int)
	var oldDet6Sum float64

	for fid := range affectedIDs {
		tri := s.Mesh.Triangles[fid]
		old := tri.Vertices()
		oldDet6Sum += det6(s.Mesh.Vertices[old[0]].Pos, s.Mesh.Vertices[old[1]].Pos, s.Mesh.Vertices[old[2]].Pos)
		hasRemove := old[0] == remove || old[1] == remove || old[2] == remove
		hasKeep := old[0] == keep || old[1] == keep || old[2] == keep
		if hasRemove && hasKeep {
			removedFaces = append(removedFaces, fid)
			continue
		}

		next := old
		if hasRemove {
			for i := range next {
				if next[i] == remove {
					next[i] = keep
				}
			}
			changes = append(changes, FaceChange{FaceID: fid, Old: old, New: next})
		}
		if next[0] == next[1] || next[1] == next[2] || next[0] == next[2] {
			return nil, false
		}
		key := canonicalTriKey(next)
		newKeys[key]++
		if newKeys[key] > 1 {
			return nil, false
		}

		a := s.Mesh.Vertices[old[0]].Pos
		b := s.Mesh.Vertices[old[1]].Pos
		c := s.Mesh.Vertices[old[2]].Pos
		n := b.Sub(a).Cross(c.Sub(a))
		simFaces = append(simFaces, simulatedFace{
			id: fid, old: old, next: next, oldNormal: n, oldArea2: n.Len(),
			initialNormal: s.initialNormals[fid],
		})
	}
	if len(removedFaces) == 0 || len(simFaces) < 1 {
		return nil, false
	}
	sort.Ints(removedFaces)
	sort.Slice(changes, func(i, j int) bool { return changes[i].FaceID < changes[j].FaceID })

	var volumeGradient Vec3
	for _, sf := range simFaces {
		var j, k int
		n := sf.next
		switch keep {
		case n[0]:
			j, k = n[1], n[2]
		case n[1]:
			j, k = n[2], n[0]
		default:
			j, k = n[0], n[1]
		}
		volumeGradient = volumeGradient.Add(s.Mesh.Vertices[j].Pos.Cross(s.Mesh.Vertices[k].Pos))
	}

	removePos := s.Mesh.Vertices[remove].Pos
	keepPos := s.Mesh.Vertices[keep].Pos
	baseQ := s.Q[remove].Added(s.Q[keep])
	solveQ := baseQ
	var volumeQ Quadric
	hasVolumePlane := volumeGradient.Len() > 1e-12*s.scale*s.scale
	if hasVolumePlane {
		planeN := volumeGradient.Normalized()
		planeD := -oldDet6Sum / volumeGradient.Len()
		volumeQ = PlaneQuadric(planeN, planeD, s.constraintAreaWeight())
		solveQ.Add(volumeQ)
	}

	candidates := []Vec3{removePos, keepPos, removePos.Lerp(keepPos, 0.5)}
	if s.Opt.FreezeLockedPositions && s.IsVertexLocked(keep) {
		candidates = []Vec3{keepPos}
	} else if optPos, ok := solveQ.Optimal(); ok {
		candidates = append(candidates, optPos)
	}

	var best *CollapseRecord
	bestCost := math.Inf(1)
	for _, p := range candidates {
		if !p.finite() || p.Distance(keepPos) > 100*s.scale {
			continue
		}
		quadricCost := math.Max(0, baseQ.Eval(p))
		volumeCost := 0.0
		if hasVolumePlane {
			volumeCost = math.Max(0, volumeQ.Eval(p))
		}

		normalCost := 0.0
		legal := true
		for _, sf := range simFaces {
			if sf.oldArea2 <= 1e-14*s.scale*s.scale {
				legal = false
				break
			}
			pos := func(id int) Vec3 {
				if id == keep {
					return p
				}
				return s.Mesh.Vertices[id].Pos
			}
			a := pos(sf.next[0])
			b := pos(sf.next[1])
			c := pos(sf.next[2])
			newN := b.Sub(a).Cross(c.Sub(a))
			newArea2 := newN.Len()
			if newArea2 <= 1e-14*s.scale*s.scale {
				legal = false
				break
			}
			newUnit := newN.Mul(1 / newArea2)
			oldUnit := sf.oldNormal.Mul(1 / sf.oldArea2)
			dot := oldUnit.Dot(newUnit)
			if math.IsNaN(dot) {
				legal = false
				break
			}
			dot = math.Max(-1, math.Min(1, dot))
			if dot <= s.Opt.FlipDotLimit {
				legal = false
				break
			}
			if sf.initialNormal.LenSquared() > 0 {
				initialDot := sf.initialNormal.Dot(newUnit)
				if math.IsNaN(initialDot) || initialDot <= s.Opt.FlipDotLimit {
					legal = false
					break
				}
			}
			deviation := 1 - dot
			area := sf.oldArea2 * 0.5
			normalCost += s.Opt.NormalWeight * s.scale * s.scale * area * deviation * deviation
		}
		if !legal {
			continue
		}

		deltaVolume6 := p.Dot(volumeGradient) - oldDet6Sum
		deltaVolume := deltaVolume6 / 6
		projectedPhysical := s.volumeSign * (s.volumeSigned6 + deltaVolume6) / 6
		volumeFloor := s.initialVolume - s.allowedLoss
		if projectedPhysical < volumeFloor {
			continue
		}

		cost := quadricCost + volumeCost + normalCost
		if cost < bestCost {
			bestCost = cost
			best = &CollapseRecord{
				Edge: e, RemovedVertex: remove, KeptVertex: keep,
				RemovedOldPos:  removePos,
				KeptOldPos:     keepPos,
				NewPosition:    p,
				KeptOldQuadric: s.Q[keep],
				RemovedFaces:   append([]int(nil), removedFaces...),
				MovedFaces:     append([]FaceChange(nil), changes...),
				Cost:           cost,
				QuadricCost:    quadricCost,
				VolumeCost:     volumeCost,
				NormalCost:     normalCost,
				DeltaVolume:    s.volumeSign * deltaVolume,
				VolumeAfter:    projectedPhysical,
			}
		}
	}
	return best, best != nil
}
