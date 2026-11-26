package qem

import (
	"errors"
	"math"
	"sort"
)

var (
	ErrNoCollapsibleEdge = errors.New("qem: no legal collapsible edge")
	ErrNothingToUndo     = errors.New("qem: no collapse record to undo")
)

// Options controls QEM simplification and hard preservation constraints.
type Options struct {
	ProtectBoundary         bool    `json:"protect_boundary"`
	ProtectMaterialSeams    bool    `json:"protect_material_seams"`
	ProtectSharpNormals     bool    `json:"protect_sharp_normals"`
	SharpNormalAngleDegrees float64 `json:"sharp_normal_angle_degrees"`
	LockProtectedVertices   bool    `json:"lock_protected_vertices"`
	FreezeLockedPositions   bool    `json:"freeze_locked_positions"`
	EdgeConstraintWeight    float64 `json:"edge_constraint_weight"`
	VolumeWeight            float64 `json:"volume_weight"`
	NormalWeight            float64 `json:"normal_weight"`
	MaxVolumeLoss           float64 `json:"max_volume_loss"`
	FlipDotLimit            float64 `json:"flip_dot_limit"`
}

func DefaultOptions() Options {
	return Options{
		ProtectBoundary:         true,
		ProtectMaterialSeams:    true,
		ProtectSharpNormals:     true,
		SharpNormalAngleDegrees: 70,
		LockProtectedVertices:   true,
		FreezeLockedPositions:   true,
		EdgeConstraintWeight:    1e4,
		VolumeWeight:            100,
		NormalWeight:            20,
		MaxVolumeLoss:           0.005,
		FlipDotLimit:            1e-6,
	}
}

// FaceChange stores one triangle's vertices before a collapse. Undo restores
// the exact stable IDs.
type FaceChange struct {
	FaceID int    `json:"face_id"`
	Old    [3]int `json:"old"`
	New    [3]int `json:"new"`
}

// CollapseRecord is a reversible progressive-mesh operation.
type CollapseRecord struct {
	Index          int          `json:"index"`
	Edge           Edge         `json:"edge"`
	RemovedVertex  int          `json:"removed_vertex"`
	KeptVertex     int          `json:"kept_vertex"`
	RemovedOldPos  Vec3         `json:"removed_old_pos"`
	KeptOldPos     Vec3         `json:"kept_old_pos"`
	NewPosition    Vec3         `json:"new_position"`
	KeptOldQuadric Quadric      `json:"kept_old_quadric"`
	RemovedFaces   []int        `json:"removed_faces"`
	MovedFaces     []FaceChange `json:"moved_faces"`
	Cost           float64      `json:"cost"`
	QuadricCost    float64      `json:"quadric_cost"`
	VolumeCost     float64      `json:"volume_cost"`
	NormalCost     float64      `json:"normal_cost"`
	DeltaVolume    float64      `json:"delta_volume"`
	VolumeAfter    float64      `json:"volume_after"`
	ProtectedEdge  bool         `json:"protected_edge,omitempty"`
}

// Simplifier owns mutable mesh state, quadrics, topology, and history.
type Simplifier struct {
	Mesh           *Mesh             `json:"mesh"`
	Opt            Options           `json:"options"`
	Topo           *Topology         `json:"-"`
	Q              []Quadric         `json:"-"`
	Records        []*CollapseRecord `json:"records"`
	initialNormals []Vec3            `json:"-"`

	protected map[Edge]string
	locked    map[int]string
	edgeVer   map[Edge]uint64
	heap      candidateHeap
	heapReady bool

	scale         float64
	initialVolume float64
	volumeSign    float64
	allowedLoss   float64
	volumeSigned6 float64
}

func NewSimplifier(m *Mesh, opt Options) (*Simplifier, error) {
	if m == nil || m.ActiveTriangleCount() == 0 {
		return nil, ErrNoCollapsibleEdge
	}
	if opt.SharpNormalAngleDegrees == 0 {
		opt.SharpNormalAngleDegrees = DefaultOptions().SharpNormalAngleDegrees
	}
	if opt.FlipDotLimit == 0 {
		opt.FlipDotLimit = 1e-6
	}
	s := &Simplifier{
		Mesh:      m,
		Opt:       opt,
		Topo:      BuildTopology(m),
		Q:         make([]Quadric, len(m.Vertices)),
		protected: make(map[Edge]string),
		locked:    make(map[int]string),
		edgeVer:   make(map[Edge]uint64),
		scale:     m.MaxDimension(),
	}
	s.volumeSigned6 = m.SignedVolume6()
	s.initialNormals = make([]Vec3, len(m.Triangles))
	for fid, tri := range m.Triangles {
		if tri.Active {
			s.initialNormals[fid] = m.FaceNormal(fid)
		}
	}
	s.initialVolume = math.Abs(s.volumeSigned6) / 6
	if s.volumeSigned6 < 0 {
		s.volumeSign = -1
	} else {
		s.volumeSign = 1
	}
	volumeBasis := math.Max(math.Abs(s.volumeSigned6)/6, 1e-3*math.Pow(s.scale, 3))
	s.allowedLoss = opt.MaxVolumeLoss * volumeBasis

	s.markProtectedEdges()
	s.initQuadrics()
	if err := s.rebuildHeap(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Simplifier) MarkProtectedEdge(a, b int, reason string) {
	e := NewEdge(a, b)
	if reason == "" {
		reason = "user"
	}
	s.protected[e] = reason
	if s.Opt.LockProtectedVertices {
		s.locked[e.A] = reason
		s.locked[e.B] = reason
	}
	s.edgeVer[e]++
	s.heapReady = false
}

func (s *Simplifier) ProtectedReason(e Edge) string {
	return s.protected[e]
}

func (s *Simplifier) IsVertexLocked(v int) bool {
	_, ok := s.locked[v]
	return ok
}

func (s *Simplifier) markProtectedEdges() {
	sharpCos := math.Cos(s.Opt.SharpNormalAngleDegrees * math.Pi / 180)
	edges := make([]Edge, 0, len(s.Topo.EdgeFaces))
	for e := range s.Topo.EdgeFaces {
		edges = append(edges, e)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].A != edges[j].A {
			return edges[i].A < edges[j].A
		}
		return edges[i].B < edges[j].B
	})
	for _, e := range edges {
		faces := s.Topo.edgeFaceList(e)
		reason := ""
		switch {
		case len(faces) == 1 && s.Opt.ProtectBoundary:
			reason = "boundary"
		case len(faces) > 2:
			reason = "non_manifold"
		case len(faces) == 2:
			t0 := s.Mesh.Triangles[faces[0]]
			t1 := s.Mesh.Triangles[faces[1]]
			if s.Opt.ProtectMaterialSeams && t0.Material != t1.Material {
				reason = "material_seam"
			} else if s.Opt.ProtectSharpNormals {
				n0 := s.Mesh.FaceNormal(faces[0])
				n1 := s.Mesh.FaceNormal(faces[1])
				if n0.Dot(n1) < sharpCos {
					reason = "sharp_normal"
				}
			}
		}
		if reason != "" {
			s.protected[e] = reason
			if s.Opt.LockProtectedVertices {
				s.locked[e.A] = reason
				s.locked[e.B] = reason
			}
		}
	}
}

func (s *Simplifier) initQuadrics() {
	for id, tri := range s.Mesh.Triangles {
		if !tri.Active {
			continue
		}
		n, area2 := s.Mesh.FaceGeometry(id)
		if area2 <= 1e-14*s.scale*s.scale {
			continue
		}
		normal := n.Mul(1 / area2)
		d := -normal.Dot(s.Mesh.Vertices[tri.A].Pos)
		q := PlaneQuadric(normal, d, area2*0.5)
		s.Q[tri.A].Add(q)
		s.Q[tri.B].Add(q)
		s.Q[tri.C].Add(q)
	}

	edgeWeight := s.Opt.EdgeConstraintWeight * s.scale * s.scale
	for e := range s.protected {
		faces := s.Topo.edgeFaceList(e)
		edgeDir := s.Mesh.Vertices[e.B].Pos.Sub(s.Mesh.Vertices[e.A].Pos).Normalized()
		if edgeDir.Len() == 0 {
			continue
		}
		for _, fid := range faces {
			if !s.Mesh.Triangles[fid].Active {
				continue
			}
			faceN := s.Mesh.FaceNormal(fid)
			constraintN := edgeDir.Cross(faceN).Normalized()
			if constraintN.Len() == 0 {
				continue
			}
			d := -constraintN.Dot(s.Mesh.Vertices[e.A].Pos)
			q := PlaneQuadric(constraintN, d, edgeWeight)
			s.Q[e.A].Add(q)
			s.Q[e.B].Add(q)
		}
	}
}

func (s *Simplifier) constraintAreaWeight() float64 {
	return s.Opt.VolumeWeight * s.scale * s.scale
}
