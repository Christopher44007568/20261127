package main

import "math"

// Contour is a closed polygon without a duplicated closing point.
type Contour []Vec2

// Outline is one outer contour and its counter (hole) contours.
type Outline struct {
	Outer Contour   `json:"outer"`
	Holes []Contour `json:"holes"`
}

func (c Contour) Clone() Contour {
	out := make(Contour, len(c))
	copy(out, c)
	return out
}

func (o Outline) Contours() []Contour {
	result := make([]Contour, 0, 1+len(o.Holes))
	result = append(result, o.Outer)
	result = append(result, o.Holes...)
	return result
}

func (o Outline) Clone() Outline {
	holes := make([]Contour, len(o.Holes))
	for i, h := range o.Holes {
		holes[i] = h.Clone()
	}
	return Outline{Outer: o.Outer.Clone(), Holes: holes}
}

// SignedArea returns twice the shoelace area. Positive is CCW in font
// coordinates where Y points upward.
func (c Contour) SignedArea() float64 {
	if len(c) < 3 {
		return 0
	}
	sum := 0.0
	for i := range c {
		a := c[i]
		b := c[(i+1)%len(c)]
		sum += a.X*b.Y - b.X*a.Y
	}
	return sum * 0.5
}

func (c Contour) Area() float64 { return math.Abs(c.SignedArea()) }

func (c Contour) Bounds() Bounds {
	b := NewBounds()
	for _, p := range c {
		b = b.Include(p)
	}
	return b
}

func (o Outline) Bounds() Bounds {
	b := o.Outer.Bounds()
	for _, h := range o.Holes {
		hb := h.Bounds()
		b.Min.X = math.Min(b.Min.X, hb.Min.X)
		b.Min.Y = math.Min(b.Min.Y, hb.Min.Y)
		b.Max.X = math.Max(b.Max.X, hb.Max.X)
		b.Max.Y = math.Max(b.Max.Y, hb.Max.Y)
	}
	return b
}

// Canonical returns a topology-preserving copy with explicit orientations:
// outer is CCW and holes are CW. For both loops, RightNormal then points
// away from filled material, i.e. the direction of weight expansion.
func (o Outline) Canonical() Outline {
	out := o.Clone()
	out.Outer = RemoveClosingDuplicate(out.Outer)
	if out.Outer.SignedArea() < 0 {
		out.Outer.Reverse()
	}
	for i := range out.Holes {
		out.Holes[i] = RemoveClosingDuplicate(out.Holes[i])
		if out.Holes[i].SignedArea() > 0 {
			out.Holes[i].Reverse()
		}
	}
	return out
}

func (c Contour) Reverse() {
	for i, j := 0, len(c)-1; i < j; i, j = i+1, j-1 {
		c[i], c[j] = c[j], c[i]
	}
}

func RemoveClosingDuplicate(c Contour) Contour {
	if len(c) >= 2 && c[0].ApproxEqual(c[len(c)-1], 1e-12) {
		c = c[:len(c)-1]
	}
	return c
}

// CleanContour removes duplicated and extremely short redundant edges.
func CleanContour(c Contour, eps float64) Contour {
	if eps <= 0 {
		eps = 1e-9
	}
	out := make(Contour, 0, len(c))
	for _, p := range c {
		if len(out) == 0 || !out[len(out)-1].ApproxEqual(p, eps) {
			out = append(out, p)
		}
	}
	if len(out) >= 2 && out[0].ApproxEqual(out[len(out)-1], eps) {
		out = out[:len(out)-1]
	}
	return out
}

// Contains reports whether p is inside the contour by ray casting.
func (c Contour) Contains(p Vec2) bool {
	inside := false
	n := len(c)
	for i := 0; i < n; i++ {
		a := c[i]
		b := c[(i+1)%n]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := a.X + (p.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			if x > p.X {
				inside = !inside
			}
		}
	}
	return inside
}

// Contains uses even-odd filling, matching outline holes.
func (o Outline) Contains(p Vec2) bool {
	inside := o.Outer.Contains(p)
	for _, h := range o.Holes {
		if h.Contains(p) {
			inside = !inside
		}
	}
	return inside
}

func (o Outline) Scale(s float64) Outline {
	sc := func(c Contour) Contour {
		r := make(Contour, len(c))
		for i, p := range c {
			r[i] = p.Scale(s)
		}
		return r
	}
	holes := make([]Contour, len(o.Holes))
	for i, h := range o.Holes {
		holes[i] = sc(h)
	}
	return Outline{Outer: sc(o.Outer), Holes: holes}
}

// CounterArea returns total positive hole area in square design units.
func (o Outline) CounterArea() float64 {
	total := 0.0
	for _, h := range o.Holes {
		total += h.Area()
	}
	return total
}

func centroid(c Contour) Vec2 {
	var sum Vec2
	for _, p := range c {
		sum = sum.Add(p)
	}
	if len(c) == 0 {
		return Vec2{}
	}
	return sum.Scale(1 / float64(len(c)))
}

// NestingSignature describes loop count and containment depth. Vertex count
// is intentionally absent because corner correction may add bevel points.
func (o Outline) NestingSignature() []int {
	contours := o.Contours()
	sig := make([]int, len(contours))
	for i, c := range contours {
		p := centroid(c)
		depth := 0
		for j, other := range contours {
			if i != j && other.Contains(p) {
				depth++
			}
		}
		sig[i] = depth
	}
	return sig
}
