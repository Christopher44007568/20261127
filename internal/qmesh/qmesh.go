package qmesh

import (
	"fmt"
	"math"

	"meshdiagnostics/internal/mesh"
)

// Face is a quad with a consistently wound boundary.
type Face [4]int

// Mesh is an indexed quadrilateral mesh.
type Mesh struct {
	Vertices []mesh.Vec3
	Faces    []Face
	Features map[mesh.EdgeKey]bool
}

// Metrics summarizes the local shape of one quad.
type Metrics struct {
	Normal       mesh.Vec3
	AnglesDeg    [4]float64
	MinAngleDeg  float64
	MaxAngleDeg  float64
	Area         float64
	Planarity    float64
	Aspect       float64
	MinEdge      float64
	MaxEdge      float64
	Convex       bool
	SignedArea2D float64
}

func (m *Mesh) FaceVertices(f Face) [4]mesh.Vec3 {
	return [4]mesh.Vec3{
		m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]], m.Vertices[f[3]],
	}
}

// FaceNormal returns the Newell normal of a potentially non-planar quad.
func (m *Mesh) FaceNormal(f Face) mesh.Vec3 {
	var n mesh.Vec3
	p := m.FaceVertices(f)
	for i := 0; i < 4; i++ {
		q := p[(i+1)%4]
		n.X += (p[i].Y - q.Y) * (p[i].Z + q.Z)
		n.Y += (p[i].Z - q.Z) * (p[i].X + q.X)
		n.Z += (p[i].X - q.X) * (p[i].Y + q.Y)
	}
	// Fallback for quads whose four vertices are nearly collinear.
	if n.Len2() < 1e-24 {
		for i := 1; i < 3; i++ {
			n = n.Add(p[0].Sub(p[i]).Cross(p[i+1].Sub(p[i])))
		}
	}
	return n
}

func (m *Mesh) FaceCentroid(f Face) mesh.Vec3 {
	p := m.FaceVertices(f)
	return p[0].Add(p[1]).Add(p[2]).Add(p[3]).Scale(0.25)
}

func (m *Mesh) FaceArea(f Face) float64 {
	p := m.FaceVertices(f)
	return 0.5 * (p[1].Sub(p[0]).Cross(p[2].Sub(p[0])).Len() +
		p[2].Sub(p[0]).Cross(p[3].Sub(p[0])).Len())
}

// Metrics computes corner angles in the quad's least-squares support plane.
// This is important for non-planar organic quads: an angle measured directly
// in 3D is not the interior angle of the face.
func (m *Mesh) Metrics(f Face) Metrics {
	p := m.FaceVertices(f)
	n := m.FaceNormal(f).Normalize()
	if n.Len2() < 1e-20 {
		return Metrics{MinAngleDeg: 0, MaxAngleDeg: 0, Convex: false}
	}

	u := p[1].Sub(p[0])
	u = u.Sub(n.Scale(u.Dot(n)))
	if u.Len2() < 1e-20 {
		u = p[2].Sub(p[0])
		u = u.Sub(n.Scale(u.Dot(n)))
	}
	u = u.Normalize()
	v := n.Cross(u)
	if v.Len2() < 1e-20 {
		v = n.Cross(mesh.V(0, 1, 0))
		if v.Len2() < 1e-20 {
			v = n.Cross(mesh.V(1, 0, 0))
		}
		v = v.Normalize()
	}

	var xy [4][2]float64
	var avgEdge float64
	for i := 0; i < 4; i++ {
		d := p[i].Sub(p[0])
		xy[i] = [2]float64{d.Dot(u), d.Dot(v)}
		if i > 0 {
			avgEdge += p[i].Sub(p[i-1]).Len()
		}
	}
	avgEdge += p[0].Sub(p[3]).Len()
	avgEdge *= 0.25

	var signedArea float64
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		signedArea += xy[i][0]*xy[j][1] - xy[j][0]*xy[i][1]
	}
	signedArea *= 0.5

	out := Metrics{Normal: n, SignedArea2D: signedArea, Convex: true}
	turnSign := 0.0
	out.MinAngleDeg = 180
	out.MaxAngleDeg = 0
	for i := 0; i < 4; i++ {
		prev := xy[(i+3)%4]
		cur := xy[i]
		next := xy[(i+1)%4]
		e1 := [2]float64{prev[0] - cur[0], prev[1] - cur[1]}
		e2 := [2]float64{next[0] - cur[0], next[1] - cur[1]}
		l1 := math.Hypot(e1[0], e1[1])
		l2 := math.Hypot(e2[0], e2[1])
		if l1 < 1e-14 || l2 < 1e-14 {
			out.Convex = false
			out.AnglesDeg[i] = 0
			out.MinAngleDeg = 0
			continue
		}
		dot := (e1[0]*e2[0] + e1[1]*e2[1]) / (l1 * l2)
		angle := math.Acos(math.Max(-1, math.Min(1, dot)))
		turn := e1[0]*e2[1] - e1[1]*e2[0]
		if sign(turn) != 0 {
			if turnSign == 0 {
				turnSign = sign(turn)
			} else if sign(turn) != turnSign {
				out.Convex = false
			}
		}
		// At a CCW vertex the incoming edge is traversed forward but our
		// e1 points backward, so a convex corner has turn opposite in sign
		// to the polygon area. A matching sign is a reflex corner.
		if turn != 0 && sign(turn) == sign(signedArea) {
			angle = math.Pi*2 - angle
		}
		deg := angle * 180 / math.Pi
		out.AnglesDeg[i] = deg
		out.MinAngleDeg = math.Min(out.MinAngleDeg, deg)
		out.MaxAngleDeg = math.Max(out.MaxAngleDeg, deg)
	}
	out.Area = m.FaceArea(f)
	out.MinEdge, out.MaxEdge = math.Inf(1), 0
	for i := 0; i < 4; i++ {
		l := p[(i+1)%4].Sub(p[i]).Len()
		out.MinEdge = math.Min(out.MinEdge, l)
		out.MaxEdge = math.Max(out.MaxEdge, l)
	}
	if out.MinEdge > 1e-20 {
		out.Aspect = out.MaxEdge / out.MinEdge
	}
	if avgEdge > 1e-20 {
		var maxDist float64
		for i := 0; i < 4; i++ {
			maxDist = math.Max(maxDist, math.Abs(p[i].Sub(p[0]).Dot(n)))
		}
		out.Planarity = maxDist / avgEdge
	}
	if signedArea < 0 {
		// Newell normal and basis can point inward; metrics are independent of
		// that sign, but retain it for callers that need orientation.
		out.SignedArea2D = signedArea
	}
	return out
}

func sign(v float64) float64 {
	if v > 1e-12 {
		return 1
	}
	if v < -1e-12 {
		return -1
	}
	return 0
}

// Edges returns unique undirected edges with their incident face count.
func (m *Mesh) Edges() map[mesh.EdgeKey]int {
	out := make(map[mesh.EdgeKey]int, len(m.Faces)*2)
	for _, f := range m.Faces {
		for i := 0; i < 4; i++ {
			out[mesh.Edge(f[i], f[(i+1)%4])]++
		}
	}
	return out
}

// Triangles fan-triangulates each quad for rendering and numerical analysis.
func (m *Mesh) Triangles() [][3]int {
	out := make([][3]int, 0, len(m.Faces)*2)
	for _, f := range m.Faces {
		out = append(out, [3]int{f[0], f[1], f[2]}, [3]int{f[0], f[2], f[3]})
	}
	return out
}

// WriteOBJ saves the quad mesh. Feature edges are also emitted as a comment
// summary so downstream tools can recover the preserved crease set.
func (m *Mesh) WriteOBJ(path string) error {
	faces := make([][]int, len(m.Faces))
	for i, f := range m.Faces {
		faces[i] = []int{f[0], f[1], f[2], f[3]}
	}
	return mesh.WriteOBJ(path, m.Vertices, faces)
}

// Validate reports topological problems useful for tests and CLI diagnostics.
func (m *Mesh) Validate() error {
	if len(m.Vertices) == 0 || len(m.Faces) == 0 {
		return fmt.Errorf("empty quad mesh")
	}
	for i, f := range m.Faces {
		if f[0] < 0 || f[1] < 0 || f[2] < 0 || f[3] < 0 ||
			f[0] >= len(m.Vertices) || f[1] >= len(m.Vertices) ||
			f[2] >= len(m.Vertices) || f[3] >= len(m.Vertices) {
			return fmt.Errorf("face %d has an out-of-range vertex", i)
		}
		if f[0] == f[1] || f[1] == f[2] || f[2] == f[3] || f[3] == f[0] {
			return fmt.Errorf("face %d has a repeated vertex", i)
		}
	}
	return nil
}
