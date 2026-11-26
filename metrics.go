package main

import "math"

type GeometryScore struct {
	ErrorStats
	SignedMean float64 `json:"signed_mean"`
}

func ScoreDemoGeometry(m *Mesh, scene *DemoScene, boundaryMask []bool) GeometryScore {
	var unsigned []float64
	signedSum := 0.0
	signedCount := 0
	for i, p := range m.P {
		if i < len(boundaryMask) && boundaryMask[i] {
			continue
		}
		if p.X < scene.XMin+0.06 || p.X > scene.XMax-0.06 ||
			p.Y < scene.YMin+0.06 || p.Y > scene.YMax-0.06 {
			continue
		}
		if scene.InHole(p.X, p.Y) {
			continue
		}
		z := demoHeight(p.X, p.Y)
		truth := V3{p.X, p.Y, z}
		n := demoNormal(p.X, p.Y)
		signed := Dot(p.Sub(truth), n)
		unsigned = append(unsigned, signed)
		signedSum += signed
		signedCount++
	}
	stats := GeometryScore{ErrorStats: statsFromValues(unsigned)}
	if signedCount > 0 {
		stats.SignedMean = signedSum / float64(signedCount)
	}
	return stats
}

type HoleScore struct {
	Index            int     `json:"index"`
	Radius           float64 `json:"radius"`
	CentroidHits     int     `json:"centroid_hits"`
	CoreTriangleHits int     `json:"core_triangle_hits"`
	DeepestInset     float64 `json:"deepest_inset"`
	OpenCore         bool    `json:"open_core"`
}

func ScoreHoles(m *Mesh, holes []Hole) []HoleScore {
	out := make([]HoleScore, 0, len(holes))
	for hi, h := range holes {
		score := HoleScore{Index: hi, Radius: h.R, DeepestInset: 0, OpenCore: true}
		minDist := math.Inf(1)
		for _, t := range m.Tri {
			c := m.P[t[0]].Add(m.P[t[1]]).Add(m.P[t[2]]).Mul(1.0 / 3.0)
			dx := c.X - h.X
			dy := c.Y - h.Y
			d := math.Hypot(dx, dy)
			if d < minDist {
				minDist = d
			}
			if h.Inside(c.X, c.Y) {
				score.CentroidHits++
			}
			if d < h.R*0.55 {
				score.CoreTriangleHits++
				score.OpenCore = false
			}
		}
		if !math.IsInf(minDist, 1) && minDist < h.R {
			score.DeepestInset = h.R - minDist
		}
		out = append(out, score)
	}
	return out
}

func MeshArea(m *Mesh) float64 {
	area := 0.0
	for _, t := range m.Tri {
		a := m.P[t[0]]
		b := m.P[t[1]]
		c := m.P[t[2]]
		area += 0.5 * Norm(Cross(b.Sub(a), c.Sub(a)))
	}
	return area
}
