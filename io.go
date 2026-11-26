package skinsss

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// WritePNG encodes an image to path, creating parent directories as needed.
func WritePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ComposeGrid arranges a set of same-sized square images in a cols x rows grid
// with a thin dark gutter between tiles.
func ComposeGrid(tiles []*image.RGBA, cols, rows, gutter int) *image.RGBA {
	tw, th := tiles[0].Bounds().Dx(), tiles[0].Bounds().Dy()
	w := cols*tw + (cols+1)*gutter
	h := rows*th + (rows+1)*gutter
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, tile := range tiles {
		col := i % cols
		row := i / cols
		ox := gutter + col*(tw+gutter)
		oy := gutter + row*(th+gutter)
		copyTile(out, tile, ox, oy)
	}
	return out
}

func copyTile(dst *image.RGBA, src *image.RGBA, ox, oy int) {
	sb := src.Bounds()
	for y := sb.Min.Y; y < sb.Max.Y; y++ {
		for x := sb.Min.X; x < sb.Max.X; x++ {
			dst.Set(ox+x, oy+y, src.RGBAAt(x, y))
		}
	}
}
