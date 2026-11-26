package main

import "math"

type Camera struct {
	Name             string
	Eye              Vec3
	Forward          Vec3
	Right            Vec3
	Up               Vec3
	FX, FY           float64
	CX, CY           float64
	Width, Height    int
	Near, Far        float64
	Azimuth          float64
	PoseErrorMeters  float64
	PoseErrorRadians float64
}

func LookAtCamera(name string, eye, target Vec3, fovDeg float64, width, height int) Camera {
	forward := target.Sub(eye).Norm()
	worldUp := V3(0, 0, 1)
	right := forward.Cross(worldUp).Norm()
	if right.Len() < 1e-6 {
		right = V3(1, 0, 0)
	}
	up := right.Cross(forward).Norm()
	fy := 0.5 * float64(height) / math.Tan(degToRad(fovDeg)*0.5)
	fx := fy
	return Camera{
		Name:    name,
		Eye:     eye,
		Forward: forward,
		Right:   right,
		Up:      up,
		FX:      fx,
		FY:      fy,
		CX:      float64(width) / 2,
		CY:      float64(height) / 2,
		Width:   width,
		Height:  height,
		Near:    0.05,
		Far:     8.0,
	}
}

func degToRad(v float64) float64 { return v * math.Pi / 180 }

func radToDeg(v float64) float64 { return v * 180 / math.Pi }

func OrbitCamera(name string, center Vec3, radius, azimuthDeg, height, fovDeg float64, w, h int) Camera {
	az := degToRad(azimuthDeg)
	eye := V3(
		center.X+radius*math.Sin(az),
		center.Y-radius*math.Cos(az),
		center.Z+height,
	)
	c := LookAtCamera(name, eye, center, fovDeg, w, h)
	c.Azimuth = azimuthDeg
	return c
}

func rotateAroundAxis(v, axis Vec3, angle float64) Vec3 {
	if math.Abs(angle) < 1e-12 {
		return v
	}
	c := math.Cos(angle)
	s := math.Sin(angle)
	return v.Mul(c).Add(axis.Cross(v).Mul(s)).Add(axis.Mul(axis.Dot(v) * (1 - c)))
}

func (c Camera) WithPoseError(eyeOffset Vec3, yaw, pitch float64) Camera {
	out := c
	out.Eye = c.Eye.Add(eyeOffset)
	worldZ := V3(0, 0, 1)
	out.Forward = rotateAroundAxis(c.Forward, worldZ, yaw)
	out.Right = rotateAroundAxis(c.Right, worldZ, yaw)
	out.Up = rotateAroundAxis(c.Up, worldZ, yaw)
	out.Forward = rotateAroundAxis(out.Forward, out.Right, pitch).Norm()
	out.Up = out.Right.Cross(out.Forward).Norm()
	out.PoseErrorMeters = eyeOffset.Len()
	out.PoseErrorRadians = math.Hypot(yaw, pitch)
	return out
}

func (c Camera) RayAtPixel(ix, iy int) Ray {
	u := float64(ix) + 0.5
	v := float64(iy) + 0.5
	x := (u - c.CX) / c.FX
	y := -(v - c.CY) / c.FY
	dir := c.Forward.Add(c.Right.Mul(x)).Add(c.Up.Mul(y)).Norm()
	return Ray{Origin: c.Eye, Dir: dir}
}

func (c Camera) Project(p Vec3) (u, v, z float64, visible bool) {
	d := p.Sub(c.Eye)
	z = d.Dot(c.Forward)
	if z <= c.Near || z >= c.Far {
		return 0, 0, z, false
	}
	x := d.Dot(c.Right)
	y := d.Dot(c.Up)
	u = c.CX + c.FX*x/z
	v = c.CY - c.FY*y/z
	if u < 0 || u >= float64(c.Width) || v < 0 || v >= float64(c.Height) {
		return u, v, z, false
	}
	return u, v, z, true
}

func (c Camera) UnprojectDepth(ix, iy int, z float64) Vec3 {
	u := float64(ix) + 0.5
	v := float64(iy) + 0.5
	x := (u - c.CX) * z / c.FX
	y := -(v - c.CY) * z / c.FY
	return c.Eye.Add(c.Right.Mul(x)).Add(c.Up.Mul(y)).Add(c.Forward.Mul(z))
}
