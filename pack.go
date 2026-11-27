package main

import (
	"math"
	"sort"
)

type packRect struct {
	island int
	wedges []int
	min    Vec2
	max    Vec2
	w, h   float64
	x, y   float64
}

func PackAtlas(a *Atlas) {
	const (
		unit = 210.0
		pad  = 22.0
		binW = 1280.0
	)
	var rects []*packRect
	for id, faces := range a.Islands {
		seen := map[int]bool{}
		r := &packRect{island: id, min: Vec2{math.Inf(1), math.Inf(1)}, max: Vec2{math.Inf(-1), math.Inf(-1)}}
		for _, f := range faces {
			for _, w := range a.FaceWedge[f] {
				if seen[w] {
					continue
				}
				seen[w] = true
				r.wedges = append(r.wedges, w)
				p := a.MetricUV[w]
				r.min.X = math.Min(r.min.X, p.X)
				r.min.Y = math.Min(r.min.Y, p.Y)
				r.max.X = math.Max(r.max.X, p.X)
				r.max.Y = math.Max(r.max.Y, p.Y)
			}
		}
		r.w = (r.max.X-r.min.X)*unit + 2*pad
		r.h = (r.max.Y-r.min.Y)*unit + 2*pad
		rects = append(rects, r)
	}
	sort.Slice(rects, func(i, j int) bool { return rects[i].h > rects[j].h })

	x, y := 0.0, 0.0
	shelfH := 0.0
	usedW := 1.0
	usedH := 1.0
	for _, r := range rects {
		if x > 0 && x+r.w > binW {
			x = 0
			y += shelfH
			shelfH = 0
		}
		r.x = x
		r.y = y
		x += r.w
		shelfH = math.Max(shelfH, r.h)
		usedW = math.Max(usedW, r.x+r.w)
		usedH = math.Max(usedH, r.y+r.h)
	}
	usedH = math.Max(usedH, y+shelfH)

	uvScale := math.Min((1-0.035)/usedW, (1-0.035)/usedH)
	drawW := usedW * uvScale
	drawH := usedH * uvScale
	ox := (1 - drawW) * 0.5
	oy := (1 - drawH) * 0.5
	if a.WedgeUV == nil || len(a.WedgeUV) < len(a.MetricUV) {
		a.WedgeUV = make([]Vec2, len(a.MetricUV))
	}
	for _, r := range rects {
		for _, w := range r.wedges {
			p := a.MetricUV[w]
			px := r.x + pad + (p.X-r.min.X)*unit
			py := r.y + pad + (r.max.Y-p.Y)*unit
			a.WedgeUV[w] = Vec2{
				X: ox + px*uvScale,
				Y: 1 - oy - py*uvScale,
			}
		}
	}
}
