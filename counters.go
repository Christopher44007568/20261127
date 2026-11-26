package main

import (
	"image"
	"math"
)

// CounterStat describes one enclosed background region (a glyph counter).
type CounterStat struct {
	AreaRegular int
	AreaBold    int
	Retention   float64
	CoreDepth   int
}

// BoldReport is the counter-preservation decision for one rune.
type BoldReport struct {
	Rune       rune
	HasOutline bool
	TargetEm   float64
	ChosenEm   float64
	Limited    bool
	AtTarget   []CounterStat
	AtChosen   []CounterStat
}

const boldAnalysisPpem = 240.0

func (fs *FontSet) boldCacheMap() map[rune]BoldReport {
	if fs.boldCache == nil {
		fs.boldCache = map[rune]BoldReport{}
	}
	return fs.boldCache
}

// BoldRadiusEm returns the counter-safe embolden radius in em units.
func (fs *FontSet) BoldRadiusEm(r rune) float64 {
	return fs.BoldReport(r).ChosenEm
}

// BoldReport returns (cached) counter-preservation metrics for a rune.
func (fs *FontSet) BoldReport(r rune) BoldReport {
	cache := fs.boldCacheMap()
	if rep, ok := cache[r]; ok {
		return rep
	}
	const target = 0.05 // radius em; centered stroke width = 0.10em
	rep := BoldReport{Rune: r, TargetEm: target, ChosenEm: target}
	_, g, o0 := fs.OutlineFor(r, boldAnalysisPpem, 0.2)
	if len(o0.Contours) == 0 {
		cache[r] = rep
		return rep
	}
	rep.HasOutline = true
	regMask, ox, oy, mw, mh := buildGlyphMask(o0, target*boldAnalysisPpem+8)
	regContours := translateContours(o0.Contours, ox, oy)
	FillGray(regMask, regContours, 255)
	regs := findCounters(regMask, mw, mh)

	eval := func(rEm float64) []CounterStat {
		rad := rEm * boldAnalysisPpem
		mask := image.NewGray(image.Rect(0, 0, mw, mh))
		FillGray(mask, regContours, 255)
		if rad > 0 {
			strokeContoursOnMask(mask, regContours, rad)
		}
		stats := make([]CounterStat, 0, len(regs))
		for _, comp := range regs {
			areaBold := 0
			for _, idx := range comp.pixels {
				if mask.Pix[idx] < 128 {
					areaBold++
				}
			}
			ret := 0.0
			if comp.area > 0 {
				ret = float64(areaBold) / float64(comp.area)
			}
			core := survivingCoreDepth(mask, comp.pixels)
			stats = append(stats, CounterStat{
				AreaRegular: comp.area,
				AreaBold:    areaBold,
				Retention:   ret,
				CoreDepth:   core,
			})
		}
		return stats
	}

	safe := func(stats []CounterStat) bool {
		for _, st := range stats {
			// A counter must keep at least 38% area and a deep enough core
			// to stay visibly open at text sizes.
			if st.Retention < 0.38 || st.AreaBold < 28 || st.CoreDepth < 2 {
				return false
			}
		}
		return true
	}

	rep.AtTarget = eval(target)
	if safe(rep.AtTarget) {
		rep.AtChosen = rep.AtTarget
		cache[r] = rep
		return rep
	}
	rep.Limited = true
	chosen := 0.0
	var chosenStats []CounterStat
	for rEm := target - 0.005; rEm >= 0.004; rEm -= 0.005 {
		st := eval(rEm)
		if safe(st) {
			chosen = rEm
			chosenStats = st
			break
		}
	}
	if chosen == 0 {
		chosenStats = eval(0)
	}
	rep.ChosenEm = chosen
	rep.AtChosen = chosenStats
	_ = g
	cache[r] = rep
	return rep
}

// EvaluateRadiusEm exposes the counter result for an arbitrary radius, used by
// the rejected-overbold diagnostic panel.
func (fs *FontSet) EvaluateRadiusEm(r rune, rEm float64) (BoldReport, []CounterStat, *image.Gray, float64, float64, int, int) {
	_, _, o0 := fs.OutlineFor(r, boldAnalysisPpem, 0.2)
	pad := rEm*boldAnalysisPpem + 8
	mask, ox, oy, mw, mh := buildGlyphMask(o0, pad)
	translated := translateContours(o0.Contours, ox, oy)
	FillGray(mask, translated, 255)
	rad := rEm * boldAnalysisPpem
	if rad > 0 {
		strokeContoursOnMask(mask, translated, rad)
	}
	regs := findCountersFromOutline(o0, ox, oy, mw, mh)
	stats := make([]CounterStat, 0, len(regs))
	for _, comp := range regs {
		areaBold := 0
		for _, idx := range comp.pixels {
			if mask.Pix[idx] < 128 {
				areaBold++
			}
		}
		ret := 0.0
		if comp.area > 0 {
			ret = float64(areaBold) / float64(comp.area)
		}
		stats = append(stats, CounterStat{AreaRegular: comp.area, AreaBold: areaBold, Retention: ret})
	}
	return BoldReport{Rune: r, TargetEm: rEm, ChosenEm: rEm}, stats, mask, ox, oy, mw, mh
}

func buildGlyphMask(o Outline, pad float64) (*image.Gray, float64, float64, int, int) {
	minX := o.Bounds.MinX
	minY := o.Bounds.MinY
	maxX := o.Bounds.MaxX
	maxY := o.Bounds.MaxY
	if len(o.Contours) == 0 {
		minX, minY, maxX, maxY = -50, -200, 200, 50
	}
	w := int(math.Ceil(maxX - minX + 2*pad))
	h := int(math.Ceil(maxY - minY + 2*pad))
	if w < 8 {
		w = 8
	}
	if h < 8 {
		h = 8
	}
	ox := pad - minX
	oy := pad - minY
	return image.NewGray(image.Rect(0, 0, w, h)), ox, oy, w, h
}

func translateContours(cs []Contour, dx, dy float64) []Contour {
	out := make([]Contour, len(cs))
	for i, c := range cs {
		out[i] = Contour{Closed: c.Closed, Pts: make([]Pt, len(c.Pts))}
		for j, p := range c.Pts {
			out[i].Pts[j] = Pt{p.X + dx, p.Y + dy}
		}
	}
	return out
}

func strokeContoursOnMask(mask *image.Gray, cs []Contour, radius float64) {
	for _, c := range cs {
		pts := c.Pts
		for i := 0; i < len(pts)-1; i++ {
			StampPathGray(mask, []Pt{pts[i], pts[i+1]}, radius, 255)
		}
		if c.Closed && len(pts) > 2 {
			StampPathGray(mask, []Pt{pts[len(pts)-1], pts[0]}, radius, 255)
		}
		for _, p := range pts {
			stampDiscGray(mask, p, radius, 255)
		}
	}
}

type pixelComp struct {
	pixels []int
	area   int
}

// findCounters returns background components not connected to the mask border.
func findCounters(mask *image.Gray, w, h int) []pixelComp {
	b := mask.Bounds()
	W, H := b.Dx(), b.Dy()
	visited := make([]bool, W*H)
	isBg := func(x, y int) bool {
		if x < 0 || y < 0 || x >= W || y >= H {
			return false
		}
		return mask.GrayAt(x+b.Min.X, y+b.Min.Y).Y < 128
	}
	// Mark every background component touching the border as exterior.
	var queue []int
	mark := func(start int) {
		if visited[start] {
			return
		}
		sx, sy := start%W, start/W
		if !isBg(sx, sy) {
			return
		}
		visited[start] = true
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			idx := queue[0]
			queue = queue[1:]
			x, y := idx%W, idx/W
			nb := []int{x - 1 + y*W, x + 1 + y*W, x + (y-1)*W, x + (y+1)*W}
			for _, nidx := range nb {
				if nidx < 0 || nidx >= W*H || visited[nidx] {
					continue
				}
				nx, ny := nidx%W, nidx/W
				if !isBg(nx, ny) {
					continue
				}
				visited[nidx] = true
				queue = append(queue, nidx)
			}
		}
	}
	for x := 0; x < W; x++ {
		mark(x)
		mark(x + (H-1)*W)
	}
	for y := 0; y < H; y++ {
		mark(y * W)
		mark(W - 1 + y*W)
	}
	var counters []pixelComp
	for idx := 0; idx < W*H; idx++ {
		if visited[idx] || mask.Pix[idx] >= 128 {
			continue
		}
		// Unvisited background => enclosed counter; flood its pixels.
		comp := pixelComp{}
		visited[idx] = true
		q := []int{idx}
		for len(q) > 0 {
			cur := q[len(q)-1]
			q = q[:len(q)-1]
			comp.pixels = append(comp.pixels, cur)
			x, y := cur%W, cur/W
			for _, nidx := range []int{x - 1 + y*W, x + 1 + y*W, x + (y-1)*W, x + (y+1)*W} {
				if nidx < 0 || nidx >= W*H || visited[nidx] {
					continue
				}
				nx, ny := nidx%W, nidx/W
				if !isBg(nx, ny) {
					continue
				}
				visited[nidx] = true
				q = append(q, nidx)
			}
		}
		comp.area = len(comp.pixels)
		if comp.area >= 16 {
			counters = append(counters, comp)
		}
	}
	return counters
}

func findCountersFromOutline(o Outline, ox, oy float64, w, h int) []pixelComp {
	mask := image.NewGray(image.Rect(0, 0, w, h))
	FillGray(mask, translateContours(o.Contours, ox, oy), 255)
	return findCounters(mask, w, h)
}

// survivingCoreDepth computes a conservative erosion depth of the surviving
// background pixels that belong to compPixels.
func survivingCoreDepth(mask *image.Gray, compPixels []int) int {
	W := mask.Stride
	H := mask.Bounds().Dy()
	inComp := make(map[int]bool, len(compPixels))
	cur := make([]int, 0, len(compPixels))
	for _, idx := range compPixels {
		if mask.Pix[idx] < 128 {
			inComp[idx] = true
			cur = append(cur, idx)
		}
	}
	depth := 0
	for len(cur) > 0 {
		var next []int
		for _, idx := range cur {
			x, y := idx%W, idx/W
			ok := true
			for _, nidx := range []int{x - 1 + y*W, x + 1 + y*W, x + (y-1)*W, x + (y+1)*W} {
				if nidx < 0 || nidx >= W*H || !inComp[nidx] || mask.Pix[nidx] >= 128 {
					ok = false
					break
				}
			}
			if ok {
				next = append(next, idx)
			}
		}
		if len(next) == 0 {
			break
		}
		depth++
		// Erode: only pixels that had all neighbors survive remain.
		newSet := make(map[int]bool, len(next))
		for _, idx := range next {
			newSet[idx] = true
		}
		inComp = newSet
		cur = next
		if depth > 40 {
			break
		}
	}
	return depth
}
