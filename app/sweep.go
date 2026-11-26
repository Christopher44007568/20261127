package app

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"

	"coverlab/cover"
	"coverlab/scenes"
	"coverlab/viz"
)

func sweepIndex(scene cover.Scene) int {
	return scenes.SweepPixelY*scene.W + scenes.SweepPixelX
}

func coverageAt(scene cover.Scene, method string) float64 {
	_, alpha, _ := cover.RenderMethod(scene, method, viz.White)
	return alpha[sweepIndex(scene)]
}

func coarseDys() []float64 {
	dys := []float64{}
	for i := -10; i <= 10; i++ {
		dys = append(dys, float64(i)/10)
	}
	return dys
}

func toPoints(dys []float64, cov func(dy float64) float64) []Point {
	pts := make([]Point, 0, len(dys))
	for _, dy := range dys {
		c := cov(dy)
		pts = append(pts, Point{Dy: dy, Coverage: c, Gray255: 255 * (1 - c)})
	}
	return pts
}

func maxJump(pts []Point) float64 {
	mj := 0.0
	for i := 1; i < len(pts); i++ {
		if d := absf(pts[i].Coverage - pts[i-1].Coverage); d > mj {
			mj = d
		}
	}
	return mj
}

func renderSweep(outDir string) (SweepSummary, error) {
	dys := coarseDys()
	sum := SweepSummary{
		PixelX:   scenes.SweepPixelX,
		PixelY:   scenes.SweepPixelY,
		Step:     0.1,
		Analytic: toPoints(dys, func(dy float64) float64 { return coverageAt(scenes.SweepScene(dy), mAnalytic) }),
		SS2:      toPoints(dys, func(dy float64) float64 { return coverageAt(scenes.SweepScene(dy), mSS2) }),
		SS3:      toPoints(dys, func(dy float64) float64 { return coverageAt(scenes.SweepScene(dy), mSS3) }),
		MaxJump:  map[string]float64{},
	}
	sum.MaxJump[mAnalytic] = maxJump(sum.Analytic)
	sum.MaxJump[mSS2] = maxJump(sum.SS2)
	sum.MaxJump[mSS3] = maxJump(sum.SS3)

	if err := writeSweepCSV(outDir, sum); err != nil {
		return sum, err
	}
	if err := writeSweepChart(outDir, sum); err != nil {
		return sum, err
	}
	return sum, writeSweepStrip(outDir)
}

func writeSweepCSV(outDir string, sum SweepSummary) error {
	p := filepath.Join(outDir, "sweep", "sweep.csv")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintln(f, "dy,analytic,ss2,ss3")
	for i := range sum.Analytic {
		fmt.Fprintf(f, "%.1f,%.6f,%.6f,%.6f\n",
			sum.Analytic[i].Dy, sum.Analytic[i].Coverage, sum.SS2[i].Coverage, sum.SS3[i].Coverage)
	}
	return nil
}

func writeSweepChart(outDir string, sum SweepSummary) error {
	dense := []float64{}
	for i := -50; i <= 50; i++ {
		dense = append(dense, float64(i)/50)
	}
	anaDense := toPoints(dense, func(dy float64) float64 { return coverageAt(scenes.SweepScene(dy), mAnalytic) })

	toXY := func(pts []Point) [][2]float64 {
		out := make([][2]float64, len(pts))
		for i, p := range pts {
			out[i] = [2]float64{p.Dy, p.Coverage}
		}
		return out
	}
	chart := viz.LineChart(
		"THIN LINE SUBPIXEL SWEEP: FIXED-PIXEL COVERAGE",
		"LINE CENTER OFFSET DY (PX, STEP 0.1)", "COVERAGE (ALPHA)",
		-1, 1, 0, 1.0,
		[]viz.Series{
			{Name: "ANALYTIC", Color: color.NRGBA{R: 18, G: 18, B: 24, A: 255}, Pts: toXY(anaDense), Width: 4},
			{Name: "SS 2X2", Color: color.NRGBA{R: 31, G: 119, B: 180, A: 255}, Pts: toXY(sum.SS2), Step: true, Square: true, Width: 2},
			{Name: "SS 3X3", Color: color.NRGBA{R: 214, G: 39, B: 40, A: 255}, Pts: toXY(sum.SS3), Step: true, Width: 2},
		})
	return viz.SavePNG(filepath.Join(outDir, "sweep", "continuity_chart.png"), chart)
}

func writeSweepStrip(outDir string) error {
	const cell = 28
	rowDys := []float64{}
	for i := -5; i <= 5; i++ {
		rowDys = append(rowDys, float64(i)/10)
	}
	methods := []string{mAnalytic, mSS2, mSS3}
	base := scenes.SweepScene(0)
	x0c, y0c, x1c, y1c, cw, ch := cropOf(base)
	_ = x0c
	_ = y0c
	_ = x1c
	_ = y1c
	tileW, tileH := cw*cell, ch*cell
	const pad = 18
	const labelW = 92
	const colGap = 26
	headerH := 64
	rowGap := 12
	W := pad + labelW + len(methods)*tileW + (len(methods)-1)*colGap + pad
	H := headerH + len(rowDys)*(tileH+rowGap) + pad
	img := image.NewNRGBA(image.Rect(0, 0, W, H))
	viz.Fill(img, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	viz.DrawText(img, pad, 14, 3, "SUBPIXEL SWEEP / RED BOX = TRACKED PIXEL",
		color.NRGBA{R: 20, G: 20, B: 26, A: 255})

	mark := image.Pt(scenes.SweepPixelX, scenes.SweepPixelY)
	for col, m := range methods {
		hx := pad + labelW + col*(tileW+colGap)
		viz.DrawText(img, hx+tileW/2-viz.TextWidth(methodLabel(m), 2)/2, 40, 2,
			methodLabel(m), color.NRGBA{R: 40, G: 42, B: 52, A: 255})
		for row, dy := range rowDys {
			scene := scenes.SweepScene(dy)
			rgb, alpha, _ := cover.RenderMethod(scene, m, viz.White)
			tile := viz.ColorTile(scene, rgb, cell, &mark)
			x := hx
			y := headerH + row*(tileH+rowGap)
			draw.Draw(img, image.Rect(x, y, x+tileW, y+tileH), tile, image.Point{}, draw.Over)
			c := alpha[sweepIndex(scene)]
			viz.Label(img, x+6, y+8, 2, fmt.Sprintf("C %.2f", c),
				color.NRGBA{R: 20, G: 20, B: 26, A: 255},
				color.NRGBA{R: 255, G: 255, B: 255, A: 210})
			if col == 0 {
				ly := y + tileH/2 - 6
				viz.DrawText(img, pad, ly, 2, fmt.Sprintf("DY %+.1f", dy),
					color.NRGBA{R: 60, G: 62, B: 72, A: 255})
			}
		}
	}
	return viz.SavePNG(filepath.Join(outDir, "sweep", "continuity_strip.png"), img)
}

func cropOf(scene cover.Scene) (x0, y0, x1, y1, cw, ch int) {
	c := scene.Crop
	return c[0], c[1], c[2], c[3], c[2] - c[0], c[3] - c[1]
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
