package stroke

import "math"

// DetectOverlaps reports triangle pairs whose interiors intersect with
// positive area. Shared edges and shared vertices are not overlaps.
func DetectOverlaps(mesh Mesh, tolerance float64) []TrianglePair {
	if tolerance < 0 {
		tolerance = 0
	}
	bounds := make([]Bounds, len(mesh.Triangles))
	for i, t := range mesh.Triangles {
		bounds[i] = triangleBoundsBox(t)
	}
	var pairs []TrianglePair
	for i := 0; i < len(mesh.Triangles); i++ {
		ti := mesh.Triangles[i]
		if math.Abs(signedArea(ti.A, ti.B, ti.C)) <= tolerance {
			continue
		}
		for j := i + 1; j < len(mesh.Triangles); j++ {
			if !boundsOverlap(bounds[i], bounds[j], tolerance) {
				continue
			}
			tj := mesh.Triangles[j]
			if math.Abs(signedArea(tj.A, tj.B, tj.C)) <= tolerance {
				continue
			}
			if trianglesOverlap(ti, tj, tolerance) {
				pairs = append(pairs, TrianglePair{A: i, B: j})
			}
		}
	}
	return pairs
}

func triangleBoundsBox(t Triangle) Bounds {
	return Bounds{
		Min: Point{X: math.Min(t.A.X, math.Min(t.B.X, t.C.X)),
			Y: math.Min(t.A.Y, math.Min(t.B.Y, t.C.Y))},
		Max: Point{X: math.Max(t.A.X, math.Max(t.B.X, t.C.X)),
			Y: math.Max(t.A.Y, math.Max(t.B.Y, t.C.Y))},
	}
}

func boundsOverlap(a, b Bounds, tolerance float64) bool {
	return math.Min(a.Max.X, b.Max.X)-math.Max(a.Min.X, b.Min.X) > tolerance &&
		math.Min(a.Max.Y, b.Max.Y)-math.Max(a.Min.Y, b.Min.Y) > tolerance
}

func trianglesOverlap(a, b Triangle, tolerance float64) bool {
	axes := [6]Point{
		edgeNormal(a.A, a.B),
		edgeNormal(a.B, a.C),
		edgeNormal(a.C, a.A),
		edgeNormal(b.A, b.B),
		edgeNormal(b.B, b.C),
		edgeNormal(b.C, b.A),
	}
	for _, axis := range axes {
		if length2(axis) <= 1e-24 {
			continue
		}
		minA, maxA := projectTriangle(a, axis)
		minB, maxB := projectTriangle(b, axis)
		overlap := math.Min(maxA, maxB) - math.Max(minA, minB)
		if overlap <= tolerance {
			return false
		}
	}
	return true
}

func edgeNormal(a, b Point) Point {
	d := sub(b, a)
	return Point{X: -d.Y, Y: d.X}
}

func projectTriangle(t Triangle, axis Point) (float64, float64) {
	a := dot(t.A, axis)
	b := dot(t.B, axis)
	c := dot(t.C, axis)
	return math.Min(a, math.Min(b, c)), math.Max(a, math.Max(b, c))
}

func length2(p Point) float64 {
	return p.X*p.X + p.Y*p.Y
}

func overlapIntersectionTriangles(mesh Mesh, pairs []TrianglePair) []Triangle {
	out := make([]Triangle, 0, len(pairs)*4)
	for _, pair := range pairs {
		poly := clipTriangle(mesh.Triangles[pair.A], mesh.Triangles[pair.B])
		for i := 1; i+1 < len(poly); i++ {
			t := Triangle{A: poly[0], B: poly[i], C: poly[i+1]}
			if math.Abs(signedArea(t.A, t.B, t.C)) > 1e-12 {
				out = append(out, t)
			}
		}
	}
	return out
}

func clipTriangle(subject, clip Triangle) []Point {
	poly := []Point{subject.A, subject.B, subject.C}
	orientation := 1.0
	if signedArea(clip.A, clip.B, clip.C) < 0 {
		orientation = -1
	}
	edges := [3][2]Point{
		{clip.A, clip.B},
		{clip.B, clip.C},
		{clip.C, clip.A},
	}
	for _, edge := range edges {
		if len(poly) == 0 {
			break
		}
		poly = clipPolygonHalfPlane(poly, edge[0], edge[1], orientation)
	}
	return poly
}

func clipPolygonHalfPlane(poly []Point, a, b Point, orientation float64) []Point {
	if len(poly) == 0 {
		return nil
	}
	out := make([]Point, 0, len(poly)+2)
	prev := poly[len(poly)-1]
	prevD := orientation * cross(sub(b, a), sub(prev, a))
	prevIn := prevD >= -1e-12
	for _, cur := range poly {
		curD := orientation * cross(sub(b, a), sub(cur, a))
		curIn := curD >= -1e-12
		switch {
		case prevIn && curIn:
			out = append(out, cur)
		case prevIn && !curIn:
			out = append(out, lineIntersection(prev, cur, prevD, curD))
		case !prevIn && curIn:
			out = append(out, lineIntersection(prev, cur, prevD, curD))
			out = append(out, cur)
		}
		prev, prevD, prevIn = cur, curD, curIn
	}
	return out
}

func lineIntersection(a, b Point, da, db float64) Point {
	den := da - db
	if math.Abs(den) <= 1e-15 {
		return a
	}
	t := da / den
	return add(a, scale(sub(b, a), t))
}
