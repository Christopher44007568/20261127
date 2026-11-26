package render

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

type Panel struct {
	Image *image.RGBA
	Label string
}

// ComposePanels places renders side by side with readable captions. It keeps
// captions outside the image area so they never alter the rendered comparison.
func ComposePanels(panels []Panel, gap int) *image.RGBA {
	if len(panels) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	const header = 48
	cellWidth, imageHeight := 0, 0
	for _, panel := range panels {
		if panel.Image == nil {
			continue
		}
		if panel.Image.Bounds().Dx() > cellWidth {
			cellWidth = panel.Image.Bounds().Dx()
		}
		if panel.Image.Bounds().Dy() > imageHeight {
			imageHeight = panel.Image.Bounds().Dy()
		}
	}
	totalWidth := len(panels)*cellWidth + gap*(len(panels)-1)
	canvas := image.NewRGBA(image.Rect(0, 0, totalWidth, header+imageHeight))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{7, 10, 15, 255}}, image.Point{}, draw.Src)

	accent := color.RGBA{80, 164, 179, 255}
	ink := color.RGBA{221, 229, 232, 255}
	for i, panel := range panels {
		if panel.Image == nil {
			continue
		}
		x := i * (cellWidth + gap)
		drawRect(canvas, x, 0, cellWidth, 2, accent)
		label := panel.Label
		if label == "" {
			label = "UNTITLED"
		}
		scale := 2
		tx := x + (cellWidth-textWidth(label, scale))/2
		drawText(canvas, tx, 19, label, scale, ink)
		draw.Draw(canvas, image.Rect(x, header, x+panel.Image.Bounds().Dx(), header+panel.Image.Bounds().Dy()), panel.Image, panel.Image.Bounds().Min, draw.Src)
	}
	return canvas
}

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
