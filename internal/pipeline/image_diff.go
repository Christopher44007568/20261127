package pipeline

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"materiallod/internal/chart"
	"materiallod/material"
)

func renderBrightnessDiffPNG(dir string, art *Artifacts) error {
	const width, height = 1120, 880
	cv := chart.New(width, height, chart.Hex(pageBG))
	cv.Text(34, 34, "BRIGHTNESS DIFFERENCE", 4, chart.Hex(ink))
	cv.Text(36, 82, "ALIGNED SPHERE LUMA AFTER SHARED TONEMAP - ABSOLUTE DIFFERENCE VS HIGH", 1, chart.Hex(muted))

	cv.TextCenter(300, 116, "LOW - HIGH", 2, chart.Hex(lowColor))
	cv.TextCenter(560, 116, "MEDIUM - HIGH", 2, chart.Hex(midColor))
	drawColorBar(cv, 1052, 150, 20, 190)

	for row, mat := range art.Materials {
		y := 142 + row*238
		cv.Text(38, y+72, panelTitle(mat.Kind), 3, chart.Hex(ink))
		cv.Text(40, y+118, mat.ID, 1, chart.Hex(muted))

		bufs := art.Spheres[mat.ID]
		lowDiff := material.DiffImage(bufs[0], bufs[2], DiffFullScale)
		midDiff := material.DiffImage(bufs[1], bufs[2], DiffFullScale)
		drawSphereAt(cv, lowDiff, 210, y)
		drawSphereAt(cv, midDiff, 470, y)

		lowStats := findStat(art.Brightness, mat.ID, "low_vs_high")
		midStats := findStat(art.Brightness, mat.ID, "medium_vs_high")
		x := 720
		cv.Text(x, y+8, "LOW-HIGH", 2, chart.Hex(ink))
		cv.Text(x, y+38, fmt.Sprintf("MAE  %.4f", lowStats.MAE), 1, chart.Hex(muted))
		cv.Text(x, y+56, fmt.Sprintf("RMSE %.4f", lowStats.RMSE), 1, chart.Hex(muted))
		cv.Text(x, y+74, fmt.Sprintf("MAX  %.4f", lowStats.MaxAbs), 1, chart.Hex(muted))

		cv.Text(x, y+112, "MID-HIGH", 2, chart.Hex(ink))
		cv.Text(x, y+142, fmt.Sprintf("MAE  %.4f", midStats.MAE), 1, chart.Hex(muted))
		cv.Text(x, y+160, fmt.Sprintf("RMSE %.4f", midStats.RMSE), 1, chart.Hex(muted))
		cv.Text(x, y+178, fmt.Sprintf("MAX  %.4f", midStats.MaxAbs), 1, chart.Hex(muted))
	}

	return savePNG(dir, "brightness_diff.png", cv.Img)
}

func drawSphereAt(cv *chart.Canvas, sphere image.Image, x, y int) {
	cv.Rect(x-8, y-8, x+material.SphereSize+8, y+material.SphereSize+8, chart.Hex(cardBG))
	draw.Draw(cv.Img, image.Rect(x, y, x+sphere.Bounds().Dx(), y+sphere.Bounds().Dy()),
		sphere, image.Point{}, draw.Over)
}

func findStat(stats []material.BrightnessStats, id, compare string) material.BrightnessStats {
	for _, s := range stats {
		if s.MaterialID == id && s.Compare == compare {
			return s
		}
	}
	return material.BrightnessStats{}
}

func drawColorBar(cv *chart.Canvas, x, y, w, h int) {
	cv.Text(x-4, y-28, "ABS", 1, chart.Hex(muted))
	for row := 0; row < h; row++ {
		t := 1 - float64(row)/float64(h-1)
		c := material.HeatColor(t)
		cv.Rect(x, y+row, x+w, y+row+1, colorRGBA(c))
	}
	cv.Frame(x, y, x+w, y+h, chart.Hex(0xC8D0DF), 1)
	cv.Text(x+w+8, y-4, f2(DiffFullScale), 1, chart.Hex(muted))
	cv.Text(x+w+8, y+h-8, "0", 1, chart.Hex(muted))
}

func colorRGBA(c material.Color) color.RGBA {
	return color.RGBA{R: byte(c.R * 255), G: byte(c.G * 255), B: byte(c.B * 255), A: 255}
}
