package main

import "math"

type Ray3 struct {
	O, D Vec3
}

func orientNormal3(d, n Vec3) Vec3 {
	if d.Dot(n) > 0 {
		return n.Mul(-1)
	}
	return n
}

func Refract3(d, normal Vec3, n1, n2 float64) (Vec3, bool) {
	n := orientNormal3(d, normal)
	cosI := clamp(-d.Dot(n), 0, 1)
	eta := n1 / n2
	k := 1 - eta*eta*(1-cosI*cosI)
	if k < 0 {
		return Vec3{}, false
	}
	return d.Mul(eta).Add(n.Mul(eta*cosI - math.Sqrt(k))).Norm(), true
}

func intersectSphere3(r Ray3, center Vec3, radius float64) (float64, Vec3) {
	oc := r.O.Sub(center)
	b := oc.Dot(r.D)
	disc := b*b - oc.Dot(oc) + radius*radius
	if disc < 0 {
		return -1, Vec3{}
	}
	s := math.Sqrt(disc)
	t := -b - s
	if t < 1e-5 {
		t = -b + s
	}
	if t < 1e-5 {
		return -1, Vec3{}
	}
	p := r.O.Add(r.D.Mul(t))
	return t, p.Sub(center).Norm()
}

// SphereBackground maps an orthographic image point through a dispersive glass
// sphere to the procedural background plane. The three display channels are
// evaluated at three physical wavelengths, producing real chromatic edges.
func SphereBackground(p Vec2, sphere Circle2) Vec3 {
	center := v3(sphere.C.X, sphere.C.Y, 0)
	backZ := -2.65
	channelLambda := [3]float64{645, 540, 455}
	var rgb Vec3

	for channel, lambda := range channelLambda {
		fallback := channelValue(hexLinear(0x9edff5), channel) * 0.22
		ray := Ray3{O: v3(p.X, p.Y, 4.2), D: v3(0, 0, -1)}
		frontT, normal := intersectSphere3(ray, center, sphere.R)
		if frontT <= 0 {
			return benchColor(p)
		}
		index := RefractiveIndex(lambda)
		hit := ray.O.Add(ray.D.Mul(frontT))
		inDir, ok := Refract3(ray.D, normal, 1, index)
		if !ok {
			inDir = ray.D
		}

		inside := Ray3{O: hit.Add(inDir.Mul(1e-4)), D: inDir}
		totalDistance := 0.0
		var exit, outDir Vec3
		escaped := false
		for bounce := 0; bounce < 4; bounce++ {
			insideT, outNormal := intersectSphere3(inside, center, sphere.R)
			if insideT <= 0 {
				break
			}
			totalDistance += insideT
			surface := inside.O.Add(inside.D.Mul(insideT))
			transmitted, ok := Refract3(inside.D, outNormal, index, 1)
			if ok {
				exit = surface
				outDir = transmitted
				escaped = true
				break
			}
			reflected := Reflect3(inside.D, outNormal)
			inside = Ray3{O: surface.Add(reflected.Mul(1e-4)), D: reflected}
		}

		value := fallback
		if escaped && outDir.Z < -1e-4 {
			tBack := (backZ - exit.Z) / outDir.Z
			if tBack > 0 {
				q := exit.Add(outDir.Mul(tBack))
				bg := benchColor(v2(q.X, q.Y))
				value = channelValue(bg, channel) * BeerTransmission(lambda, totalDistance)
			}
		}
		rgb = addChannel(rgb, channel, value)
	}
	return rgb
}

func Reflect3(d, n Vec3) Vec3 {
	n = orientNormal3(d, n)
	return d.Sub(n.Mul(2 * d.Dot(n))).Norm()
}

func channelValue(c Vec3, channel int) float64 {
	switch channel {
	case 0:
		return c.X
	case 1:
		return c.Y
	default:
		return c.Z
	}
}

func addChannel(c Vec3, channel int, value float64) Vec3 {
	switch channel {
	case 0:
		c.X = value
	case 1:
		c.Y = value
	default:
		c.Z = value
	}
	return c
}
