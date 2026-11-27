package main

import (
	"fmt"
	"math"
)

type FaceMetric struct {
	Area3D    float64
	AreaUV    float64
	AreaRatio float64
	SMin      float64
	SMax      float64
	L2        float64
	LInf      float64
	Heat      float64
	Flipped   bool
}

type Atlas struct {
	FaceWedge  [][3]int
	WedgeUV    []Vec2
	MetricUV   []Vec2
	Islands    [][]int
	Seam       []bool
	Metrics    []FaceMetric
	Repairs    int
	Iterations []int
}

type UnwrapConfig struct {
	AOSamples  int
	MaxRepairs int
	SeedSeams  bool
}

func DefaultUnwrapConfig() UnwrapConfig {
	return UnwrapConfig{AOSamples: 18, MaxRepairs: 18, SeedSeams: true}
}

func Unwrap(m *Mesh, cfg UnwrapConfig) (*Atlas, *Signals, error) {
	tp := BuildTopology(m)
	sig := ComputeSignals(m, tp, cfg.AOSamples)
	seam := make([]bool, len(tp.Edge))
	if cfg.SeedSeams {
		plan := PlanSeams(m, tp, sig)
		copy(seam, plan.EdgeSeam)
	}

	atlas := &Atlas{Seam: seam}
	lastFlipped := -1
	for round := 0; ; round++ {
		islands := tp.Islands(seam)
		faceWedge, wedgeVertex := BuildWedges(tp, seam)
		uv := make([]Vec2, len(wedgeVertex))
		var iterations []int
		for _, faces := range islands {
			res, err := SolveIslandLSCM(m, faceWedge, wedgeVertex, faces)
			if err != nil {
				return nil, sig, err
			}
			iterations = append(iterations, res.Iterations)
			for _, f := range faces {
				for _, w := range faceWedge[f] {
					uv[w] = res.UV[w]
				}
			}
		}

		flipped := flippedFaces(m, faceWedge, uv)
		atlas.Islands = islands
		atlas.FaceWedge = faceWedge
		atlas.WedgeUV = uv
		atlas.Iterations = iterations
		if len(flipped) == 0 {
			break
		}
		if round >= cfg.MaxRepairs {
			return nil, sig, fmt.Errorf("LSCM still has %d flipped faces after %d repair rounds", len(flipped), round)
		}
		added := addRepairSeams(tp, sig, seam, flipped)
		if added == 0 {
			return nil, sig, fmt.Errorf("cannot repair %d flipped faces without cutting protected front edges", len(flipped))
		}
		if len(flipped) == lastFlipped && round > 4 {
			// The edge-choice heuristic can circle; broaden the repair cuts.
			addAllFlipEdges(tp, sig, seam, flipped)
		}
		lastFlipped = len(flipped)
		atlas.Repairs++
	}

	rawArea := 0.0
	for f := range m.Tri {
		ws := atlas.FaceWedge[f]
		rawArea += uvTriangleArea(atlas.WedgeUV[ws[0]], atlas.WedgeUV[ws[1]], atlas.WedgeUV[ws[2]])
	}
	area3D := m.TotalArea()
	metricScale := 1.0
	if rawArea > 1e-10 {
		metricScale = math.Sqrt(area3D / rawArea)
	}
	atlas.MetricUV = make([]Vec2, len(atlas.WedgeUV))
	for i, uv := range atlas.WedgeUV {
		atlas.MetricUV[i] = uv.Mul(metricScale)
	}
	atlas.Metrics = evaluateMetrics(m, atlas.FaceWedge, atlas.MetricUV)
	PackAtlas(atlas)
	return atlas, sig, nil
}

func uvTriangleArea(a, b, c Vec2) float64 {
	return 0.5 * (b.Sub(a).X*(c.Sub(a).Y) - b.Sub(a).Y*(c.Sub(a).X))
}

func flippedFaces(m *Mesh, fw [][3]int, uv []Vec2) []int {
	var out []int
	for f := range m.Tri {
		ws := fw[f]
		if uvTriangleArea(uv[ws[0]], uv[ws[1]], uv[ws[2]]) < -1e-9 {
			out = append(out, f)
		}
	}
	return out
}

func addRepairSeams(tp *Topology, sig *Signals, seam []bool, bad []int) int {
	added := 0
	badSet := map[int]bool{}
	for _, f := range bad {
		badSet[f] = true
	}
	for _, f := range bad {
		best := -1
		bestCost := math.Inf(1)
		for _, eid := range tp.FaceEdge[f] {
			if seam[eid] || sig.Protected[eid] {
				continue
			}
			cost := sig.EdgeCost[eid]
			for _, nf := range tp.EdgeFaces[eid] {
				if nf != f && !badSet[nf] {
					cost -= 0.7
				}
			}
			if cost < bestCost {
				bestCost = cost
				best = eid
			}
		}
		if best >= 0 && !seam[best] {
			seam[best] = true
			added++
		}
	}
	return added
}

func addAllFlipEdges(tp *Topology, sig *Signals, seam []bool, bad []int) int {
	added := 0
	for _, f := range bad {
		for _, eid := range tp.FaceEdge[f] {
			if !seam[eid] && !sig.Protected[eid] {
				seam[eid] = true
				added++
			}
		}
	}
	return added
}

func evaluateMetrics(m *Mesh, fw [][3]int, uv []Vec2) []FaceMetric {
	out := make([]FaceMetric, len(m.Tri))
	for f, t := range m.Tri {
		p0 := m.Pos[t.A]
		p1 := m.Pos[t.B]
		p2 := m.Pos[t.C]
		e1 := p1.Sub(p0)
		e2 := p2.Sub(p0)
		q0 := uv[fw[f][0]]
		q1 := uv[fw[f][1]]
		q2 := uv[fw[f][2]]
		d1 := q1.Sub(q0)
		d2 := q2.Sub(q0)
		area3 := 0.5 * e1.Cross(e2).Len()
		signedUV2 := d1.X*d2.Y - d1.Y*d2.X
		areaUV := 0.5 * math.Abs(signedUV2)
		ratio := 1.0
		if area3 > 1e-12 {
			ratio = areaUV / area3
		}
		a := e1.Dot(e1)
		b := e1.Dot(e2)
		c := e2.Dot(e2)
		aa := d1.Dot(d1)
		bb := d1.Dot(d2)
		cc := d2.Dot(d2)
		det3 := math.Max(1e-14, a*c-b*b)
		detUV := math.Max(0, aa*cc-bb*bb)
		detRatio := detUV / det3
		trace := (a*cc + c*aa - 2*b*bb) / det3
		disc := math.Max(0, trace*trace-4*detRatio)
		lambdaHi := 0.5 * (trace + math.Sqrt(disc))
		lambdaLo := math.Max(0, 0.5*(trace-math.Sqrt(disc)))
		sMax := math.Sqrt(math.Max(0, lambdaHi))
		sMin := math.Sqrt(lambdaLo)
		l2 := math.Sqrt(0.5 * (lambdaHi + lambdaLo))
		lInf := sMax
		if sMin > 1e-8 && 1/sMin > lInf {
			lInf = 1 / sMin
		}
		areaLog := math.Abs(math.Log(math.Max(ratio, 1e-6)))
		angleLog := math.Log(math.Max(1, sMax/math.Max(sMin, 1e-6)))
		out[f] = FaceMetric{
			Area3D: area3, AreaUV: areaUV, AreaRatio: ratio,
			SMin: sMin, SMax: sMax, L2: l2, LInf: lInf,
			Heat:    clampFloat(0.65*areaLog+0.55*angleLog, 0, 4),
			Flipped: signedUV2 < -1e-9,
		}
	}
	return out
}
