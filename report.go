package main

import (
	"encoding/json"
	"image"
	"math"
	"os"
)

// OutputReport is diff_report.json.
type OutputReport struct {
	GeneratedAt string        `json:"generatedAt"`
	OutDir      string        `json:"outDir"`
	Scene       SceneInfo     `json:"scene"`
	Browser     *BrowserInfo  `json:"browser,omitempty"`
	Diff        *DiffReport   `json:"diff"`
	Checks      []CheckResult `json:"checks"`
	ReloadAudit *SVGAudit     `json:"reloadAudit"`
	BadBaseline *SVGAudit     `json:"badBaselineAudit"`
	FaultDemo   *FaultInfo    `json:"faultDemo,omitempty"`
	Files       []string      `json:"files"`
}

// SceneInfo summarizes the source scene.
type SceneInfo struct {
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	ObjectCount int      `json:"objectCount"`
	IDs         []string `json:"ids"`
	Gradients   int      `json:"gradients"`
	ClipPaths   int      `json:"clipPaths"`
}

// FaultInfo records the injected-error attribution proof.
type FaultInfo struct {
	InjectedID            string `json:"injectedId"`
	AttributedID          string `json:"attributedId"`
	AttributedErrorPixels int    `json:"attributedErrorPixels"`
	Pass                  bool   `json:"pass"`
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func meanImageDelta(a, b *image.RGBA) float64 {
	if a.Bounds() != b.Bounds() {
		return -1
	}
	var sum float64
	n := a.Bounds().Dx() * a.Bounds().Dy()
	for i := 0; i < len(a.Pix); i += 4 {
		dr := float64(a.Pix[i]) - float64(b.Pix[i])
		dg := float64(a.Pix[i+1]) - float64(b.Pix[i+1])
		db := float64(a.Pix[i+2]) - float64(b.Pix[i+2])
		sum += math.Sqrt((dr*dr + dg*dg + db*db) / 3)
	}
	return sum / float64(n)
}
