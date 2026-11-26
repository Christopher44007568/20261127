package qem

// Topology stores updatable incidence sets. Sets make local collapse updates
// and stale heap validation straightforward.
type Topology struct {
	EdgeFaces map[Edge]map[int]struct{} `json:"-"`
	VertFaces map[int]map[int]struct{}  `json:"-"`
}

func NewTopology() *Topology {
	return &Topology{
		EdgeFaces: make(map[Edge]map[int]struct{}),
		VertFaces: make(map[int]map[int]struct{}),
	}
}

func BuildTopology(m *Mesh) *Topology {
	t := NewTopology()
	for id, tri := range m.Triangles {
		if tri.Active {
			t.AddFace(id, tri.Vertices())
		}
	}
	return t
}

func (t *Topology) addSet(m map[int]struct{}, id int) map[int]struct{} {
	if m == nil {
		m = make(map[int]struct{})
	}
	m[id] = struct{}{}
	return m
}

func (t *Topology) AddFace(faceID int, v [3]int) {
	edges := [3]Edge{NewEdge(v[0], v[1]), NewEdge(v[1], v[2]), NewEdge(v[2], v[0])}
	for _, e := range edges {
		set := t.EdgeFaces[e]
		if set == nil {
			set = make(map[int]struct{})
			t.EdgeFaces[e] = set
		}
		set[faceID] = struct{}{}
	}
	for _, vid := range v {
		set := t.VertFaces[vid]
		if set == nil {
			set = make(map[int]struct{})
			t.VertFaces[vid] = set
		}
		set[faceID] = struct{}{}
	}
}

func (t *Topology) RemoveFace(faceID int, v [3]int) {
	edges := [3]Edge{NewEdge(v[0], v[1]), NewEdge(v[1], v[2]), NewEdge(v[2], v[0])}
	for _, e := range edges {
		if set := t.EdgeFaces[e]; set != nil {
			delete(set, faceID)
			if len(set) == 0 {
				delete(t.EdgeFaces, e)
			}
		}
	}
	for _, vid := range v {
		if set := t.VertFaces[vid]; set != nil {
			delete(set, faceID)
			if len(set) == 0 {
				delete(t.VertFaces, vid)
			}
		}
	}
}

func (t *Topology) edgeFaceList(e Edge) []int {
	set := t.EdgeFaces[e]
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func (t *Topology) vertexFaceList(v int) []int {
	set := t.VertFaces[v]
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}
