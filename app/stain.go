package app

import (
	"fmt"
	"path/filepath"

	"coverlab/cover"
	"coverlab/scenes"
	"coverlab/viz"
)

func renderStain(outDir string) (map[string]cover.StainReport, GuardReport, map[string]bool, error) {
	scene := scenes.Segregate()
	redIdx := -1
	for i, l := range scene.Layers {
		if l.Name == "red" {
			redIdx = i
		}
	}
	refRedCov := cover.AnalyticLayer(scene.Layers[redIdx], scene.W, scene.H)
	noRedScene := scene
	noRedScene.Layers = append([]cover.Layer{}, scene.Layers...)
	noRedScene.Layers = append(noRedScene.Layers[:redIdx], noRedScene.Layers[redIdx+1:]...)

	type pair struct {
		full    []cover.RGB
		noLayer []cover.RGB
	}
	pairs := map[string]pair{}
	guards := map[string]cover.RGB{}

	for _, m := range []string{mAnalytic, mSS2, mSS3} {
		frgb, _, _ := cover.RenderMethod(scene, m, viz.White)
		nrgb, _, _ := cover.RenderMethod(noRedScene, m, viz.White)
		pairs[m] = pair{frgb, nrgb}
		guards[m] = frgb[scenes.GuardPixelY*scene.W+scenes.GuardPixelX]
	}
	fb, _ := cover.RenderBlur(scene, BlurSigma, viz.White, -1)
	nb, _ := cover.RenderBlur(scene, BlurSigma, viz.White, redIdx)
	pairs[mBlur] = pair{fb, nb}
	guards[mBlur] = fb[scenes.GuardPixelY*scene.W+scenes.GuardPixelX]

	rep := GuardReport{
		X: scenes.GuardPixelX, Y: scenes.GuardPixelY,
		Analytic: guards[mAnalytic], SS2: guards[mSS2], SS3: guards[mSS3], Blur: guards[mBlur],
	}

	stains := map[string]cover.StainReport{}
	stainBufs := map[string][]float64{}
	vmax := 0.0
	for _, m := range []string{mAnalytic, mSS2, mSS3, mBlur} {
		p := pairs[m]
		st := cover.MeasureStain("red", refRedCov, p.full, p.noLayer)
		stains[m] = st
		buf := make([]float64, scene.W*scene.H)
		for i := range buf {
			a, b := p.full[i], p.noLayer[i]
			d := absf(a.R - b.R)
			d = maxf(d, absf(a.G-b.G))
			d = maxf(d, absf(a.B-b.B))
			buf[i] = d / 255
		}
		stainBufs[m] = buf
		if bufMax := st.MaxStain255 / 255; bufMax > vmax {
			vmax = bufMax
		}
	}

	for _, m := range []string{mAnalytic, mSS2, mSS3, mBlur} {
		st := stains[m]
		title := fmt.Sprintf("RED CROSSTALK / %s", methodLabel(m))
		statsLine := fmt.Sprintf("MAX STAIN %.1f/255   MEAN %.2f", st.MaxStain255, st.MeanStain)
		panel := viz.StainPanel(scene, stainBufs[m], refRedCov, scene.Layers[redIdx].Poly, vmax, title, statsLine)
		if err := viz.SavePNG(filepath.Join(outDir, "stain", "segregate__red_"+m+".png"), panel); err != nil {
			return stains, rep, nil, err
		}
	}

	checks := map[string]bool{
		"analytic_guard_pixel_white": nearWhite(rep.Analytic),
		"ss2_guard_pixel_white":      nearWhite(rep.SS2),
		"ss3_guard_pixel_white":      nearWhite(rep.SS3),
		"analytic_zero_stain":        stains[mAnalytic].MaxStain255 == 0,
		"ss2_zero_stain":             stains[mSS2].MaxStain255 == 0,
		"ss3_zero_stain":             stains[mSS3].MaxStain255 == 0,
		"blur_baseline_leaks":        stains[mBlur].MaxStain255 > 1.0,
	}
	return stains, rep, checks, nil
}

func nearWhite(c cover.RGB) bool {
	return c.R >= 254.5 && c.G >= 254.5 && c.B >= 254.5
}
