// Package geom contains small 2-D vector/polygon primitives and the exact
// polygon-vs-pixel area routine used by the analytic coverage backend.
package geom

import "math"

// Vec is a 2-D point in continuous pixel coordinates. Pixel (ix,iy) covers
// the square [ix,ix+1] x [iy,iy+1].
type Vec struct {
	X, Y float64
}

// BBox is an axis-aligned bounding box (inclusive min, exclusive-ish max).
type BBox struct {
	MinX, MinY float64
	MaxX, MaxY float64
}

// Valid reports whether the box has positive extent.
func (b BBox) Valid() bool { return b.MaxX > b.MinX && b.MaxY > b.MinY }

// PolyBBox returns the bounding box of a polygon.
func PolyBBox(poly []Vec) BBox {
	if len(poly) == 0 {
		return BBox{}
	}
	b := BBox{poly[0].X, poly[0].Y, poly[0].X, poly[0].Y}
	for _, p := range poly[1:] {
		b.MinX = math.Min(b.MinX, p.X)
		b.MinY = math.Min(b.MinY, p.Y)
		b.MaxX = math.Max(b.MaxX, p.X)
		b.MaxY = math.Max(b.MaxY, p.Y)
	}
	return b
}

// PolygonArea returns the unsigned shoelace area of a simple polygon.
func PolygonArea(poly []Vec) float64 {
	if len(poly) < 3 {
		return 0
	}
	s := 0.0
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		s += a.X*b.Y - b.X*a.Y
	}
	return math.Abs(s) * 0.5
}

// SegmentOutline builds the rectangular outline polygon of a stroke along
// segment a-b with the given width. capExt extends both ends along the
// tangent: 0 gives butt caps, width/2 gives square caps.
func SegmentOutline(a, b Vec, width, capExt float64) []Vec {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := math.Hypot(dx, dy)
	if l < 1e-12 {
		r := width * 0.5
		return []Vec{{a.X - r, a.Y - r}, {a.X + r, a.Y - r}, {a.X + r, a.Y + r}, {a.X - r, a.Y + r}}
	}
	ux, uy := dx/l, dy/l
	nx, ny := -uy*width*0.5, ux*width*0.5
	ex, ey := ux*capExt, uy*capExt
	a0 := Vec{a.X - ex + nx, a.Y - ey + ny}
	a1 := Vec{a.X - ex - nx, a.Y - ey - ny}
	b1 := Vec{b.X + ex - nx, b.Y + ey - ny}
	b0 := Vec{b.X + ex + nx, b.Y + ey + ny}
	return []Vec{a0, a1, b1, b0}
}

// plane identifies one clipping half-plane of a pixel square.
type plane int

const (
	planeLeft   plane = iota // x >= bound
	planeRight               // x <= bound
	planeTop                 // y >= bound
	planeBottom              // y <= bound
)

func insidePlane(p Vec, pl plane, bound float64) bool {
	switch pl {
	case planeLeft:
		return p.X >= bound
	case planeRight:
		return p.X <= bound
	case planeTop:
		return p.Y >= bound
	default:
		return p.Y <= bound
	}
}

// intersectPlane returns the intersection of segment s-e with the plane.
// It is only called when s and e lie on different sides, so the denominator
// is non-zero.
func intersectPlane(s, e Vec, pl plane, bound float64) Vec {
	switch pl {
	case planeLeft, planeRight:
		t := (bound - s.X) / (e.X - s.X)
		return Vec{X: bound, Y: s.Y + t*(e.Y-s.Y)}
	default:
		t := (bound - s.Y) / (e.Y - s.Y)
		return Vec{X: s.X + t*(e.X-s.X), Y: bound}
	}
}

// clipPlane applies one Sutherland-Hodgman half-plane clip.
func clipPlane(poly []Vec, pl plane, bound float64) []Vec {
	if len(poly) == 0 {
		return nil
	}
	out := make([]Vec, 0, len(poly)+4)
	s := poly[len(poly)-1]
	sIn := insidePlane(s, pl, bound)
	for _, e := range poly {
		eIn := insidePlane(e, pl, bound)
		switch {
		case eIn && sIn:
			out = append(out, e)
		case eIn && !sIn:
			out = append(out, intersectPlane(s, e, pl, bound), e)
		case !eIn && sIn:
			out = append(out, intersectPlane(s, e, pl, bound))
		}
		s, sIn = e, eIn
	}
	return out
}

// PixelCoverage returns the exact area of intersection between poly and the
// unit pixel square [ix,ix+1] x [iy,iy+1], i.e. the analytic box-filter
// coverage. The result is in [0,1] for any simple polygon.
func PixelCoverage(poly []Vec, ix, iy int) float64 {
	x0, y0 := float64(ix), float64(iy)
	clipped := poly
	clipped = clipPlane(clipped, planeLeft, x0)
	clipped = clipPlane(clipped, planeRight, x0+1)
	clipped = clipPlane(clipped, planeTop, y0)
	clipped = clipPlane(clipped, planeBottom, y0+1)
	c := PolygonArea(clipped)
	if c < 0 {
		c = 0
	}
	if c > 1 {
		c = 1
	}
	return c
}

// PointInPoly reports whether p lies inside or on the boundary of poly using
// the even-odd ray cast. Polygons here are simple outlines.
func PointInPoly(p Vec, poly []Vec) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a, b := poly[j], poly[i]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			xCross := a.X + (p.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			if xCross >= p.X {
				inside = !inside
			}
		}
	}
	return inside
}
