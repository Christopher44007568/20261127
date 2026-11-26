package main

import (
	"fmt"
	"math"
	"os"
)

func main() {
	outDir := "out"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	scene := demoScene()
	tl := Timeline{Duration: 12, FPS: 4, Shutter: 0.75, ShutterSteps: 6}

	report, err := Export(scene, tl, outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "export failed:", err)
		os.Exit(1)
	}

	printSummary(report, outDir)
	if len(report.Collisions) > 0 || len(report.OutOfBounds) > 0 {
		// Non-zero status makes the conflict visible to automated pipelines.
		os.Exit(2)
	}
}

// demoScene intentionally includes an out-of-bounds camera segment, a camera
// spline that passes through an obstacle, and character pauses during which
// the camera must keep moving.
func demoScene() Scene {
	bounds := Rect{Min: Vec2{0, 0}, Max: Vec2{100, 75}}
	obstacles := []Obstacle{
		{Name: "pillar-a", Rect: Rect{Min: Vec2{18, 30}, Max: Vec2{28, 40}}},
		{Name: "wall-b", Rect: Rect{Min: Vec2{55, 35}, Max: Vec2{70, 48}}},
	}
	camera := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{10, 12}, Heading: 0, Zoom: 1},
		{Time: 3, Pos: Vec2{-3, 30}, Heading: math.Pi / 8, Zoom: 1.1}, // leaves bounds (left edge)
		{Time: 5, Pos: Vec2{25, 35}, Heading: math.Pi / 4, Zoom: 1.2}, // crosses pillar-a
		{Time: 7, Pos: Vec2{62, 42}, Heading: math.Pi / 6, Zoom: 1},   // crosses wall-b
		{Time: 10, Pos: Vec2{60, 60}, Heading: 0, Zoom: 0.9},
		{Time: 12, Pos: Vec2{35, 50}, Heading: -math.Pi / 10, Zoom: 1},
	}}
	characters := []CharacterTrack{
		{Name: "hero", Keys: []CharacterKey{
			{Time: 0, Pos: Vec2{8, 8}},
			{Time: 4, Pos: Vec2{30, 18}},
			{Time: 7, Pos: Vec2{30, 18}}, // pause: camera still moves 4s->7s
			{Time: 10, Pos: Vec2{50, 45}},
			{Time: 12, Pos: Vec2{50, 45}}, // second pause
		}},
		{Name: "scout", Keys: []CharacterKey{
			{Time: 0, Pos: Vec2{90, 10}},
			{Time: 6, Pos: Vec2{75, 25}},
			{Time: 9, Pos: Vec2{75, 25}}, // pause shared on the same timeline
			{Time: 12, Pos: Vec2{45, 30}},
		}},
	}
	renderer := Renderer{
		Width: 320, Height: 240, Bounds: bounds, Obstacles: obstacles,
		ViewWidth: 45,
	}
	return Scene{
		Camera: camera, Characters: characters, Obstacles: obstacles,
		Bounds: bounds, Renderer: renderer,
	}
}

func printSummary(r *Report, outDir string) {
	fmt.Printf("offscreen export: %d frames @ %.0f fps, %.1fs, shutter %.0f%% (%d steps)\n",
		r.Frames, r.FPS, r.Duration, r.Shutter*100, r.ShutterSteps)
	fmt.Printf("outputs in %s: thumbnails/*.png, offscreen.gif, path_map.png, report.json\n", outDir)
	for name, pauses := range r.CharacterPauses {
		for _, p := range pauses {
			fmt.Printf("character %s paused on shared timeline: %.2fs-%.2fs (camera continues)\n",
				name, p[0], p[1])
		}
	}
	if len(r.OutOfBounds) == 0 {
		fmt.Println("bounds: camera stayed inside the playable rectangle")
	}
	for _, ev := range r.OutOfBounds {
		fmt.Printf("CONFLICT out-of-bounds: camera outside bounds %.2fs-%.2fs\n",
			ev.EnterTime, ev.ExitTime)
	}
	if len(r.Collisions) == 0 {
		fmt.Println("collisions: none")
	}
	for _, ev := range r.Collisions {
		fmt.Printf("CONFLICT collision: camera crossed obstacle %q %.2fs-%.2fs, max penetration %.2f @ %.2fs\n",
			ev.Obstacle, ev.EnterTime, ev.ExitTime, ev.MaxDepth, ev.AtMaxTime)
	}
}
