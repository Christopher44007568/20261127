package main

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type ImageComparison struct {
	TotalPixels     int
	DifferentPixels int
	MaxChannelDiff  int
	RMSE            float64
	PSNR            float64
	Exact           bool
}

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}

func SaveGIF(path string, frames []*image.RGBA, delay int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	anim := &gif.GIF{LoopCount: 0}
	for _, frame := range frames {
		paletted := image.NewPaletted(frame.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, frame.Bounds(), frame, image.Point{})
		anim.Image = append(anim.Image, paletted)
		anim.Delay = append(anim.Delay, delay)
		anim.Disposal = append(anim.Disposal, gif.DisposalBackground)
	}
	return gif.EncodeAll(f, anim)
}

func CompareImages(a, b *image.RGBA) ImageComparison {
	result := ImageComparison{Exact: true}
	bounds := a.Bounds().Intersect(b.Bounds())
	result.TotalPixels = bounds.Dx() * bounds.Dy()
	var squared float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ia := a.RGBAAt(x, y)
			ib := b.RGBAAt(x, y)
			d0 := absInt(int(ia.R) - int(ib.R))
			d1 := absInt(int(ia.G) - int(ib.G))
			d2 := absInt(int(ia.B) - int(ib.B))
			d := max(d0, max(d1, d2))
			if d != 0 {
				result.DifferentPixels++
				result.Exact = false
			}
			result.MaxChannelDiff = max(result.MaxChannelDiff, d)
			squared += float64(d0*d0 + d1*d1 + d2*d2)
		}
	}
	if result.TotalPixels > 0 {
		result.RMSE = math.Sqrt(squared / float64(result.TotalPixels*3))
	}
	if result.RMSE == 0 {
		result.PSNR = math.Inf(1)
	} else {
		result.PSNR = 20 * math.Log10(255/result.RMSE)
	}
	return result
}

func DiffImage(a, b *image.RGBA, gain float64) *image.RGBA {
	bounds := a.Bounds().Intersect(b.Bounds())
	out := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ia := a.RGBAAt(x, y)
			ib := b.RGBAAt(x, y)
			d := absInt(int(ia.R) - int(ib.R))
			d = max(d, absInt(int(ia.G)-int(ib.G)))
			d = max(d, absInt(int(ia.B)-int(ib.B)))
			v := uint8(clamp(float64(d)*gain, 0, 255))
			if d == 0 {
				v = 0
			}
			out.SetRGBA(x, y, color.RGBA{R: v, G: uint8(float64(v) * 0.18), B: uint8(float64(v) * 0.55), A: 255})
		}
	}
	return out
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func imageSize(img image.Image) string {
	b := img.Bounds()
	return fmt.Sprintf("%dx%d", b.Dx(), b.Dy())
}

func drawText(img *image.RGBA, x, y, scale int, text string, c color.RGBA) {
	cursor := x
	for _, r := range text {
		glyph, ok := font5x7[r]
		if !ok {
			glyph = font5x7[' ']
		}
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if glyph[row][col] != '1' {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						px, py := cursor+col*scale+sx, y+row*scale+sy
						if image.Pt(px, py).In(img.Bounds()) {
							img.SetRGBA(px, py, c)
						}
					}
				}
			}
		}
		cursor += 6 * scale
	}
}

func textWidth(text string, scale int) int {
	return len(text) * 6 * scale
}

func drawRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func BuildHeatmap(grid *FroxelGrid, cellPx int) *image.RGBA {
	cellPx = max(cellPx, 8)
	cols := 4
	rows := (grid.Config.Slices + cols - 1) / cols
	tileW := grid.Config.TilesX * cellPx
	tileH := grid.Config.TilesY * cellPx
	const captionH = 20
	const gap = 12
	const margin = 16
	const titleH = 48
	const legendH = 54
	panelW := tileW
	panelH := captionH + tileH
	width := margin*2 + cols*panelW + (cols-1)*gap
	height := titleH + rows*panelH + (rows-1)*gap + legendH
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	background := color.RGBA{R: 8, G: 11, B: 18, A: 255}
	drawRect(img, 0, 0, width, height, background)
	drawText(img, margin, 12, 2, "FROXEL LIGHT COUNT", color.RGBA{R: 225, G: 235, B: 245, A: 255})
	subtitle := fmt.Sprintf("TILES %d X %d   DEPTH SLICES %d   MAX %d",
		grid.Config.TilesX, grid.Config.TilesY, grid.Config.Slices, grid.Stats.MaxCount)
	drawText(img, margin, 31, 1, subtitle, color.RGBA{R: 110, G: 135, B: 165, A: 255})

	maxCount := max(grid.Stats.MaxCount, 1)
	for tz := 0; tz < grid.Config.Slices; tz++ {
		col := tz % cols
		row := tz / cols
		x0 := margin + col*(panelW+gap)
		y0 := titleH + row*(panelH+gap)
		caption := fmt.Sprintf("Z%02d", tz)
		if tz == 0 {
			caption += "  NEAR"
		} else if tz == grid.Config.Slices-1 {
			caption += "  FAR"
		}
		drawText(img, x0, y0, 1, caption, color.RGBA{R: 170, G: 195, B: 220, A: 255})
		for ty := 0; ty < grid.Config.TilesY; ty++ {
			for tx := 0; tx < grid.Config.TilesX; tx++ {
				cell, _ := grid.InspectCell(tx, ty, tz)
				px0 := x0 + tx*cellPx
				py0 := y0 + captionH + ty*cellPx
				fill := heatColor(float64(cell.Count) / float64(maxCount))
				drawRect(img, px0, py0, px0+cellPx-1, py0+cellPx-1, fill)
				if cell.Fallback {
					border := color.RGBA{R: 255, G: 70, B: 190, A: 255}
					drawRect(img, px0, py0, px0+cellPx, py0+2, border)
					drawRect(img, px0, py0+cellPx-2, px0+cellPx, py0+cellPx, border)
				}
				if cellPx >= 13 {
					textColor := color.RGBA{R: 8, G: 10, B: 15, A: 255}
					if cell.Count < maxCount/2 {
						textColor = color.RGBA{R: 225, G: 235, B: 240, A: 255}
					}
					label := fmt.Sprintf("%d", cell.Count)
					txText := px0 + (cellPx-textWidth(label, 1))/2
					tyText := py0 + (cellPx-7)/2
					drawText(img, txText, tyText, 1, label, textColor)
				}
			}
		}
	}
	legendY := height - legendH + 8
	drawText(img, margin, legendY, 1, "LIGHT COUNT", color.RGBA{R: 170, G: 190, B: 210, A: 255})
	barX := margin + 86
	barW := 240
	for x := 0; x < barW; x++ {
		t := float64(x) / float64(barW-1)
		drawRect(img, barX+x, legendY, barX+x+1, legendY+14, heatColor(t))
	}
	drawText(img, barX, legendY+20, 1, "0", color.RGBA{R: 150, G: 170, B: 190, A: 255})
	maxLabel := fmt.Sprintf("%d", grid.Stats.MaxCount)
	drawText(img, barX+barW-textWidth(maxLabel, 1), legendY+20, 1, maxLabel,
		color.RGBA{R: 220, G: 175, B: 120, A: 255})
	return img
}

func heatColor(t float64) color.RGBA {
	type stop struct {
		t float64
		c Vec3
	}
	stops := []stop{
		{0.0, V(0.035, 0.055, 0.12)},
		{0.2, V(0.02, 0.24, 0.55)},
		{0.4, V(0.0, 0.67, 0.70)},
		{0.62, V(0.70, 0.92, 0.30)},
		{0.82, V(1.0, 0.48, 0.08)},
		{1.0, V(0.93, 0.08, 0.13)},
	}
	t = clamp(t, 0, 1)
	for i := 1; i < len(stops); i++ {
		if t > stops[i].t {
			continue
		}
		u := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
		c := stops[i-1].c.Lerp(stops[i].c, u)
		return color.RGBA{R: uint8(c.X * 255), G: uint8(c.Y * 255), B: uint8(c.Z * 255), A: 255}
	}
	return color.RGBA{R: 237, G: 20, B: 33, A: 255}
}
