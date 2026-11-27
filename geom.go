package main

// Rect is an axis-aligned rectangle in pixel coordinates.
// The range is half-open: [X0, X1) x [Y0, Y1), so a rectangle covering
// pixels x=10..19 has X0=10, X1=20.
type Rect struct {
	X0, Y0, X1, Y1 int
}

// RectWH builds a half-open rectangle from x/y and width/height.
func RectWH(x, y, w, h int) Rect {
	return Rect{X0: x, Y0: y, X1: x + w, Y1: y + h}
}

// Empty reports whether the rectangle contains zero pixels.
func (r Rect) Empty() bool {
	return r.X1 <= r.X0 || r.Y1 <= r.Y0
}

func (r Rect) Width() int  { return r.X1 - r.X0 }
func (r Rect) Height() int { return r.Y1 - r.Y0 }

// Contains reports whether pixel (x, y) lies inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X0 && x < r.X1 && y >= r.Y0 && y < r.Y1
}

// Intersect returns the overlap of a and b. The result is Empty when the
// rectangles do not share at least one pixel.
func (a Rect) Intersect(b Rect) Rect {
	r := Rect{
		X0: maxInt(a.X0, b.X0),
		Y0: maxInt(a.Y0, b.Y0),
		X1: minInt(a.X1, b.X1),
		Y1: minInt(a.Y1, b.Y1),
	}
	if r.Empty() {
		return Rect{}
	}
	return r
}

// Union returns the bounding rectangle covering both a and b.
// Empty rectangles are treated as missing operands.
func (a Rect) Union(b Rect) Rect {
	if a.Empty() {
		return b
	}
	if b.Empty() {
		return a
	}
	return Rect{
		X0: minInt(a.X0, b.X0),
		Y0: minInt(a.Y0, b.Y0),
		X1: maxInt(a.X1, b.X1),
		Y1: maxInt(a.Y1, b.Y1),
	}
}

// Inflate expands the rectangle by n pixels on every side.
func (r Rect) Inflate(n int) Rect {
	if r.Empty() {
		return r
	}
	return Rect{X0: r.X0 - n, Y0: r.Y0 - n, X1: r.X1 + n, Y1: r.Y1 + n}
}

// Offset moves the rectangle by (dx, dy).
func (r Rect) Offset(dx, dy int) Rect {
	return Rect{X0: r.X0 + dx, Y0: r.Y0 + dy, X1: r.X1 + dx, Y1: r.Y1 + dy}
}

// Pixels returns the number of integer pixels in the rectangle.
func (r Rect) Pixels() int {
	if r.Empty() {
		return 0
	}
	return r.Width() * r.Height()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MergeRects returns a conservative set of pairwise disjoint rectangles
// covering every input rectangle. Overlapping inputs are replaced by their
// bounding union; the returned area may be slightly larger than the exact
// region, which is safe for dirty-rectangle repainting.
func MergeRects(in []Rect) []Rect {
	rs := make([]Rect, 0, len(in))
	for _, r := range in {
		if !r.Empty() {
			rs = append(rs, r)
		}
	}

	for {
		mergedAny := false
		for i := 0; i < len(rs) && !mergedAny; i++ {
			for j := i + 1; j < len(rs); j++ {
				if !rs[i].Intersect(rs[j]).Empty() {
					rs[i] = rs[i].Union(rs[j])
					rs = append(rs[:j], rs[j+1:]...)
					mergedAny = true
					break
				}
			}
		}
		if !mergedAny {
			return rs
		}
	}
}

// RectsDisjoint is a test/helper invariant check.
func RectsDisjoint(rs []Rect) bool {
	for i := 0; i < len(rs); i++ {
		for j := i + 1; j < len(rs); j++ {
			if !rs[i].Intersect(rs[j]).Empty() {
				return false
			}
		}
	}
	return true
}
