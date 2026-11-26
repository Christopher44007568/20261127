package pipeline

import "math"

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) MulV(b Vec3) Vec3   { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Length() float64    { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Normalize() Vec3    { return a.Mul(1 / a.Length()) }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}
func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Mul(t)) }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// ShadowBuffer is written only by the shadow pass and read only by scene.
type ShadowBuffer struct {
	Width      int
	Height     int
	Visibility []float64
}

// SceneHDR is the linear-light output of the scene pass.
type SceneHDR struct {
	Width  int
	Height int
	Pixels []Vec3
}

// SceneLDR is the display-space scene produced by tonemap.
type SceneLDR struct {
	Width  int
	Height int
	Pixels []Vec3
}

// UIImage is kept separate from the scene so its position in the schedule
// cannot determine whether the tonemapper sees UI values.
type UIImage struct {
	Width  int
	Height int
	Pixels []UIPixel
}

type UIPixel struct {
	R, G, B, A uint8
}

type Frame struct {
	Width  int
	Height int
	RGB    []byte
}

type Outputs struct {
	Shadow    *ShadowBuffer
	Scene     *SceneHDR
	Tonemaped *SceneLDR
	UI        *UIImage
}

type passInputs struct {
	shadow    *ShadowBuffer
	scene     *SceneHDR
	tonemaped *SceneLDR
	ui        *UIImage
}

type passResult struct {
	shadow    *ShadowBuffer
	scene     *SceneHDR
	tonemaped *SceneLDR
	ui        *UIImage
}
