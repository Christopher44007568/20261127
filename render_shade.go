package main

import (
	"image/color"
	"math"
)

func shadePixel(mode shadeMode, uv, ddxUV, ddyUV Vec2, n Vec3, heat, ao float64) color.RGBA {
	key := V3(-0.38, 0.72, 0.88).Normalized()
	fill := V3(0.72, 0.18, 0.45).Normalized()
	diff := math.Max(0, n.Dot(key))
	fillDiff := math.Max(0, n.Dot(fill))
	aoLight := 1 - 0.38*clampFloat(ao, 0, 1)
	light := (0.40+0.62*diff+0.20*fillDiff)*aoLight + 0.10*(0.5+0.5*n.Y)

	switch mode {
	case shadeHeat:
		t := clampFloat(heat, 0, 1)
		base := heatRamp(t)
		l := 0.62 + 0.42*diff
		return scaleColor(base, l)
	case shadeAO:
		v := uint8(clampFloat(246*(0.70+0.30*(1-clampFloat(ao, 0, 1))), 0, 255))
		return rgba(v, v, v)
	default:
		base := checkerColorAA(uv, ddxUV, ddyUV)
		return scaleColor(base, clampFloat(light, 0.25, 1.22))
	}
}

func scaleColor(c color.RGBA, s float64) color.RGBA {
	return rgba(
		uint8(clampFloat(float64(c.R)*s, 0, 255)),
		uint8(clampFloat(float64(c.G)*s, 0, 255)),
		uint8(clampFloat(float64(c.B)*s, 0, 255)),
	)
}

func checkerColorAA(uv, ddx, ddy Vec2) color.RGBA {
	offsets := [4][2]float64{{-0.42, -0.42}, {0.42, -0.42}, {-0.42, 0.42}, {0.42, 0.42}}
	var r, g, b float64
	for _, o := range offsets {
		sampleUV := Vec2{
			X: uv.X + ddx.X*o[0] + ddy.X*o[1],
			Y: uv.Y + ddx.Y*o[0] + ddy.Y*o[1],
		}
		col := checkerColor(sampleUV)
		r += float64(col.R)
		g += float64(col.G)
		b += float64(col.B)
	}
	return rgba(uint8(r/4), uint8(g/4), uint8(b/4))
}

func checkerColor(uv Vec2) color.RGBA {
	const tiles = 17.0
	u := uv.X * tiles
	v := uv.Y * tiles
	iu := int(math.Floor(u))
	iv := int(math.Floor(v))
	fu := u - math.Floor(u)
	fv := v - math.Floor(v)
	col := rgba(14, 165, 233)
	if (iu+iv)&1 == 0 {
		col = rgba(248, 250, 252)
	}
	edge := math.Min(math.Min(fu, 1-fu), math.Min(fv, 1-fv))
	if edge < 0.035 {
		line := rgba(15, 23, 42)
		t := clampFloat((0.035-edge)/0.035*0.72, 0, 0.72)
		return mixColor(col, line, t)
	}
	return col
}

func mixColor(a, b color.RGBA, t float64) color.RGBA {
	return rgba(
		uint8(lerpFloat(float64(a.R), float64(b.R), t)),
		uint8(lerpFloat(float64(a.G), float64(b.G), t)),
		uint8(lerpFloat(float64(a.B), float64(b.B), t)),
	)
}
