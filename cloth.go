package main

import "math"

const (
	clothHalfWidth  = 2.05
	clothHalfHeight = 1.50
	clothCorner     = 0.16
)

type ClothHit struct {
	Position Vec3
	EdgeDist float64
}

func clothHeight(x, y float64) float64 {
	phase := 2.35*x + 0.20*math.Sin(1.35*y) + 0.15
	return 0.235*math.Sin(phase) +
		0.060*math.Sin(5.1*x-0.42*y+1.15) +
		0.016*math.Sin(9.4*x+1.1*y+0.5)
}

func clothGradient(x, y float64) (float64, float64) {
	phase := 2.35*x + 0.20*math.Sin(1.35*y) + 0.15
	wave := math.Cos(phase)
	second := math.Cos(5.1*x - 0.42*y + 1.15)
	third := math.Cos(9.4*x + 1.1*y + 0.5)
	dx := 0.235*2.35*wave + 0.060*5.1*second + 0.016*9.4*third
	dy := 0.235*0.20*1.35*math.Cos(1.35*y)*wave -
		0.060*0.42*second + 0.016*1.1*third
	return dx, dy
}

func clothSDF(x, y float64) float64 {
	px := math.Abs(x) - clothHalfWidth + clothCorner
	py := math.Abs(y) - clothHalfHeight + clothCorner
	qx := math.Max(px, 0)
	qy := math.Max(py, 0)
	return math.Hypot(qx, qy) + math.Min(math.Max(px, py), 0) - clothCorner
}

func clothNormals(x, y, fiberAngle float64) (Vec3, Vec3, float64) {
	dx, dy := clothGradient(x, y)
	normal := V(-dx, -dy, 1).Normalize()
	tangentY := V(0, 1, dy).Normalize()
	bitangent := normal.Cross(tangentY).Normalize()
	angle := fiberAngle + 0.045*math.Sin(4.1*x+2.7*y+0.4)
	tangent := tangentY.Scale(math.Cos(angle)).Add(bitangent.Scale(math.Sin(angle))).Normalize()
	return normal, tangent, -clothSDF(x, y)
}

func intersectCloth(origin, direction Vec3) (ClothHit, bool) {
	if math.Abs(direction.Z) < 1e-8 {
		return ClothHit{}, false
	}
	tPlane := -origin.Z / direction.Z
	if tPlane < 0.2 {
		return ClothHit{}, false
	}
	start := math.Max(0.2, tPlane-0.9)
	end := tPlane + 0.9
	const steps = 72
	prevT := start
	if rayHeightGap(origin, direction, prevT) <= 0 {
		return ClothHit{}, false
	}
	for i := 1; i <= steps; i++ {
		t := start + (end-start)*float64(i)/steps
		if rayHeightGap(origin, direction, t) <= 0 {
			lo, hi := prevT, t
			for j := 0; j < 12; j++ {
				mid := 0.5 * (lo + hi)
				if rayHeightGap(origin, direction, mid) > 0 {
					lo = mid
				} else {
					hi = mid
				}
			}
			tHit := 0.5 * (lo + hi)
			p := origin.Add(direction.Scale(tHit))
			if clothSDF(p.X, p.Y) > 0.012 {
				return ClothHit{}, false
			}
			edge := -clothSDF(p.X, p.Y)
			return ClothHit{Position: p, EdgeDist: edge}, true
		}
		prevT = t
	}
	return ClothHit{}, false
}

func rayHeightGap(origin, direction Vec3, t float64) float64 {
	p := origin.Add(direction.Scale(t))
	return p.Z - clothHeight(p.X, p.Y)
}
