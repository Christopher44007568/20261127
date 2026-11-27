package main

import (
	"image"
	"log"
)

type scaleSpec struct {
	factor int
	view   string
	radius int
	tau    float64
}

func scaleSpecs() []scaleSpec {
	return []scaleSpec{
		{1, "close", 10, 26},
		{4, "mid", 5, 20},
		{16, "far", 2, 14},
	}
}

func runScales(orig, processed *image.RGBA, rep *report) (heatsOrig, heatsProc []*HeatMap) {
	for _, sc := range scaleSpecs() {
		oi, pi := orig, processed
		px, py := TileW, TileH
		if sc.factor > 1 {
			oi = BoxDownscale(orig, sc.factor)
			pi = BoxDownscale(processed, sc.factor)
			px = TileW / sc.factor
			py = TileH / sc.factor
		}
		ho := RepetitionHeat(oi, px, py, sc.radius, sc.tau)
		hp := RepetitionHeat(pi, px, py, sc.radius, sc.tau)
		heatsOrig = append(heatsOrig, ho)
		heatsProc = append(heatsProc, hp)
		if sc.factor == 1 {
			mustSave(outPath("heatmap_original.png"), ho.Colorize())
			mustSave(outPath("heatmap_processed.png"), hp.Colorize())
		}
		cut := 0.0
		if ho.MeanScore > 0 {
			cut = (1 - hp.MeanScore/ho.MeanScore) * 100
		}
		rep.Scales = append(rep.Scales, scaleReport{
			Downscale: sc.factor, View: sc.view, PeriodX: px, PeriodY: py,
			Radius: sc.radius, Tau: sc.tau,
			OrigMeanDelta: ho.MeanDelta, ProcMeanDelta: hp.MeanDelta,
			OrigMeanRepetition: ho.MeanScore, ProcMeanRepetition: hp.MeanScore,
			RepetitionCutPct: cut,
		})
		log.Printf("scale x%-2d %-5s orig rep=%.3f (d=%5.1f)  proc rep=%.3f (d=%5.1f)  cut=%.0f%%",
			sc.factor, sc.view, ho.MeanScore, ho.MeanDelta,
			hp.MeanScore, hp.MeanDelta, cut)
	}
	return heatsOrig, heatsProc
}

func mustSave(path string, img image.Image) {
	if err := savePNG(path, img); err != nil {
		log.Fatal(err)
	}
}
