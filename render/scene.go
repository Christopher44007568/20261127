package render

import "math"

type Ray struct {
	Origin    Vec3
	Direction Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.Origin.Add(r.Direction.Mul(t))
}

type Camera struct {
	Position    Vec3
	Target      Vec3
	VerticalFOV float64
}

func NewCamera(position, target Vec3, verticalFOV float64) Camera {
	return Camera{Position: position, Target: target, VerticalFOV: verticalFOV}
}

func (c Camera) Ray(u, v, aspect float64) Ray {
	forward := c.Target.Sub(c.Position).Normalize()
	right := forward.Cross(V(0, 1, 0)).Normalize()
	up := right.Cross(forward).Normalize()
	halfHeight := math.Tan(c.VerticalFOV * math.Pi / 360)

	direction := forward.
		Add(right.Mul(u * halfHeight * aspect)).
		Add(up.Mul(v * halfHeight)).
		Normalize()
	return Ray{Origin: c.Position, Direction: direction}
}

type Sphere struct {
	Center            Vec3
	Radius            float64
	Material          Material
	RoughnessGradient bool
	GradientMin       float64
	GradientMax       float64
}

func (s Sphere) Hit(ray Ray) (float64, bool) {
	oc := ray.Origin.Sub(s.Center)
	b := oc.Dot(ray.Direction)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return 0, false
	}

	root := math.Sqrt(disc)
	t := -b - root
	if t > 1e-6 {
		return t, true
	}
	t = -b + root
	return t, t > 1e-6
}

func (s Sphere) Occludes(origin, direction Vec3, maxDistance float64) bool {
	ray := Ray{Origin: origin, Direction: direction}
	t, ok := s.Hit(ray)
	return ok && t < maxDistance
}

func (s Sphere) RoughnessAt(normal Vec3) float64 {
	base := s.Material.roughness()
	if !s.RoughnessGradient {
		return base
	}
	lo, hi := s.GradientMin, s.GradientMax
	if hi <= 0 {
		lo, hi = MinCoatingRoughness, 0.9
	}
	phase := Smoothstep(-0.78, 0.78, normal.X)
	return Lerp(lo, hi, phase)
}

type Scene struct {
	Sphere Sphere
	FloorY float64
}

type RenderMode int

const (
	RenderComposite RenderMode = iota
	RenderDiffuse
	RenderClearcoat
)

type RenderOptions struct {
	Width    int
	Height   int
	Samples  int
	Mode     RenderMode
	Isolated bool
}

func DefaultOptions(width, height int, mode RenderMode) RenderOptions {
	return RenderOptions{
		Width:   width,
		Height:  height,
		Samples: 2,
		Mode:    mode,
	}
}
