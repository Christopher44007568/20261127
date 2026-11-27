package main

import "math"

type Camera struct {
	Position Vec3
	Target   Vec3
	Up       Vec3
	FovY     float64
	Aspect   float64
	Near     float64
	Far      float64

	right   Vec3
	up      Vec3
	forward Vec3
}

func NewCamera(position, target, up Vec3, fovYDeg, aspect, near, far float64) *Camera {
	c := &Camera{
		Position: position,
		Target:   target,
		Up:       up,
		FovY:     fovYDeg * math.Pi / 180,
		Aspect:   aspect,
		Near:     near,
		Far:      far,
	}
	c.forward = target.Sub(position).Normalize()
	c.right = up.Cross(c.forward).Normalize()
	c.up = c.forward.Cross(c.right).Normalize()
	return c
}

func (c *Camera) Basis() (right, up, forward Vec3) {
	return c.right, c.up, c.forward
}

func (c *Camera) HorizontalFOV() float64 {
	return 2 * math.Atan(math.Tan(c.FovY*0.5)*c.Aspect)
}

func (c *Camera) View(p Vec3) Vec3 {
	d := p.Sub(c.Position)
	return V(d.Dot(c.right), d.Dot(c.up), d.Dot(c.forward))
}

func (c *Camera) ViewZ(p Vec3) float64 {
	return p.Sub(c.Position).Dot(c.forward)
}

func (c *Camera) RayAt(px, py, width, height float64) Ray {
	ndcX := 2*px/width - 1
	ndcY := 1 - 2*py/height
	tanHalf := math.Tan(c.FovY * 0.5)
	dir := c.forward.
		Add(c.right.Mul(ndcX * tanHalf * c.Aspect)).
		Add(c.up.Mul(ndcY * tanHalf)).
		Normalize()
	return Ray{Origin: c.Position, Direction: dir}
}

type Material struct {
	Albedo    Vec3
	Specular  float64
	Shininess float64
	Emissive  Vec3
	Checker   float64
}

type LightKind int

const (
	PointLight LightKind = iota
	SpotLight
)

type Light struct {
	Kind      LightKind
	Position  Vec3
	Direction Vec3
	Color     Vec3
	Intensity float64
	Range     float64
	InnerCos  float64
	OuterCos  float64
}

func newPointLight(position, color Vec3, intensity, radius float64) Light {
	return Light{
		Kind:      PointLight,
		Position:  position,
		Color:     color,
		Intensity: intensity,
		Range:     radius,
		InnerCos:  1,
		OuterCos:  0,
	}
}

func newSpotLight(position, direction, color Vec3, intensity, radius, innerDeg, outerDeg float64) Light {
	return Light{
		Kind:      SpotLight,
		Position:  position,
		Direction: direction.Normalize(),
		Color:     color,
		Intensity: intensity,
		Range:     radius,
		InnerCos:  math.Cos(innerDeg * math.Pi / 180),
		OuterCos:  math.Cos(outerDeg * math.Pi / 180),
	}
}

// InfluenceSphere conservatively contains every point at which the light can
// contribute. The solver uses it for cluster culling; it never uses it to
// approximate the lighting itself.
func (l Light) InfluenceSphere() (Vec3, float64) {
	if l.Kind == PointLight {
		return l.Position, l.Range
	}
	if l.OuterCos <= 1e-4 {
		return l.Position, math.Inf(1)
	}
	outer := clamp(l.OuterCos, 0, 1)
	baseRadius := l.Range * math.Sqrt(math.Max(0, 1-outer*outer)) / outer
	center := l.Position.Add(l.Direction.Mul(l.Range * 0.5))
	radius := math.Hypot(l.Range*0.5, baseRadius)
	return center, radius
}

func (l Light) Attenuation(p Vec3) float64 {
	toLight := l.Position.Sub(p)
	d2 := toLight.LengthSquared()
	r2 := l.Range * l.Range
	if d2 >= r2 {
		return 0
	}
	window := 1 - d2/r2
	att := l.Intensity * window * window / (1 + d2*0.32)
	if l.Kind == SpotLight {
		cosAngle := toLight.Normalize().Dot(l.Direction)
		att *= smoothstep(l.OuterCos, l.InnerCos, cosAngle)
	}
	return att
}

type Sphere struct {
	Center   Vec3
	Radius   float64
	Material Material
}

type Box struct {
	Bounds   AABB
	Material Material
}

type Volume struct {
	Bounds     AABB
	Density    float64
	Anisotropy float64
}

type Scene struct {
	Spheres []Sphere
	Boxes   []Box
	Lights  []Light
	Volume  *Volume
}

type Hit struct {
	T        float64
	Point    Vec3
	Normal   Vec3
	Material Material
	ObjectID int
	IsSphere bool
}

func (s *Scene) Intersect(r Ray) (Hit, bool) {
	best := Hit{T: math.Inf(1)}
	found := false
	for i := range s.Spheres {
		if t, ok := intersectSphere(r, s.Spheres[i]); ok && t < best.T {
			best = Hit{
				T:        t,
				Point:    r.At(t),
				Normal:   r.At(t).Sub(s.Spheres[i].Center).Normalize(),
				Material: s.Spheres[i].Material,
				ObjectID: i,
				IsSphere: true,
			}
			found = true
		}
	}
	for i := range s.Boxes {
		if t, n, ok := intersectBox(r, s.Boxes[i].Bounds); ok && t < best.T {
			best = Hit{
				T:        t,
				Point:    r.At(t),
				Normal:   n,
				Material: s.Boxes[i].Material,
				ObjectID: len(s.Spheres) + i,
			}
			found = true
		}
	}
	return best, found
}

func (s *Scene) Occluded(r Ray, maxT float64) bool {
	for i := range s.Spheres {
		if s.Spheres[i].Material.Emissive.LengthSquared() > 0 {
			continue
		}
		if t, ok := intersectSphere(r, s.Spheres[i]); ok && t > 1e-5 && t < maxT {
			return true
		}
	}
	for i := range s.Boxes {
		if t, _, ok := intersectBox(r, s.Boxes[i].Bounds); ok && t > 1e-5 && t < maxT {
			return true
		}
	}
	return false
}

func intersectSphere(r Ray, s Sphere) (float64, bool) {
	oc := r.Origin.Sub(s.Center)
	b := oc.Dot(r.Direction)
	c := oc.LengthSquared() - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return 0, false
	}
	root := math.Sqrt(disc)
	t0 := -b - root
	if t0 > 1e-6 {
		return t0, true
	}
	t1 := -b + root
	if t1 > 1e-6 {
		return t1, true
	}
	return 0, false
}

func intersectBox(r Ray, box AABB) (float64, Vec3, bool) {
	tmin, tmax := -math.Inf(1), math.Inf(1)
	normalAxis, normalSign := 0, -1.0
	origin := [3]float64{r.Origin.X, r.Origin.Y, r.Origin.Z}
	dir := [3]float64{r.Direction.X, r.Direction.Y, r.Direction.Z}
	lo := [3]float64{box.Min.X, box.Min.Y, box.Min.Z}
	hi := [3]float64{box.Max.X, box.Max.Y, box.Max.Z}
	for axis := 0; axis < 3; axis++ {
		if math.Abs(dir[axis]) < 1e-12 {
			if origin[axis] < lo[axis] || origin[axis] > hi[axis] {
				return 0, Vec3{}, false
			}
			continue
		}
		inv := 1 / dir[axis]
		t0 := (lo[axis] - origin[axis]) * inv
		t1 := (hi[axis] - origin[axis]) * inv
		sign := -1.0
		if t0 > t1 {
			t0, t1 = t1, t0
			sign = 1
		}
		if t0 > tmin {
			tmin = t0
			normalAxis = axis
			normalSign = sign
		}
		tmax = math.Min(tmax, t1)
	}
	if tmax < tmin || tmax < 1e-6 {
		return 0, Vec3{}, false
	}
	t := tmin
	if t < 1e-6 {
		t = tmax
		if t < 1e-6 {
			return 0, Vec3{}, false
		}
		normalAxis = 0
		normalSign = 1
		if math.Abs(r.Origin.Y-box.Min.Y) < math.Abs(r.Origin.X-box.Min.X) &&
			math.Abs(r.Origin.Y-box.Min.Y) < math.Abs(r.Origin.Z-box.Min.Z) {
			normalAxis = 1
		} else if math.Abs(r.Origin.Z-box.Min.Z) < math.Abs(r.Origin.X-box.Min.X) {
			normalAxis = 2
		}
	}
	n := Vec3{}
	switch normalAxis {
	case 0:
		n.X = normalSign
	case 1:
		n.Y = normalSign
	default:
		n.Z = normalSign
	}
	return t, n, true
}

func NewGalleryScene() *Scene {
	floor := Material{Albedo: V(0.16, 0.18, 0.21), Specular: 0.12, Shininess: 48, Checker: 1.7}
	wall := Material{Albedo: V(0.13, 0.14, 0.16), Specular: 0.04, Shininess: 24}
	pillar := Material{Albedo: V(0.25, 0.23, 0.21), Specular: 0.08, Shininess: 32}
	copper := Material{Albedo: V(0.72, 0.24, 0.08), Specular: 0.65, Shininess: 96}
	ceramic := Material{Albedo: V(0.08, 0.42, 0.52), Specular: 0.55, Shininess: 80}
	cream := Material{Albedo: V(0.62, 0.55, 0.42), Specular: 0.22, Shininess: 42}
	black := Material{Albedo: V(0.035, 0.04, 0.05), Specular: 0.35, Shininess: 70}

	scene := &Scene{
		Boxes: []Box{
			{Bounds: AABB{Min: V(-24, -1, -12), Max: V(24, 0, 28)}, Material: floor},
			{Bounds: AABB{Min: V(-16, 0, 20), Max: V(16, 9, 21)}, Material: wall},
			{Bounds: AABB{Min: V(-11, 0, 1), Max: V(-10, 9, 21)}, Material: wall},
			{Bounds: AABB{Min: V(10, 0, 1), Max: V(11, 9, 21)}, Material: wall},
			{Bounds: AABB{Min: V(-6.3, 0, 12.4), Max: V(-5.2, 7.2, 13.3)}, Material: pillar},
			{Bounds: AABB{Min: V(5.2, 0, 12.4), Max: V(6.3, 7.2, 13.3)}, Material: pillar},
			{Bounds: AABB{Min: V(-4.3, 0, 6.8), Max: V(-2.1, 1.05, 9.2)}, Material: black},
			{Bounds: AABB{Min: V(2.15, 0, 9.5), Max: V(4.35, 0.72, 11.7)}, Material: pillar},
			{Bounds: AABB{Min: V(-0.8, 0.0, 13.1), Max: V(0.8, 1.65, 14.7)}, Material: cream},
		},
		Spheres: []Sphere{
			{Center: V(-3.2, 2.0, 8.0), Radius: 0.95, Material: copper},
			{Center: V(3.25, 1.55, 10.6), Radius: 0.82, Material: ceramic},
			{Center: V(-0.25, 0.52, 4.8), Radius: 0.52, Material: ceramic},
		},
		Volume: &Volume{
			Bounds:     AABB{Min: V(-7.5, 0.06, 3.2), Max: V(7.5, 3.5, 17.5)},
			Density:    0.095,
			Anisotropy: 0.45,
		},
	}
	scene.Lights = []Light{
		newPointLight(V(-4.8, 3.45, 4.4), V(1.0, 0.32, 0.055), 25, 11.5),
		newPointLight(V(4.9, 2.7, 6.9), V(0.03, 0.52, 1.0), 23, 12),
		newPointLight(V(1.4, 5.35, 13.4), V(0.72, 0.10, 1.0), 30, 14),
		newPointLight(V(-1.2, 7.15, 7.2), V(1.0, 0.78, 0.48), 18, 13),
		newPointLight(V(-2.0, 0.52, 11.9), V(0.12, 1.0, 0.28), 8, 5.5),
		newPointLight(V(3.7, 0.48, 14.2), V(0.08, 0.22, 1.0), 9, 6.5),
		newPointLight(V(5.8, 3.8, 16.2), V(1.0, 0.34, 0.06), 16, 9),
		newSpotLight(V(0, 7.6, 4.4), V(0, -1, 0.22), V(1.0, 0.88, 0.68), 70, 16, 16, 31),
		newSpotLight(V(-7.2, 6.4, 8.0), V(0.82, -0.60, 0.15), V(0.08, 0.48, 1.0), 105, 18, 12, 27),
		newSpotLight(V(7.0, 6.1, 10.0), V(-0.76, -0.64, 0.08), V(1.0, 0.19, 0.045), 100, 18, 14, 31),
		newSpotLight(V(0.0, 6.3, 18.9), V(0, -0.82, -0.57), V(1.0, 0.25, 0.68), 88, 15, 18, 39),
	}
	return scene
}
