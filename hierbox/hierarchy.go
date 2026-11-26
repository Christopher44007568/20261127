package hierbox

import (
	"fmt"
	"sort"
)

// Node is persistent storage for one bone's hierarchical bounding box.
type Node struct {
	ID       BoneID
	Name     string
	Parent   BoneID
	Children []BoneID
	Depth    int

	Local Box3 // box in the bone's local coordinate frame
	World Box3 // conservative AABB after the current pose

	LocalMatrix Mat4
	WorldMatrix Mat4

	matrixDirty bool
	boundsDirty bool
}

// UpdateStats reports the amount of work performed by one incremental update.
type UpdateStats struct {
	Nodes              int      `json:"nodes"`
	MatrixUpdates      int      `json:"matrixUpdates"`
	BoundsUpdates      int      `json:"boundsUpdates"`
	StructuralRebuilds int      `json:"structuralRebuilds"`
	FullTreeRebuilt    bool     `json:"fullTreeRebuilt"`
	ChangedBones       []BoneID `json:"changedBones,omitempty"`
	BoundsOrder        []BoneID `json:"boundsOrder,omitempty"`
}

// Hierarchy is a bone-topology bounding volume hierarchy. The node slice is
// allocated once; pose updates mutate nodes and never rebuild the tree.
type Hierarchy struct {
	Skeleton *Skeleton
	Nodes    []Node
	Padding  float64

	structuralRebuilds int
	lastStats          UpdateStats
	frame              uint64
	orderIndex         []int
	pending            map[BoneID]struct{}
	listedFrame        []uint64
}

// BVH is a naming alias for the bone-topology bounding volume hierarchy.
type BVH = Hierarchy

// NewBVH is the conventional constructor alias for NewHierarchy.
func NewBVH(s *Skeleton, local []Box3, padding float64) (*BVH, error) {
	return NewHierarchy(s, local, padding)
}

// NewHierarchy creates persistent nodes from local bone boxes.
func NewHierarchy(s *Skeleton, local []Box3, padding float64) (*Hierarchy, error) {
	if s == nil {
		return nil, fmt.Errorf("hierbox: nil skeleton")
	}
	if len(local) != len(s.Bones) {
		return nil, fmt.Errorf("hierbox: got %d local boxes for %d bones", len(local), len(s.Bones))
	}
	if padding < 0 {
		return nil, fmt.Errorf("hierbox: padding must be non-negative")
	}
	n := len(s.Bones)
	orderIndex := make([]int, n)
	for index, id := range s.Order {
		orderIndex[id] = index
	}
	h := &Hierarchy{
		Skeleton:    s,
		Nodes:       make([]Node, n),
		Padding:     padding,
		orderIndex:  orderIndex,
		pending:     make(map[BoneID]struct{}, n),
		listedFrame: make([]uint64, n),
	}
	for _, b := range s.Bones {
		box := local[b.ID]
		if padding > 0 && !box.Empty {
			box = box.Expanded(padding)
		}
		h.Nodes[b.ID] = Node{
			ID:          b.ID,
			Name:        b.Name,
			Parent:      b.Parent,
			Children:    append([]BoneID(nil), s.Children[b.ID]...),
			Depth:       s.Depth[b.ID],
			Local:       box,
			LocalMatrix: TRS(b.Rest.Position, b.Rest.Rotation, b.Rest.Scale),
			matrixDirty: true,
			boundsDirty: true,
		}
		h.pending[b.ID] = struct{}{}
	}
	if _, err := h.Update(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Hierarchy) checkID(id BoneID) error {
	if int(id) < 0 || int(id) >= len(h.Nodes) {
		return fmt.Errorf("hierbox: invalid bone id %d", id)
	}
	return nil
}

// SetLocal replaces one bone-local TRS transform and marks that bone dirty.
// Descendant matrices are handled during Update by topology traversal.
func (h *Hierarchy) SetLocal(id BoneID, tr Transform) error {
	if err := h.checkID(id); err != nil {
		return err
	}
	h.Nodes[id].LocalMatrix = PoseMatrix(tr)
	h.Nodes[id].matrixDirty = true
	h.Nodes[id].boundsDirty = true
	h.pending[id] = struct{}{}
	return nil
}

// SetLocals batches a complete pose. It preserves all nodes and topology.
func (h *Hierarchy) SetLocals(poses []Transform) error {
	if len(poses) != len(h.Nodes) {
		return fmt.Errorf("hierbox: pose has %d transforms for %d nodes", len(poses), len(h.Nodes))
	}
	changed := make([]BoneID, 0, len(poses))
	for id, tr := range poses {
		if err := h.SetLocal(BoneID(id), tr); err != nil {
			return err
		}
		changed = append(changed, BoneID(id))
	}
	return nil
}

// SetLocalBox refits a bone's local box and marks its bounds dirty. This is
// useful when a rigid or skinned leaf chunk is explicitly refit.
func (h *Hierarchy) SetLocalBox(id BoneID, box Box3) error {
	if err := h.checkID(id); err != nil {
		return err
	}
	if h.Padding > 0 && !box.Empty {
		box = box.Expanded(h.Padding)
	}
	h.Nodes[id].Local = box
	h.Nodes[id].boundsDirty = true
	h.pending[id] = struct{}{}
	return nil
}

func (h *Hierarchy) WorldBox(id BoneID) Box3 { return h.Nodes[id].World }

func (h *Hierarchy) RootBox() Box3 {
	if len(h.Skeleton.Roots) == 1 {
		return h.Nodes[h.Skeleton.Roots[0]].World
	}
	box := EmptyBox()
	for _, root := range h.Skeleton.Roots {
		box = box.AddBox(h.Nodes[root].World)
	}
	return box
}

func (h *Hierarchy) LastStats() UpdateStats { return h.lastStats }

// Update recomputes dirty world matrices top-down, then dirty bounds
// bottom-up. A changed node updates its descendant transforms and itself plus
// its ancestors' bounds; unaffected branches are not boxed or counted.
func (h *Hierarchy) Update() (UpdateStats, error) {
	h.frame++
	stats := UpdateStats{Nodes: len(h.Nodes), StructuralRebuilds: h.structuralRebuilds}
	ids := make([]BoneID, 0, len(h.pending))
	for id := range h.pending {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return h.orderIndex[ids[i]] < h.orderIndex[ids[j]] })

	var updateMatrixSubtree func(BoneID)
	updateMatrixSubtree = func(id BoneID) {
		n := &h.Nodes[id]
		if n.Parent == NoBone {
			n.WorldMatrix = n.LocalMatrix
		} else {
			n.WorldMatrix = h.Nodes[n.Parent].WorldMatrix.Mul(n.LocalMatrix)
		}
		n.matrixDirty = false
		stats.MatrixUpdates++
		stats.ChangedBones = append(stats.ChangedBones, id)
		for _, child := range n.Children {
			updateMatrixSubtree(child)
		}
	}

	for _, id := range ids {
		if h.Nodes[id].matrixDirty {
			updateMatrixSubtree(id)
		}
	}

	boundsIDs := make([]BoneID, 0, len(ids)*2)
	collectAncestors := func(start BoneID) {
		for id := start; id != NoBone; id = h.Nodes[id].Parent {
			if h.listedFrame[id] == h.frame {
				return
			}
			h.listedFrame[id] = h.frame
			boundsIDs = append(boundsIDs, id)
		}
	}
	for _, id := range stats.ChangedBones {
		collectAncestors(id)
	}
	for _, id := range ids {
		collectAncestors(id)
	}
	sort.Slice(boundsIDs, func(i, j int) bool {
		return h.orderIndex[boundsIDs[i]] > h.orderIndex[boundsIDs[j]]
	})

	for _, id := range boundsIDs {
		n := &h.Nodes[id]
		world := n.Local.Transform(n.WorldMatrix)
		for _, child := range n.Children {
			world = world.AddBox(h.Nodes[child].World)
		}
		n.World = world
		n.boundsDirty = false
		stats.BoundsUpdates++
		stats.BoundsOrder = append(stats.BoundsOrder, id)
	}

	clear(h.pending)
	h.lastStats = stats
	return stats, nil
}

// LayerBoxes returns one union box per topology depth. These are unions, not
// sums, and are useful for visual layer extents.
func (h *Hierarchy) LayerBoxes() []Box3 {
	maxDepth := h.Skeleton.MaxDepth()
	layers := make([]Box3, maxDepth+1)
	for i := range layers {
		layers[i] = EmptyBox()
	}
	for _, n := range h.Nodes {
		layers[n.Depth] = layers[n.Depth].AddBox(n.World)
	}
	return layers
}
