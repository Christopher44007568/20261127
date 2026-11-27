package main

import (
	"encoding/json"
	"image"
	"math"
	"os"
	"path/filepath"
)

type ManifestImage struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ManifestCamera struct {
	Position         Vec3    `json:"position"`
	Target           Vec3    `json:"target"`
	FovYDegrees      float64 `json:"fov_y_degrees"`
	HorizontalFovDeg float64 `json:"horizontal_fov_degrees"`
	Near             float64 `json:"near"`
	Far              float64 `json:"far"`
}

type ManifestComparison struct {
	Exact           bool    `json:"exact"`
	TotalPixels     int     `json:"total_pixels"`
	DifferentPixels int     `json:"different_pixels"`
	MaxChannelDiff  int     `json:"max_channel_diff"`
	RMSE            float64 `json:"rmse"`
	PSNR            float64 `json:"psnr_db"`
}

type ManifestFrame struct {
	Index      int                `json:"index"`
	Path       string             `json:"path"`
	Camera     ManifestCamera     `json:"camera"`
	Froxel     FroxelStats        `json:"froxel"`
	Render     RenderStats        `json:"render"`
	Comparison ManifestComparison `json:"comparison"`
}

type ArtifactManifest struct {
	Title            string             `json:"title"`
	GeneratedUTC     string             `json:"generated_utc"`
	Renderer         string             `json:"renderer"`
	Scene            map[string]any     `json:"scene"`
	Camera           ManifestCamera     `json:"camera"`
	Froxel           FroxelStats        `json:"froxel"`
	Reference        ManifestImage      `json:"reference"`
	FroxelImage      ManifestImage      `json:"froxel_image"`
	Heatmap          ManifestImage      `json:"heatmap"`
	Diff             ManifestImage      `json:"diff"`
	Comparison       ManifestComparison `json:"comparison"`
	ReferenceRender  RenderStats        `json:"reference_render"`
	FroxelRender     RenderStats        `json:"froxel_render"`
	WideFOV          *WideFOVArtifact   `json:"wide_fov,omitempty"`
	Frames           []ManifestFrame    `json:"frames"`
	GIF              ManifestImage      `json:"gif"`
	FallbackPolicies []string           `json:"fallback_policies"`
}

type WideFOVArtifact struct {
	Camera          ManifestCamera     `json:"camera"`
	Froxel          FroxelStats        `json:"froxel"`
	ReferenceImage  ManifestImage      `json:"reference_image"`
	FroxelImage     ManifestImage      `json:"froxel_image"`
	ReferenceRender RenderStats        `json:"reference_render"`
	FroxelRender    RenderStats        `json:"froxel_render"`
	Comparison      ManifestComparison `json:"comparison"`
}

func manifestCamera(c *Camera) ManifestCamera {
	return ManifestCamera{
		Position:         c.Position,
		Target:           c.Target,
		FovYDegrees:      c.FovY * 180 / 3.141592653589793,
		HorizontalFovDeg: c.HorizontalFOV() * 180 / 3.141592653589793,
		Near:             c.Near,
		Far:              c.Far,
	}
}

func manifestImage(path string, img interface{ Bounds() image.Rectangle }) ManifestImage {
	b := img.Bounds()
	return ManifestImage{Path: path, Width: b.Dx(), Height: b.Dy()}
}

func manifestComparison(c ImageComparison) ManifestComparison {
	psnr := c.PSNR
	if math.IsInf(psnr, 1) {
		psnr = 99.99
	}
	return ManifestComparison{
		Exact:           c.Exact,
		TotalPixels:     c.TotalPixels,
		DifferentPixels: c.DifferentPixels,
		MaxChannelDiff:  c.MaxChannelDiff,
		RMSE:            c.RMSE,
		PSNR:            psnr,
	}
}

func WriteManifest(path string, m ArtifactManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
