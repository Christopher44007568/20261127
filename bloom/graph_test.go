package bloom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGraphNegotiatesAndRendersEveryFormat(t *testing.T) {
	graph, err := BuildBloomGraph(64, 40)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := RenderReference(graph)
	if err != nil {
		t.Fatal(err)
	}
	var renders []*FormatRender
	for _, format := range AllFormats() {
		plan, err := graph.Negotiate(format)
		if err != nil {
			t.Fatalf("negotiate %s: %v", format, err)
		}
		for id, outputFormat := range plan.OutputFormats {
			if outputFormat != format {
				t.Fatalf("node %q selected %s for a %s graph", id, outputFormat, format)
			}
		}
		render, err := RenderFormat(graph, format, reference)
		if err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		for id, out := range render.Execution.Nodes {
			if out.Texture == nil || out.Texture.Format != format {
				t.Fatalf("node %q texture format = %v, want %s", id, out.Texture, format)
			}
		}
		if render.Execution.Final.Texture.Format != format {
			t.Fatalf("final format = %s, want %s", render.Execution.Final.Texture.Format, format)
		}
		if render.FinalError.MeanAbs < 0 || render.FinalError.NonFiniteChannels != 0 {
			t.Fatalf("invalid error stats for %s: %+v", format, render.FinalError)
		}
		if render.Execution.Range.ClampCount() == 0 {
			t.Fatalf("%s did not report source/range clamping", format)
		}
		renders = append(renders, render)
	}
	if renders[0].FinalError.MeanAbs >= renders[1].FinalError.MeanAbs {
		t.Fatalf("RGBA16F mean error %g should be below R11G11B10 %g",
			renders[0].FinalError.MeanAbs, renders[1].FinalError.MeanAbs)
	}
	if renders[2].FinalError.MeanAbs <= renders[1].FinalError.MeanAbs {
		t.Fatalf("sRGB8 mean error %g should exceed R11G11B10 %g",
			renders[2].FinalError.MeanAbs, renders[1].FinalError.MeanAbs)
	}
}

func TestNegotiationRejectsUnknownFormat(t *testing.T) {
	graph, err := BuildBloomGraph(32, 32)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := graph.Negotiate(FormatID("rgba32f")); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestGenerateWritesIsolatedArtifacts(t *testing.T) {
	root := t.TempDir()
	summary, err := Generate(root, 96, 54)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Formats) != len(AllFormats()) {
		t.Fatalf("got %d format reports, want %d", len(summary.Formats), len(AllFormats()))
	}
	seenHDR := make(map[string]FormatID)
	for _, report := range summary.Formats {
		dir := filepath.Join(root, string(report.Format))
		for _, name := range []string{
			"hdr_compare.pfm", "hdr_compare.png", "quantization_error.png", "reload.ppm", "stats.json",
		} {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Fatalf("%s/%s missing: %v", report.Format, name, err)
			}
		}
		if report.PPMReload.MismatchCount != 0 {
			t.Fatalf("%s PPM reload mismatches = %d", report.Format, report.PPMReload.MismatchCount)
		}
		if report.PPMReload.MaxVal != 65535 || report.PPMReload.Width != 96 || report.PPMReload.Height != 54 {
			t.Fatalf("%s PPM metadata is wrong: %+v", report.Format, report.PPMReload)
		}
		hdr, err := ReadPFM(filepath.Join(dir, "hdr_compare.pfm"))
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Width != 96 || hdr.Height != 54 {
			t.Fatalf("%s PFM is %dx%d", report.Format, hdr.Width, hdr.Height)
		}
		stats, err := os.ReadFile(filepath.Join(dir, "stats.json"))
		if err != nil {
			t.Fatal(err)
		}
		var decoded FormatArtifactReport
		if err := json.Unmarshal(stats, &decoded); err != nil {
			t.Fatalf("%s stats.json: %v", report.Format, err)
		}
		if decoded.Format != report.Format {
			t.Fatalf("%s stats reported format %s", report.Format, decoded.Format)
		}
		if previous, ok := seenHDR[report.PPMReload.SHA256]; ok {
			t.Fatalf("%s and %s produced the same reload bitmap", previous, report.Format)
		}
		seenHDR[report.PPMReload.SHA256] = report.Format
	}
	if _, err := os.Stat(filepath.Join(root, "summary.json")); err != nil {
		t.Fatal(err)
	}
}

type unsupportedOutputNode struct {
	NodeBase
}

func (unsupportedOutputNode) InputCount() int { return 0 }

func (unsupportedOutputNode) SupportsOutput(f FormatID) bool { return f == FormatRGBA16F }

func (unsupportedOutputNode) Evaluate(_ []*Image, _ FormatID) (*Image, error) {
	return NewImage(1, 1)
}

func TestNegotiationChecksNodeCapabilities(t *testing.T) {
	graph := NewGraph("source")
	if err := graph.AddNode("source", unsupportedOutputNode{NodeBase{KindName: "limited"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := graph.Negotiate(FormatSRGB8); err == nil {
		t.Fatal("expected node output capability failure")
	}
}
