package studio

import "strings"

func ResizeBox(src *FloatImage, w, h int) *FloatImage {
	out := NewImage(w, h)
	for y := 0; y < h; y++ {
		sy0 := y * src.H / h
		sy1 := (y + 1) * src.H / h
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < w; x++ {
			sx0 := x * src.W / w
			sx1 := (x + 1) * src.W / w
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, a float32
			n := float32(0)
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					p := src.At(sx, sy)
					r += p.R
					g += p.G
					b += p.B
					a += p.A
					n++
				}
			}
			out.Set(x, y, RGBA{R: r / n, G: g / n, B: b / n, A: a / n})
		}
	}
	return out
}

func Checker(w, h, cell int) *FloatImage {
	im := NewImage(w, h)
	DrawChecker(im, cell, Hex("111722"), Hex("1a2331"))
	return im
}

func layerPreview(layer *FloatImage) *FloatImage {
	bg := Checker(layer.W, layer.H, 32)
	bg.Blend(layer, Normal)
	return bg
}

func maskPreview(mask []float32, w, h int) *FloatImage {
	im := NewImage(w, h)
	for i, v := range mask {
		c := Hex("05070b")
		if v > 0 {
			c = lerpRGBA(c, Hex("eaf3ff"), clamp01f(v))
		}
		im.SetStraight(i%w, i/w, c)
	}
	return im
}

func drawThumb(im *FloatImage, src *FloatImage, x, y, w, h int) {
	scaled := ResizeBox(src, w, h)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			im.Set(x+px, y+py, scaled.At(px, py))
		}
	}
}

// LayerSheet is a compact contact sheet; the individual full-resolution layer
// PNGs remain available under out/layers.
func LayerSheet(scene *Scene) *FloatImage {
	im := NewImage(CanvasW, CanvasH)
	im.Clear(Hex("06090f"))
	white := Hex("eef5ff")
	muted := Hex("8a9bb4")
	accent := Hex("5be7d4")
	DrawLabel(im, scene.Font, "LAYER STACK / PREMULTIPLIED", 34, 24, 28, white)
	DrawLabel(im, scene.Font, "CLIP STACK: FRAME INTERSECT GROUP MASK", 36, 58, 13, muted)

	final := scene.BuildFinal(Normal, MaskLast)
	items := []struct {
		label string
		img   *FloatImage
		tint  RGBA
	}{
		{"00 BACKGROUND", scene.Background, muted},
		{"01 GROUP CLIP MASK", maskPreview(scene.Group, CanvasW, CanvasH), accent},
		{"02 LINEAR / GROUP", layerPreview(scene.Linear), Hex("45d9ff")},
		{"03 RADIAL / GROUP", layerPreview(scene.Radial), Hex("ffb45c")},
		{"04 ALPHA MASK", maskPreview(scene.Alpha, CanvasW, CanvasH), Hex("ff6b8d")},
		{"05 FINAL / NORMAL", final, white},
	}

	const cols = 3
	const cellW, cellH = 512, 438
	const gap = 16
	startY := 88
	for i, item := range items {
		col := i % cols
		row := i / cols
		x := gap + col*(cellW+gap)
		y := startY + row*(cellH+gap)
		DrawLabel(im, scene.Font, item.label, float32(x+8), float32(y+4), 13, item.tint)
		thumbW, thumbH := 480, 300
		tx := x + (cellW-thumbW)/2
		ty := y + 34
		FillRect(im, tx-4, ty-4, thumbW+8, thumbH+8, Hex("02040a"))
		StrokeRect(im, tx-4, ty-4, thumbW+8, thumbH+8, Hex("6f86a52e"))
		drawThumb(im, item.img, tx, ty, thumbW, thumbH)
		StrokeRect(im, tx, ty, thumbW, thumbH, Hex("aec4de2b"))
		if i == 5 {
			DrawLabel(im, scene.Font, "PRE-MUL + DITHER", float32(tx), float32(ty+thumbH+18), 12, muted)
		}
	}
	return im
}

func BlendSheet(scene *Scene) *FloatImage {
	im := NewImage(CanvasW, 620)
	im.Clear(Hex("06090f"))
	DrawLabel(im, scene.Font, "BLEND MODES / SAME CLIP + MASK STACK", 34, 26, 28, Hex("eef5ff"))
	modes := []BlendMode{Normal, Multiply, Screen}
	colors := []RGBA{Hex("5be7d4"), Hex("ffb45c"), Hex("ff6b8d")}
	for i, mode := range modes {
		x := 34 + i*512
		DrawLabel(im, scene.Font, strings.ToUpper(mode.String()), float32(x), 78, 19, colors[i])
		src := scene.BuildFinal(mode, MaskLast)
		drawThumb(im, src, x, 112, 480, 300)
		StrokeRect(im, x, 112, 480, 300, Hex("aec4de2e"))
		desc := "SOURCE OVER"
		if mode == Multiply {
			desc = "LINEAR MULTIPLY"
		}
		if mode == Screen {
			desc = "LINEAR SCREEN"
		}
		DrawLabel(im, scene.Font, desc, float32(x), 430, 13, Hex("8a9bb4"))
	}
	DrawLabel(im, scene.Font, "ALPHA MASK: DESTINATION-IN AT THE END OF EVERY MODE", 34, 510, 15, Hex("8a9bb4"))
	return im
}

func OrderSheet(scene *Scene) *FloatImage {
	im := NewImage(CanvasW, 900)
	im.Clear(Hex("06090f"))
	DrawLabel(im, scene.Font, "ORDER MATTERS / TRANSPARENT EDGES", 34, 22, 30, Hex("eef5ff"))
	DrawLabel(im, scene.Font, "NORMAL RADIAL BLEND / SAME SOURCE LAYERS", 36, 60, 13, Hex("8a9bb4"))
	last := scene.BuildFinal(Normal, MaskLast)
	first := scene.BuildFinal(Normal, MaskFirst)
	for i, item := range []struct {
		label string
		img   *FloatImage
		c     RGBA
	}{
		{"LINEAR TO RADIAL TO MASK", last, Hex("5be7d4")},
		{"LINEAR TO MASK TO RADIAL", first, Hex("ffb45c")},
	} {
		x := 50 + i*800
		DrawLabel(im, scene.Font, item.label, float32(x), 96, 18, item.c)
		drawThumb(im, item.img, x, 128, 700, 438)
		StrokeRect(im, x, 128, 700, 438, Hex("aec4de2e"))
	}
	DrawLabel(im, scene.Font, "MASK LAST: DESTINATION-IN NARROWS THE FINAL COMPOSITE BOUNDARY.", 50, 592, 14, Hex("8a9bb4"))
	DrawLabel(im, scene.Font, "MASK FIRST: THE LATER RADIAL BLEND REINTRODUCES COVERAGE.", 50, 618, 14, Hex("8a9bb4"))
	lastEdge := EdgeCrop(last, 335, 405, 80, 80, 2)
	firstEdge := EdgeCrop(first, 335, 405, 80, 80, 2)
	drawThumb(im, lastEdge, 50, 666, 160, 160)
	drawThumb(im, firstEdge, 850, 666, 160, 160)
	StrokeRect(im, 50, 666, 160, 160, Hex("aec4de2e"))
	StrokeRect(im, 850, 666, 160, 160, Hex("aec4de2e"))
	DrawLabel(im, scene.Font, "EDGE CROP 2X / MASK LAST", 232, 670, 15, Hex("5be7d4"))
	DrawLabel(im, scene.Font, "EDGE CROP 2X / MASK FIRST", 1032, 670, 15, Hex("ffb45c"))
	DrawLabel(im, scene.Font, "MASK-LAST PRESERVES ANTI-ALIASED PREMULTIPLIED EDGES.", 232, 704, 12, Hex("8a9bb4"))
	DrawLabel(im, scene.Font, "MASK-FIRST HAS HIGHER ALPHA BEHIND THE RADIAL LAYER.", 1032, 704, 12, Hex("8a9bb4"))
	DrawLabel(im, scene.Font, "SAME PREMULTIPLIED SAMPLER / DIFFERENT STACK ORDER", 50, 856, 12, Hex("8a9bb4"))
	return im
}
