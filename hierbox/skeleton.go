package hierbox

import (
	"fmt"
	"sort"
)

type BoneID int

const NoBone BoneID = -1

// Transform is a bone-local TRS pose.
type Transform struct {
	Position Vec3 `json:"position"`
	Rotation Quat `json:"rotation"`
	Scale    Vec3 `json:"scale"`
}

func DefaultTransform() Transform {
	return Transform{Rotation: IdentityQuat(), Scale: V(1, 1, 1)}
}

// Bone describes one node in the skeleton topology.
type Bone struct {
	ID     BoneID    `json:"id"`
	Name   string    `json:"name"`
	Parent BoneID    `json:"parent"`
	Rest   Transform `json:"rest"`
}

// Skeleton is an immutable-after-construction bone topology.
type Skeleton struct {
	Bones    []Bone     `json:"bones"`
	Children [][]BoneID `json:"children"`
	Roots    []BoneID   `json:"roots"`
	Order    []BoneID   `json:"order"`
	Depth    []int      `json:"depth"`
}

func NewSkeleton(bones []Bone) (*Skeleton, error) {
	if len(bones) == 0 {
		return nil, fmt.Errorf("hierbox: skeleton has no bones")
	}
	n := len(bones)
	s := &Skeleton{
		Bones:    make([]Bone, n),
		Children: make([][]BoneID, n),
		Depth:    make([]int, n),
	}
	copy(s.Bones, bones)
	seen := make(map[BoneID]bool, n)
	for i := range s.Bones {
		id := s.Bones[i].ID
		if int(id) < 0 || int(id) >= n {
			return nil, fmt.Errorf("hierbox: bone %q has out-of-range id %d", s.Bones[i].Name, id)
		}
		if seen[id] {
			return nil, fmt.Errorf("hierbox: duplicate bone id %d", id)
		}
		seen[id] = true
		if s.Bones[i].Rest.Rotation.W == 0 && s.Bones[i].Rest.Rotation.X == 0 &&
			s.Bones[i].Rest.Rotation.Y == 0 && s.Bones[i].Rest.Rotation.Z == 0 {
			rest := s.Bones[i].Rest
			rest.Rotation = IdentityQuat()
			if rest.Scale.X == 0 && rest.Scale.Y == 0 && rest.Scale.Z == 0 {
				rest.Scale = V(1, 1, 1)
			}
			s.Bones[i].Rest = rest
		}
	}
	for i := range s.Bones {
		b := &s.Bones[i]
		if b.Parent == NoBone {
			s.Roots = append(s.Roots, b.ID)
			s.Depth[b.ID] = 0
			continue
		}
		if int(b.Parent) < 0 || int(b.Parent) >= n || !seen[b.Parent] {
			return nil, fmt.Errorf("hierbox: bone %q has invalid parent %d", b.Name, b.Parent)
		}
		if b.Parent == b.ID {
			return nil, fmt.Errorf("hierbox: bone %q is its own parent", b.Name)
		}
		s.Children[b.Parent] = append(s.Children[b.Parent], b.ID)
	}

	visited := make([]bool, n)
	var visit func(id BoneID, depth int) error
	visit = func(id BoneID, depth int) error {
		if visited[id] {
			return fmt.Errorf("hierbox: cycle or repeated node at bone %d", id)
		}
		visited[id] = true
		s.Depth[id] = depth
		s.Order = append(s.Order, id)
		for _, child := range s.Children[id] {
			if err := visit(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range s.Roots {
		if err := visit(root, 0); err != nil {
			return nil, err
		}
	}
	if len(s.Order) != n {
		return nil, fmt.Errorf("hierbox: skeleton topology contains a cycle")
	}
	sort.SliceStable(s.Roots, func(i, j int) bool { return s.Roots[i] < s.Roots[j] })
	return s, nil
}

func (s *Skeleton) Bone(name string) (BoneID, bool) {
	for _, b := range s.Bones {
		if b.Name == name {
			return b.ID, true
		}
	}
	return NoBone, false
}

func (s *Skeleton) MaxDepth() int {
	max := 0
	for _, d := range s.Depth {
		if d > max {
			max = d
		}
	}
	return max
}

// WorldMatrices evaluates a complete pose without using a Hierarchy. It is
// useful for tests and one-off vertex transformations.
func (s *Skeleton) WorldMatrices(locals []Transform) ([]Mat4, error) {
	if len(locals) != len(s.Bones) {
		return nil, fmt.Errorf("hierbox: pose has %d transforms, skeleton has %d bones", len(locals), len(s.Bones))
	}
	world := make([]Mat4, len(s.Bones))
	for _, id := range s.Order {
		local := PoseMatrix(locals[id])
		if s.Bones[id].Parent == NoBone {
			world[id] = local
		} else {
			world[id] = world[s.Bones[id].Parent].Mul(local)
		}
	}
	return world, nil
}
