package main

import "math"

type Tri struct {
	A, B, C int
}

type MeshHints struct {
	Crown    int
	Chin     int
	Nape     int
	Bottom   int
	NeckRing []int
	LeftEar  int
	RightEar int
}

type Mesh struct {
	Name  string
	Pos   []Vec3
	Tri   []Tri
	Norm  []Vec3
	Hints MeshHints
}

func (m *Mesh) FaceVert(f int, k int) int {
	t := m.Tri[f]
	switch k {
	case 0:
		return t.A
	case 1:
		return t.B
	default:
		return t.C
	}
}

func (m *Mesh) FaceNormal(f int) Vec3 {
	a := m.Pos[m.Tri[f].A]
	b := m.Pos[m.Tri[f].B]
	c := m.Pos[m.Tri[f].C]
	return b.Sub(a).Cross(c.Sub(a)).Normalized()
}

func (m *Mesh) FaceArea(f int) float64 {
	a := m.Pos[m.Tri[f].A]
	b := m.Pos[m.Tri[f].B]
	c := m.Pos[m.Tri[f].C]
	return 0.5 * b.Sub(a).Cross(c.Sub(a)).Len()
}

func (m *Mesh) TotalArea() float64 {
	total := 0.0
	for f := range m.Tri {
		total += m.FaceArea(f)
	}
	return total
}

// EnsureOutward makes a closed procedural mesh consistently outward facing.
// The signed volume test is robust for the star-shaped head and avoids
// depending on the orientation of every individual ring construction.
func (m *Mesh) EnsureOutward() {
	vol := 0.0
	for _, t := range m.Tri {
		a, b, c := m.Pos[t.A], m.Pos[t.B], m.Pos[t.C]
		vol += a.Dot(b.Cross(c)) / 6.0
	}
	if vol < 0 {
		for i := range m.Tri {
			m.Tri[i].B, m.Tri[i].C = m.Tri[i].C, m.Tri[i].B
		}
	}
	m.ComputeNormals()
}

func (m *Mesh) ComputeNormals() {
	m.Norm = make([]Vec3, len(m.Pos))
	for f, t := range m.Tri {
		n := m.FaceNormal(f)
		m.Norm[t.A] = m.Norm[t.A].Add(n)
		m.Norm[t.B] = m.Norm[t.B].Add(n)
		m.Norm[t.C] = m.Norm[t.C].Add(n)
	}
	for i := range m.Norm {
		m.Norm[i] = m.Norm[i].Normalized()
	}
}

type profilePoint struct {
	y, rx, rz float64
}

var headProfile = []profilePoint{
	{1.34, 0.00, 0.00},
	{1.24, 0.18, 0.20},
	{1.08, 0.39, 0.41},
	{0.86, 0.56, 0.58},
	{0.60, 0.67, 0.69},
	{0.30, 0.73, 0.75},
	{0.02, 0.74, 0.78},
	{-0.25, 0.69, 0.73},
	{-0.48, 0.62, 0.66},
	{-0.68, 0.52, 0.55},
	{-0.84, 0.42, 0.44},
	{-0.98, 0.35, 0.37},
	{-1.20, 0.34, 0.36},
	{-1.43, 0.36, 0.38},
	{-1.54, 0.34, 0.36},
}

func profileAt(y float64) (float64, float64) {
	if y >= headProfile[0].y {
		return headProfile[0].rx, headProfile[0].rz
	}
	last := headProfile[len(headProfile)-1]
	if y <= last.y {
		return last.rx, last.rz
	}
	for i := 0; i < len(headProfile)-1; i++ {
		a := headProfile[i]
		b := headProfile[i+1]
		if y <= a.y && y >= b.y {
			u := (a.y - y) / (a.y - b.y)
			p0 := headProfile[maxInt(i-1, 0)]
			p1 := a
			p2 := b
			p3 := headProfile[minInt(i+2, len(headProfile)-1)]
			return catmull(p0.rx, p1.rx, p2.rx, p3.rx, u),
				catmull(p0.rz, p1.rz, p2.rz, p3.rz, u)
		}
	}
	return last.rx, last.rz
}

func catmull(p0, p1, p2, p3, t float64) float64 {
	tt := t * t
	ttt := tt * t
	return 0.5 * ((2 * p1) + (-p0+p2)*t +
		(2*p0-5*p1+4*p2-p3)*tt +
		(-p0+3*p1-3*p2+p3)*ttt)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func facialDisplacement(x, y, cz float64) float64 {
	front := clampFloat((cz-0.05)/0.45, 0, 1)
	if front <= 0 {
		return 0
	}
	d := 0.0
	d += 0.105 * gauss2(x, y, 0.00, 0.285, 0.085, 0.220)
	d += 0.185 * gauss2(x, y, 0.00, 0.035, 0.145, 0.125)
	d += 0.055 * gauss2(x, y, -0.110, -0.060, 0.065, 0.055)
	d += 0.055 * gauss2(x, y, 0.110, -0.060, 0.065, 0.055)
	d += 0.060 * gauss2(x, y, -0.275, 0.405, 0.180, 0.075)
	d += 0.060 * gauss2(x, y, 0.275, 0.405, 0.180, 0.075)
	d += 0.060 * gauss2(x, y, -0.360, -0.080, 0.190, 0.150)
	d += 0.060 * gauss2(x, y, 0.360, -0.080, 0.190, 0.150)
	d += 0.060 * gauss2(x, y, 0.000, -0.405, 0.225, 0.085)
	d += 0.080 * gauss2(x, y, 0.000, -0.625, 0.210, 0.110)
	d -= 0.095 * gauss2(x, y, -0.270, 0.225, 0.170, 0.100)
	d -= 0.095 * gauss2(x, y, 0.270, 0.225, 0.170, 0.100)
	d -= 0.035 * gauss2(x, y, 0.000, -0.335, 0.205, 0.026)
	d -= 0.040 * gauss2(x, y, -0.075, -0.080, 0.038, 0.028)
	d -= 0.040 * gauss2(x, y, 0.075, -0.080, 0.038, 0.028)
	return d * front
}

const (
	headSegments = 72
	headRings    = 52
)

func NewOrganicHead() *Mesh {
	m := &Mesh{Name: "curvature-occlusion-head"}
	s := headSegments
	r := headRings

	crown := len(m.Pos)
	m.Pos = append(m.Pos, Vec3{0, 1.36, -0.03})
	m.Hints.Crown = crown

	ring := make([][]int, r)
	bestNeckDiff := 1e9
	for i := 0; i < r; i++ {
		y := lerpFloat(1.25, -1.53, float64(i)/float64(r-1))
		rx, rz := profileAt(y)
		ring[i] = make([]int, s)
		if math.Abs(y+0.87) < bestNeckDiff {
			bestNeckDiff = math.Abs(y + 0.87)
			m.Hints.NeckRing = m.Hints.NeckRing[:0]
		}
		for j := 0; j < s; j++ {
			theta := 2 * math.Pi * float64(j) / float64(s)
			sn, cs := math.Sin(theta), math.Cos(theta)
			x := rx * sn
			z := rz * cs
			if cs < 0 {
				z *= 1.08
			}
			earAngle := math.Pow(math.Abs(sn), 10)
			earY := math.Exp(-math.Pow((y-0.02)/0.23, 2))
			ear := earAngle * earY
			if sn != 0 {
				x += math.Copysign(0.18*ear, sn)
			}
			z -= 0.075 * ear
			z += facialDisplacement(x, y, z)
			id := len(m.Pos)
			m.Pos = append(m.Pos, Vec3{x, y, z})
			ring[i][j] = id

			if i >= 0 && math.Abs(y+0.87) <= bestNeckDiff+1e-9 {
				m.Hints.NeckRing = append(m.Hints.NeckRing, id)
			}
			if j == 0 && math.Abs(y+0.66) < 0.04 {
				m.Hints.Chin = id
			}
			if j == s/2 && math.Abs(y+0.87) < 0.08 {
				m.Hints.Nape = id
			}
			if j == s/4 && math.Abs(y-0.02) < 0.04 {
				m.Hints.RightEar = id
			}
			if j == 3*s/4 && math.Abs(y-0.02) < 0.04 {
				m.Hints.LeftEar = id
			}
		}
	}

	bottom := len(m.Pos)
	m.Pos = append(m.Pos, Vec3{0, -1.585, -0.01})
	m.Hints.Bottom = bottom

	for j := 0; j < s; j++ {
		j1 := (j + 1) % s
		m.Tri = append(m.Tri, Tri{crown, ring[0][j], ring[0][j1]})
	}
	for i := 0; i+1 < r; i++ {
		for j := 0; j < s; j++ {
			j1 := (j + 1) % s
			upper0 := ring[i][j]
			upper1 := ring[i][j1]
			lower0 := ring[i+1][j]
			lower1 := ring[i+1][j1]
			m.Tri = append(m.Tri,
				Tri{upper0, lower0, upper1},
				Tri{upper1, lower0, lower1},
			)
		}
	}
	last := ring[r-1]
	for j := 0; j < s; j++ {
		j1 := (j + 1) % s
		m.Tri = append(m.Tri, Tri{bottom, last[j1], last[j]})
	}

	if m.Hints.Chin == 0 {
		m.Hints.Chin = ring[r/2][0]
	}
	if m.Hints.Nape == 0 {
		m.Hints.Nape = ring[r*2/3][s/2]
	}
	if len(m.Hints.NeckRing) == 0 {
		m.Hints.NeckRing = ring[r*2/3]
	}
	m.EnsureOutward()
	return m
}

func (m *Mesh) Bounds() (Vec3, Vec3) {
	if len(m.Pos) == 0 {
		return Vec3{}, Vec3{}
	}
	mn, mx := m.Pos[0], m.Pos[0]
	for _, p := range m.Pos[1:] {
		mn = Vec3{math.Min(mn.X, p.X), math.Min(mn.Y, p.Y), math.Min(mn.Z, p.Z)}
		mx = Vec3{math.Max(mx.X, p.X), math.Max(mx.Y, p.Y), math.Max(mx.Z, p.Z)}
	}
	return mn, mx
}

func (m *Mesh) Center() Vec3 {
	mn, mx := m.Bounds()
	return mn.Add(mx).Mul(0.5)
}

func WriteOBJ(path string, m *Mesh, wedgeUV []Vec2, faceWedge [][3]int) error {
	return writeOBJFile(path, m, wedgeUV, faceWedge)
}
