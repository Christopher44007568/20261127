package hierbox

import (
	"fmt"
	"math"
)

type Violation struct {
	Vertex       int     `json:"vertex"`
	Bone         BoneID  `json:"bone"`
	Point        Vec3    `json:"point"`
	OwnerOutside float64 `json:"ownerOutside"`
	RootOutside  float64 `json:"rootOutside"`
}

type CoverageReport struct {
	VertexCount         int         `json:"vertexCount"`
	Epsilon             float64     `json:"epsilon"`
	Covered             bool        `json:"covered"`
	OwnerViolations     int         `json:"ownerViolations"`
	RootViolations      int         `json:"rootViolations"`
	HierarchyViolations int         `json:"hierarchyViolations"`
	MaxOwnerOutside     float64     `json:"maxOwnerOutside"`
	MaxRootOutside      float64     `json:"maxRootOutside"`
	MaxHierarchyOutside float64     `json:"maxHierarchyOutside"`
	Violations          []Violation `json:"violations,omitempty"`
}

type NaiveReport struct {
	Box            Box3    `json:"box"`
	Covered        bool    `json:"covered"`
	Violations     int     `json:"violations"`
	MaxOutside     float64 `json:"maxOutside"`
	ComparisonNote string  `json:"comparisonNote"`
}

type PoseResult struct {
	Name         string         `json:"name"`
	Stats        UpdateStats    `json:"updateStats"`
	Coverage     CoverageReport `json:"coverage"`
	RootOnly     NaiveReport    `json:"rootOnly"`
	RootBox      Box3           `json:"rootBox"`
	LayerVolumes []LayerVolume  `json:"layerVolumes"`

	Pose     []Transform `json:"-"`
	Worlds   []Mat4      `json:"-"`
	Vertices []Vec3      `json:"-"`
}

type LayerVolume struct {
	Depth       int     `json:"depth"`
	Nodes       int     `json:"nodes"`
	VolumeSum   float64 `json:"volumeSum"`
	UnionVolume float64 `json:"unionVolume"`
}

func errUnknownPose(name string) error {
	return fmt.Errorf("hierbox: unknown runner pose %q", name)
}

func (q Quat) Conjugate() Quat { return Quat{X: -q.X, Y: -q.Y, Z: -q.Z, W: q.W} }

// RotateVector rotates a direction by q. The quaternion is expected to be a
// unit quaternion, as all rig rotations are.
func (q Quat) RotateVector(v Vec3) Vec3 {
	qp := Quat{X: v.X, Y: v.Y, Z: v.Z, W: 0}
	r := q.Mul(qp).Mul(q.Conjugate())
	return V(r.X, r.Y, r.Z)
}

func BoxDistance(b Box3, p Vec3) float64 {
	if b.Empty {
		return math.Inf(1)
	}
	var d2 float64
	if p.X < b.Min.X {
		d := b.Min.X - p.X
		d2 += d * d
	} else if p.X > b.Max.X {
		d := p.X - b.Max.X
		d2 += d * d
	}
	if p.Y < b.Min.Y {
		d := b.Min.Y - p.Y
		d2 += d * d
	} else if p.Y > b.Max.Y {
		d := p.Y - b.Max.Y
		d2 += d * d
	}
	if p.Z < b.Min.Z {
		d := b.Min.Z - p.Z
		d2 += d * d
	} else if p.Z > b.Max.Z {
		d := p.Z - b.Max.Z
		d2 += d * d
	}
	return math.Sqrt(d2)
}

func (r *Runner) RootOnlyBox(pose []Transform) Box3 {
	bindWorlds, err := r.Skeleton.WorldMatrices(r.RestPose)
	if err != nil {
		panic(err)
	}
	restRoot := r.RestPose[BoneRoot]
	invRot := restRoot.Rotation.Conjugate()
	local := EmptyBox()
	for _, v := range r.Mesh.Vertices {
		world := bindWorlds[v.Bone].TransformPoint(v.Local)
		rel := world.Sub(restRoot.Position)
		rootSpace := invRot.RotateVector(rel)
		local = local.AddPoint(rootSpace)
	}
	rootMatrix := PoseMatrix(pose[BoneRoot])
	return local.Transform(rootMatrix)
}

func (r *Runner) LayerVolumes() []LayerVolume {
	maxDepth := r.Skeleton.MaxDepth()
	out := make([]LayerVolume, maxDepth+1)
	for d := range out {
		out[d].Depth = d
		union := EmptyBox()
		for _, id := range r.Skeleton.Order {
			n := r.Hierarchy.Nodes[id]
			if n.Depth != d {
				continue
			}
			out[d].Nodes++
			out[d].VolumeSum += n.World.Volume()
			union = union.AddBox(n.World)
		}
		out[d].UnionVolume = union.Volume()
	}
	return out
}

// EvaluatePose applies a pose, checks every deformed vertex against its owner
// bone box and the root hierarchy box, and builds chart/culling input data.
func (r *Runner) EvaluatePose(name string) (*PoseResult, error) {
	pose, ok := r.Poses[name]
	if !ok {
		return nil, errUnknownPose(name)
	}
	worlds, err := r.Skeleton.WorldMatrices(pose)
	if err != nil {
		return nil, err
	}
	stats, err := r.ApplyPose(pose)
	if err != nil {
		return nil, err
	}
	vertices := r.Mesh.PosedVertices(worlds)
	eps := 1e-7
	cov := CoverageReport{VertexCount: len(vertices), Epsilon: eps, Covered: true}
	rootBox := r.Hierarchy.RootBox()

	for i, vertex := range vertices {
		bone := r.Mesh.Vertices[i].Bone
		owner := r.Hierarchy.WorldBox(bone)
		ownerOutside := BoxDistance(owner, vertex)
		rootOutside := BoxDistance(rootBox, vertex)
		if ownerOutside > cov.MaxOwnerOutside {
			cov.MaxOwnerOutside = ownerOutside
		}
		if rootOutside > cov.MaxRootOutside {
			cov.MaxRootOutside = rootOutside
		}
		hierarchyBad := false
		for ancestor := bone; ancestor != NoBone; ancestor = r.Hierarchy.Nodes[ancestor].Parent {
			ancestorOutside := BoxDistance(r.Hierarchy.WorldBox(ancestor), vertex)
			if ancestorOutside > cov.MaxHierarchyOutside {
				cov.MaxHierarchyOutside = ancestorOutside
			}
			if !r.Hierarchy.WorldBox(ancestor).ContainsPoint(vertex, eps) {
				cov.HierarchyViolations++
				hierarchyBad = true
			}
		}
		ownerBad := !owner.ContainsPoint(vertex, eps)
		rootBad := !rootBox.ContainsPoint(vertex, eps)
		if ownerBad {
			cov.OwnerViolations++
		}
		if rootBad {
			cov.RootViolations++
		}
		if ownerBad || rootBad || hierarchyBad {
			cov.Covered = false
		}
		if (ownerBad || rootBad || hierarchyBad) && len(cov.Violations) < 20 {
			cov.Violations = append(cov.Violations, Violation{
				Vertex: i, Bone: bone, Point: vertex,
				OwnerOutside: ownerOutside, RootOutside: rootOutside,
			})
		}
	}

	naiveBox := r.RootOnlyBox(pose)
	naive := NaiveReport{Box: naiveBox, Covered: true}
	for _, vertex := range vertices {
		d := BoxDistance(naiveBox, vertex)
		if d > naive.MaxOutside {
			naive.MaxOutside = d
		}
		if !naiveBox.ContainsPoint(vertex, eps) {
			naive.Violations++
			naive.Covered = false
		}
	}
	naive.ComparisonNote = "bind-pose character AABB transformed only by root TRS"

	return &PoseResult{
		Name: name, Stats: stats, Coverage: cov, RootOnly: naive,
		RootBox: rootBox, LayerVolumes: r.LayerVolumes(),
		Pose: pose, Worlds: worlds, Vertices: vertices,
	}, nil
}
