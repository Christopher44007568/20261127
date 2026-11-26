// Package report renders acceptance figures (model, feature overlay and
// endpoint-error diagrams) and writes JSON/Markdown summaries.
package report

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"featurelines/internal/gfx"
)

// Panel dimensions.
const (
	PW = 980
	PH = 700
)

// Palette keeps all figures visually consistent.
var Palette = struct {
	BG     color.RGBA
	Ink    color.RGBA
	Muted  color.RGBA
	Grid   color.RGBA
	Blue   color.RGBA
	Green  color.RGBA
	Red    color.RGBA
	Orange color.RGBA
	Amber  color.RGBA
	Purple color.RGBA
	Gray   color.RGBA
	Panel  color.RGBA
}{
	BG:     color.RGBA{238, 242, 247, 255},
	Ink:    color.RGBA{28, 38, 54, 255},
	Muted:  color.RGBA{104, 116, 134, 255},
	Grid:   color.RGBA{214, 222, 232, 255},
	Blue:   color.RGBA{18, 88, 214, 255},
	Green:  color.RGBA{18, 158, 92, 255},
	Red:    color.RGBA{214, 48, 49, 255},
	Orange: color.RGBA{232, 126, 4, 255},
	Amber:  color.RGBA{240, 180, 20, 255},
	Purple: color.RGBA{122, 64, 200, 255},
	Gray:   color.RGBA{150, 160, 172, 255},
	Panel:  color.RGBA{255, 255, 255, 255},
}

func newScene(bg color.RGBA) *gfx.Scene {
	return gfx.NewScene(PW, PH, bg)
}

// header paints the title band shared by every panel.
func header(s *gfx.Scene, tag, title, subtitle string, accent color.RGBA) {
	img := s.Img
	draw.Draw(img, image.Rect(0, 0, PW, 78), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 78, PW, 80), &image.Uniform{Palette.Grid}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, 10, 78), &image.Uniform{accent}, image.Point{}, draw.Src)
	gfx.DrawText(img, tag, 30, 14, 2, accent)
	gfx.DrawText(img, title, 30, 40, 3, Palette.Ink)
	if subtitle != "" {
		gfx.DrawText(img, subtitle, 30+gfx.TextWidth(title, 3)+26, 50, 2, Palette.Muted)
	}
}

// footer writes a bottom metrics strip.
func footer(s *gfx.Scene, text string, bg color.RGBA) {
	y := PH - 40
	fillRect(s.Img, 0, y, PW, PH, bg)
	draw.Draw(s.Img, image.Rect(0, y, PW, y+2), &image.Uniform{Palette.Grid}, image.Point{}, draw.Src)
	gfx.DrawText(s.Img, text, 24, y+13, 2, Palette.Ink)
}

// SavePNG writes img to path, creating parent directories as needed.
func SavePNG(img *image.RGBA, path string) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

// fillRect alpha-composites c over the canvas and keeps every pixel opaque so
// PNG encoding never hits premultiplied/straight alpha conversion artifacts.
func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > img.Rect.Max.X {
		x1 = img.Rect.Max.X
	}
	if y1 > img.Rect.Max.Y {
		y1 = img.Rect.Max.Y
	}
	a := uint32(c.A)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := img.PixOffset(x, y)
			dr, dg, db := uint32(img.Pix[i]), uint32(img.Pix[i+1]), uint32(img.Pix[i+2])
			img.Pix[i] = uint8((uint32(c.R)*a + dr*(255-a)) / 255)
			img.Pix[i+1] = uint8((uint32(c.G)*a + dg*(255-a)) / 255)
			img.Pix[i+2] = uint8((uint32(c.B)*a + db*(255-a)) / 255)
			img.Pix[i+3] = 255
		}
	}
}
