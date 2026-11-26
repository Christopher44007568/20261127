package qem

import (
	"container/heap"
	"errors"
)

func (s *Simplifier) refreshEdge(e Edge) {
	ver := s.bumpEdge(e)
	rec, ok := s.prepareCollapse(e)
	if !ok {
		return
	}
	heap.Push(&s.heap, &heapItem{edge: e, cost: rec.Cost, version: ver})
}

// CollapseOne performs the lowest-cost legal edge collapse and records enough
// state to reverse it with Undo.
func (s *Simplifier) CollapseOne() (*CollapseRecord, error) {
	if !s.heapReady {
		if err := s.rebuildHeap(); err != nil {
			return nil, err
		}
	}

	triedRebuild := false
	for {
		for s.heap.Len() > 0 {
			item := heap.Pop(&s.heap).(*heapItem)
			if item.version != s.edgeVer[item.edge] {
				continue
			}
			if _, ok := s.Topo.EdgeFaces[item.edge]; !ok {
				continue
			}
			rec, ok := s.prepareCollapse(item.edge)
			if !ok {
				continue
			}
			if err := s.applyCollapse(rec); err != nil {
				continue
			}
			return rec, nil
		}
		if triedRebuild {
			return nil, ErrNoCollapsibleEdge
		}
		triedRebuild = true
		if err := s.rebuildHeap(); err != nil {
			return nil, err
		}
	}
}

func (s *Simplifier) applyCollapse(rec *CollapseRecord) error {
	m := s.Mesh
	for _, fid := range rec.RemovedFaces {
		tri := m.Triangles[fid]
		s.Topo.RemoveFace(fid, tri.Vertices())
		tri.Active = false
		m.Triangles[fid] = tri
	}
	for _, change := range rec.MovedFaces {
		s.Topo.RemoveFace(change.FaceID, change.Old)
		tri := m.Triangles[change.FaceID]
		tri.SetVertices(change.New)
		m.Triangles[change.FaceID] = tri
		s.Topo.AddFace(change.FaceID, change.New)
	}

	m.Vertices[rec.KeptVertex].Pos = rec.NewPosition
	m.Vertices[rec.KeptVertex].Active = true
	m.Vertices[rec.RemovedVertex].Active = false

	s.Q[rec.KeptVertex] = rec.KeptOldQuadric
	s.Q[rec.KeptVertex].Add(s.Q[rec.RemovedVertex])
	s.volumeSigned6 += 6 * s.volumeSign * rec.DeltaVolume

	dirty := make(map[Edge]struct{})
	for _, fid := range s.Topo.vertexFaceList(rec.KeptVertex) {
		tri := m.Triangles[fid]
		if !tri.Active {
			continue
		}
		vs := tri.Vertices()
		dirty[NewEdge(vs[0], vs[1])] = struct{}{}
		dirty[NewEdge(vs[1], vs[2])] = struct{}{}
		dirty[NewEdge(vs[2], vs[0])] = struct{}{}
	}
	for e := range dirty {
		s.refreshEdge(e)
	}

	rec.Index = len(s.Records)
	s.Records = append(s.Records, rec)
	return nil
}

// CollapseToTriangles collapses until at most target active triangles remain or
// no legal collapse exists. The returned count is the number of collapses.
func (s *Simplifier) CollapseToTriangles(target int) (int, error) {
	steps := 0
	for s.Mesh.ActiveTriangleCount() > target {
		_, err := s.CollapseOne()
		if err != nil {
			if errors.Is(err, ErrNoCollapsibleEdge) {
				return steps, err
			}
			return steps, err
		}
		steps++
	}
	return steps, nil
}

// Undo reverses the most recent collapse exactly.
func (s *Simplifier) Undo() (*CollapseRecord, error) {
	if len(s.Records) == 0 {
		return nil, ErrNothingToUndo
	}
	last := len(s.Records) - 1
	rec := s.Records[last]
	m := s.Mesh

	for _, change := range rec.MovedFaces {
		s.Topo.RemoveFace(change.FaceID, change.New)
		s.Topo.AddFace(change.FaceID, change.Old)
	}
	for _, fid := range rec.RemovedFaces {
		s.Topo.AddFace(fid, m.Triangles[fid].Vertices())
	}

	for _, change := range rec.MovedFaces {
		tri := m.Triangles[change.FaceID]
		tri.SetVertices(change.Old)
		tri.Active = true
		m.Triangles[change.FaceID] = tri
	}
	for _, fid := range rec.RemovedFaces {
		m.Triangles[fid].Active = true
	}
	m.Vertices[rec.KeptVertex].Pos = rec.KeptOldPos
	m.Vertices[rec.KeptVertex].Active = true
	m.Vertices[rec.RemovedVertex].Pos = rec.RemovedOldPos
	m.Vertices[rec.RemovedVertex].Active = true
	s.Q[rec.KeptVertex] = rec.KeptOldQuadric
	s.volumeSigned6 -= 6 * s.volumeSign * rec.DeltaVolume

	s.Records = s.Records[:last]
	s.heap = nil
	s.heapReady = false
	s.edgeVer = make(map[Edge]uint64)
	return rec, nil
}

// UndoAll restores the original mesh state through vertex splits.
func (s *Simplifier) UndoAll() error {
	for len(s.Records) > 0 {
		if _, err := s.Undo(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Simplifier) CollapsedEdges() []Edge {
	out := make([]Edge, 0, len(s.Records))
	for _, rec := range s.Records {
		out = append(out, rec.Edge)
	}
	return out
}
