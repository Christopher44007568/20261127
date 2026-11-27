package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

func main() {
	out := flag.String("out", "outputs", "directory for PNG/JSON artifacts")
	seed := flag.Int64("seed", 20261127, "deterministic random seed")
	flag.Parse()
	outputDir = *out
	if err := run(uint64(*seed)); err != nil {
		log.Fatal(err)
	}
}

type scaleReport struct {
	Downscale          int     `json:"downscale"`
	View               string  `json:"view"`
	PeriodX            int     `json:"period_x"`
	PeriodY            int     `json:"period_y"`
	Radius             int     `json:"radius"`
	Tau                float64 `json:"tau"`
	OrigMeanDelta      float64 `json:"orig_mean_delta"`
	ProcMeanDelta      float64 `json:"proc_mean_delta"`
	OrigMeanRepetition float64 `json:"orig_mean_repetition"`
	ProcMeanRepetition float64 `json:"proc_mean_repetition"`
	RepetitionCutPct   float64 `json:"repetition_cut_pct"`
}

type report struct {
	Seed   uint64                   `json:"seed"`
	Scales []scaleReport            `json:"scales"`
	Joints map[string]JointContrast `json:"joint_contrast"`
	Seams  map[string][3]float64    `json:"boundary_continuity"`
}

func run(seed uint64) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	tile := BakeTile(seed ^ 0xB41C)
	mustSave(outPath("brick_tile.png"), tile)

	orig := TileRepeat(tile, OutW, OutH)
	mustSave(outPath("01_original_tiling.png"), orig)

	params := DefaultParams()
	params.Seed = seed
	processed := ProcessWall(tile, OutW, OutH, params)
	mustSave(outPath("02_processed_result.png"), processed)

	naive := NaiveRotatedTiling(tile, OutW, OutH, seed^77)
	mustSave(outPath("06_naive_rotation_bad.png"), naive)

	rep := report{Seed: seed, Joints: map[string]JointContrast{}, Seams: map[string][3]float64{}}
	heatsO, heatsP := runScales(orig, processed, &rep)
	mustSave(outPath("03_repetition_heatmap.png"),
		BuildHeatSheet(heatsO[0], heatsP[0], rep.Scales[0]))
	mustSave(outPath("04_distant_view.png"),
		BuildDistantSheet(orig, processed, rep.Scales[2]))

	rep.Joints["original"] = MeasureJointContrast(orig)
	rep.Joints["processed"] = MeasureJointContrast(processed)
	rep.Joints["naive_rotation"] = MeasureJointContrast(naive)
	seamOrig := seamTriple(BoundaryContinuity(orig, TileW))
	seamProc := seamTriple(BoundaryContinuity(processed, TileW))
	seamNaive := seamTriple(BoundaryContinuity(naive, TileW))
	rep.Seams["original"] = seamOrig
	rep.Seams["processed"] = seamProc
	rep.Seams["naive_rotation"] = seamNaive

	mustSave(outPath("05_joint_check.png"),
		BuildJointSheet(orig, processed, rep.Joints["original"], rep.Joints["processed"]))

	logSummary(&rep)
	data, _ := json.MarshalIndent(rep, "", "  ")
	return os.WriteFile(outPath("metrics.json"), data, 0o644)
}

func seamTriple(seam, control, ratio float64) [3]float64 {
	return [3]float64{seam, control, ratio}
}

func logSummary(rep *report) {
	for _, s := range rep.Scales {
		log.Printf("%-5s x%-2d repetition %.3f -> %.3f (-%.0f%%)",
			s.View, s.Downscale, s.OrigMeanRepetition, s.ProcMeanRepetition, s.RepetitionCutPct)
	}
	j := rep.Joints
	log.Printf("joint strong coverage orig=%.2f proc=%.2f naive=%.2f",
		j["original"].StrongFrac, j["processed"].StrongFrac, j["naive_rotation"].StrongFrac)
	log.Printf("seam ratio orig=%.2f proc=%.2f naive=%.2f",
		rep.Seams["original"][2], rep.Seams["processed"][2], rep.Seams["naive_rotation"][2])
}
