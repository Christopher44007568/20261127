package main

import (
	"flag"
	"fmt"
	"image"
	"os"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNG/JSON deliverables")
	flag.Parse()
	if err := run(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	fs, err := LoadFontSet()
	if err != nil {
		return err
	}
	fmt.Printf("Latin font: %s (upem=%d)\n", fs.Latin.Path, fs.Latin.UnitsPerEm)
	if fs.Han != nil {
		fmt.Printf("Han font:   %s (upem=%d)\n", fs.Han.Path, fs.Han.UnitsPerEm)
	} else {
		fmt.Println("Han font:   NOT FOUND")
	}

	artifacts := []string{
		"01_original_outlines.png",
		"02_outline_points.png",
		"03_layout_result.png",
		"04_layout_clean.png",
		"05_stroke_preview.png",
		"06_zoom_linegap.png",
		"07_zoom_tracking.png",
		"08_zoom_counters.png",
		"09_zoom_points.png",
		"10_overview.png",
		"report.json",
		"outline_data.json",
	}

	imgOrig := RenderOriginalOutlines(fs)
	imgPts := RenderOutlineSampling(fs)
	sheet := RenderLayoutSheet(fs)
	clean := RenderCleanLayout(fs)
	stroke := RenderStrokePreview(fs)
	counters := RenderCounterZoom(fs).Render()

	lgCrop := Crop(sheet.Img, sheet.LineGapCrop)
	tkCrop := Crop(sheet.Img, sheet.TrackingCrop)
	lgZoom := ComposeZoomFrame(fs,
		"局部放大 1：上升部与下降部之间存在正间隙",
		"3× 最近邻取材自 03_layout_result.png 的 A 区；橙线=上行下降墨迹底，青线=下行上升墨迹顶",
		"间隙由真实 glyf 包围盒逐行计算：step = max(1.18em, descent + minGap + ascent)，因此相邻行永不相交",
		lgCrop, 1560)
	tkZoom := ComposeZoomFrame(fs,
		"局部放大 2：字距 T 作用于轮廓笔位",
		"取材自 03_layout_result.png 的 B 区；蓝框=真实字形墨迹边界，灰=advance，紫=额外 tracking",
		"x(n+1) = x(n) + advance(n) + T；字距是轮廓坐标推进量，不是文本框的内边距",
		tkCrop, 1280)

	ptsCrop := Crop(imgPts, PointsZoomCrop(imgPts))
	ptsZoom := ComposeZoomFrame(fs,
		"局部放大 3：轮廓布点与法向",
		"取材自 02_outline_points.png 中字母 g 的字腔区域；红点按弧长等距分布",
		"先累计折线弧长，再按目标弧长反解参数位置；采样间距误差写入 outline_data.json",
		ptsCrop, 1500)

	images := map[string]*image.RGBA{
		"01_original_outlines.png": imgOrig,
		"02_outline_points.png":    imgPts,
		"03_layout_result.png":     sheet.Img,
		"04_layout_clean.png":      clean,
		"05_stroke_preview.png":    stroke,
		"06_zoom_linegap.png":      lgZoom,
		"07_zoom_tracking.png":     tkZoom,
		"08_zoom_counters.png":     counters,
		"09_zoom_points.png":       ptsZoom,
	}
	for name, img := range images {
		if err := SaveImagePNG(outDir+"/"+name, img); err != nil {
			return err
		}
	}

	overview := RenderOverview(fs, images)
	if err := SaveImagePNG(outDir+"/10_overview.png", overview); err != nil {
		return err
	}

	lay := fs.Layout(demoParagraph(), mainLayoutStyle())
	rep := buildReport(fs, lay, artifacts)
	if err := writeJSON(outDir+"/report.json", rep); err != nil {
		return err
	}
	if err := writeJSON(outDir+"/outline_data.json", buildOutlineData(fs)); err != nil {
		return err
	}

	allPass := true
	for _, c := range rep.Checks {
		status := "PASS"
		if !c.Pass {
			status = "FAIL"
			allPass = false
		}
		fmt.Printf("CHECK %s  %s (%.2f)\n", status, c.Name, c.Value)
	}
	if !allPass {
		return fmt.Errorf("typography invariant check failed")
	}
	fmt.Printf("min line ink gap: %.2f px (overlap=%v)\n", lay.MinPairGap, lay.Overlap)
	fmt.Println("wrote", len(artifacts), "artifacts to", outDir)
	return nil
}
