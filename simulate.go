package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Config controls a deterministic drag simulation.
type Config struct {
	Width          int
	Height         int
	Updates        int
	OutDir         string
	WriteArtifacts bool
}

// DefaultConfig is the large-canvas demo used by the CLI.
func DefaultConfig() Config {
	return Config{
		Width:          1280,
		Height:         800,
		Updates:        24,
		OutDir:         "artifacts",
		WriteArtifacts: true,
	}
}

// FrameStat records the pixel-equivalence and dirty-area result of one step.
type FrameStat struct {
	Step           int     `json:"step"`
	StatusText     string  `json:"status_text"`
	RawRects       int     `json:"raw_rects"`
	RepaintRects   int     `json:"repaint_rects"`
	DirtyPixels    int     `json:"dirty_pixels"`
	CanvasPixels   int     `json:"canvas_pixels"`
	DirtyRatioPct  float64 `json:"dirty_ratio_pct"`
	MismatchPixels int     `json:"mismatch_pixels"`
	PixelIdentical bool    `json:"pixel_identical"`
}

// Report is the machine-readable result of a run.
type Report struct {
	Width               int         `json:"width"`
	Height              int         `json:"height"`
	Updates             int         `json:"updates"`
	CanvasPixels        int         `json:"canvas_pixels"`
	Frames              []FrameStat `json:"frames"`
	TotalMismatchPixels int         `json:"total_mismatch_pixels"`
	PixelPerfect        bool        `json:"pixel_perfect"`
	AvgDirtyRatioPct    float64     `json:"avg_dirty_ratio_pct"`
	LastDirtyRatioPct   float64     `json:"last_dirty_ratio_pct"`
	Files               []string    `json:"files"`
	OutDir              string      `json:"out_dir"`
}

type demoScene struct {
	sc     *Scene
	font   *Font
	a      *ShapeObject
	b      *ShapeObject
	c      *ShapeObject
	status *LabelObject
	hud    *LabelObject
}

var statusSequence = []string{
	"DRAG ME!",
	"MOVING...",
	"GO!",
	"CACHE",
	"A TO B",
	"HI!",
	"DIRTY RECT",
	"X:042",
	"SHORTER?",
	"OK",
	"PIXEL-PERFECT",
	"DONE!",
}

func newDemoScene(w, h int, font *Font) *demoScene {
	bg := BuildBackground(w, h, font)
	sc := &Scene{Width: w, Height: h, Bg: bg}

	a := NewShape("shape-a", shapeRect, 240, 300, Color{R: 255, G: 159, B: 28}, font, "A X:0240 Y:0300")
	b := NewShape("shape-b", shapeDisk, 1040, 560, Color{R: 46, G: 196, B: 182}, font, "B X:1040 Y:0560")
	c := NewShape("shape-c", shapeDiamond, 640, 430, Color{R: 131, G: 56, B: 236}, font, "C X:0640 Y:0430")
	sc.add(a)
	sc.add(b)
	sc.add(c)

	status := NewLabel("status", 48, 112, false, font, 3,
		Color{R: 255, G: 224, B: 130}, Color{R: 20, G: 32, B: 64}, Color{R: 120, G: 150, B: 220},
		statusSequence[0])
	sc.add(status)

	hud := NewLabel("hud", w-48, 44, true, font, 2,
		Color{R: 220, G: 232, B: 255}, Color{R: 20, G: 32, B: 64}, Color{R: 120, G: 150, B: 220},
		"FRAME 00/24")
	sc.add(hud)

	return &demoScene{sc: sc, font: font, a: a, b: b, c: c, status: status, hud: hud}
}

// pingPong maps step i to an integer triangle wave in [lo, hi].
func pingPong(i, period, phase, lo, hi int) int {
	if period < 2 {
		return lo
	}
	p := ((i+phase)%period + period) % period
	half := period / 2
	if p <= half {
		return lo + (hi-lo)*p/half
	}
	return hi - (hi-lo)*(p-half)/(period-half)
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

// step advances the scripted drag by one frame.
func (d *demoScene) step(i, total int) string {
	w, h := d.sc.Width, d.sc.Height

	ax := pingPong(i, 44, 0, 190, w-190)
	ay := clampInt(300+int(90*math.Sin(float64(i)*0.34)), 215, 395)
	d.a.SetCenter(ax, ay, fmt.Sprintf("A X:%04d Y:%04d", ax, ay))

	bx := pingPong(i, 36, 13, 150, w-150)
	by := clampInt(560+int(60*math.Cos(float64(i)*0.27)), 480, 625)
	d.b.SetCenter(bx, by, fmt.Sprintf("B X:%04d Y:%04d", bx, by))

	cx := pingPong(i, 29, 7, 210, w-210)
	cy := pingPong(i, 19, 3, 250, h-250)
	d.c.SetCenter(cx, cy, fmt.Sprintf("C X:%04d Y:%04d", cx, cy))

	text := statusSequence[(i-1)%len(statusSequence)]
	d.status.SetText(text)
	d.hud.SetText(fmt.Sprintf("FRAME %02d/%02d", i, total))
	return text
}

// Run executes the simulation, verifies every incremental frame against a
// full repaint, and optionally writes all PPM/PNG/report artifacts.
func Run(cfg Config) (*Report, error) {
	if cfg.Width <= 0 {
		cfg.Width = 1280
	}
	if cfg.Height <= 0 {
		cfg.Height = 800
	}
	if cfg.Updates <= 0 {
		cfg.Updates = 24
	}

	font := DefaultFont()
	demo := newDemoScene(cfg.Width, cfg.Height, font)
	sc := demo.sc

	// Initial frame: a single full paint, then tracker baseline.
	front := RenderFull(sc)
	tracker := NewDirtyTracker()
	tracker.Snapshot(sc)

	rep := &Report{
		Width:        cfg.Width,
		Height:       cfg.Height,
		Updates:      cfg.Updates,
		CanvasPixels: cfg.Width * cfg.Height,
		PixelPerfect: true,
	}

	var lastRegion Region
	var lastMerged []Rect
	var fullFinal *Framebuffer

	for i := 1; i <= cfg.Updates; i++ {
		status := demo.step(i, cfg.Updates)

		region := tracker.Collect(sc)
		merged := region.Merged()
		for _, rc := range merged {
			PaintScene(front, sc, rc)
		}

		full := RenderFull(sc)
		mismatch := MismatchCount(front, full)
		dirtyPixels := 0
		for _, rc := range merged {
			dirtyPixels += rc.Pixels()
		}

		stat := FrameStat{
			Step:           i,
			StatusText:     status,
			RawRects:       len(region.Rects),
			RepaintRects:   len(merged),
			DirtyPixels:    dirtyPixels,
			CanvasPixels:   rep.CanvasPixels,
			DirtyRatioPct:  100 * float64(dirtyPixels) / float64(rep.CanvasPixels),
			MismatchPixels: mismatch,
			PixelIdentical: mismatch == 0,
		}
		rep.Frames = append(rep.Frames, stat)
		rep.TotalMismatchPixels += mismatch
		if mismatch != 0 {
			rep.PixelPerfect = false
		}
		rep.AvgDirtyRatioPct += stat.DirtyRatioPct

		if i == cfg.Updates {
			lastRegion = region
			lastMerged = merged
			fullFinal = full
		}
	}

	if cfg.Updates > 0 {
		rep.AvgDirtyRatioPct /= float64(cfg.Updates)
		rep.LastDirtyRatioPct = rep.Frames[len(rep.Frames)-1].DirtyRatioPct
	}

	if cfg.WriteArtifacts {
		files, err := writeArtifacts(cfg.OutDir, rep, front, fullFinal, lastRegion, lastMerged, font)
		if err != nil {
			return rep, err
		}
		rep.Files = files
		rep.OutDir = cfg.OutDir
	}
	return rep, nil
}

func writeArtifacts(outDir string, rep *Report, front, full *Framebuffer, region Region, merged []Rect, font *Font) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	dirtyMap := RenderDirtyMap(front, merged)
	mask := RenderDirtyMask(front.W, front.H, merged)
	diff := RenderDiff(front, full)
	sheet := RenderFontSheet(font)

	type artifact struct {
		name string
		fb   *Framebuffer
		png  bool
	}
	artifacts := []artifact{
		{"dirty_region.ppm", dirtyMap, true},
		{"dirty.ppm", dirtyMap, false},
		{"dirty_rect.ppm", dirtyMap, false},
		{"dirty_map.ppm", dirtyMap, false},
		{"dirty_mask.ppm", mask, true},
		{"incremental_result.ppm", front, true},
		{"incremental.ppm", front, false},
		{"full_reference.ppm", full, true},
		{"reference.ppm", full, false},
		{"full.ppm", full, false},
		{"diff.ppm", diff, false},
		{"font_sheet.ppm", sheet, true},
	}

	var files []string
	for _, a := range artifacts {
		p := filepath.Join(outDir, a.name)
		if err := WritePPM(p, a.fb); err != nil {
			return files, fmt.Errorf("write %s: %w", p, err)
		}
		files = append(files, p)
		if a.png {
			pngPath := trimExt(p) + ".png"
			if err := WritePNG(pngPath, a.fb); err != nil {
				return files, fmt.Errorf("write %s: %w", pngPath, err)
			}
			files = append(files, pngPath)
		}
	}

	if err := writeReportFiles(outDir, rep, region, merged); err != nil {
		return files, err
	}
	files = append(files, filepath.Join(outDir, "report.txt"), filepath.Join(outDir, "report.json"))
	return files, nil
}

func trimExt(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '.' {
			return p[:i]
		}
		if os.IsPathSeparator(p[i]) {
			break
		}
	}
	return p
}

func writeReportFiles(outDir string, rep *Report, region Region, merged []Rect) error {
	jsonBytes, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "report.json"), jsonBytes, 0o644); err != nil {
		return err
	}

	txt, err := formatReport(rep, region, merged)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "report.txt"), []byte(txt), 0o644)
}

func formatReport(rep *Report, region Region, merged []Rect) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Dirty-Rectangle Canvas Report\n")
	fmt.Fprintf(&b, "canvas: %dx%d (%d pixels), drag steps: %d\n\n", rep.Width, rep.Height, rep.CanvasPixels, rep.Updates)
	fmt.Fprintf(&b, "pixel-perfect vs full repaint: %v (total mismatching pixels: %d)\n", rep.PixelPerfect, rep.TotalMismatchPixels)
	fmt.Fprintf(&b, "average dirty area: %.2f%% of canvas; last step: %.2f%%\n\n", rep.AvgDirtyRatioPct, rep.LastDirtyRatioPct)

	fmt.Fprintf(&b, "last step dirty rectangles (old bounds + new bounds, merged to %d disjoint repaint rects):\n", len(merged))
	for i, r := range merged {
		fmt.Fprintf(&b, "  rect[%02d] x=%d y=%d w=%d h=%d pixels=%d\n", i, r.X0, r.Y0, r.Width(), r.Height(), r.Pixels())
	}
	fmt.Fprintf(&b, "\nraw tracked rectangles: %d\n\n", len(region.Rects))

	fmt.Fprintf(&b, "step  status         raw  merged  dirty%%   mismatch  result\n")
	for _, s := range rep.Frames {
		fmt.Fprintf(&b, "%4d  %-14s %4d  %6d  %6.2f  %8d  %s\n",
			s.Step, s.StatusText, s.RawRects, s.RepaintRects, s.DirtyRatioPct, s.MismatchPixels,
			map[bool]string{true: "IDENTICAL", false: "DIFFER"}[s.PixelIdentical])
	}
	return b.String(), nil
}
