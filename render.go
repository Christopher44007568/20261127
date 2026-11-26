package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type RenderOptions struct {
	Width    int
	Height   int
	AA       int
	Exposure float64
}

type ImageMetrics struct {
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	Coverage           float64 `json:"coverage"`
	MeanClothLinear    float64 `json:"mean_cloth_linear"`
	MeanEdgeLinear     float64 `json:"mean_edge_linear"`
	MeanInteriorLinear float64 `json:"mean_interior_linear"`
	EdgeInteriorRatio  float64 `json:"edge_interior_ratio"`
	MaxLinear          float64 `json:"max_linear"`
	OverOneFraction    float64 `json:"over_one_fraction"`
}

type RenderResult struct {
	Pixels   []Vec3
	Coverage []float64
	Metrics  ImageMetrics
}

func renderView(camera Camera, view View, material Material, opts RenderOptions) RenderResult {
	width, height := opts.Width, opts.Height
	aa := opts.AA
	pixels := make([]Vec3, width*height)
	coverage := make([]float64, width*height)
	metrics := ImageMetrics{Width: width, Height: height}
	var clothSamples, edgeSamples, interiorSamples, overOneSamples float64
	var clothSum, edgeSum, interiorSum float64

	for py := 0; py < height; py++ {
		v := (float64(py) + 0.5) / float64(height)
		for px := 0; px < width; px++ {
			u := (float64(px) + 0.5) / float64(width)
			var color Vec3
			var hits int
			for sy := 0; sy < aa; sy++ {
				for sx := 0; sx < aa; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/float64(aa)
					fy := float64(py) + (float64(sy)+0.5)/float64(aa)
					dir := camera.RayAt(fx, fy, width, height)
					hit, ok := intersectCloth(camera.Position, dir)
					if !ok {
						color = color.Add(studioBackground(u, v, view.Backlit))
						continue
					}
					n, tangent, edge := clothNormals(hit.Position.X, hit.Position.Y, view.FiberAngle)
					shaded := material.Shade(ShadePoint{
						Normal: n, Tangent: tangent, View: dir.Scale(-1), EdgeDist: edge,
					}, view.Lights)
					color = color.Add(shaded)
					hits++
					lum := shaded.Luminance()
					clothSamples++
					clothSum += lum
					metrics.MaxLinear = math.Max(metrics.MaxLinear, shaded.MaxComponent())
					if shaded.MaxComponent() > 1 {
						overOneSamples++
					}
					if edge <= 0.12 {
						edgeSamples++
						edgeSum += lum
					} else if edge >= 0.24 {
						interiorSamples++
						interiorSum += lum
					}
				}
			}
			samples := float64(aa * aa)
			pixels[py*width+px] = color.Scale(1 / samples)
			coverage[py*width+px] = float64(hits) / samples
		}
	}
	metrics.Coverage = coverageAverage(coverage)
	metrics.MeanClothLinear = safeRatio(clothSum, clothSamples)
	metrics.MeanEdgeLinear = safeRatio(edgeSum, edgeSamples)
	metrics.MeanInteriorLinear = safeRatio(interiorSum, interiorSamples)
	metrics.EdgeInteriorRatio = safeRatio(metrics.MeanEdgeLinear, metrics.MeanInteriorLinear)
	metrics.OverOneFraction = safeRatio(overOneSamples, clothSamples)
	return RenderResult{Pixels: pixels, Coverage: coverage, Metrics: metrics}
}

func coverageAverage(coverage []float64) float64 {
	if len(coverage) == 0 {
		return 0
	}
	inside := 0.0
	for _, c := range coverage {
		if c > 0.1 {
			inside++
		}
	}
	return inside / float64(len(coverage))
}

func safeRatio(a, b float64) float64 {
	if b <= 1e-12 {
		return 0
	}
	return a / b
}

func writePPM(path string, result RenderResult, exposure float64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", result.Metrics.Width, result.Metrics.Height); err != nil {
		return err
	}
	data := make([]byte, result.Metrics.Width*result.Metrics.Height*3)
	for i, c := range result.Pixels {
		r, g, b := displayRGB(c, exposure)
		data[i*3] = r
		data[i*3+1] = g
		data[i*3+2] = b
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return w.Flush()
}

func displayRGB(c Vec3, exposure float64) (byte, byte, byte) {
	r := displayChannel(c.X * exposure)
	g := displayChannel(c.Y * exposure)
	b := displayChannel(c.Z * exposure)
	return byte(math.Round(r * 255)), byte(math.Round(g * 255)), byte(math.Round(b * 255))
}

func displayChannel(x float64) float64 {
	x = math.Max(x, 0)
	y := (x * (2.51*x + 0.03)) / (x*(2.43*x+0.59) + 0.14)
	return math.Pow(clamp(y, 0, 1), 1/2.2)
}
