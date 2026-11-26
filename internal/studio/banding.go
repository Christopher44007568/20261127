package studio

import (
	"fmt"
	"image"
)

// RampNRGBA produces the deliberately smooth test ramp shown in the
// diagnostic sheet. With dither=false each quantisation plateau becomes a
// visible vertical band; with dither=true the same ramp is noise-shaped.
func RampNRGBA(w, h int, dither bool) *image.NRGBA {
	im := NewImage(w, h)
	stops := []Stop{
		{Pos: 0.00, Color: Hex("303030")},
		{Pos: 0.52, Color: Hex("686868")},
		{Pos: 1.00, Color: Hex("a0a0a0")},
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := (float32(x) + 0.5) / float32(w)
			im.Set(x, y, Premultiply(sampleStops(stops, t)))
		}
	}
	return ToNRGBA(im, dither)
}

func packRGB(img *image.NRGBA, x, y int) uint32 {
	i := y*img.Stride + x*4
	return uint32(img.Pix[i])<<16 | uint32(img.Pix[i+1])<<8 | uint32(img.Pix[i+2])
}

// DetectBandMask flags horizontal plateaus at least minRun pixels long. A
// smooth dithered ramp has sub-minRun plateaus, while an un-dithered 8-bit
// ramp contains whole columns of repeated values.
func DetectBandMask(img *image.NRGBA, minRun int) ([]float32, int) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	mask := make([]float32, w*h)
	if minRun < 2 {
		minRun = 2
	}
	centerRuns := 0
	for y := 0; y < h; y++ {
		start := 0
		last := packRGB(img, 0, y)
		for x := 1; x <= w; x++ {
			var cur uint32
			if x < w {
				cur = packRGB(img, x, y)
			}
			if x < w && cur == last {
				continue
			}
			run := x - start
			if run >= minRun {
				for k := 0; k < 2; k++ {
					if start+k < x {
						mask[y*w+start+k] = 1
					}
					if x-1-k >= start {
						mask[y*w+x-1-k] = 1
					}
				}
				if y == h/2 {
					centerRuns++
				}
			}
			start = x
			last = cur
		}
	}
	return mask, centerRuns
}

func overlayMaskRed(canvas *FloatImage, x0, y0, w, h int, mask []float32, c RGBA, strength float32) {
	p := Premultiply(c)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m := clamp01f(mask[y*w+x] * strength)
			if m <= 0 {
				continue
			}
			q := RGBA{R: p.R * m, G: p.G * m, B: p.B * m, A: p.A * m}
			canvas.OverPixel(x0+x, y0+y, q)
		}
	}
}

func drawDimPanel(im *FloatImage, x, y, w, h int) {
	FillRect(im, x, y, w, h, Hex("0a0e16"))
	StrokeRect(im, x, y, w, h, Hex("7b8ca52e"))
}

// BandingSheet renders both the diagnosis and the seamless-tile proof. The
// final encode for this sheet is un-dithered so the tool's own quantisation
// does not hide the demonstration.
func BandingSheet(scene *Scene) (*FloatImage, map[string]float64) {
	im := NewImage(CanvasW, CanvasH)
	im.Clear(Hex("06090f"))

	white := Hex("eef5ff")
	muted := Hex("8394ad")
	accent := Hex("5be7d4")
	red := Hex("ff4265")
	DrawLabel(im, scene.Font, "BANDING + SEAM INSPECTION", 48, 42, 34, white)
	DrawLabel(im, scene.Font, "FLOAT COMPOSITE TO 8-BIT PNG / IGN + BAYER", 50, 92, 16, muted)

	const panelW, panelH = 736, 210
	leftX, rightX := 48, 816
	topY := 166
	drawDimPanel(im, leftX, topY, panelW, panelH)
	drawDimPanel(im, rightX, topY, panelW, panelH)
	DrawLabel(im, scene.Font, "RAW 8-BIT / VISIBLE BANDS", float32(leftX), 134, 16, red)
	DrawLabel(im, scene.Font, "DITHERED 8-BIT / NO BANDS", float32(rightX), 134, 16, accent)
	raw := RampNRGBA(panelW, panelH, false)
	dithered := RampNRGBA(panelW, panelH, true)
	DrawImageAt(im, raw, leftX, topY)
	DrawImageAt(im, dithered, rightX, topY)

	rawMask, rawRuns := DetectBandMask(raw, 8)
	ditherMask, ditherRuns := DetectBandMask(dithered, 8)
	detY := 445
	drawDimPanel(im, leftX, detY, panelW, 190)
	drawDimPanel(im, rightX, detY, panelW, 190)
	DrawImageAt(im, raw, leftX, detY)
	DrawImageAt(im, dithered, rightX, detY)
	BlendRect(im, leftX, detY, panelW, 190, Hex("000000b8"))
	BlendRect(im, rightX, detY, panelW, 190, Hex("000000b8"))
	DrawLabel(im, scene.Font, "DETECTED BAND EDGES / RAW", float32(leftX), 413, 16, red)
	DrawLabel(im, scene.Font, "DETECTED BAND EDGES / DITHERED", float32(rightX), 413, 16, accent)
	overlayMaskRed(im, leftX, detY, panelW, 190, rawMask, red, 1)
	overlayMaskRed(im, rightX, detY, panelW, 190, ditherMask, accent, 1)
	StrokeRect(im, leftX, detY, panelW, 190, Hex("7b8ca52e"))
	StrokeRect(im, rightX, detY, panelW, 190, Hex("7b8ca52e"))

	tileW, tileH, cols, rows := 320, 100, 3, 2
	tileImg := TileImage(tileW, tileH, cols, rows)
	tile8 := ToNRGBA(tileImg, true)
	tileX, tileY := 48, 735
	DrawImageAt(im, tile8, tileX, tileY)
	StrokeRect(im, tileX, tileY, tileW*cols, tileH*rows, Hex("9db8d92b"))
	for i := 1; i < cols; i++ {
		x := tileX + i*tileW
		BlendRect(im, x-1, tileY-12, 2, 12, accent)
	}
	DrawLabel(im, scene.Font, "SEAMLESS PERIODIC TILE / 3 X 2", float32(tileX), 698, 18, white)
	DrawLabel(im, scene.Font, "CYAN TICKS MARK SEAMS", float32(tileX), 946, 12, muted)

	floatSeam := tileBoundaryDelta(tileW, tileH)
	seam8 := seamResidual(ToNRGBA(tileImg, false), tileW, tileH, cols, rows)

	metricsX := 1062
	DrawLabel(im, scene.Font, "CHECKS", float32(metricsX), 704, 20, accent)
	lines := []string{
		fmt.Sprintf("RAW BANDS %d", rawRuns),
		fmt.Sprintf("DITHERED BANDS %d", ditherRuns),
		fmt.Sprintf("TILE DELTA %.1E", floatSeam),
		fmt.Sprintf("SEAM RESIDUAL %d LSB", seam8),
		"EDGE MODE PREMUL",
		"GRADIENT 8-BIT + DITHER",
	}
	for i, line := range lines {
		DrawLabel(im, scene.Font, line, float32(metricsX), 744+float32(i)*31, 17, white)
	}
	BlendRect(im, metricsX, 930, 438, 1, Hex("7b8ca52e"))
	DrawLabel(im, scene.Font, "SEAM DELTA IS MEASURED ON THE IDEAL FLOAT TILE", float32(metricsX), 950, 12, muted)

	stats := map[string]float64{
		"raw_band_runs_per_row": float64(rawRuns),
		"dithered_band_runs":    float64(ditherRuns),
		"tile_boundary_delta":   floatSeam,
		"seam_residual_lsb":     float64(seam8),
	}
	return im, stats
}

func seamResidual(img *image.NRGBA, tileW, tileH, cols, rows int) int {
	maxResidual := 0
	for row := 0; row < rows; row++ {
		for col := 1; col < cols; col++ {
			x := col * tileW
			for y := row * tileH; y < (row+1)*tileH; y++ {
				for ch := 0; ch < 3; ch++ {
					step := absByteDelta(img, y, x-1, x, ch)
					left := absByteDelta(img, y, x-2, x-1, ch)
					right := absByteDelta(img, y, x, x+1, ch)
					baseline := left
					if right > baseline {
						baseline = right
					}
					residual := step - baseline
					if residual > maxResidual {
						maxResidual = residual
					}
				}
			}
		}
	}
	return maxResidual
}

func absByteDelta(img *image.NRGBA, y, x0, x1, ch int) int {
	i0 := y*img.Stride + x0*4 + ch
	i1 := y*img.Stride + x1*4 + ch
	d := int(img.Pix[i0]) - int(img.Pix[i1])
	if d < 0 {
		return -d
	}
	return d
}
