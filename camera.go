package main

import "math"

type Camera struct {
	Origin     Vec3
	Target     Vec3
	Up         Vec3
	FovY       float64
	Width      int
	Height     int
	right      Vec3
	up         Vec3
	forward    Vec3
	viewHeight float64
	viewWidth  float64
}

func newCamera(origin, target Vec3, fovY float64, w, h int) *Camera {
	c := &Camera{
		Origin: origin,
		Target: target,
		Up:     Vec3{0, 1, 0},
		FovY:   fovY,
		Width:  w,
		Height: h,
	}
	c.forward = target.Sub(origin).Normalize()
	c.right = cross(c.forward, c.Up).Normalize()
	c.up = cross(c.right, c.forward).Normalize()
	c.viewHeight = 2 * math.Tan(fovY*0.5)
	c.viewWidth = c.viewHeight * float64(w) / float64(h)
	return c
}

func (c *Camera) ray(nx, ny float64) Ray {
	// nx, ny in [-1, 1].
	dir := c.forward.
		Add(c.right.Scale(nx * c.viewWidth * 0.5)).
		Add(c.up.Scale(ny * c.viewHeight * 0.5)).
		Normalize()
	return Ray{Orig: c.Origin, Dir: dir}
}
