package main

import (
	"image"
	"image/color"
)

// NaiveRotatedTiling is the anti-pattern: each tile cell gets a random
// 90-degree rotation / flip. The tile is square so indexing stays in
// bounds, but running-bond bricks turn vertical and cross-cell joints
// are destroyed. It exists only as a visual control.
func NaiveRotatedTiling(tile *image.RGBA, w, h int, seed uint64) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	cellsX, cellsY := w/TileW, h/TileH
	for cy := 0; cy < cellsY; cy++ {
		for cx := 0; cx < cellsX; cx++ {
			cs := splitmix64(seed ^ uint64(cx)*73856093 ^ uint64(cy)*19349663)
			rot := int(frand(cs, 1)*4) % 4
			flip := frand(cs, 2) < 0.5
			for ly := 0; ly < TileH; ly++ {
				for lx := 0; lx < TileW; lx++ {
					sx, sy := lx, ly
					if flip {
						sx = TileW - 1 - sx
					}
					switch rot {
					case 1: // 90 clockwise
						sx, sy = TileW-1-ly, lx
					case 2:
						sx, sy = TileW-1-lx, TileH-1-ly
					case 3: // 270 clockwise
						sx, sy = ly, TileH-1-lx
					}
					c := tile.RGBAAt(sx, sy)
					out.SetRGBA(cx*TileW+lx, cy*TileH+ly,
						color.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
				}
			}
		}
	}
	return out
}
