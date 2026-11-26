package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	src := NewSourceSkeleton()
	dst := NewTargetSkeleton()

	sourcePoses := GenerateSourceMotion(src)
	targetPoses := RetargetMotion(src, dst, sourcePoses)

	frames := make([]Frame, FrameCount)
	for f := 0; f < FrameCount; f++ {
		rate := 0.0
		if f > 0 {
			rate = sourcePoses[f].Heading - sourcePoses[f-1].Heading
		}
		frames[f] = Frame{
			Source:   sourcePoses[f],
			Target:   targetPoses[f],
			TurnRate: rate,
		}
	}

	renderer := NewRenderer()
	buffers := make([]*FrameBuffer, FrameCount)
	for f := 0; f < FrameCount; f++ {
		buffers[f] = renderer.renderFrame(frames, f)
	}

	outDir := "artifacts"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	gifBytes, err := EncodeGIF(buffers, 3) // ~30 fps
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "retarget.gif"), gifBytes, 0o644); err != nil {
		fatal(err)
	}

	// Persist a mid-turn frame as a still preview.
	mid := (TurnStartF + TurnEndF) / 2
	if err := writePNG(filepath.Join(outDir, "frame_mid_turn.png"), buffers[mid]); err != nil {
		fatal(err)
	}
	if err := writePNG(filepath.Join(outDir, "frame_start.png"), buffers[0]); err != nil {
		fatal(err)
	}
	if err := writePNG(filepath.Join(outDir, "frame_late.png"), buffers[FrameCount-16]); err != nil {
		fatal(err)
	}

	csvPath := filepath.Join(outDir, "metrics.csv")
	if err := WriteMetricsCSV(csvPath, src, dst, frames); err != nil {
		fatal(err)
	}

	fmt.Printf("wrote %s/retarget.gif (%d frames), 3 preview PNGs, metrics.csv\n",
		outDir, FrameCount)
	fmt.Printf("source leg %.3fm, target leg %.3fm, root scale %.3f\n",
		src.legLength(), dst.legLength(), dst.legLength()/src.legLength())
}

func writePNG(path string, fb *FrameBuffer) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, fb.W, fb.H))
	copy(img.Pix, fb.Pix)
	return png.Encode(f, img)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
