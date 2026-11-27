package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

var (
	colWhite  = color.RGBA{238, 244, 252, 255}
	colMuted  = color.RGBA{154, 168, 188, 255}
	colYellow = color.RGBA{255, 224, 92, 255}
	colCyan   = color.RGBA{91, 231, 255, 255}
	colRed    = color.RGBA{255, 116, 102, 255}
	colGreen  = color.RGBA{112, 240, 176, 255}
	colInk    = color.RGBA{8, 11, 17, 245}
	colBorder = color.RGBA{62, 76, 98, 255}
)

func drawGlobalHeader(img *image.RGBA, r *SceneResult) {
	fillRect(img, Rect{X: 0, Y: 0, W: imageW, H: 62}, color.RGBA{13, 18, 27, 255})
	drawText(img, 24, 18, r.Scene.Title, 2, colWhite)
	right := "PURE GO STDLIB  //  OCTREE + QEF + CRACK STITCH"
	drawText(img, imageW-textWidth(right, 1)-24, 14, right, 1, colCyan)
	sub := "same camera / same octree: left uses mass-point vertices, right uses normal tangent-plane positioning"
	drawText(img, imageW-textWidth(sub, 1)-24, 34, sub, 1, colMuted)
}

func drawPanelHUD(img *image.RGBA, p Rect, r *SceneResult, baseline bool) {
	hud := Rect{X: p.X + 12, Y: p.Y + p.H - 104, W: p.W - 24, H: 90}
	fillRect(img, hud, colInk)
	strokeRect(img, hud, colBorder)
	x := hud.X + 12
	y := hud.Y + 12

	if baseline {
		baseSharp := 0.0
		for _, b := range r.SharpBins {
			baseSharp = math.Max(baseSharp, b.BaselineMax)
		}
		if len(r.SharpBins) == 0 {
			baseSharp = r.BaselineMax
		}
		drawText(img, x, y, "BASELINE: AVERAGE EDGE CROSSINGS", 1, colRed)
		drawText(img, x, y+18, fmt.Sprintf("SHARP MAX |SDF| = %.6f", baseSharp), 2, colWhite)
		drawText(img, x, y+42, fmt.Sprintf("GLOBAL MAX = %.6f", r.BaselineMax), 1, colMuted)
		drawText(img, x, y+60, "ROUNDING/CHAMFER VISIBLE VS YELLOW TRUE EDGES", 1, colMuted)
	} else {
		mean := 0.0
		count := 0
		for _, b := range r.SharpBins {
			mean += b.SumSDF
			count += b.Count
		}
		if count > 0 {
			mean /= float64(count)
		}
		bin := ErrorBin{}
		if len(r.SharpBins) > 0 {
			bin = r.SharpBins[len(r.SharpBins)-1]
		}
		level := "NO CREASE / SMOOTH CONTROL"
		if len(r.SharpBins) > 0 {
			level = fmt.Sprintf("LOCAL LEVEL D%d  H=%.4f", bin.Depth, bin.H)
		}
		drawText(img, x, y, "NORMAL-QEF: TANGENT PLANES LOCATE CREASE", 1, colGreen)
		drawText(img, x, y+18, level, 2, colYellow)
		if r.Scene.Kind == KindSphere {
			drawText(img, x, y+42, fmt.Sprintf("MAX E=%.6f  V=%d Q=%d", r.MaxSDF, r.VertexCount, len(r.Top.Quads)), 1, colMuted)
			drawText(img, x, y+60, fmt.Sprintf("CRACK BOUNDARY %d -> %d", r.NaiveBoundary, r.FinalBoundary), 1, colMuted)
		} else {
			drawText(img, x, y+42, fmt.Sprintf("SHARP N=%d  MAX E=%.6f  MEAN=%.6f", count, r.MaxSDF, mean), 1, colWhite)
			drawText(img, x, y+60, fmt.Sprintf("NAIVE CRACKS=%d  FINAL=%d  PATCHES=%d", r.NaiveBoundary, r.FinalBoundary, r.Top.RepairTriangles), 1, colMuted)
		}
	}
}

func drawFooter(img *image.RGBA, r *SceneResult) {
	y := 752
	drawText(img, 24, y, "SHARP FEATURE LOCAL ERROR PLOTTED WITH LOCAL VOXEL H; REFINEMENT IS LOCAL, NOT HIDDEN BY GLOBAL 128^3 RESAMPLING", 1, colCyan)
	if len(r.SharpBins) == 0 {
		msg := fmt.Sprintf("%s: SMOOTH SPHERE CONTROL  MAX E=%.6f  H_FINEST=%.5f  LEAVES=%d", r.Scene.ID, r.MaxSDF, r.Oct.HFinest, r.LeafCount)
		drawText(img, 24, y+24, msg, 2, colWhite)
		return
	}
	x := 24
	for _, b := range r.SharpBins {
		msg := fmt.Sprintf("D%d H=%.4f N=%d MAXE=%.6f MEANE=%.6f BASEMAX=%.6f", b.Depth, b.H, b.Count, b.MaxSDF, b.MeanSDF(), b.BaselineMax)
		drawText(img, x, y+26, msg, 2, colYellow)
		x += textWidth(msg, 2) + 28
	}
	note := "YELLOW CROSSES = TRUE CORNER NEAREST-QEF VERTEX; LABELS SHOW E AND H"
	drawText(img, 24, y+58, note, 1, colMuted)
}

func drawReferenceFeatures(img *image.RGBA, r Rect, cam Camera, res *SceneResult, depth []float64, xray bool) {
	s := res.Scene
	if s.Kind == KindSphere {
		return
	}
	drawBoxEdgeNetwork(img, r, cam, depth, s.Half, colYellow, false, s.Clip, xray)
	if s.Kind == KindShell {
		inner := s.Half.Sub(V(s.Thickness, s.Thickness, s.Thickness))
		drawBoxEdgeNetwork(img, r, cam, depth, inner, colCyan, true, s.Clip, xray)
	}
}

func drawBoxEdgeNetwork(img *image.RGBA, vp Rect, cam Camera, depth []float64, b Vec3, c color.RGBA, inner, clip, xray bool) {
	for axis := 0; axis < 3; axis++ {
		a1 := (axis + 1) % 3
		a2 := (axis + 2) % 3
		for _, s1 := range []float64{-1, 1} {
			for _, s2 := range []float64{-1, 1} {
				var p0, p1 Vec3
				setCoordPtr(&p0, axis, -coord(b, axis))
				setCoordPtr(&p1, axis, coord(b, axis))
				setCoordPtr(&p0, a1, s1*coord(b, a1))
				setCoordPtr(&p1, a1, s1*coord(b, a1))
				setCoordPtr(&p0, a2, s2*coord(b, a2))
				setCoordPtr(&p1, a2, s2*coord(b, a2))
				drawLine3D(img, vp, cam, depth, p0, p1, c, clip, xray)
			}
		}
	}
}

func drawLine3D(img *image.RGBA, vp Rect, cam Camera, depth []float64, a, b Vec3, c color.RGBA, clip, xray bool) {
	pa, oka := cam.Project(a)
	pb, okb := cam.Project(b)
	if !oka || !okb {
		return
	}
	steps := int(math.Max(math.Abs(pb.X-pa.X), math.Abs(pb.Y-pa.Y))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		sx := pa.X + (pb.X-pa.X)*t
		sy := pa.Y + (pb.Y-pa.Y)*t
		w := pa.W + (pb.W-pa.W)*t
		world := a.Add(b.Sub(a).Mul(t))
		if clip && world.X+world.Z > 0.72 {
			continue
		}
		px, py := int(math.Round(sx)), int(math.Round(sy))
		if !vp.Contains(px, py) {
			continue
		}
		di := (py-vp.Y)*vp.W + (px - vp.X)
		if !xray && depth[di] > w+0.006 {
			continue
		}
		setPixelSafe(img, px, py, c)
		setPixelSafe(img, px+1, py, c)
		setPixelSafe(img, px, py+1, c)
	}
}

func setPixelSafe(img *image.RGBA, x, y int, c color.RGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetRGBA(x, y, c)
	}
}
