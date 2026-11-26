package geom

import "math"

type V struct {
	X, Y float64
}

func (a V) Add(b V) V       { return V{a.X + b.X, a.Y + b.Y} }
func (a V) Sub(b V) V       { return V{a.X - b.X, a.Y - b.Y} }
func (a V) Mul(s float64) V { return V{a.X * s, a.Y * s} }
func (a V) Dot(b V) float64 { return a.X*b.X + a.Y*b.Y }
func (a V) Cross(b V) float64 {
	return a.X*b.Y - a.Y*b.X
}
func (a V) Len() float64 { return math.Hypot(a.X, a.Y) }

func (a V) Norm() V {
	l := a.Len()
	if l < 1e-12 {
		return V{}
	}
	return a.Mul(1 / l)
}

func (a V) Dist(b V) float64 { return a.Sub(b).Len() }

type Box struct {
	Min V
	Max V
}

func EmptyBox() Box {
	return Box{Min: V{math.Inf(1), math.Inf(1)}, Max: V{math.Inf(-1), math.Inf(-1)}}
}

func (b *Box) Include(p V) {
	b.Min.X = math.Min(b.Min.X, p.X)
	b.Min.Y = math.Min(b.Min.Y, p.Y)
	b.Max.X = math.Max(b.Max.X, p.X)
	b.Max.Y = math.Max(b.Max.Y, p.Y)
}

func (b Box) Empty() bool { return b.Min.X > b.Max.X }

type Segment struct {
	A, B V
}

type Contour struct {
	Points []V
}

type Shape struct {
	Contours []Contour
}

func (s Shape) Empty() bool { return len(s.Contours) == 0 }

func (s Shape) Box() Box {
	box := EmptyBox()
	for _, c := range s.Contours {
		for _, p := range c.Points {
			box.Include(p)
		}
	}
	return box
}

// WindingNumber implements the standard ray-crossing winding count.
// A non-zero result is the fill rule used by TrueType glyphs.
func WindingNumber(p V, contours []Contour) int {
	winding := 0
	for _, c := range contours {
		n := len(c.Points)
		if n < 3 {
			continue
		}
		for i := 0; i < n; i++ {
			a := c.Points[i]
			b := c.Points[(i+1)%n]
			if a.Y <= p.Y {
				if b.Y > p.Y && CrossSide(a, b, p) > 0 {
					winding++
				}
			} else {
				if b.Y <= p.Y && CrossSide(a, b, p) < 0 {
					winding--
				}
			}
		}
	}
	return winding
}

func Inside(p V, contours []Contour) bool {
	return WindingNumber(p, contours) != 0
}

func CrossSide(a, b, p V) float64 {
	return b.Sub(a).Cross(p.Sub(a))
}

// PointSegment returns the true Euclidean distance and the clamped
// projection parameter in [0,1].
func PointSegment(p V, s Segment) (dist, t float64) {
	d := s.B.Sub(s.A)
	l2 := d.Dot(d)
	if l2 <= 1e-18 {
		return p.Dist(s.A), 0
	}
	t = p.Sub(s.A).Dot(d) / l2
	t = math.Max(0, math.Min(1, t))
	q := s.A.Add(d.Mul(t))
	return p.Dist(q), t
}

// PseudoLineDistance is the oriented distance to the infinite line through
// the segment. It intentionally does not clamp the projection; this is the
// continuation used by MSDF corner reconstruction.
func PseudoLineDistance(p V, s Segment) float64 {
	d := s.B.Sub(s.A)
	l := d.Len()
	if l < 1e-12 {
		return 0
	}
	return d.Cross(p.Sub(s.A)) / l
}
