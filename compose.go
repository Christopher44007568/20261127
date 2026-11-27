package main

import "image"

func newCanvas(width, height int, bg RGB) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			setRGB(img, x, y, bg)
		}
	}
	return img
}

func blit(dst, src *image.RGBA, dx, dy int) {
	for y := 0; y < src.Rect.Dy(); y++ {
		for x := 0; x < src.Rect.Dx(); x++ {
			tx := dx + x
			ty := dy + y
			if image.Pt(tx, ty).In(dst.Rect) {
				dst.SetRGBA(tx, ty, src.RGBAAt(x, y))
			}
		}
	}
}

func fillRect(img *image.RGBA, r image.Rectangle, c RGB) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if image.Pt(x, y).In(img.Rect) {
				setRGB(img, x, y, c)
			}
		}
	}
}

func makeComparison(fixed, aniso, ref, anisoErr *image.RGBA) *image.RGBA {
	const (
		panelW = 480
		panelH = 360
		margin = 10
		gap    = 10
		labelH = 28
	)
	width := margin*2 + panelW*2 + gap
	height := margin*2 + (labelH+panelH)*2 + gap
	out := newCanvas(width, height, RGB{0.015, 0.020, 0.025})
	panels := []struct {
		img   *image.RGBA
		label string
	}{
		{fixed, "FIXED 4 TEXELS"},
		{aniso, "ANISO 128 TAP BUDGET"},
		{ref, "ELLIPTICAL REFERENCE"},
		{anisoErr, "ANISO ERROR X20"},
	}
	for i, p := range panels {
		col := i % 2
		row := i / 2
		x := margin + col*(panelW+gap)
		y := margin + row*(labelH+panelH+gap)
		drawText(out, x+4, y+6, p.label, RGB{0.88, 0.93, 0.97}, 2)
		blit(out, resizeImage(p.img, panelW, panelH), x, y+labelH)
	}
	return out
}

func makeDetail(fixed, aniso, fixedErr *image.RGBA) *image.RGBA {
	const (
		panelW = 420
		panelH = 230
		margin = 8
		gap    = 8
		labelH = 28
	)
	cropRect := image.Rect(150, 270, 810, 630)
	width := margin*2 + panelW*3 + gap*2
	height := margin*2 + labelH + panelH
	out := newCanvas(width, height, RGB{0.015, 0.020, 0.025})
	panels := []struct {
		img   *image.RGBA
		label string
	}{
		{fixed, "FIXED 4: ALIAS MOIRE"},
		{aniso, "ANISO: LONG RUNS KEPT"},
		{fixedErr, "FIXED 4 ERROR FIELD"},
	}
	for i, p := range panels {
		x := margin + i*(panelW+gap)
		drawText(out, x+4, margin+5, p.label, RGB{0.88, 0.93, 0.97}, 2)
		cropped := cropImage(p.img, cropRect)
		blit(out, resizeImage(cropped, panelW, panelH), x, margin+labelH)
	}
	return out
}

func makeBudgetStrip(images map[int]*image.RGBA, budgets []int) *image.RGBA {
	const (
		panelW = 240
		panelH = 180
		margin = 8
		gap    = 8
		labelH = 28
	)
	width := margin*2 + len(budgets)*panelW + (len(budgets)-1)*gap
	height := margin*2 + labelH + panelH
	out := newCanvas(width, height, RGB{0.015, 0.020, 0.025})
	for i, budget := range budgets {
		x := margin + i*(panelW+gap)
		label := "BUDGET " + itoa(budget)
		drawText(out, x+4, margin+5, label, RGB{0.88, 0.93, 0.97}, 2)
		blit(out, resizeImage(images[budget], panelW, panelH), x, margin+labelH)
	}
	return out
}

func textureImage(tex *Texture) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, tex.Size, tex.Size))
	data := tex.Levels[0]
	for y := 0; y < tex.Size; y++ {
		for x := 0; x < tex.Size; x++ {
			i := (y*tex.Size + x) * 4
			setRGB(img, x, y, RGB{float64(data[i]), float64(data[i+1]), float64(data[i+2])})
		}
	}
	return img
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
