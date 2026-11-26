package studio

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
)

type FileReport struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int64  `json:"bytes"`
}

type Checks struct {
	PremultipliedLinear bool    `json:"premultiplied_linear"`
	Dither              string  `json:"dither"`
	RawBandRuns         float64 `json:"raw_band_runs_per_row"`
	DitheredBandRuns    float64 `json:"dithered_band_runs"`
	TileBoundaryDelta   float64 `json:"tile_boundary_delta"`
	SeamResidualLSB     float64 `json:"seam_residual_lsb"`
}

type Report struct {
	Canvas     [2]int       `json:"canvas"`
	BlendModes []string     `json:"blend_modes"`
	ClipStack  []string     `json:"clip_stack"`
	Order      []string     `json:"order_variants"`
	Checks     Checks       `json:"checks"`
	Files      []FileReport `json:"files"`
}

func Generate(outDir string) (*Report, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	scene := NewScene()
	report := &Report{
		Canvas:     [2]int{CanvasW, CanvasH},
		BlendModes: []string{"normal", "multiply", "screen"},
		ClipStack:  []string{"frame", "group mask (circles union text)"},
		Order:      []string{"linear > radial > alpha mask", "linear > alpha mask > radial"},
		Checks: Checks{
			PremultipliedLinear: true,
			Dither:              "bayer8+ign",
		},
	}

	saves := []struct {
		rel  string
		img  image.Image
		w, h int
	}{
		{"layers/00_background.png", ToNRGBA(scene.Background, true), CanvasW, CanvasH},
		{"layers/01_group_mask.png", ToGray(scene.Group, CanvasW, CanvasH), CanvasW, CanvasH},
		{"layers/02_linear_group.png", ToNRGBA(scene.Linear, true), CanvasW, CanvasH},
		{"layers/03_radial_group.png", ToNRGBA(scene.Radial, true), CanvasW, CanvasH},
		{"layers/04_alpha_mask.png", ToGray(scene.Alpha, CanvasW, CanvasH), CanvasW, CanvasH},
		{"layers/05_premask_stack.png", ToNRGBA(scene.BuildPremask(), true), CanvasW, CanvasH},
	}
	for _, item := range saves {
		if err := SavePNG(filepath.Join(outDir, item.rel), item.img); err != nil {
			return nil, err
		}
		if err := report.addFile(outDir, item.rel, item.w, item.h); err != nil {
			return nil, err
		}
	}
	normal := scene.BuildFinal(Normal, MaskLast)
	finalNormal := ToNRGBA(normal, true)
	if err := SavePNG(filepath.Join(outDir, "layers/06_final_normal.png"), finalNormal); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "layers/06_final_normal.png", CanvasW, CanvasH); err != nil {
		return nil, err
	}

	for _, mode := range []BlendMode{Normal, Multiply, Screen} {
		img := scene.BuildFinal(mode, MaskLast)
		rel := "final_" + mode.String() + ".png"
		if err := SavePNG(filepath.Join(outDir, rel), ToNRGBA(img, true)); err != nil {
			return nil, err
		}
		if err := report.addFile(outDir, rel, CanvasW, CanvasH); err != nil {
			return nil, err
		}
	}
	if err := SavePNG(filepath.Join(outDir, "final.png"), finalNormal); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "final.png", CanvasW, CanvasH); err != nil {
		return nil, err
	}

	layerSheet := LayerSheet(scene)
	if err := SavePNG(filepath.Join(outDir, "layer_sheet.png"), ToNRGBA(layerSheet, true)); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "layer_sheet.png", CanvasW, CanvasH); err != nil {
		return nil, err
	}

	blendSheet := BlendSheet(scene)
	if err := SavePNG(filepath.Join(outDir, "blend_modes.png"), ToNRGBA(blendSheet, true)); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "blend_modes.png", CanvasW, blendSheet.H); err != nil {
		return nil, err
	}

	orderSheet := OrderSheet(scene)
	if err := SavePNG(filepath.Join(outDir, "order_compare.png"), ToNRGBA(orderSheet, true)); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "order_compare.png", CanvasW, orderSheet.H); err != nil {
		return nil, err
	}

	banding, stats := BandingSheet(scene)
	if err := SavePNG(filepath.Join(outDir, "banding_detection.png"), ToNRGBA(banding, false)); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "banding_detection.png", CanvasW, CanvasH); err != nil {
		return nil, err
	}
	report.Checks.RawBandRuns = stats["raw_band_runs_per_row"]
	report.Checks.DitheredBandRuns = stats["dithered_band_runs"]
	report.Checks.TileBoundaryDelta = stats["tile_boundary_delta"]
	report.Checks.SeamResidualLSB = stats["seam_residual_lsb"]

	tile := TileImage(320, 100, 3, 2)
	if err := SavePNG(filepath.Join(outDir, "seamless_tile.png"), ToNRGBA(tile, true)); err != nil {
		return nil, err
	}
	if err := report.addFile(outDir, "seamless_tile.png", tile.W, tile.H); err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "report.json"), data, 0o644); err != nil {
		return nil, err
	}
	return report, nil
}

func (r *Report) addFile(root, rel string, w, h int) error {
	info, err := os.Stat(filepath.Join(root, rel))
	if err != nil {
		return err
	}
	r.Files = append(r.Files, FileReport{
		Path:   filepath.ToSlash(rel),
		Width:  w,
		Height: h,
		Bytes:  info.Size(),
	})
	return nil
}
