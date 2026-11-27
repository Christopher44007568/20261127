package main

import (
	"image"
	"image/color"
)

// BakeTile renders the seamlessly tileable source brick texture.
func BakeTile(seed uint64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileW, TileH))
	for y := 0; y < TileH; y++ {
		for x := 0; x < TileW; x++ {
			info := BrickAt(x, y)
			kc, kr := TileKey(info)
			keySeed := splitmix64(seed ^ uint64(kc)*73856093 ^ uint64(kr)*19349663)

			var c RGB
			if info.Mortar {
				c = bakeMortar(x, y, info, keySeed)
			} else {
				c = bakeBrick(x, y, info, keySeed)
			}
			r, g, b, a := rgbToRGBA8(c)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}

func bakeBrick(x, y int, info BrickInfo, seed uint64) RGB {
	pick := frand(seed, 1)
	var base RGB
	switch {
	case pick < 0.12: // tan / buff bricks
		base = hslToRGB(0.075+frand(seed, 2)*0.030, 0.32+frand(seed, 3)*0.16, 0.44+frand(seed, 4)*0.10)
	case pick < 0.22: // dark purplish-red clinker
		base = hslToRGB(0.985+frand(seed, 2)*0.020, 0.22+frand(seed, 3)*0.16, 0.25+frand(seed, 4)*0.08)
	default: // common terracotta reds
		base = hslToRGB(0.012+frand(seed, 2)*0.045, 0.44+frand(seed, 3)*0.18, 0.36+frand(seed, 4)*0.11)
	}

	// Broad clay mottling (periodic -> seamless across tile edges).
	mottle := TiledFBMOverTile(x, y, 77, 5, 8)
	fine := TiledFBMOverTile(x+3, y+11, 91, 3, 4)
	c := shiftHSL(base,
		(mottle-0.5)*0.015,
		(fine-0.5)*0.10,
		(mottle-0.5)*0.16+(fine-0.5)*0.07)

	// Per-brick directional kiln gradient.
	gx := float64(info.Lx) / BrickW
	gy := float64(info.Ly) / BrickH
	dir := (frand(seed, 5) - 0.5) * 0.10
	c = shiftHSL(c, 0, 0, (0.5-gy)*0.07+(gx-0.5)*dir)

	// Soft bevel + ambient occlusion near brick edges.
	e := info.FaceEdge
	switch {
	case e <= 1:
		c = mulRGB(c, 0.70)
	case e <= 3:
		c = mulRGB(c, lerp(0.80, 0.92, (e-1)/2))
	case e <= 7:
		c = mulRGB(c, lerp(0.92, 1.0, (e-3)/4))
	}
	// Top-inner highlight gives a subtle lit lip.
	if info.Ly-Joint >= 1 && info.Ly-Joint <= 2 {
		c = mixRGB(c, RGB{1, 0.93, 0.82}, 0.10)
	}

	// Occasional kiln scorch mark, stable per brick.
	if frand(seed, 6) < 0.35 {
		cx := Joint + 6 + frand(seed, 7)*float64(BrickW-2*Joint-12)
		cy := Joint + 4 + frand(seed, 8)*float64(BrickH-2*Joint-8)
		rad := 4 + frand(seed, 9)*9
		dx := float64(info.Lx) - cx
		dy := float64(info.Ly) - cy
		stain := smoothStep01(1 - (dx*dx+dy*dy)/(rad*rad))
		c = mixRGB(c, RGB{0.16, 0.10, 0.08}, stain*0.28)
	}

	// Fine per-pixel grain.
	grain := hashFloat(int64(x), int64(y), 1234) - 0.5
	c = shiftHSL(c, 0, 0, grain*0.05)
	return c
}

func bakeMortar(x, y int, info BrickInfo, seed uint64) RGB {
	base := hslToRGB(0.09+frand(seed, 20)*0.02, 0.10+frand(seed, 21)*0.08, 0.50+frand(seed, 22)*0.08)
	large := TiledFBMOverTile(x, y, 303, 4, 16)
	grit := TiledFBMOverTile(x, y, 404, 3, 4)
	c := shiftHSL(base, 0, (grit-0.5)*0.06, (large-0.5)*0.14+(grit-0.5)*0.10)

	// Darken both lips of each mortar strip for recessed depth.
	lx, ly := info.Lx, info.Ly
	edge := 99.0
	if lx < Joint {
		edge = float64(minInt(lx, Joint-1-lx))
	}
	if ly < Joint {
		if d := minInt(ly, Joint-1-ly); float64(d) < edge {
			edge = float64(d)
		}
	}
	if edge < 2 {
		c = mulRGB(c, 0.78)
	} else if edge < 4 {
		c = mulRGB(c, 0.90)
	}

	grain := hashFloat(int64(x), int64(y), 5678) - 0.5
	c = shiftHSL(c, 0, 0, grain*0.06)
	return c
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
