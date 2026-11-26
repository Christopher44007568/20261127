package main

import "math"

type Camera struct {
	Position Vec3
	Forward  Vec3
	Right    Vec3
	Up       Vec3
	Fov      float64
}

func makeCamera(position, target Vec3, fovDegrees float64) Camera {
	worldUp := V(0, 1, 0)
	forward := target.Sub(position).Normalize()
	right := forward.Cross(worldUp).Normalize()
	up := right.Cross(forward).Normalize()
	return Camera{
		Position: position,
		Forward:  forward,
		Right:    right,
		Up:       up,
		Fov:      fovDegrees * math.Pi / 180,
	}
}

func (c Camera) RayAt(x, y float64, width, height int) Vec3 {
	aspect := float64(width) / float64(height)
	ndcX := x/float64(width)*2 - 1
	ndcY := 1 - y/float64(height)*2
	tanHalf := math.Tan(c.Fov / 2)
	direction := c.Forward.
		Add(c.Right.Scale(ndcX * tanHalf * aspect)).
		Add(c.Up.Scale(ndcY * tanHalf))
	return direction.Normalize()
}

type View struct {
	Name       string
	FiberAngle float64
	Lights     []Light
	Backlit    bool
}

func velvetViews() []View {
	front := []Light{
		{Direction: V(-0.58, 0.72, 0.72).Normalize(), Radiance: V(3.00, 2.82, 2.62)},
		{Direction: V(0.82, -0.10, 0.88).Normalize(), Radiance: V(0.62, 0.68, 0.82)},
		{Direction: V(0.12, -0.78, 0.46).Normalize(), Radiance: V(0.22, 0.24, 0.29)},
	}
	back := []Light{
		{Direction: V(0.26, 0.55, -0.92).Normalize(), Radiance: V(3.10, 2.05, 1.56), Behind: true},
		{Direction: V(-0.70, 0.24, -0.78).Normalize(), Radiance: V(0.92, 0.52, 0.37), Behind: true},
		{Direction: V(0.08, 0.15, 0.98).Normalize(), Radiance: V(0.12, 0.13, 0.17)},
	}
	return []View{
		{Name: "front", FiberAngle: 0, Lights: front},
		{Name: "backlit", FiberAngle: 0, Lights: back, Backlit: true},
		{Name: "rotation", FiberAngle: 0.82, Lights: front},
	}
}

func studioBackground(u, v float64, backlit bool) Vec3 {
	cx := u - 0.50
	cy := v - 0.44
	halo := math.Exp(-(cx*cx/0.26 + cy*cy/0.38))
	if backlit {
		rimHalo := math.Exp(-(cx*cx/0.11 + cy*cy/0.17))
		return V(0.007, 0.008, 0.013).
			Add(V(0.014, 0.010, 0.009).Scale(halo)).
			Add(V(0.025, 0.010, 0.006).Scale(rimHalo))
	}
	return V(0.010, 0.011, 0.016).
		Add(V(0.032, 0.025, 0.020).Scale(halo)).
		Add(V(0.010, 0.008, 0.006).Scale(math.Exp(-(cx*cx/0.04 + cy*cy/0.035))))
}
