package main

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

const (
	sheetGap    = 8
	sheetTop    = 28
	panelLabelY = 10
)

func toImage(buf []color.RGBA, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, c := range buf {
		o := i * 4
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, c.A
	}
	return img
}

// ContactSheet lays out current frame, history reprojection, velocity
// map and ghost mask in a 2x2 panel with bitmap labels.
func ContactSheet(f *Frame, reproj, vel, mask []color.RGBA, title string) *image.RGBA {
	W := ViewW*2 + sheetGap*3
	H := ViewH*2 + sheetGap*3 + sheetTop + 24
	sheet := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{6, 8, 14, 255}}, image.Point{}, draw.Src)

	labelColor := color.RGBA{225, 230, 240, 255}
	panels := []struct {
		buf   []color.RGBA
		label string
		x, y  int
	}{
		{f.Color, "1 CURRENT FRAME", sheetGap, sheetTop},
		{reproj, "2 HISTORY REPROJECTION", ViewW + 2*sheetGap, sheetTop},
		{vel, "3 SKINNED VELOCITY (RG)", sheetGap, ViewH + 2*sheetGap + sheetTop},
		{mask, "4 GHOST MASK", ViewW + 2*sheetGap, ViewH + 2*sheetGap + sheetTop},
	}
	for _, p := range panels {
		draw.Draw(sheet, image.Rect(p.x, p.y, p.x+ViewW, p.y+ViewH),
			toImage(p.buf, ViewW, ViewH), image.Point{}, draw.Src)
		drawText(sheet, strings.ToUpper(p.label), p.x, p.y-panelLabelY, 1, labelColor)
	}
	legend(sheet, sheetGap, H-16)
	drawText(sheet, strings.ToUpper(title), sheetGap, 6, 1, color.RGBA{240, 190, 70, 255})
	return sheet
}

// legend explains the velocity and mask color coding along the bottom.
func legend(img *image.RGBA, x, y int) {
	type swatch struct {
		c     color.Color
		label string
	}
	items := []swatch{
		{color.RGBA{128, 128, 0, 255}, "VEL ZERO"},
		{color.RGBA{255, 128, 0, 255}, "VEL +X"},
		{color.RGBA{128, 255, 0, 255}, "VEL +Y"},
		{color.RGBA{0, 0, 0, 255}, "HUD NO VEL"},
		{ghostColor, "GHOST TRAIL"},
		{disocclColor, "DISOCCLUSION"},
		{okTint, "VALID HISTORY"},
	}
	pen := x
	for _, it := range items {
		draw.Draw(img, image.Rect(pen, y, pen+8, y+8),
			&image.Uniform{it.c}, image.Point{}, draw.Src)
		drawText(img, it.label, pen+11, y+1, 1, color.RGBA{200, 205, 215, 255})
		pen += 11 + len(it.label)*(FontWidth+1) + 14
	}
}

// CompareSheet places the history reprojection from per-vertex skinned
// velocity next to the root-only baseline reprojection. The root-only
// panel visibly double-images the fast hand, while the skinned panel
// stays aligned.
func CompareSheet(skinned *Frame, skinnedReproj []color.RGBA, root *Frame, rootReproj []color.RGBA,
	skinnedGhost, rootGhost, frame int, handVel, rootHandVel float64) *image.RGBA {
	W := ViewW*2 + sheetGap*3
	H := ViewH + sheetTop + sheetGap + 14
	sheet := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.RGBA{6, 8, 14, 255}}, image.Point{}, draw.Src)
	drawText(sheet, fmt.Sprintf("FRAME %02d HISTORY REPROJECTION COMPARISON", frame),
		sheetGap, 6, 1, color.RGBA{240, 190, 70, 255})

	left := toImage(skinnedReproj, ViewW, ViewH)
	right := toImage(rootReproj, ViewW, ViewH)
	overlaySilhouette(left, skinned)
	overlaySilhouette(right, root)
	draw.Draw(sheet, image.Rect(sheetGap, sheetTop, sheetGap+ViewW, sheetTop+ViewH),
		left, image.Point{}, draw.Src)
	draw.Draw(sheet, image.Rect(ViewW+2*sheetGap, sheetTop, ViewW*2+2*sheetGap, sheetTop+ViewH),
		right, image.Point{}, draw.Src)
	drawText(sheet, fmt.Sprintf("PER VERTEX SKINNED  HAND VEL %.2f  GHOST %d", handVel, skinnedGhost),
		sheetGap, sheetTop-panelLabelY, 1,
		color.RGBA{225, 230, 240, 255})
	drawText(sheet, fmt.Sprintf("ROOT ONLY BASELINE  HAND VEL %.2f  GHOST %d", rootHandVel, rootGhost),
		ViewW+2*sheetGap, sheetTop-panelLabelY, 1,
		color.RGBA{225, 230, 240, 255})
	return sheet
}

// overlaySilhouette draws a white outline of the current character so
// reprojection misalignment is directly visible against the history.
func overlaySilhouette(img *image.RGBA, f *Frame) {
	edge := color.RGBA{245, 245, 250, 255}
	for y := 1; y < f.H-1; y++ {
		for x := 1; x < f.W-1; x++ {
			i := f.idx(x, y)
			if !f.Cover[i] || f.Hud[i] {
				continue
			}
			if !f.Cover[f.idx(x-1, y)] || !f.Cover[f.idx(x+1, y)] ||
				!f.Cover[f.idx(x, y-1)] || !f.Cover[f.idx(x, y+1)] {
				img.Set(x, y, edge)
			}
		}
	}
}

func drawText(img *image.RGBA, text string, x, y, scale int, c color.Color) {
	pen := x
	for _, r := range text {
		g, ok := FontGlyphs[r]
		if !ok {
			pen += 4 * scale
			continue
		}
		for row := 0; row < FontHeight; row++ {
			for col := 0; col < FontWidth; col++ {
				if g[row]&(1<<(FontWidth-1-col)) != 0 {
					px := pen + col*scale
					py := y + row*scale
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							img.Set(px+dx, py+dy, c)
						}
					}
				}
			}
		}
		pen += (FontWidth + 1) * scale
	}
}

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return png.Encode(fh, img)
}

// SaveGIF quantizes each panel image to the built-in Plan 9 palette;
// no third-party encoder is involved.
func SaveGIF(path string, frames []*image.RGBA, delayCS int) error {
	g := &gif.GIF{LoopCount: 0}
	for _, src := range frames {
		b := src.Bounds()
		dst := image.NewPaletted(b, palette.Plan9)
		draw.Draw(dst, b, src, b.Min, draw.Over)
		g.Image = append(g.Image, dst)
		g.Delay = append(g.Delay, delayCS)
	}
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return gif.EncodeAll(fh, g)
}
