package flakepaint

import "math"

func rotateXYZ(p Vec3, rx, ry, rz float64) Vec3 {
	cy, sy := math.Cos(ry), math.Sin(ry)
	p = V(cy*p.X+sy*p.Z, p.Y, -sy*p.X+cy*p.Z)
	cx, sx := math.Cos(rx), math.Sin(rx)
	p = V(p.X, cx*p.Y-sx*p.Z, sx*p.Y+cx*p.Z)
	cz, sz := math.Cos(rz), math.Sin(rz)
	return V(cz*p.X-sz*p.Y, sz*p.X+cz*p.Y, p.Z)
}

func invRotateXYZ(p Vec3, rx, ry, rz float64) Vec3 {
	cz, sz := math.Cos(rz), math.Sin(rz)
	p = V(cz*p.X+sz*p.Y, -sz*p.X+cz*p.Y, p.Z)
	cx, sx := math.Cos(rx), math.Sin(rx)
	p = V(p.X, cx*p.Y+sx*p.Z, -sx*p.Y+cx*p.Z)
	cy, sy := math.Cos(ry), math.Sin(ry)
	return V(cy*p.X-sy*p.Z, p.Y, sy*p.X+cy*p.Z)
}

func sdRoundBox(p, b Vec3, r float64) float64 {
	q := V(math.Abs(p.X)-b.X+r, math.Abs(p.Y)-b.Y+r, math.Abs(p.Z)-b.Z+r)
	outside := q.Max0().Length()
	inside := math.Min(math.Max(q.X, math.Max(q.Y, q.Z)), 0)
	return outside + inside - r
}

type panelObject struct {
	center     Vec3
	rx, ry, rz float64
	half       Vec3
	radius     float64
}

func newPanelObject() panelObject {
	return panelObject{
		center: V(0, 0, 0),
		rx:     0.055,
		ry:     0.135,
		rz:     -0.032,
		half:   V(0.315, 0.187, 0.029),
		radius: 0.018,
	}
}

func (o panelObject) local(p Vec3) Vec3 {
	return invRotateXYZ(p.Sub(o.center), o.rx, o.ry, o.rz)
}

func (o panelObject) worldDir(p Vec3) Vec3 {
	return rotateXYZ(p, o.rx, o.ry, o.rz)
}

func (o panelObject) sdfLocal(p Vec3) float64 {
	curved := p
	curved.Z -= 0.018*(p.X/o.half.X)*(p.X/o.half.X) + 0.011*(p.Y/o.half.Y)*(p.Y/o.half.Y)
	return sdRoundBox(curved, o.half, o.radius)
}

func (o panelObject) hit(ray Ray) (bool, float64, Vec3) {
	ro := o.local(ray.Origin)
	rd := invRotateXYZ(ray.Dir, o.rx, o.ry, o.rz)
	t := 0.0
	for i := 0; i < 96; i++ {
		p := ro.Add(rd.Mul(t))
		d := o.sdfLocal(p)
		if d < 0.00018 {
			return true, t, o.localToWorldPoint(p)
		}
		t += math.Max(d*0.82, 0.00012)
		if t > 2.2 {
			break
		}
	}
	return false, 0, Vec3{}
}

func (o panelObject) localToWorldPoint(p Vec3) Vec3 {
	return o.center.Add(rotateXYZ(p, o.rx, o.ry, o.rz))
}

func (o panelObject) normal(localP Vec3) Vec3 {
	const e = 0.00022
	n := V(
		o.sdfLocal(localP.Add(V(e, 0, 0)))-o.sdfLocal(localP.Sub(V(e, 0, 0))),
		o.sdfLocal(localP.Add(V(0, e, 0)))-o.sdfLocal(localP.Sub(V(0, e, 0))),
		o.sdfLocal(localP.Add(V(0, 0, e)))-o.sdfLocal(localP.Sub(V(0, 0, e))),
	).Normalize()
	return o.worldDir(n)
}

func panelBackground(rd Vec3) Vec3 {
	d := rd.Normalize()
	up := smoothstep(-0.7, 0.9, d.Y)
	bg := V(0.0015, 0.0025, 0.0055).Lerp(V(0.010, 0.019, 0.031), up)
	bg = bg.Add(V(0.011, 0.015, 0.022).Mul(smoothstep(0.10, 0.85, d.Y)))

	// Out-of-focus vertical studio strips behind the sample.
	left := math.Exp(-0.5 * math.Pow((d.X+0.55)/0.045, 2))
	left *= smoothstep(-0.35, 0.10, d.Y) * (1 - smoothstep(0.75, 0.98, d.Y))
	right := math.Exp(-0.5 * math.Pow((d.X-0.63)/0.06, 2))
	right *= smoothstep(-0.50, -0.05, d.Y) * (1 - smoothstep(0.55, 0.90, d.Y))
	bg = bg.Add(V(0.08, 0.18, 0.28).Mul(left * 0.32))
	bg = bg.Add(V(0.16, 0.10, 0.055).Mul(right * 0.20))
	return bg
}

func nearPanelShader(cam Camera, paint Paint, obj panelObject, width, height int) SampleShader {
	return func(u, v, aspect, _ float64) Vec3 {
		ray := cam.Ray(u, v, aspect)
		ok, t, worldP := obj.hit(ray)
		if !ok {
			return panelBackground(ray.Dir)
		}

		localP := obj.local(worldP)
		geoN := obj.normal(localP)
		nLocal := invRotateXYZ(geoN, obj.rx, obj.ry, obj.rz)
		// Gentle panel crown: normals fan out from the center.
		nLocal = nLocal.Add(V(
			0.28*(localP.X/obj.half.X),
			0.18*(localP.Y/obj.half.Y),
			0,
		)).Normalize()
		crease := math.Exp(-0.5 * math.Pow((localP.Y+0.012)/0.008, 2))
		nLocal = nLocal.Add(V(0, 0.048*crease, 0)).Normalize()
		nWorld := obj.worldDir(nLocal)
		tangent := obj.worldDir(V(1, 0, 0))
		tangent = tangent.Sub(nWorld.Mul(tangent.Dot(nWorld))).Normalize()
		bitangent := nWorld.Cross(tangent).Normalize()
		frame := Frame{N: nWorld, T: tangent, B: bitangent}

		uv := V2(localP.X*1000, localP.Y*1000)
		foot := t * cam.PixelAngle(height) / math.Max(0.24, math.Abs(ray.Dir.Dot(geoN))) * 1000
		flake := paint.Flakes.Sample(uv, frame, foot)
		view := ray.Dir.Mul(-1)
		ndv := clamp(geoN.Dot(view), 0, 1)
		edge := 0.28 * smoothstep(0.25, 0.85, 1-ndv)
		edge += 0.16 * smoothstep(0.035, 0.11, math.Abs(localP.X)-obj.half.X)

		return paint.Shade(ShadeInput{
			Normal:    flake.N,
			GeoNormal: geoN,
			ViewDir:   view,
			Flake:     flake,
			EdgeLight: edge,
		})
	}
}

func NearPanelCamera() Camera {
	return Camera{
		Position: V(0.055, 0.105, 0.735),
		Target:   V(0, -0.005, 0),
		Up:       V(0, 1, 0),
		FovY:     35 * math.Pi / 180,
	}
}

func RenderNearPanel(w, h, samples int, seed uint64) *HDRBuffer {
	paint := DefaultPaint(seed)
	obj := newPanelObject()
	cam := NearPanelCamera()
	return RenderRows(w, h, samples, nearPanelShader(cam, paint, obj, w, h))
}
