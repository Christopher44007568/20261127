package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
)

const frameCount = 48

type stats struct {
	frame          int
	maxHandSpeed   float64
	avgBodySpeed   float64
	hudVelocity    float64
	ghostPixels    int
	disocclPixels  int
	coveredPixels  int
	rootGhost      int
	handRegion     float64
	rootHandRegion float64
}

func main() {
	outDir := filepath.Join("artifacts")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}
	sk := NewSkeleton()
	mesh := BuildHumanoid(sk)

	var prevColor []color.RGBA
	var prevCover []bool
	var gifFrames []*image.RGBA
	snapshots := map[int]bool{
		0:              true,
		frameCount / 6: true,
		frameCount / 2: true,
		frameCount - 1: true,
	}
	var all []stats

	for f := 0; f < frameCount; f++ {
		pose := Animate(f, frameCount)
		prev := pose
		if f > 0 {
			prev = Animate(f-1, frameCount)
		}
		frame := RenderFrame(mesh, sk, pose, prev)
		DrawHud(frame)

		if prevColor == nil {
			prevColor = append([]color.RGBA{}, frame.Color...)
			prevCover = append([]bool{}, frame.Cover...)
		}
		reproj := Reproject(frame, prevColor, prevCover)
		mask := GhostMask(frame, prevCover)
		vel := VelocityImage(frame)
		st := measure(frame, mask)
		st.frame = f
		rootFrame := RenderRootBaseline(mesh, sk, pose, prev)
		DrawHud(rootFrame)
		rootReproj := Reproject(rootFrame, prevColor, prevCover)
		rootMask := GhostMask(rootFrame, prevCover)
		st.rootGhost = countColor(rootMask, ghostColor)
		rx0, ry0, rx1, ry1 := ViewW/2, ViewH/4, ViewW, ViewH
		st.handRegion = RegionSpeed(frame, rx0, ry0, rx1, ry1)
		st.rootHandRegion = RegionSpeed(rootFrame, rx0, ry0, rx1, ry1)
		all = append(all, st)

		title := fmt.Sprintf("FRAME %02d/%d  HAND %.1f  BODY %.2f PX  HUD VEL %.1f",
			f, frameCount, st.maxHandSpeed, st.avgBodySpeed, st.hudVelocity)
		sheet := ContactSheet(frame, reproj, vel, mask, title)
		gifFrames = append(gifFrames, nearestScale(sheet, 0.5))
		if snapshots[f] {
			if err := SavePNG(filepath.Join(outDir, fmt.Sprintf("frame_%02d.png", f)), sheet); err != nil {
				fatal(err)
			}
			compare := CompareSheet(frame, reproj, rootFrame, rootReproj,
				st.ghostPixels, st.rootGhost, f, st.handRegion, st.rootHandRegion)
			if err := SavePNG(filepath.Join(outDir, fmt.Sprintf("root_vs_skinned_%02d.png", f)), compare); err != nil {
				fatal(err)
			}
		}

		prevColor = append([]color.RGBA{}, frame.Color...)
		prevCover = append([]bool{}, frame.Cover...)
	}

	if err := SaveGIF(filepath.Join(outDir, "skinning_velocity.gif"), gifFrames, 4); err != nil {
		fatal(err)
	}
	writeReport(filepath.Join(outDir, "report.txt"), all)
	printSummary(all)
}

func measure(f *Frame, mask []color.RGBA) stats {
	var st stats
	var bodySum float64
	bodyN := 0
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			i := f.idx(x, y)
			if f.Hud[i] {
				st.hudVelocity += math.Hypot(f.VelX[i], f.VelY[i])
				continue
			}
			if !f.Cover[i] {
				if mask[i] == ghostColor {
					st.ghostPixels++
				}
				continue
			}
			st.coveredPixels++
			speed := math.Hypot(f.VelX[i], f.VelY[i])
			bodySum += speed
			bodyN++
			if speed > st.maxHandSpeed {
				st.maxHandSpeed = speed
			}
			switch mask[i] {
			case ghostColor:
				st.ghostPixels++
			case disocclColor:
				st.disocclPixels++
			}
		}
	}
	if bodyN > 0 {
		st.avgBodySpeed = bodySum / float64(bodyN)
	}
	return st
}

// nearestScale resizes an RGBA image with integer-friendly nearest
// sampling using only the standard library.
func nearestScale(src *image.RGBA, scale float64) *image.RGBA {
	b := src.Bounds()
	nw := int(float64(b.Dx()) * scale)
	nh := int(float64(b.Dy()) * scale)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := int(float64(y)/scale) + b.Min.Y
		for x := 0; x < nw; x++ {
			sx := int(float64(x)/scale) + b.Min.X
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func writeReport(path string, all []stats) {
	fh, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer fh.Close()
	fmt.Fprintln(fh, "frame,max_speed,avg_body_speed,hud_velocity,ghost,root_baseline_ghost,disocclusion,covered,hand_region_vel,root_hand_region_vel")
	for _, s := range all {
		fmt.Fprintf(fh, "%d,%.3f,%.3f,%.3f,%d,%d,%d,%d,%.3f,%.3f\n",
			s.frame, s.maxHandSpeed, s.avgBodySpeed, s.hudVelocity,
			s.ghostPixels, s.rootGhost, s.disocclPixels, s.coveredPixels,
			s.handRegion, s.rootHandRegion)
	}
}

func printSummary(all []stats) {
	var peakHand, avgBody, hud float64
	var ghost, rootGhost, disoccl int
	var hand, rootHand float64
	for _, s := range all {
		peakHand = math.Max(peakHand, s.maxHandSpeed)
		avgBody += s.avgBodySpeed
		hud += s.hudVelocity
		ghost += s.ghostPixels
		rootGhost += s.rootGhost
		disoccl += s.disocclPixels
		hand += s.handRegion
		rootHand += s.rootHandRegion
	}
	n := float64(len(all))
	fmt.Printf("frames=%d peak_hand_px=%.2f avg_body_px=%.3f hud_velocity=%.1f\n",
		len(all), peakHand, avgBody/n, hud)
	fmt.Printf("hand_region_skinned_px=%.3f hand_region_root_px=%.3f ratio=%.2fx ghost=%d disocclusion=%d\n",
		hand/n, rootHand/n, hand/math.Max(rootHand, 1e-9), ghost, disoccl)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
