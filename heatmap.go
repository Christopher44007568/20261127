package flakepaint

import (
	"image"
	"image/color"
	"math"
	"sort"
)

type densityGrid struct {
	W, H       int
	Min, Max   Vec2
	BinW       float64
	BinH       float64
	Vals       []float64
	FlakeCount int
	TotalArea  float64
}

func buildDensityGrid(field FlakeField, min, max Vec2, cols, rows int) *densityGrid {
	g := &densityGrid{
		W: cols, H: rows, Min: min, Max: max,
		BinW: (max.X - min.X) / float64(cols),
		BinH: (max.Y - min.Y) / float64(rows),
		Vals: make([]float64, cols*rows),
	}
	field.EnumerateFlakes(min, max, func(f FlakeInstance) {
		a := math.Pi * f.RX * f.RY
		g.FlakeCount++
		g.TotalArea += a
		cx := (f.Center.X - min.X) / g.BinW
		cy := (f.Center.Y - min.Y) / g.BinH
		sigma := math.Max(f.RX, f.RY) * 0.72 / math.Min(g.BinW, g.BinH)
		if sigma < 0.45 {
			sigma = 0.45
		}
		radius := int(math.Ceil(2.8 * sigma))
		ix := int(math.Floor(cx))
		iy := int(math.Floor(cy))
		var norm float64
		for yy := iy - radius; yy <= iy+radius; yy++ {
			for xx := ix - radius; xx <= ix+radius; xx++ {
				if xx < 0 || yy < 0 || xx >= cols || yy >= rows {
					continue
				}
				dx := (float64(xx) + 0.5 - cx) / sigma
				dy := (float64(yy) + 0.5 - cy) / sigma
				norm += math.Exp(-0.5 * (dx*dx + dy*dy))
			}
		}
		if norm <= 0 {
			return
		}
		for yy := iy - radius; yy <= iy+radius; yy++ {
			for xx := ix - radius; xx <= ix+radius; xx++ {
				if xx < 0 || yy < 0 || xx >= cols || yy >= rows {
					continue
				}
				dx := (float64(xx) + 0.5 - cx) / sigma
				dy := (float64(yy) + 0.5 - cy) / sigma
				w := math.Exp(-0.5*(dx*dx+dy*dy)) / norm
				g.Vals[yy*cols+xx] += a * w
			}
		}
	})
	binArea := g.BinW * g.BinH
	for i := range g.Vals {
		g.Vals[i] /= binArea
	}
	blurDensityGrid(g, 1.8)
	return g
}

func blurDensityGrid(g *densityGrid, sigma float64) {
	radius := int(math.Ceil(3 * sigma))
	kernel := make([]float64, radius*2+1)
	var norm float64
	for i := -radius; i <= radius; i++ {
		w := math.Exp(-0.5 * float64(i*i) / (sigma * sigma))
		kernel[i+radius] = w
		norm += w
	}
	for i := range kernel {
		kernel[i] /= norm
	}
	tmp := make([]float64, len(g.Vals))
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			var sum float64
			for k := -radius; k <= radius; k++ {
				xx := clampInt(x+k, 0, g.W-1)
				sum += g.Vals[y*g.W+xx] * kernel[k+radius]
			}
			tmp[y*g.W+x] = sum
		}
	}
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			var sum float64
			for k := -radius; k <= radius; k++ {
				yy := clampInt(y+k, 0, g.H-1)
				sum += tmp[yy*g.W+x] * kernel[k+radius]
			}
			g.Vals[y*g.W+x] = sum
		}
	}
}

func (g *densityGrid) sample(x, y float64) float64 {
	fx := ((x - g.Min.X) / g.BinW) - 0.5
	fy := ((y - g.Min.Y) / g.BinH) - 0.5
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := fx - float64(x0)
	ty := fy - float64(y0)
	get := func(xx, yy int) float64 {
		xx = int(clamp(float64(xx), 0, float64(g.W-1)))
		yy = int(clamp(float64(yy), 0, float64(g.H-1)))
		return g.Vals[yy*g.W+xx]
	}
	a := mix(get(x0, y0), get(x0+1, y0), tx)
	b := mix(get(x0, y0+1), get(x0+1, y0+1), tx)
	return mix(a, b, ty)
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(clamp(p*float64(len(sorted)-1), 0, float64(len(sorted)-1)))
	return sorted[i]
}

func heatColor(t float64) color.RGBA {
	t = clamp(t, 0, 1)
	type stop struct {
		p float64
		c [3]float64
	}
	stops := []stop{
		{0.00, [3]float64{0.010, 0.020, 0.035}},
		{0.18, [3]float64{0.020, 0.090, 0.115}},
		{0.38, [3]float64{0.025, 0.280, 0.285}},
		{0.58, [3]float64{0.080, 0.610, 0.500}},
		{0.76, [3]float64{0.560, 0.770, 0.300}},
		{0.90, [3]float64{0.950, 0.620, 0.155}},
		{1.00, [3]float64{1.000, 0.255, 0.075}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].p {
			f := (t - stops[i-1].p) / (stops[i].p - stops[i-1].p)
			return color.RGBA{
				R: uint8(math.Round(255 * mix(stops[i-1].c[0], stops[i].c[0], f))),
				G: uint8(math.Round(255 * mix(stops[i-1].c[1], stops[i].c[1], f))),
				B: uint8(math.Round(255 * mix(stops[i-1].c[2], stops[i].c[2], f))),
				A: 255,
			}
		}
	}
	c := stops[len(stops)-1].c
	return color.RGBA{uint8(c[0] * 255), uint8(c[1] * 255), uint8(c[2] * 255), 255}
}

func heatmapBackground(img *image.RGBA) {
	for y := 0; y < img.Rect.Dy(); y++ {
		f := float64(y) / float64(img.Rect.Dy()-1)
		c := color.RGBA{
			R: uint8(mix(5, 10, f)),
			G: uint8(mix(9, 17, f)),
			B: uint8(mix(14, 24, f)),
			A: 255,
		}
		fillRect(img, 0, y, img.Rect.Dx(), y+1, c)
	}
}

// RenderDensityMap renders an informational frame from the actual active
// flake cells, including a continuous coverage field and isolines.
func RenderDensityMap(outW, outH int, field FlakeField) *image.RGBA {
	w, h := 1200, 760
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	heatmapBackground(img)
	accent := color.RGBA{100, 225, 210, 255}
	dim := color.RGBA{105, 137, 151, 255}
	bright := color.RGBA{225, 239, 240, 255}
	grid := color.RGBA{145, 190, 198, 32}

	min := V2(-180, -110)
	max := V2(180, 110)
	dg := buildDensityGrid(field, min, max, 120, 73)
	sorted := append([]float64(nil), dg.Vals...)
	sort.Float64s(sorted)
	p02 := percentile(sorted, 0.02)
	p98 := percentile(sorted, 0.98)
	mean := 0.0
	for _, v := range dg.Vals {
		mean += v
	}
	mean /= float64(len(dg.Vals))

	drawTextShadow(img, 72, 48, "FLAKE DENSITY", 3, bright)
	drawText(img, 74, 82, "WORLD-SPACE CELL HISTOGRAM / FOUR LAYER STACK", 1, dim)
	drawText(img, 72, 114, "03 / DENSITY", 1, accent)
	drawText(img, w-240, 62, "MM COORDINATES", 1, dim)
	drawText(img, w-240, 82, "DETERMINISTIC / STATIC", 1, accent)
	drawLine(img, 72, 138, w-72, 138, color.RGBA{100, 160, 170, 110})

	mx0, my0 := 74, 178
	mw, mh := 820, 500
	for py := 0; py < mh; py++ {
		for px := 0; px < mw; px++ {
			u := float64(px) / float64(mw)
			v := 1 - float64(py)/float64(mh)
			val := dg.sample(u*(max.X-min.X)+min.X, v*(max.Y-min.Y)+min.Y)
			t := (val - p02) / math.Max(1e-9, p98-p02)
			t = math.Pow(clamp(t, 0, 1), 0.82)
			c := heatColor(t)
			putPixel(img, mx0+px, my0+py, c)
		}
	}

	// Faint 20 mm measurement grid.
	for x := 0.0; x <= max.X-min.X; x += 20 {
		px := mx0 + int(math.Round(x/float64(max.X-min.X)*float64(mw)))
		for py := my0; py < my0+mh; py++ {
			putPixel(img, px, py, grid)
		}
	}
	for y := 0.0; y <= max.Y-min.Y; y += 20 {
		py := my0 + int(math.Round((1-y/float64(max.Y-min.Y))*float64(mh)))
		for px := mx0; px < mx0+mw; px++ {
			putPixel(img, px, py, grid)
		}
	}

	// Isolines are computed from the same splatted coverage buffer.
	for py := 1; py < mh; py++ {
		for px := 1; px < mw; px++ {
			u := float64(px) / float64(mw)
			v := 1 - float64(py)/float64(mh)
			val := dg.sample(u*(max.X-min.X)+min.X, v*(max.Y-min.Y)+min.Y)
			n := int(math.Floor(((val - p02) / math.Max(1e-9, p98-p02)) * 9))
			u0 := float64(px-1) / float64(mw)
			val0 := dg.sample(u0*(max.X-min.X)+min.X, v*(max.Y-min.Y)+min.Y)
			n0 := int(math.Floor(((val0 - p02) / math.Max(1e-9, p98-p02)) * 9))
			if n != n0 {
				putPixel(img, mx0+px, my0+py, color.RGBA{230, 245, 230, 44})
			}
		}
	}

	// Sparse static centers show the distribution behind the smooth coverage.
	field.EnumerateFlakes(min, max, func(f FlakeInstance) {
		px := mx0 + int((f.Center.X-min.X)/(max.X-min.X)*float64(mw))
		py := my0 + int((1-(f.Center.Y-min.Y)/(max.Y-min.Y))*float64(mh))
		putPixel(img, px, py, color.RGBA{245, 250, 224, 35})
	})

	drawLine(img, mx0, my0, mx0+mw, my0, color.RGBA{180, 235, 235, 150})
	drawLine(img, mx0, my0+mh, mx0+mw, my0+mh, color.RGBA{180, 235, 235, 90})
	drawLine(img, mx0, my0, mx0, my0+mh, color.RGBA{180, 235, 235, 90})
	drawLine(img, mx0+mw, my0, mx0+mw, my0+mh, color.RGBA{180, 235, 235, 90})
	drawText(img, mx0, my0+mh+12, "-180", 1, dim)
	drawText(img, mx0+mw-34, my0+mh+12, "+180", 1, dim)
	drawText(img, mx0-48, my0+4, "110", 1, dim)
	drawText(img, mx0-42, my0+mh-13, "-110", 1, dim)

	// Vertical color bar.
	cx := 952
	cy0, cyh := 188, 400
	for i := 0; i < cyh; i++ {
		t := 1 - float64(i)/float64(cyh-1)
		c := heatColor(math.Pow(t, 0.82))
		fillRect(img, cx, cy0+i, cx+34, cy0+i+1, c)
	}
	for i := 0; i <= 4; i++ {
		y := cy0 + i*cyh/4
		drawLine(img, cx-7, y, cx+40, y, color.RGBA{170, 215, 218, 140})
		level := p98 + (p02-p98)*float64(i)/4
		drawText(img, cx+50, y-3, formatInt(int(math.Round(level*100))), 1, dim)
	}
	drawText(img, cx, cy0-28, "COVERAGE %", 1, bright)
	drawText(img, cx, cy0+cyh+16, "P02-P98 RANGE", 1, dim)

	drawText(img, 952, 624, formatInt(dg.FlakeCount), 2, accent)
	drawLine(img, 952, 648, w-74, 648, color.RGBA{100, 160, 170, 90})
	drawText(img, 952, 656, "ACTIVE FLAKES", 1, dim)
	drawText(img, 952, 682, formatFloat1(mean*100), 2, color.RGBA{235, 225, 190, 255})
	drawText(img, 952, 707, "MEAN COVERAGE %", 1, dim)

	drawLine(img, 74, 718, w-74, 718, color.RGBA{100, 160, 170, 90})
	drawText(img, 74, 730, "360 X 220 MM FIELD   /   P02-P98 AUTO RANGE   /   3 MM GRID + 6 MM DISPLAY KERNEL", 1, dim)
	if outW != w || outH != h {
		return scaleRGBA(img, outW, outH)
	}
	return img
}

func formatFloat1(v float64) string {
	whole := int(math.Floor(v))
	frac := int(math.Round((v - float64(whole)) * 10))
	if frac >= 10 {
		whole++
		frac = 0
	}
	return formatInt(whole) + "." + formatInt(frac)
}

func formatInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
