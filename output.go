package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func toNRGBA(p pixel) color.NRGBA {
	clamp := func(v float32) uint8 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 255
		}
		return uint8(v*255 + 0.5)
	}
	return color.NRGBA{clamp(p.R), clamp(p.G), clamp(p.B), clamp(p.A)}
}

// writePNG saves a color surface as a PNG.
func writePNG(path string, surf *Surface) error {
	img := image.NewNRGBA(image.Rect(0, 0, surf.W, surf.H))
	for y := 0; y < surf.H; y++ {
		for x := 0; x < surf.W; x++ {
			img.SetNRGBA(x, y, toNRGBA(surf.color[y*surf.W+x]))
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// tileMapPNG renders the tile segmentation: each tile is tinted by its
// final-pass color store decision, with grid lines on tile boundaries.
//
//	green - tile stored (touched, contents consumed/presented)
//	gray  - store elided (untouched or dead store, safe optimization)
//	blue  - load elided as well (fully dead tile this pass)
func tileMapPNG(path string, fb *Framebuffer, dec [][]tileDecision) error {
	img := image.NewNRGBA(image.Rect(0, 0, fb.W, fb.H))
	last := dec[len(dec)-1]
	for ti := range fb.tiles {
		t := &fb.tiles[ti]
		var c color.NRGBA
		switch {
		case last[ti].store[AttColor] == StoreStore:
			c = color.NRGBA{40, 200, 90, 255}
		case last[ti].load[AttColor] == LoadDontCare:
			c = color.NRGBA{60, 90, 200, 255}
		default:
			c = color.NRGBA{120, 120, 120, 255}
		}
		for y := t.Bounds.MinY; y < t.Bounds.MaxY; y++ {
			for x := t.Bounds.MinX; x < t.Bounds.MaxX; x++ {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	for ti := range fb.tiles {
		t := &fb.tiles[ti]
		for x := t.Bounds.MinX; x < t.Bounds.MaxX; x++ {
			img.SetNRGBA(x, t.Bounds.MinY, color.NRGBA{0, 0, 0, 255})
		}
		for y := t.Bounds.MinY; y < t.Bounds.MaxY; y++ {
			img.SetNRGBA(t.Bounds.MinX, y, color.NRGBA{0, 0, 0, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// errorMapPNG highlights, per tile, pixels where the tested surface differs
// from the reference: differing pixels are tinted magenta over the
// reference image and every tile containing an error gets a red border.
// Returns the number of differing pixels.
func errorMapPNG(path string, fb *Framebuffer, ref, got *Surface) (int, error) {
	img := image.NewNRGBA(image.Rect(0, 0, fb.W, fb.H))
	badTile := make([]bool, len(fb.tiles))
	badPixels := 0
	for y := 0; y < fb.H; y++ {
		for x := 0; x < fb.W; x++ {
			i := y*fb.W + x
			d := ref.color[i]
			g := got.color[i]
			diff := abs32(d.R-g.R) + abs32(d.G-g.G) + abs32(d.B-g.B)
			if diff > 0.02 {
				badPixels++
				badTile[(y/fb.TilePx)*fb.tilesX+x/fb.TilePx] = true
				img.SetNRGBA(x, y, color.NRGBA{255, 0, 255, 255})
			} else {
				img.SetNRGBA(x, y, toNRGBA(ref.color[i]))
			}
		}
	}
	for ti, bad := range badTile {
		if !bad {
			continue
		}
		t := &fb.tiles[ti]
		for k := 0; k < 2; k++ {
			for x := t.Bounds.MinX; x < t.Bounds.MaxX; x++ {
				img.SetNRGBA(x, t.Bounds.MinY+k, color.NRGBA{255, 0, 0, 255})
				img.SetNRGBA(x, t.Bounds.MaxY-1-k, color.NRGBA{255, 0, 0, 255})
			}
			for y := t.Bounds.MinY; y < t.Bounds.MaxY; y++ {
				img.SetNRGBA(t.Bounds.MinX+k, y, color.NRGBA{255, 0, 0, 255})
				img.SetNRGBA(t.Bounds.MaxX-1-k, y, color.NRGBA{255, 0, 0, 255})
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return badPixels, png.Encode(f, img)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// countBadTiles returns how many tiles contain at least one pixel that
// differs between the reference and the tested surface.
func countBadTiles(fb *Framebuffer, ref, got *Surface) int {
	bad := make([]bool, len(fb.tiles))
	for y := 0; y < fb.H; y++ {
		for x := 0; x < fb.W; x++ {
			i := y*fb.W + x
			d := ref.color[i]
			g := got.color[i]
			if abs32(d.R-g.R)+abs32(d.G-g.G)+abs32(d.B-g.B) > 0.02 {
				bad[(y/fb.TilePx)*fb.tilesX+x/fb.TilePx] = true
			}
		}
	}
	n := 0
	for _, b := range bad {
		if b {
			n++
		}
	}
	return n
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
