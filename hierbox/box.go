package hierbox

import "math"

// Box3 is an axis-aligned bounding box in world or local coordinates.
type Box3 struct {
	Min   Vec3   `json:"min"`
	Max   Vec3   `json:"max"`
	Empty bool   `json:"empty"`
	Label string `json:"label,omitempty"`
}

func EmptyBox() Box3 {
	return Box3{Empty: true, Min: V(math.Inf(1), math.Inf(1), math.Inf(1)), Max: V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
}

func BoxPoint(p Vec3) Box3 {
	return Box3{Min: p, Max: p}
}

func (b Box3) AddPoint(p Vec3) Box3 {
	if b.Empty {
		return BoxPoint(p)
	}
	b.Min = b.Min.Min(p)
	b.Max = b.Max.Max(p)
	return b
}

func (b Box3) AddBox(c Box3) Box3 {
	if c.Empty {
		return b
	}
	if b.Empty {
		return c
	}
	b.Min = b.Min.Min(c.Min)
	b.Max = b.Max.Max(c.Max)
	return b
}

func (b Box3) Center() Vec3 {
	return b.Min.Add(b.Max).Scale(0.5)
}

func (b Box3) Size() Vec3 {
	if b.Empty {
		return Vec3{}
	}
	return b.Max.Sub(b.Min)
}

func (b Box3) Volume() float64 {
	if b.Empty {
		return 0
	}
	s := b.Size()
	if s.X < 0 || s.Y < 0 || s.Z < 0 {
		return 0
	}
	return s.X * s.Y * s.Z
}

func (b Box3) SurfaceArea() float64 {
	s := b.Size()
	return 2 * (s.X*s.Y + s.Y*s.Z + s.Z*s.X)
}

func (b Box3) Corners() [8]Vec3 {
	return [8]Vec3{
		V(b.Min.X, b.Min.Y, b.Min.Z),
		V(b.Max.X, b.Min.Y, b.Min.Z),
		V(b.Min.X, b.Max.Y, b.Min.Z),
		V(b.Max.X, b.Max.Y, b.Min.Z),
		V(b.Min.X, b.Min.Y, b.Max.Z),
		V(b.Max.X, b.Min.Y, b.Max.Z),
		V(b.Min.X, b.Max.Y, b.Max.Z),
		V(b.Max.X, b.Max.Y, b.Max.Z),
	}
}

// Transform returns the axis-aligned box enclosing all eight transformed
// corners. It is conservative for any rigid transform represented by Mat4.
func (b Box3) Transform(m Mat4) Box3 {
	if b.Empty {
		return b
	}
	out := EmptyBox()
	for _, c := range b.Corners() {
		out = out.AddPoint(m.TransformPoint(c))
	}
	return out
}

// Expanded returns a box grown by absolute padding in every direction.
func (b Box3) Expanded(pad float64) Box3 {
	if b.Empty {
		return b
	}
	return Box3{Min: V(b.Min.X-pad, b.Min.Y-pad, b.Min.Z-pad), Max: V(b.Max.X+pad, b.Max.Y+pad, b.Max.Z+pad)}
}

func (b Box3) ContainsPoint(p Vec3, eps float64) bool {
	if b.Empty {
		return false
	}
	return p.X >= b.Min.X-eps && p.X <= b.Max.X+eps &&
		p.Y >= b.Min.Y-eps && p.Y <= b.Max.Y+eps &&
		p.Z >= b.Min.Z-eps && p.Z <= b.Max.Z+eps
}

func (a Box3) Intersects(b Box3) bool {
	if a.Empty || b.Empty {
		return false
	}
	return a.Min.X <= b.Max.X && a.Max.X >= b.Min.X &&
		a.Min.Y <= b.Max.Y && a.Max.Y >= b.Min.Y &&
		a.Min.Z <= b.Max.Z && a.Max.Z >= b.Min.Z
}

// UnionBox returns the union of many boxes.
func UnionBox(boxes ...Box3) Box3 {
	out := EmptyBox()
	for _, b := range boxes {
		out = out.AddBox(b)
	}
	return out
}
