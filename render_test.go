package qem

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderMaskPPMAndIdenticalSilhouette(t *testing.T) {
	mesh := NewDetailOrb(2)
	frame := NewViewFrame(mesh, 96, 96)
	view := View{Yaw: 0.4, Pitch: 0.1}
	mask := RenderMask(mesh, frame, view)
	if mask.PixelCount() < 500 {
		t.Fatalf("silhouette too small: %d", mask.PixelCount())
	}

	rows, overlays, summary := CompareSilhouettes(mesh, mesh, frame, []View{view, {Yaw: 1.2}})
	if len(rows) != 2 || len(overlays) != 2 {
		t.Fatalf("expected two metrics and overlays")
	}
	if summary.MeanAreaRatio != 1 || summary.MeanIoU != 1 {
		t.Fatalf("identical silhouette mismatch: %+v", summary)
	}
	if summary.MaxHausdorffPixels != 0 || summary.MeanChamferPixels != 0 {
		t.Fatalf("identical contour distance not zero: %+v", summary)
	}

	img := RenderShaded(mesh, frame, view)
	var buf bytes.Buffer
	if _, err := img.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	header := buf.String()
	if !strings.HasPrefix(header, "P6\n96 96\n255\n") {
		t.Fatalf("bad PPM header: %q", header[:20])
	}
}

func TestOBJRoundTripActiveGeometry(t *testing.T) {
	mesh := NewWavyPatch(5, 5)
	var buf bytes.Buffer
	if _, err := mesh.WriteOBJTo(&buf); err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseOBJ(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ActiveTriangleCount() != mesh.ActiveTriangleCount() {
		t.Fatalf("triangle count changed: %d vs %d", parsed.ActiveTriangleCount(), mesh.ActiveTriangleCount())
	}
}
