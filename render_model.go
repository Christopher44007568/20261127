package main

import (
	"image/color"
	"math"
)

type Camera struct {
	Target Vec3
	Yaw    float64
	Pitch  float64
	Dist   float64
	FovY   float64
}

type projectedVertex struct {
	sx, sy float64
	z      float64
	invZ   float64
	uv     Vec2
	n      Vec3
	heat   float64
	ao     float64
}

func (cam Camera) basis() (Vec3, Vec3, Vec3, Vec3, float64) {
	cp := math.Cos(cam.Pitch)
	eye := cam.Target.Add(Vec3{
		X: math.Sin(cam.Yaw) * cp * cam.Dist,
		Y: math.Sin(cam.Pitch)*cam.Dist + cam.Target.Y*0,
		Z: math.Cos(cam.Yaw) * cp * cam.Dist,
	})
	eye.Y += cam.Target.Y
	forward := cam.Target.Sub(eye).Normalized()
	right := forward.Cross(Vec3{0, 1, 0}).Normalized()
	up := right.Cross(forward).Normalized()
	focal := 0.5 / math.Tan(cam.FovY*0.5*math.Pi/180)
	return eye, forward, right, up, focal
}

func (cam Camera) project(p Vec3, w, h int) (x, y, z float64, ok bool) {
	eye, fwd, right, up, focalNorm := cam.basis()
	rel := p.Sub(eye)
	z = rel.Dot(fwd)
	if z <= 0.05 {
		return 0, 0, z, false
	}
	focal := focalNorm * float64(h)
	x = float64(w)*0.5 + rel.Dot(right)/z*focal
	y = float64(h)*0.5 - rel.Dot(up)/z*focal
	return x, y, z, true
}

type shadeMode int

const (
	shadeChecker shadeMode = iota
	shadeHeat
	shadeAO
)

func RenderModelView(w, h int, m *Mesh, atlas *Atlas, sig *Signals, mode shadeMode, cam Camera) *Canvas {
	c := NewCanvas(w, h)
	c.Clear(rgba(15, 23, 42))
	blendVerticalBackground(c, rgba(30, 41, 59, 40), rgba(2, 6, 23, 0))
	for i := range c.Depth {
		c.Depth[i] = math.Inf(1)
	}

	for f, t := range m.Tri {
		ws := atlas.FaceWedge[f]
		verts := [3]projectedVertex{}
		ids := [3]int{t.A, t.B, t.C}
		okAll := true
		for k := 0; k < 3; k++ {
			p := m.Pos[ids[k]]
			sx, sy, z, ok := cam.project(p, w, h)
			if !ok {
				okAll = false
				break
			}
			invZ := 1 / z
			heat := 0.0
			ao := 0.0
			if f < len(atlas.Metrics) {
				heat = atlas.Metrics[f].Heat / heatDisplayScale
			}
			if sig != nil && ids[k] < len(sig.AO) {
				ao = sig.AO[ids[k]]
			}
			verts[k] = projectedVertex{
				sx: sx, sy: sy, z: z, invZ: invZ,
				uv:   atlas.WedgeUV[ws[k]],
				n:    m.Norm[ids[k]],
				heat: heat, ao: ao,
			}
		}
		if !okAll {
			continue
		}
		rasterProjectedTriangle(c, verts, mode)
	}
	drawModelSeams(c, m, atlas, cam)
	return c
}

func blendVerticalBackground(c *Canvas, top, bottom color.RGBA) {
	for y := 0; y < c.H; y++ {
		t := float64(y) / math.Max(1, float64(c.H-1))
		col := color.RGBA{
			R: uint8(lerpFloat(float64(top.R), float64(bottom.R), t)),
			G: uint8(lerpFloat(float64(top.G), float64(bottom.G), t)),
			B: uint8(lerpFloat(float64(top.B), float64(bottom.B), t)),
			A: uint8(lerpFloat(float64(top.A), float64(bottom.A), t)),
		}
		for x := 0; x < c.W; x++ {
			c.Blend(x, y, col, float64(col.A)/255)
		}
	}
}

func rasterProjectedTriangle(c *Canvas, v [3]projectedVertex, mode shadeMode) {
	minX := int(math.Floor(math.Min(v[0].sx, math.Min(v[1].sx, v[2].sx))))
	maxX := int(math.Ceil(math.Max(v[0].sx, math.Max(v[1].sx, v[2].sx))))
	minY := int(math.Floor(math.Min(v[0].sy, math.Min(v[1].sy, v[2].sy))))
	maxY := int(math.Ceil(math.Max(v[0].sy, math.Max(v[1].sy, v[2].sy))))
	minX = clampInt(minX, 0, c.W-1)
	maxX = clampInt(maxX, 0, c.W-1)
	minY = clampInt(minY, 0, c.H-1)
	maxY = clampInt(maxY, 0, c.H-1)
	area := (v[1].sx-v[0].sx)*(v[2].sy-v[0].sy) - (v[1].sy-v[0].sy)*(v[2].sx-v[0].sx)
	if math.Abs(area) < 1e-8 {
		return
	}
	ddxUV, ddyUV := uvScreenDerivatives(v)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			w0 := ((v[1].sx-px)*(v[2].sy-py) - (v[1].sy-py)*(v[2].sx-px)) / area
			w1 := ((v[2].sx-px)*(v[0].sy-py) - (v[2].sy-py)*(v[0].sx-px)) / area
			w2 := 1 - w0 - w1
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			winv := w0*v[0].invZ + w1*v[1].invZ + w2*v[2].invZ
			if winv <= 0 {
				continue
			}
			depth := 1 / winv
			idx := y*c.W + x
			if depth >= c.Depth[idx] {
				continue
			}
			uv := Vec2{
				X: (w0*v[0].uv.X*v[0].invZ + w1*v[1].uv.X*v[1].invZ + w2*v[2].uv.X*v[2].invZ) / winv,
				Y: (w0*v[0].uv.Y*v[0].invZ + w1*v[1].uv.Y*v[1].invZ + w2*v[2].uv.Y*v[2].invZ) / winv,
			}
			n := Vec3{
				X: (w0*v[0].n.X*v[0].invZ + w1*v[1].n.X*v[1].invZ + w2*v[2].n.X*v[2].invZ) / winv,
				Y: (w0*v[0].n.Y*v[0].invZ + w1*v[1].n.Y*v[1].invZ + w2*v[2].n.Y*v[2].invZ) / winv,
				Z: (w0*v[0].n.Z*v[0].invZ + w1*v[1].n.Z*v[1].invZ + w2*v[2].n.Z*v[2].invZ) / winv,
			}.Normalized()
			heat := (w0*v[0].heat*v[0].invZ + w1*v[1].heat*v[1].invZ + w2*v[2].heat*v[2].invZ) / winv
			ao := (w0*v[0].ao*v[0].invZ + w1*v[1].ao*v[1].invZ + w2*v[2].ao*v[2].invZ) / winv
			col := shadePixel(mode, uv, ddxUV, ddyUV, n, heat, ao)
			c.Depth[idx] = depth
			c.Img.SetRGBA(x, y, col)
		}
	}
}

func uvScreenDerivatives(v [3]projectedVertex) (Vec2, Vec2) {
	dx1 := v[1].sx - v[0].sx
	dy1 := v[1].sy - v[0].sy
	du1 := v[1].uv.X - v[0].uv.X
	dv1 := v[1].uv.Y - v[0].uv.Y
	dx2 := v[2].sx - v[0].sx
	dy2 := v[2].sy - v[0].sy
	du2 := v[2].uv.X - v[0].uv.X
	dv2 := v[2].uv.Y - v[0].uv.Y
	det := dx1*dy2 - dy1*dx2
	if math.Abs(det) < 1e-9 {
		return Vec2{}, Vec2{}
	}
	ddxU := (du1*dy2 - dy1*du2) / det
	ddyU := (dx1*du2 - du1*dx2) / det
	ddxV := (dv1*dy2 - dy1*dv2) / det
	ddyV := (dx1*dv2 - dv1*dx2) / det
	return Vec2{X: ddxU, Y: ddxV}, Vec2{X: ddyU, Y: ddyV}
}
