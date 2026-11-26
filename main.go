package main

import (
	"flag"
	"image"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := flag.String("out", "artifacts", "artifact output directory")
	ss := flag.Int("ss", 4, "supersampling factor for self rasterizer")
	threshold := flag.Float64("threshold", 42, "pixel RMS error threshold")
	skipBrowser := flag.Bool("skip-browser", false, "do not invoke headless browser")
	flag.Parse()

	if err := run(*outDir, *ss, *threshold, *skipBrowser); err != nil {
		log.Fatalf("vectorlab: %v", err)
	}
}

func run(outDir string, ss int, threshold float64, skipBrowser bool) error {
	started := time.Now()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	log.Printf("vectorlab: building scene")
	scene := BuildScene()

	svgText := ExportSVG(scene)
	svgPath := filepath.Join(outDir, "scene.svg")
	if err := os.WriteFile(svgPath, []byte(svgText), 0o644); err != nil {
		return err
	}

	reloaded, audit, err := ParseSVG(svgText)
	if err != nil {
		return err
	}

	log.Printf("vectorlab: self rasterizing (%dx SSAA)", ss)
	ownRR := Render(scene, ss)
	if err := savePNG(filepath.Join(outDir, "own.png"), ownRR.Img); err != nil {
		return err
	}

	relRR := Render(reloaded, ss)
	if err := savePNG(filepath.Join(outDir, "reloaded.png"), relRR.Img); err != nil {
		return err
	}
	reloadDelta := meanImageDelta(ownRR.Img, relRR.Img)

	var binfo *BrowserInfo
	var refRGBA *image.RGBA
	if !skipBrowser {
		log.Printf("vectorlab: capturing browser reference")
		refPath := filepath.Join(outDir, "browser_reference.png")
		binfo, err = CaptureBrowser(svgText, refPath, scene.W, scene.H)
		if err != nil {
			log.Printf("vectorlab: browser capture failed: %v", err)
		} else {
			log.Printf("vectorlab: browser %s rendered %dx%d", binfo.Mode, binfo.Width, binfo.Height)
			if img, lerr := loadPNG(refPath); lerr == nil {
				refRGBA = toRGBA(img)
			}
			_ = copyFile(refPath, filepath.Join(outDir, "browser_ref.png"))
		}
	}
	if refRGBA == nil {
		log.Printf("vectorlab: using self render as reference fallback (browser image unavailable)")
		refRGBA = ownRR.Img
	}

	log.Printf("vectorlab: computing structural diff")
	diff, rep := ComputeDiff(refRGBA, ownRR, threshold)
	heat := diff.Heatmap(3.0)
	if err := savePNG(filepath.Join(outDir, "heatmap.png"), heat); err != nil {
		return err
	}
	idmap := BuildIDMap(ownRR)
	if err := savePNG(filepath.Join(outDir, "id_map.png"), idmap); err != nil {
		return err
	}
	errLayer := BuildErrorLayer(ownRR, diff, rep, LayerOpts{})
	if err := SaveTitledImage(filepath.Join(outDir, "error_layer.png"),
		"ERROR LAYER - ORIGINAL OBJECT IDS", errLayer, inkBad); err != nil {
		return err
	}

	fault := runFaultDemo(outDir, refRGBA, scene, ss, threshold)

	ownBytes, err := os.ReadFile(filepath.Join(outDir, "own.png"))
	if err != nil {
		return err
	}
	badSVG := BuildFlattenedBadSVG(ownBytes, scene.W, scene.H)
	if err := os.WriteFile(filepath.Join(outDir, "flattened_bad.svg"), []byte(badSVG), 0o644); err != nil {
		return err
	}
	_, badAudit, err := ParseSVG(badSVG)
	if err != nil {
		return err
	}

	checks := buildChecks(scene, audit, rep, reloadDelta, badAudit, fault, binfo != nil)

	board := BuildDiffBoard(&BoardModel{
		RefImg: refRGBA, OwnImg: ownRR.Img, HeatImg: heat, IDImg: idmap,
		ErrImg: errLayer, Rep: rep, Reload: audit, ReloadDelta: reloadDelta,
		Bad: badAudit, Checks: checks,
	})
	if err := savePNG(filepath.Join(outDir, "diff.png"), board); err != nil {
		return err
	}

	aliases := map[string]string{
		"self.png":            "own.png",
		"own_render.png":      "own.png",
		"structural_diff.png": "diff.png",
		"delta_heatmap.png":   "heatmap.png",
		"id_segmentation.png": "id_map.png",
	}
	for alias, src := range aliases {
		_ = copyFile(filepath.Join(outDir, src), filepath.Join(outDir, alias))
	}

	fileList := []string{
		"scene.svg", "browser_reference.png", "browser_ref.png", "own.png", "self.png",
		"reloaded.png", "diff.png", "structural_diff.png", "error_layer.png",
		"heatmap.png", "id_map.png", "fault_demo.png", "flattened_bad.svg",
		"diff_report.json",
	}
	report := OutputReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		OutDir:      outDir,
		Scene:       makeSceneInfo(scene),
		Browser:     binfo,
		Diff:        rep,
		Checks:      checks,
		ReloadAudit: audit,
		BadBaseline: badAudit,
		FaultDemo:   fault,
		Files:       fileList,
	}
	if err := writeJSON(filepath.Join(outDir, "diff_report.json"), report); err != nil {
		return err
	}

	pass := 0
	for _, c := range checks {
		if c.Pass {
			pass++
		}
	}
	log.Printf("vectorlab: %d/%d checks passed in %s -> %s", pass, len(checks),
		time.Since(started).Round(time.Millisecond), outDir)
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func makeSceneInfo(s *Scene) SceneInfo {
	ids := make([]string, 0, len(s.All))
	for _, o := range s.All {
		ids = append(ids, o.ID)
	}
	return SceneInfo{
		Width: s.W, Height: s.H, ObjectCount: len(s.All), IDs: ids,
		Gradients: len(s.Grads), ClipPaths: len(s.Clips),
	}
}
