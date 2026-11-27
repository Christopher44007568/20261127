package flakepaint

import "math"

type sphereObject struct {
	Center Vec3
	Radius float64
}

func (s sphereObject) hit(ray Ray) (bool, float64, Vec3) {
	oc := ray.Origin.Sub(s.Center)
	b := oc.Dot(ray.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	disc := b*b - c
	if disc < 0 {
		return false, 0, Vec3{}
	}
	root := math.Sqrt(disc)
	t := -b - root
	if t < 1e-5 {
		t = -b + root
	}
	if t < 1e-5 {
		return false, 0, Vec3{}
	}
	return true, t, ray.Origin.Add(ray.Dir.Mul(t))
}

func farBackground(u, v float64, rd Vec3) Vec3 {
	d := rd.Normalize()
	up := smoothstep(-0.9, 0.9, d.Y)
	bg := V(0.0010, 0.0018, 0.0032).Lerp(V(0.006, 0.013, 0.024), up)
	bg = bg.Add(V(0.008, 0.013, 0.022).Mul(smoothstep(0.2, 1.0, d.Y)))

	dx := (u - 0.50) * 1.55
	dy := (v - 0.49) * 1.10
	halo := math.Exp(-(dx*dx + dy*dy) * 2.4)
	bg = bg.Add(V(0.025, 0.047, 0.073).Mul(halo))

	// Very faint floor marker keeps the calibration sphere grounded.
	horizon := math.Exp(-0.5 * math.Pow((v-0.825)/0.004, 2))
	bg = bg.Add(V(0.016, 0.023, 0.030).Mul(horizon))
	return bg
}

func farSphereShader(cam Camera, paint Paint, sphere sphereObject, width, height int) SampleShader {
	return func(u, v, aspect, _ float64) Vec3 {
		ray := cam.Ray(u, v, aspect)
		ok, t, p := sphere.hit(ray)
		if !ok {
			return farBackground(u, v, ray.Dir)
		}
		geoN := p.Sub(sphere.Center).Normalize()
		if geoN.Dot(ray.Dir) > 0 {
			geoN = geoN.Mul(-1)
		}
		frame := frameFromNormal(geoN)
		foot := t * cam.PixelAngle(height) / math.Max(0.32, math.Abs(ray.Dir.Dot(geoN))) * 1000

		// A stable spherical UV for the flake statistics. At this distance the
		// footprint covers many cells and Sample returns an aggregate response.
		uv := V2(math.Atan2(p.Z, p.X)*sphere.Radius*1000, math.Asin(clamp(p.Y/sphere.Radius, -1, 1))*sphere.Radius*1000)
		flake := paint.Flakes.Sample(uv, frame, foot)
		view := ray.Dir.Mul(-1)
		ndv := clamp(geoN.Dot(view), 0, 1)
		rim := 0.38 * math.Pow(1-ndv, 2.2)
		return paint.Shade(ShadeInput{
			Normal:    flake.N,
			GeoNormal: geoN,
			ViewDir:   view,
			Flake:     flake,
			EdgeLight: rim,
			Aggregate: true,
		})
	}
}

func FarSphereCamera() Camera {
	return Camera{
		Position: V(0, 0.24, 20),
		Target:   V(0, -0.01, 0),
		Up:       V(0, 1, 0),
		FovY:     8.2 * math.Pi / 180,
	}
}

func FarSphereObject() sphereObject {
	return sphereObject{Center: V(0, 0, 0), Radius: 1.30}
}

func RenderFarSphere(w, h, samples int, seed uint64) *HDRBuffer {
	paint := DefaultPaint(seed)
	sphere := FarSphereObject()
	cam := FarSphereCamera()
	return RenderRows(w, h, samples, farSphereShader(cam, paint, sphere, w, h))
}
