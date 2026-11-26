package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
)

// Scene is everything the offscreen renderer sweeps along the shared timeline.
type Scene struct {
	Camera     CameraPath
	Characters []CharacterTrack
	Obstacles  []Obstacle
	Bounds     Rect
	Renderer   Renderer
}

// FrameReport is the JSON record of one exported frame.
type FrameReport struct {
	Index         int      `json:"index"`
	Time          float64  `json:"time"`
	ShutterOpen   float64  `json:"shutterOpen"`
	ShutterClose  float64  `json:"shutterClose"`
	CameraPos     Vec2     `json:"cameraPos"`
	CameraHeading float64  `json:"cameraHeading"`
	CameraZoom    float64  `json:"cameraZoom"`
	CharacterPos  []Vec2   `json:"characterPositions"`
	InsideBounds  bool     `json:"cameraInsideBounds"`
	Collisions    []string `json:"collisions"`
}

// Report is the full machine-readable export summary.
type Report struct {
	Frames          int                     `json:"frameCount"`
	FPS             float64                 `json:"fps"`
	Duration        float64                 `json:"durationSeconds"`
	Shutter         float64                 `json:"shutterFraction"`
	ShutterSteps    int                     `json:"shutterSteps"`
	FrameReports    []FrameReport           `json:"frames"`
	Collisions      []CollisionEvent        `json:"collisions"`
	OutOfBounds     []OutOfBoundsEvent      `json:"outOfBounds"`
	CharacterPauses map[string][][2]float64 `json:"characterPauses"`
}

// Export renders every frame, writes PNG thumbnails, the animated GIF, the
// path map and the JSON report into outDir, and returns the report.
func Export(s Scene, tl Timeline, outDir string) (*Report, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	count := tl.FrameCount()
	anim := &gif.GIF{LoopCount: 0}
	report := &Report{
		Frames: count, FPS: tl.FPS, Duration: tl.Duration,
		Shutter: tl.Shutter, ShutterSteps: tl.ShutterSteps,
		CharacterPauses: map[string][][2]float64{},
	}

	thumbDir := filepath.Join(outDir, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return nil, err
	}

	for i := 0; i < count; i++ {
		t := tl.FrameTime(i)
		open, close := tl.ShutterInterval(i)

		// Camera and characters read the same timestamp: one shared timeline.
		cam := s.Camera.At(t)
		ghosts := map[string][]Vec2{}
		centers := map[string]Vec2{}
		for _, tr := range s.Characters {
			pts := make([]Vec2, 0, tl.ShutterSteps)
			for _, st := range tl.ShutterTimes(i) {
				pts = append(pts, tr.At(st))
			}
			ghosts[tr.Name] = pts
			centers[tr.Name] = tr.At(t)
		}

		img := s.Renderer.RenderFrame(FrameInput{
			Index: i, FrameCount: count, Time: t,
			Camera: cam, Ghosts: ghosts, Centers: centers,
		})
		if err := writePNG(filepath.Join(thumbDir, fmt.Sprintf("frame_%04d.png", i)), img); err != nil {
			return nil, err
		}
		anim.Image = append(anim.Image, quantize(img))
		anim.Delay = append(anim.Delay, int(100.0/tl.FPS+0.5))

		report.FrameReports = append(report.FrameReports, buildFrameReport(s, i, t, open, close, cam, centers))
	}

	gifPath := filepath.Join(outDir, "offscreen.gif")
	if err := writeGIF(gifPath, anim); err != nil {
		return nil, err
	}

	mapper := PathMap{Width: 640, Height: 480, Margin: 30, Bounds: s.Bounds, Obstacles: s.Obstacles}
	mapImg := mapper.Build(s.Camera, s.Characters, s.Renderer, 480)
	if err := writePNG(filepath.Join(outDir, "path_map.png"), mapImg); err != nil {
		return nil, err
	}

	const sweepStep = 0.01
	report.Collisions = SweepCollision(s.Camera, s.Obstacles, tl.Duration, sweepStep)
	report.OutOfBounds = SweepBounds(s.Camera, s.Bounds, tl.Duration, sweepStep)
	for _, tr := range s.Characters {
		report.CharacterPauses[tr.Name] = tr.Pauses()
	}
	if err := writeJSON(filepath.Join(outDir, "report.json"), report); err != nil {
		return nil, err
	}
	return report, nil
}

func buildFrameReport(s Scene, i int, t, open, close float64, cam CameraKey, centers map[string]Vec2) FrameReport {
	fr := FrameReport{
		Index: i, Time: t, ShutterOpen: open, ShutterClose: close,
		CameraPos: cam.Pos, CameraHeading: cam.Heading, CameraZoom: cam.Zoom,
		InsideBounds: s.Bounds.Contains(cam.Pos),
	}
	for _, ob := range s.Obstacles {
		if penetration(cam.Pos, ob.Rect) > 0 {
			fr.Collisions = append(fr.Collisions, ob.Name)
		}
	}
	for _, tr := range s.Characters {
		fr.CharacterPos = append(fr.CharacterPos, centers[tr.Name])
	}
	return fr
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// quantize maps the true-color frame onto the stdlib Plan 9 palette for GIF.
func quantize(img *image.RGBA) *image.Paletted {
	pm := image.NewPaletted(img.Bounds(), palette.Plan9)
	draw.FloydSteinberg.Draw(pm, img.Bounds(), img, img.Bounds().Min)
	return pm
}

func writeGIF(path string, anim *gif.GIF) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, anim)
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
