package main

import "math"

type Camera struct {
	Eye     Vec3
	Target  Vec3
	Forward Vec3
	Right   Vec3
	Up      Vec3
	TanHalf float64
	Near    float64
	FOVDeg  float64
}

func NewCamera(eye, target, up Vec3, fovDegrees float64) Camera {
	forward := target.Sub(eye).Normalize()
	right := forward.Cross(up).Normalize()
	if right.LengthSquared() < 1e-10 {
		right = V(1, 0, 0)
	}
	cameraUp := right.Cross(forward).Normalize()
	return Camera{
		Eye:     eye,
		Target:  target,
		Forward: forward,
		Right:   right,
		Up:      cameraUp,
		TanHalf: math.Tan(fovDegrees * math.Pi / 180 / 2),
		Near:    0.08,
		FOVDeg:  fovDegrees,
	}
}

func (c Camera) depth(p Vec3) float64 {
	return p.Sub(c.Eye).Dot(c.Forward)
}

type Frame struct {
	Width  int
	Height int
	Color  []Vec3
	Depth  []float64
}

func NewFrame(width, height int) *Frame {
	return &Frame{
		Width:  width,
		Height: height,
		Color:  make([]Vec3, width*height),
		Depth:  make([]float64, width*height),
	}
}

func (f *Frame) index(x, y int) int { return y*f.Width + x }

func (f *Frame) Clear(top, bottom Vec3) {
	for y := 0; y < f.Height; y++ {
		t := float64(y) / float64(max(f.Height-1, 1))
		background := lerpVec(top, bottom, t)
		for x := 0; x < f.Width; x++ {
			i := f.index(x, y)
			f.Color[i] = background
			f.Depth[i] = 0
		}
	}
}

type projectedVertex struct {
	sx, sy  float64
	invZ    float64
	world   Vec3
	normal  Vec3
	visible bool
}

func (c Camera) project(v Vertex, width, height int) projectedVertex {
	q := v.Position.Sub(c.Eye)
	z := q.Dot(c.Forward)
	if z < c.Near {
		return projectedVertex{visible: false}
	}
	x := q.Dot(c.Right)
	y := q.Dot(c.Up)
	pixelScale := float64(height) / (2 * c.TanHalf)
	return projectedVertex{
		sx:      float64(width)/2 + x/z*pixelScale,
		sy:      float64(height)/2 - y/z*pixelScale,
		invZ:    1 / z,
		world:   v.Position,
		normal:  v.Normal,
		visible: true,
	}
}

func edgeFunction(a, b Vec3, px, py float64) float64 {
	return (b.X-a.X)*(py-a.Y) - (b.Y-a.Y)*(px-a.X)
}

type ShadeContext struct {
	World    Vec3
	Normal   Vec3
	DWorldDX Vec3
	DWorldDY Vec3
}

type ShadeFunc func(ctx ShadeContext) Vec3

func DrawMesh(frame *Frame, camera Camera, mesh Mesh, shade ShadeFunc) {
	for tri := 0; tri < len(mesh.Indices); tri += 3 {
		i0 := mesh.Indices[tri]
		i1 := mesh.Indices[tri+1]
		i2 := mesh.Indices[tri+2]
		p0 := camera.project(mesh.Vertices[i0], frame.Width, frame.Height)
		p1 := camera.project(mesh.Vertices[i1], frame.Width, frame.Height)
		p2 := camera.project(mesh.Vertices[i2], frame.Width, frame.Height)
		if !p0.visible || !p1.visible || !p2.visible {
			continue
		}
		drawTriangle(frame, p0, p1, p2, shade)
	}
}

func drawTriangle(frame *Frame, p0, p1, p2 projectedVertex, shade ShadeFunc) {
	minX := int(math.Floor(math.Min(p0.sx, math.Min(p1.sx, p2.sx))))
	maxX := int(math.Ceil(math.Max(p0.sx, math.Max(p1.sx, p2.sx))))
	minY := int(math.Floor(math.Min(p0.sy, math.Min(p1.sy, p2.sy))))
	maxY := int(math.Ceil(math.Max(p0.sy, math.Max(p1.sy, p2.sy))))
	minX = clampInt(minX, 0, frame.Width-1)
	maxX = clampInt(maxX, 0, frame.Width-1)
	minY = clampInt(minY, 0, frame.Height-1)
	maxY = clampInt(maxY, 0, frame.Height-1)
	if minX > maxX || minY > maxY {
		return
	}

	a := V(p0.sx, p0.sy, 0)
	b := V(p1.sx, p1.sy, 0)
	c := V(p2.sx, p2.sy, 0)
	area := edgeFunction(a, b, c.X, c.Y)
	if math.Abs(area) < 1e-10 {
		return
	}

	for y := minY; y <= maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			w0 := edgeFunction(b, c, px, py) / area
			w1 := edgeFunction(c, a, px, py) / area
			w2 := edgeFunction(a, b, px, py) / area
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}

			invZ := w0*p0.invZ + w1*p1.invZ + w2*p2.invZ
			index := frame.index(x, y)
			if invZ <= frame.Depth[index] {
				continue
			}

			c0 := w0 * p0.invZ
			c1 := w1 * p1.invZ
			c2 := w2 * p2.invZ
			world := p0.world.Scale(c0).Add(p1.world.Scale(c1)).Add(p2.world.Scale(c2)).Scale(1 / invZ)
			normal := p0.normal.Scale(c0).Add(p1.normal.Scale(c1)).Add(p2.normal.Scale(c2)).Scale(1 / invZ).Normalize()

			dW0DX := (p1.sy - p2.sy) / area
			dW1DX := (p2.sy - p0.sy) / area
			dW2DX := (p0.sy - p1.sy) / area
			dW0DY := (p2.sx - p1.sx) / area
			dW1DY := (p0.sx - p2.sx) / area
			dW2DY := (p1.sx - p0.sx) / area

			dInvZDX := dW0DX*p0.invZ + dW1DX*p1.invZ + dW2DX*p2.invZ
			dInvZDY := dW0DY*p0.invZ + dW1DY*p1.invZ + dW2DY*p2.invZ
			dADX := p0.world.Scale(dW0DX * p0.invZ).
				Add(p1.world.Scale(dW1DX * p1.invZ)).
				Add(p2.world.Scale(dW2DX * p2.invZ))
			dADY := p0.world.Scale(dW0DY * p0.invZ).
				Add(p1.world.Scale(dW1DY * p1.invZ)).
				Add(p2.world.Scale(dW2DY * p2.invZ))
			dWorldDX := dADX.Sub(world.Scale(dInvZDX)).Scale(1 / invZ)
			dWorldDY := dADY.Sub(world.Scale(dInvZDY)).Scale(1 / invZ)

			frame.Depth[index] = invZ
			frame.Color[index] = shade(ShadeContext{
				World:    world,
				Normal:   normal,
				DWorldDX: dWorldDX,
				DWorldDY: dWorldDY,
			})
		}
	}
}

func clampInt(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
