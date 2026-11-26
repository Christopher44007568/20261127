package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"coverlab/cover"
	"coverlab/viz"
)

type rendered struct {
	rgb   []cover.RGB
	alpha []float64
	covs  [][]float64
}

func renderAllMethods(scene cover.Scene) map[string]rendered {
	out := map[string]rendered{}
	for _, m := range []string{mAnalytic, mSS2, mSS3} {
		rgb, alpha, covs := cover.RenderMethod(scene, m, viz.White)
		out[m] = rendered{rgb, alpha, covs}
	}
	brgb, balpha := cover.RenderBlur(scene, BlurSigma, viz.White, -1)
	out[mBlur] = rendered{rgb: brgb, alpha: balpha}
	return out
}

func renderScene(outDir string, scene cover.Scene) (SceneSummary, error) {
	res := renderAllMethods(scene)
	ana := res[mAnalytic]

	st := SceneSummary{
		Name:    scene.Name,
		Width:   scene.W,
		Height:  scene.H,
		Methods: map[string]MethodStat{},
	}

	// Coverage maps and edge close-ups for every method.
	for _, m := range []string{mAnalytic, mSS2, mSS3, mBlur} {
		r := res[m]
		name := strings.ToUpper(scene.Name)
		covTitle := fmt.Sprintf("%s / %s COVERAGE", name, methodLabel(m))
		if err := viz.SavePNG(filepath.Join(outDir, "coverage", scene.Name+"__"+m+".png"),
			viz.CoveragePanel(scene, r.alpha, covTitle)); err != nil {
			return st, err
		}
		closeTitle := fmt.Sprintf("%s / %s EDGE CLOSEUP", name, methodLabel(m))
		if err := viz.SavePNG(filepath.Join(outDir, "closeup", scene.Name+"__"+m+".png"),
			viz.CloseupPanel(scene, r.rgb, closeTitle)); err != nil {
			return st, err
		}
	}

	// Error stats vs analytic effective alpha.
	ss2Stats := cover.DiffStats(ana.alpha, res[mSS2].alpha)
	ss3Stats := cover.DiffStats(ana.alpha, res[mSS3].alpha)
	blurStats := cover.DiffStats(ana.alpha, res[mBlur].alpha)
	st.Methods[mSS2] = MethodStat{mSS2, ss2Stats}
	st.Methods[mSS3] = MethodStat{mSS3, ss3Stats}
	st.Methods[mBlur] = MethodStat{mBlur, blurStats}
	for _, n := range []int{1, 4, 8} {
		m := fmt.Sprintf("ss%d", n)
		_, alpha, _ := cover.RenderMethod(scene, m, viz.White)
		stats := cover.DiffStats(ana.alpha, alpha)
		st.Methods[m] = MethodStat{m, stats}
	}

	ssVmax := ss2Stats.MaxAbs
	if ss3Stats.MaxAbs > ssVmax {
		ssVmax = ss3Stats.MaxAbs
	}
	heat := []struct {
		m     string
		stats cover.ErrStats
		vmax  float64
	}{
		{mSS2, ss2Stats, ssVmax},
		{mSS3, ss3Stats, ssVmax},
		{mBlur, blurStats, 0},
	}
	for _, h := range heat {
		title := fmt.Sprintf("%s / %s ERROR VS ANALYTIC", strings.ToUpper(scene.Name), methodLabel(h.m))
		statsLine := fmt.Sprintf("MAX %.3f   RMSE %.4f", h.stats.MaxAbs, h.stats.RMSE)
		panel := viz.HeatmapPanel(scene, res[h.m].alpha, ana.alpha, h.vmax, title, statsLine)
		if err := viz.SavePNG(filepath.Join(outDir, "heatmap", scene.Name+"__"+h.m+"_error.png"), panel); err != nil {
			return st, err
		}
	}
	return st, nil
}

func methodLabel(m string) string {
	switch m {
	case mAnalytic:
		return "ANALYTIC"
	case mSS2:
		return "SS 2X2"
	case mSS3:
		return "SS 3X3"
	case mBlur:
		return "BLUR"
	}
	return m
}
