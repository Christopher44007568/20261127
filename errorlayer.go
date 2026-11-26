package main

import (
	"image"
	"image/color"
	"math"
)

// LayerOpts controls which IDs are force-labeled (e.g. injected faults).
type LayerOpts struct {
	ForceIDs map[string]bool
	// MarkID, if >=0, gets an emphatic pulsing-style callout ring.
	MarkID int
}

// BuildErrorLayer renders own.png dimmed, with error pixels highlighted and
// original object IDs annotated on the affected layers.
func BuildErrorLayer(rr *RenderResult, diff *pixelDiff, rep *DiffReport, opts LayerOpts) *image.RGBA {
	W, H := rr.Scene.W, rr.Scene.H
	base := toRGBA(rr.Img)
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// Dim the correct render so error overlays and labels pop.
	for i := 0; i < W*H; i++ {
		o := i * 4
		img.Pix[o] = uint8(float64(base.Pix[o]) * 0.32)
		img.Pix[o+1] = uint8(float64(base.Pix[o+1]) * 0.34)
		img.Pix[o+2] = uint8(float64(base.Pix[o+2])*0.42 + 18)
		img.Pix[o+3] = 255
	}

	// 1px halo around error mask.
	halo := make([]bool, W*H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			if !diff.errMask[y*W+x] {
				continue
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < W && ny >= 0 && ny < H {
						halo[ny*W+nx] = true
					}
				}
			}
		}
	}
	for i := 0; i < W*H; i++ {
		switch {
		case diff.errMask[i]:
			o := i * 4
			img.Pix[o], img.Pix[o+1], img.Pix[o+2] = 255, 59, 92
		case halo[i]:
			o := i * 4
			img.Pix[o], img.Pix[o+1], img.Pix[o+2] = 120, 26, 52
		}
	}

	statusColor := map[string]color.RGBA{
		"error": {225, 29, 72, 255},
		"minor": {217, 119, 6, 255},
		"match": {16, 185, 129, 255},
	}

	type label struct {
		rect image.Rectangle
		text string
		col  color.RGBA
	}
	var occupied []image.Rectangle
	var labels []label

	plateH := 7*2 + 8
	for idx, st := range rep.Objects {
		o := rr.Scene.All[idx]
		isRoot := o.Parent == nil
		forced := opts.ForceIDs != nil && opts.ForceIDs[o.ID]
		bad := st.Status != "match"
		// Roots always labeled; nested children only when implicated or forced.
		if !isRoot && !bad && !forced {
			continue
		}
		x0, y0, x1, y1 := int(st.BBox[0]), int(st.BBox[1]), int(st.BBox[2]), int(st.BBox[3])
		if x1 <= x0 {
			continue
		}
		col := statusColor[st.Status]
		if forced {
			col = color.RGBA{225, 29, 72, 255}
		}

		text := o.ID
		if st.ErrorPixels > 0 && (bad || forced) {
			text = o.ID + " x" + itoa(st.ErrorPixels)
		}
		tw := TextWidth(text, 2) + 12
		huge := (x1-x0)*(y1-y0) > 820*500
		var candidates []image.Point
		if huge {
			candidates = []image.Point{{16, 16}, {16, 44}, {16, 72}}
		} else {
			candidates = []image.Point{
				{x0, y0 - plateH - 6},
				{x0, y1 + 6},
				{x0 + 6, y0 + 6},
				{x1 - tw, y0 - plateH - 6},
				{x1 - tw, y1 + 6},
			}
		}
		var r image.Rectangle
		placed := false
		for _, c := range candidates {
			cx := clampInt(c.X, 4, W-tw-4)
			cy := clampInt(c.Y, 4, H-plateH-4)
			try := image.Rect(cx, cy, cx+tw, cy+plateH)
			coll := false
			for _, q := range occupied {
				if try.Overlaps(q.Inset(-3)) {
					coll = true
					break
				}
			}
			if !coll {
				r = try
				placed = true
				break
			}
		}
		if !placed {
			cy := 4 + len(occupied)*(plateH+4)
			cy = cy % (H - plateH - 8)
			r = image.Rect(8, cy, 8+tw, cy+plateH)
		}
		occupied = append(occupied, r)

		if !huge {
			// bbox + short leader between plate edge and bbox edge
			strokeRect(img, x0-2, y0-2, (x1-x0)+4, (y1-y0)+4, 2, withAlpha(col, 220))
			box := image.Rect(x0-2, y0-2, x1+2, y1+2)
			if !r.Overlaps(box) {
				target := closestPointOnRect(box, rectCenter(r))
				source := closestPointOnRect(r, target)
				drawLine(img, source.X, source.Y, target.X, target.Y, 1, withAlpha(col, 190))
			}
		}

		labels = append(labels, label{rect: r, text: text, col: col})
	}

	// Plates last so crossing bbox/leader lines stay hidden behind them.
	for _, lb := range labels {
		plate := withAlpha(darken(lb.col, 0.42), 240)
		DrawTextPlateAt(img, lb.rect, lb.text, 2, color.RGBA{255, 255, 255, 255}, plate)
	}

	if opts.MarkID >= 0 && opts.MarkID < len(rr.Scene.All) {
		st := rep.Objects[opts.MarkID]
		x0, y0, x1, y1 := int(st.BBox[0])-8, int(st.BBox[1])-8, int(st.BBox[2])+8, int(st.BBox[3])+8
		for k := 0; k < 3; k++ {
			strokeRect(img, x0-k*3, y0-k*3, (x1-x0)+k*6, (y1-y0)+k*6, 2,
				color.RGBA{255, 214, 0, uint8(220 - k*60)})
		}
	}
	return img
}

// DrawTextPlateAt fills a precomputed plate rect and centers text inside.
func DrawTextPlateAt(img *image.RGBA, r image.Rectangle, text string, scale int, fg, plate color.RGBA) {
	fillRect(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), plate)
	tw := TextWidth(text, scale)
	tx := r.Min.X + (r.Dx()-tw)/2
	ty := r.Min.Y + (r.Dy()-7*scale)/2
	DrawText(img, tx, ty, text, scale, fg)
}

// BuildIDMap colorizes the modal object index per pixel.
func BuildIDMap(rr *RenderResult) *image.RGBA {
	W, H := rr.Scene.W, rr.Scene.H
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	n := len(rr.Scene.All)
	colors := make([]color.RGBA, n+1)
	for i := 0; i < n; i++ {
		h := math.Mod(float64(i)*0.61803398875, 1)
		r, g, b := hslToRGB(h, 0.62, 0.55)
		colors[i] = color.RGBA{r, g, b, 255}
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			id := rr.ModalID(x, y)
			c := color.RGBA{226, 232, 240, 255}
			if id >= 0 {
				c = colors[id]
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func withAlpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

func rectCenter(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// closestPointOnRect returns the point on r's border/area nearest to p,
// clamped independently on each axis.
func closestPointOnRect(r image.Rectangle, p image.Point) image.Point {
	x := clampInt(p.X, r.Min.X, r.Max.X)
	y := clampInt(p.Y, r.Min.Y, r.Max.Y)
	return image.Pt(x, y)
}

func darken(c color.RGBA, k float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * k),
		G: uint8(float64(c.G) * k),
		B: uint8(float64(c.B) * k),
		A: c.A,
	}
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	var r, g, b float64
	if s == 0 {
		r, g, b = l, l, l
	} else {
		hue := func(p, q, t float64) float64 {
			if t < 0 {
				t++
			}
			if t > 1 {
				t--
			}
			if t < 1.0/6 {
				return p + (q-p)*6*t
			}
			if t < 0.5 {
				return q
			}
			if t < 2.0/3 {
				return p + (q-p)*(2.0/3-t)*6
			}
			return p
		}
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hue(p, q, h+1.0/3)
		g = hue(p, q, h)
		b = hue(p, q, h-1.0/3)
	}
	return uint8(r*255 + 0.5), uint8(g*255 + 0.5), uint8(b*255 + 0.5)
}
