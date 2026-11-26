package csg

import "meshbool/internal/mesh"

const epsilon = 1e-7

type Plane struct {
	Normal mesh.Vec3
	W      float64
}

func PlaneFromPoints(a, b, c mesh.Vec3) (Plane, bool) {
	normal := b.Sub(a).Cross(c.Sub(a))
	if normal.Length() <= epsilon {
		return Plane{}, false
	}
	normal = normal.Normalize()
	return Plane{Normal: normal, W: normal.Dot(a)}, true
}

func (p Plane) Flip() Plane {
	return Plane{Normal: p.Normal.Negate(), W: -p.W}
}

// Polygon is a convex triangle or clipped polygon. Source is retained for
// diagnostics; Plane is cached so splitting never re-estimates a normal.
type Polygon struct {
	Vertices []mesh.Vec3
	Plane    Plane
	Source   string
}

func (p Polygon) Flip() Polygon {
	out := Polygon{
		Vertices: make([]mesh.Vec3, len(p.Vertices)),
		Plane:    p.Plane.Flip(),
		Source:   p.Source,
	}
	for i := range p.Vertices {
		out.Vertices[len(p.Vertices)-1-i] = p.Vertices[i]
	}
	return out
}

func (p Polygon) Clone() Polygon {
	return Polygon{
		Vertices: append([]mesh.Vec3(nil), p.Vertices...),
		Plane:    p.Plane,
		Source:   p.Source,
	}
}

func (p Polygon) Triangles() []mesh.Triangle {
	if len(p.Vertices) < 3 {
		return nil
	}
	out := make([]mesh.Triangle, 0, len(p.Vertices)-2)
	for i := 1; i+1 < len(p.Vertices); i++ {
		t := mesh.Triangle{A: p.Vertices[0], B: p.Vertices[i], C: p.Vertices[i+1]}
		if !t.Degenerate() {
			out = append(out, t)
		}
	}
	return out
}

const (
	coplanar = 0
	front    = 1
	back     = 2
	spanning = 3
)

func (p Plane) classify(point mesh.Vec3) int {
	d := p.Normal.Dot(point) - p.W
	switch {
	case d < -epsilon:
		return back
	case d > epsilon:
		return front
	default:
		return coplanar
	}
}

// splitPolygon is the core coplanar-safe clipping routine. Coplanar polygons
// are routed by normal direction so union/intersection keep exactly one copy.
func splitPolygon(
	poly Polygon,
	splitter Plane,
	coplanarFront, coplanarBack, frontSide, backSide *[]Polygon,
) {
	polyType := coplanar
	types := make([]int, len(poly.Vertices))
	for i, vertex := range poly.Vertices {
		types[i] = splitter.classify(vertex)
		polyType |= types[i]
	}

	switch polyType {
	case coplanar:
		dot := poly.Plane.Normal.Dot(splitter.Normal)
		if dot > 0 {
			*coplanarFront = append(*coplanarFront, poly)
		} else {
			*coplanarBack = append(*coplanarBack, poly)
		}
	case front:
		*frontSide = append(*frontSide, poly)
	case back:
		*backSide = append(*backSide, poly)
	default:
		var frontVertices, backVertices []mesh.Vec3
		for i := range poly.Vertices {
			j := (i + 1) % len(poly.Vertices)
			ti, tj := types[i], types[j]
			vi, vj := poly.Vertices[i], poly.Vertices[j]
			if ti != back {
				frontVertices = append(frontVertices, vi)
			}
			if ti != front {
				backVertices = append(backVertices, vi)
			}
			if ti|tj == spanning {
				denominator := splitter.Normal.Dot(vj.Sub(vi))
				if abs(denominator) > epsilon {
					t := (splitter.W - splitter.Normal.Dot(vi)) / denominator
					cross := vi.Lerp(vj, clamp01(t))
					frontVertices = append(frontVertices, cross)
					backVertices = append(backVertices, cross)
				}
			}
		}
		if len(frontVertices) >= 3 {
			*frontSide = append(*frontSide, Polygon{
				Vertices: frontVertices,
				Plane:    poly.Plane,
				Source:   poly.Source,
			})
		}
		if len(backVertices) >= 3 {
			*backSide = append(*backSide, Polygon{
				Vertices: backVertices,
				Plane:    poly.Plane,
				Source:   poly.Source,
			})
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
