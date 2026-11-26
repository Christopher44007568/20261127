package render

import (
	"math"

	"meshbool/internal/mesh"
)

type Camera struct {
	Width, Height int
	Center        mesh.Vec3
	Right         mesh.Vec3
	Up            mesh.Vec3
	Forward       mesh.Vec3
	Scale         float64
	MidX, MidY    float64
}

type Projected struct {
	X, Y, Depth float64
}

func NewCamera(m mesh.Mesh, width, height int, padding float64) Camera {
	if width < 2 || height < 2 {
		width, height = 640, 480
	}
	min, max := m.Bounds()
	center := min.Add(max).Mul(0.5)
	forward := mesh.Vec3{X: 1.35, Y: 1.02, Z: 1.22}.Normalize()
	right := forward.Cross(mesh.Vec3{Y: 1}).Normalize()
	if right.Length() < 0.1 {
		right = mesh.Vec3{X: 1}
	}
	up := right.Cross(forward).Normalize()
	camera := Camera{
		Width: width, Height: height, Center: center,
		Right: right, Up: up, Forward: forward,
	}
	if len(m.Triangles) == 0 {
		camera.Scale = math.Min(float64(width), float64(height)) * 0.35
		return camera
	}

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, triangle := range m.Triangles {
		for _, point := range []mesh.Vec3{triangle.A, triangle.B, triangle.C} {
			x, y, _ := camera.rawProject(point)
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		}
	}
	extentX := math.Max(maxX-minX, 1e-6)
	extentY := math.Max(maxY-minY, 1e-6)
	availableX := math.Max(float64(width)-2*padding, 8)
	availableY := math.Max(float64(height)-2*padding, 8)
	camera.Scale = math.Min(availableX/extentX, availableY/extentY)
	camera.MidX = (minX + maxX) * 0.5
	camera.MidY = (minY + maxY) * 0.5
	return camera
}

func (c Camera) rawProject(point mesh.Vec3) (x, y, depth float64) {
	local := point.Sub(c.Center)
	return local.Dot(c.Right), -local.Dot(c.Up), local.Dot(c.Forward)
}

func (c Camera) Project(point mesh.Vec3) Projected {
	x, y, depth := c.rawProject(point)
	return Projected{
		X:     float64(c.Width)*0.5 + (x-c.MidX)*c.Scale,
		Y:     float64(c.Height)*0.5 + (y-c.MidY)*c.Scale,
		Depth: depth,
	}
}

func (c Camera) FitFactor() float64 {
	return c.Scale
}
