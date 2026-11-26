package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"postfx"
)

type nodeReport struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Outputs []string `json:"outputs"`
}

type runReport struct {
	Mode             string       `json:"mode"`
	Executed         []nodeReport `json:"executedNodes"`
	Skipped          []string     `json:"skippedNodes"`
	AllocatedOutputs []string     `json:"allocatedOutputs"`
	ReleasedOutputs  []string     `json:"releasedOutputs"`
	RetainedRoots    []string     `json:"retainedRoots"`
	PeakLiveTextures int          `json:"peakLiveTextures"`
	PeakTextureBytes int64        `json:"peakTextureBytes"`
	TempsAtEnd       int          `json:"intermediateTexturesAtEnd"`
}

type manifest struct {
	Width               int         `json:"width"`
	Height              int         `json:"height"`
	PixelIdentical      bool        `json:"pixelIdentical"`
	MaxChannelDiff      float64     `json:"maxChannelDifference"`
	Full                runReport   `json:"full"`
	Culled              runReport   `json:"culled"`
	PathPreviews        runReport   `json:"pathPreviews"`
	DisabledReturns     []string    `json:"disabledReturns"`
	DisabledUnallocated bool        `json:"disabledReturnsUnallocated"`
	Checks              checkReport `json:"checks"`
}

type checkReport struct {
	CullingOnlyReducedWork bool `json:"cullingOnlyReducedWork"`
	DisabledOutputsEmpty   bool `json:"disabledReturnsNeverAllocated"`
	AllTempsReleased       bool `json:"allIntermediatesReleased"`
	WeakOrderPreserved     bool `json:"weakOrderPreserved"`
}

func main() {
	width := flag.Int("width", 480, "image width")
	height := flag.Int("height", 270, "image height")
	outDir := flag.String("out", "artifacts/render", "output directory")
	flag.Parse()

	if err := run(*width, *height, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "postfx-demo:", err)
		os.Exit(1)
	}
}

func run(width, height int, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	comparison, err := postfx.RenderComparison(width, height)
	if err != nil {
		return err
	}
	files := []struct {
		name  string
		image *postfx.Image
	}{
		{"full.png", comparison.Full},
		{"culled.png", comparison.Culled},
		{"difference.png", comparison.Difference},
		{"bloom.png", comparison.PreviewImages[postfx.ResBloomPreview]},
		{"fog.png", comparison.PreviewImages[postfx.ResFogPreview]},
		{"ui.png", comparison.PreviewImages[postfx.ResUIPreview]},
	}
	for _, file := range files {
		if file.image == nil {
			return fmt.Errorf("missing output image %s", file.name)
		}
		if err := file.image.WritePNG(filepath.Join(outDir, file.name)); err != nil {
			return err
		}
	}

	disabledUnallocated := !containsAny(comparison.FullReport.AllocatedOutputs, comparison.DisabledReturns) &&
		!containsAny(comparison.CulledReport.AllocatedOutputs, comparison.DisabledReturns) &&
		!containsAny(comparison.PreviewReport.AllocatedOutputs, comparison.DisabledReturns)
	allTempsReleased := comparison.FullReport.TempTexturesAtEnd == 0 &&
		comparison.CulledReport.TempTexturesAtEnd == 0 &&
		comparison.PreviewReport.TempTexturesAtEnd == 0
	cullingOnlyReducedWork := comparison.PixelIdentical &&
		len(comparison.CulledReport.Executed) < len(comparison.FullReport.Executed)
	weakOrderPreserved := weakOrderAfter(comparison.CulledReport.Executed, "ui-layer", "bloom-blur-v")

	data := manifest{
		Width:               width,
		Height:              height,
		PixelIdentical:      comparison.PixelIdentical,
		MaxChannelDiff:      comparison.MaxDifference,
		Full:                summarize(comparison.FullReport),
		Culled:              summarize(comparison.CulledReport),
		PathPreviews:        summarize(comparison.PreviewReport),
		DisabledReturns:     resourceStrings(comparison.DisabledReturns),
		DisabledUnallocated: disabledUnallocated,
		Checks: checkReport{
			CullingOnlyReducedWork: cullingOnlyReducedWork,
			DisabledOutputsEmpty:   disabledUnallocated,
			AllTempsReleased:       allTempsReleased,
			WeakOrderPreserved:     weakOrderPreserved,
		},
	}
	file, err := os.Create(filepath.Join(outDir, "report.json"))
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	fmt.Printf("full executed:   %v\n", executedNames(comparison.FullReport))
	fmt.Printf("culled executed: %v\n", executedNames(comparison.CulledReport))
	fmt.Printf("preview executed:%v\n", executedNames(comparison.PreviewReport))
	fmt.Printf("max difference:  %.9f (identical=%t)\n", comparison.MaxDifference, comparison.PixelIdentical)
	fmt.Printf("culling reduced work: %t\n", cullingOnlyReducedWork)
	fmt.Printf("disabled returns unallocated: %t\n", disabledUnallocated)
	fmt.Printf("intermediates released: %t\n", allTempsReleased)
	fmt.Printf("artifacts: %s\n", outDir)
	if !cullingOnlyReducedWork || !disabledUnallocated || !allTempsReleased || !weakOrderPreserved {
		return fmt.Errorf("validation checks failed")
	}
	return nil
}

func summarize(report postfx.ExecutionReport) runReport {
	out := runReport{
		Mode:             modeName(report.Mode),
		AllocatedOutputs: resourceStrings(report.AllocatedOutputs),
		ReleasedOutputs:  resourceStrings(report.ReleasedOutputs),
		RetainedRoots:    resourceStrings(report.Roots),
		PeakLiveTextures: report.PeakLiveTextures,
		PeakTextureBytes: report.PeakTextureBytes,
		TempsAtEnd:       report.TempTexturesAtEnd,
	}
	for _, node := range report.Executed {
		out.Executed = append(out.Executed, nodeReport{
			ID:      string(node.ID),
			Name:    node.Name,
			Path:    node.Path,
			Outputs: resourceStrings(node.Outputs),
		})
	}
	for _, skipped := range report.Skipped {
		out.Skipped = append(out.Skipped, fmt.Sprintf("%s (%s)", skipped.ID, skipped.Reason))
	}
	sort.Strings(out.Skipped)
	return out
}

func modeName(mode postfx.ExecutionMode) string {
	switch mode {
	case postfx.ExecuteAllActive:
		return "all-active"
	case postfx.ExecuteReachable:
		return "reachable-only"
	default:
		return fmt.Sprintf("mode-%d", mode)
	}
}

func resourceStrings(ids []postfx.ResourceID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

func executedNames(report postfx.ExecutionReport) []string {
	names := make([]string, len(report.Executed))
	for i, node := range report.Executed {
		names[i] = string(node.ID)
	}
	return names
}

func containsAny(resources []postfx.ResourceID, values []postfx.ResourceID) bool {
	for _, resource := range resources {
		for _, value := range values {
			if resource == value {
				return true
			}
		}
	}
	return false
}

func weakOrderAfter(nodes []postfx.ExecutedNode, dependent, dependency postfx.PassID) bool {
	dependencyIndex := -1
	dependentIndex := -1
	for i, node := range nodes {
		if node.ID == dependency {
			dependencyIndex = i
		}
		if node.ID == dependent {
			dependentIndex = i
		}
	}
	if dependencyIndex < 0 || dependentIndex < 0 {
		return false
	}
	return dependencyIndex < dependentIndex
}
