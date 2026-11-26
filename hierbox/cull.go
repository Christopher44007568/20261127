package hierbox

import "math"

type CullStatus string

const (
	CullInside       CullStatus = "inside"
	CullIntersecting CullStatus = "intersecting"
	CullOutside      CullStatus = "outside"
)

// Plane is the half-space Normal.Dot(X)+Distance >= 0.
type Plane struct {
	Normal   Vec3    `json:"normal"`
	Distance float64 `json:"distance"`
}

func newPlane(normal, point Vec3) Plane {
	n := normal.Normalized()
	return Plane{Normal: n, Distance: -n.Dot(point)}
}

func (p Plane) SignedDistance(x Vec3) float64 {
	return p.Normal.Dot(x) + p.Distance
}

type Frustum struct {
	Planes [6]Plane `json:"planes"` // near, far, left, right, bottom, top
}

// Camera is a small perspective camera used for culling and report diagrams.
type Camera struct {
	Name   string  `json:"name"`
	Eye    Vec3    `json:"eye"`
	Target Vec3    `json:"target"`
	Up     Vec3    `json:"up"`
	FOVY   float64 `json:"fovYDegrees"`
	Aspect float64 `json:"aspect"`
	Near   float64 `json:"near"`
	Far    float64 `json:"far"`
}

func (c Camera) Frustum() Frustum {
	f := c.Target.Sub(c.Eye).Normalized()
	r := f.Cross(c.Up).Normalized()
	u := r.Cross(f).Normalized()

	tanHalf := math.Tan(deg(c.FOVY) * 0.5)
	hNear := tanHalf * c.Near
	wNear := hNear * c.Aspect
	nearCenter := c.Eye.Add(f.Scale(c.Near))
	farCenter := c.Eye.Add(f.Scale(c.Far))

	top := f.Scale(c.Near).Add(u.Scale(hNear))
	bottom := f.Scale(c.Near).Sub(u.Scale(hNear))
	right := f.Scale(c.Near).Add(r.Scale(wNear))
	left := f.Scale(c.Near).Sub(r.Scale(wNear))

	return Frustum{Planes: [6]Plane{
		newPlane(f, nearCenter),
		newPlane(f.Scale(-1), farCenter),
		newPlane(left.Cross(u), c.Eye),
		newPlane(u.Cross(right), c.Eye),
		newPlane(r.Cross(bottom), c.Eye),
		newPlane(top.Cross(r), c.Eye),
	}}
}

// ClassifyBox tests an AABB against all frustum planes.
func (f Frustum) ClassifyBox(b Box3, eps float64) CullStatus {
	if b.Empty {
		return CullOutside
	}
	c := b.Center()
	s := b.Size().Scale(0.5)
	allInside := true
	for _, p := range f.Planes {
		reach := s.X*math.Abs(p.Normal.X) + s.Y*math.Abs(p.Normal.Y) + s.Z*math.Abs(p.Normal.Z)
		d := p.SignedDistance(c)
		if d+reach < -eps {
			return CullOutside
		}
		if d-reach < -eps {
			allInside = false
		}
	}
	if allInside {
		return CullInside
	}
	return CullIntersecting
}

type TestedNode struct {
	ID     BoneID     `json:"id"`
	Name   string     `json:"name"`
	Depth  int        `json:"depth"`
	Status CullStatus `json:"status"`
}

type CullStats struct {
	TotalNodes         int `json:"totalNodes"`
	FrustumTests       int `json:"frustumTests"`
	InsideTested       int `json:"insideTested"`
	IntersectingTested int `json:"intersectingTested"`
	OutsideTested      int `json:"outsideTested"`
	VisibleBones       int `json:"visibleBones"`
	CulledBones        int `json:"culledBones"`
	AcceptedSubtrees   int `json:"acceptedSubtrees"`
	PrunedOutsideRoots int `json:"prunedOutsideRoots"`
	SavedFrustumTests  int `json:"savedFrustumTests"`
}

type CullResult struct {
	Camera       Camera       `json:"camera"`
	Stats        CullStats    `json:"stats"`
	Tested       []TestedNode `json:"tested"`
	VisibleBones []BoneID     `json:"visibleBones"`
	CulledBones  []BoneID     `json:"culledBones"`
}

func subtreeIDs(h *Hierarchy, id BoneID, out *[]BoneID) {
	*out = append(*out, id)
	for _, child := range h.Nodes[id].Children {
		subtreeIDs(h, child, out)
	}
}

func subtreeSize(h *Hierarchy, id BoneID) int {
	n := 1
	for _, child := range h.Nodes[id].Children {
		n += subtreeSize(h, child)
	}
	return n
}

// Cull performs topology-driven frustum culling. An outside node prunes its
// whole subtree; a fully inside node accepts its whole subtree.
func (h *Hierarchy) Cull(camera Camera) CullResult {
	frustum := camera.Frustum()
	res := CullResult{Camera: camera}
	res.Stats.TotalNodes = len(h.Nodes)

	var visit func(id BoneID)
	visit = func(id BoneID) {
		n := &h.Nodes[id]
		status := frustum.ClassifyBox(n.World, 1e-7)
		res.Stats.FrustumTests++
		res.Tested = append(res.Tested, TestedNode{ID: id, Name: n.Name, Depth: n.Depth, Status: status})

		switch status {
		case CullOutside:
			res.Stats.OutsideTested++
			res.Stats.PrunedOutsideRoots++
			before := len(res.CulledBones)
			subtreeIDs(h, id, &res.CulledBones)
			size := len(res.CulledBones) - before
			res.Stats.CulledBones += size
			res.Stats.SavedFrustumTests += size - 1
		case CullInside:
			res.Stats.InsideTested++
			res.Stats.AcceptedSubtrees++
			before := len(res.VisibleBones)
			subtreeIDs(h, id, &res.VisibleBones)
			size := len(res.VisibleBones) - before
			res.Stats.VisibleBones += size
			res.Stats.SavedFrustumTests += size - 1
		default:
			res.Stats.IntersectingTested++
			res.VisibleBones = append(res.VisibleBones, id)
			res.Stats.VisibleBones++
			for _, child := range n.Children {
				visit(child)
			}
		}
	}

	for _, root := range h.Skeleton.Roots {
		visit(root)
	}
	return res
}

// ClassifyAll returns frustum status for every node without early traversal.
// Reports use it to color the complete hierarchy picture.
func (h *Hierarchy) ClassifyAll(camera Camera) []TestedNode {
	f := camera.Frustum()
	out := make([]TestedNode, 0, len(h.Nodes))
	for _, id := range h.Skeleton.Order {
		n := &h.Nodes[id]
		out = append(out, TestedNode{
			ID: id, Name: n.Name, Depth: n.Depth,
			Status: f.ClassifyBox(n.World, 1e-7),
		})
	}
	return out
}

// RecommendedCullCamera gives a narrow three-quarter framing of the upper
// body. Feet and tail tips are then genuine hierarchy-level cull cases.
func RecommendedCullCamera() Camera {
	return Camera{
		Name: "upper_body_narrow",
		Eye:  V(2.35, 1.55, 2.15), Target: V(0.10, 1.08, 0),
		Up: V(0, 1, 0), FOVY: 20, Aspect: 1.15, Near: 0.10, Far: 8.0,
	}
}

var _ = subtreeSize
