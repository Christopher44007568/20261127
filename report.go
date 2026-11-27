package main

import (
	"encoding/json"
	"math"
	"os"
	"sort"
)

type FloatStats struct {
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P95    float64 `json:"p95"`
	Max    float64 `json:"max"`
}

type Report struct {
	Vertices               int               `json:"vertices"`
	Triangles              int               `json:"triangles"`
	Wedges                 int               `json:"wedges"`
	IslandCount            int               `json:"island_count"`
	IslandTriangles        []int             `json:"island_triangles"`
	SeamEdges              int               `json:"seam_edges"`
	SeamLength             float64           `json:"seam_length"`
	ProtectedSeamEdges     int               `json:"protected_front_seam_edges"`
	VisibleFrontSeamEdges  int               `json:"visible_front_seam_edges"`
	VisibleFrontSeamLength float64           `json:"visible_front_seam_length"`
	FlippedFaces           int               `json:"flipped_faces"`
	RepairRounds           int               `json:"repair_rounds"`
	SolverIterations       []int             `json:"solver_iterations"`
	AreaRatio              FloatStats        `json:"area_ratio"`
	L2Stretch              FloatStats        `json:"l2_stretch"`
	LInfStretch            FloatStats        `json:"linf_stretch"`
	Heat                   FloatStats        `json:"heat"`
	MeanAO                 float64           `json:"mean_ao"`
	Checks                 Checks            `json:"checks"`
	Outputs                map[string]string `json:"outputs,omitempty"`
}

type Checks struct {
	NoFlippedFaces       bool `json:"no_flipped_faces"`
	ProtectedSeamFree    bool `json:"protected_seam_free"`
	VisibleFrontSeamFree bool `json:"visible_front_seam_free"`
	AtlasUVFinite        bool `json:"atlas_uv_finite"`
	P95L2Below2          bool `json:"p95_l2_below_2"`
}

func BuildReport(m *Mesh, sig *Signals, atlas *Atlas) Report {
	tp := BuildTopology(m)
	r := Report{
		Vertices:         len(m.Pos),
		Triangles:        len(m.Tri),
		Wedges:           len(atlas.WedgeUV),
		IslandCount:      len(atlas.Islands),
		RepairRounds:     atlas.Repairs,
		SolverIterations: atlas.Iterations,
		Outputs:          map[string]string{},
	}
	for _, isl := range atlas.Islands {
		r.IslandTriangles = append(r.IslandTriangles, len(isl))
	}

	aoSum := 0.0
	aoValues := sig.AO
	if len(sig.AORaw) == len(sig.AO) {
		aoValues = sig.AORaw
	}
	for _, v := range aoValues {
		aoSum += v
	}
	r.MeanAO = aoSum / float64(len(aoValues))

	var areaRatios, l2s, linfs, heats []float64
	for f, met := range atlas.Metrics {
		if met.Flipped {
			r.FlippedFaces++
		}
		areaRatios = append(areaRatios, met.AreaRatio)
		l2s = append(l2s, met.L2)
		linfs = append(linfs, met.LInf)
		heats = append(heats, met.Heat)
		_ = f
	}
	r.AreaRatio = StatsOf(areaRatios)
	r.L2Stretch = StatsOf(l2s)
	r.LInfStretch = StatsOf(linfs)
	r.Heat = StatsOf(heats)

	for eid, isSeam := range atlas.Seam {
		if !isSeam || eid >= len(tp.Edge) {
			continue
		}
		e := tp.Edge[eid]
		l := m.Pos[e.A].Sub(m.Pos[e.B]).Len()
		r.SeamEdges++
		r.SeamLength += l
		if eid < len(sig.Protected) && sig.Protected[eid] {
			r.ProtectedSeamEdges++
		}
		if eid < len(sig.FrontFacing) && sig.FrontFacing[eid] {
			r.VisibleFrontSeamEdges++
			r.VisibleFrontSeamLength += l
		}
	}

	r.Checks.AtlasUVFinite = true
	for _, uv := range atlas.WedgeUV {
		if math.IsNaN(uv.X) || math.IsNaN(uv.Y) || math.IsInf(uv.X, 0) || math.IsInf(uv.Y, 0) {
			r.Checks.AtlasUVFinite = false
			break
		}
	}
	r.Checks.NoFlippedFaces = r.FlippedFaces == 0
	r.Checks.ProtectedSeamFree = r.ProtectedSeamEdges == 0
	r.Checks.VisibleFrontSeamFree = r.VisibleFrontSeamEdges == 0
	r.Checks.P95L2Below2 = r.L2Stretch.P95 < 2
	return r
}

func StatsOf(xs []float64) FloatStats {
	if len(xs) == 0 {
		return FloatStats{}
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	sum := 0.0
	for _, x := range cp {
		sum += x
	}
	q := func(p float64) float64 {
		idx := int(math.Floor(p * float64(len(cp)-1)))
		return cp[clampInt(idx, 0, len(cp)-1)]
	}
	return FloatStats{
		Mean:   sum / float64(len(cp)),
		Median: q(0.5),
		P95:    q(0.95),
		Max:    cp[len(cp)-1],
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func SaveJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
