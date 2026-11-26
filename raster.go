package main

import (
	"image"
	"math"
)

type flatSub struct {
	s          Sub
	minX, minY float64
	maxX, maxY float64
}

func flattenPaths(paths []*Path, tol float64) []flatSub {
	var out []flatSub
	for _, p := range paths {
		for _, s := range p.Flatten(tol) {
			fs := flatSub{s: s}
			fs.minX, fs.minY = math.Inf(1), math.Inf(1)
			fs.maxX, fs.maxY = math.Inf(-1), math.Inf(-1)
			for _, pt := range s.Pts {
				fs.minX = math.Min(fs.minX, pt.X)
				fs.maxX = math.Max(fs.maxX, pt.X)
				fs.minY = math.Min(fs.minY, pt.Y)
				fs.maxY = math.Max(fs.maxY, pt.Y)
			}
			out = append(out, fs)
		}
	}
	return out
}

func flatContains(fs []flatSub, q Pt) bool {
	wn := 0
	for _, f := range fs {
		if q.X < f.minX || q.X > f.maxX || q.Y < f.minY || q.Y > f.maxY {
			continue
		}
		pts := f.s.Pts
		n := len(pts)
		if n < 3 {
			continue
		}
		for i := 0; i < n; i++ {
			a := pts[i]
			b := pts[(i+1)%n]
			if !f.s.Closed && i == n-1 {
				break
			}
			if a.Y <= q.Y {
				if b.Y > q.Y && cross(b.sub(a), q.sub(a)) > 0 {
					wn++
				}
			} else if b.Y <= q.Y && cross(b.sub(a), q.sub(a)) < 0 {
				wn--
			}
		}
	}
	return wn != 0
}

func flatStroke(fs []flatSub, q Pt, halfW float64) bool {
	r2 := halfW * halfW
	for _, f := range fs {
		if q.X < f.minX-halfW || q.X > f.maxX+halfW ||
			q.Y < f.minY-halfW || q.Y > f.maxY+halfW {
			continue
		}
		pts := f.s.Pts
		n := len(pts)
		if n < 2 {
			continue
		}
		last := n - 1
		if f.s.Closed {
			last = n
		}
		for i := 0; i < last; i++ {
			a := pts[i%n]
			b := pts[(i+1)%n]
			if distToSeg2(q, a, b) <= r2 {
				return true
			}
		}
	}
	return false
}

func distToSeg2(p, a, b Pt) float64 {
	ab := b.sub(a)
	l2 := ab.X*ab.X + ab.Y*ab.Y
	if l2 == 0 {
		dx := p.X - a.X
		dy := p.Y - a.Y
		return dx*dx + dy*dy
	}
	t := ((p.X-a.X)*ab.X + (p.Y-a.Y)*ab.Y) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	dx := a.X + ab.X*t - p.X
	dy := a.Y + ab.Y*t - p.Y
	return dx*dx + dy*dy
}

// RenderResult holds the RGBA image plus the supersampled object ID buffer.
type RenderResult struct {
	Img   *image.RGBA
	SS    int
	idHi  []int32
	hiW   int
	hiH   int
	Subs  [][]flatSub
	Clip  map[string][]flatSub
	Scene *Scene
}

// Render rasterizes the scene with ss×ss supersampling and box downsample.
func Render(sc *Scene, ss int) *RenderResult {
	if ss < 1 {
		ss = 1
	}
	sc.indexObjects()
	r := &RenderResult{SS: ss, Scene: sc}
	W, H := sc.W, sc.H
	r.hiW, r.hiH = W*ss, H*ss
	hi := image.NewRGBA(image.Rect(0, 0, r.hiW, r.hiH))
	bg := sc.BG
	for i := range hi.Pix {
		hi.Pix[i] = 0
	}
	for y := 0; y < r.hiH; y++ {
		for x := 0; x < r.hiW; x++ {
			hi.SetRGBA(x, y, bg)
		}
	}
	r.idHi = make([]int32, r.hiW*r.hiH)
	for i := range r.idHi {
		r.idHi[i] = -1
	}

	r.Subs = make([][]flatSub, len(sc.All))
	for _, o := range sc.All {
		r.Subs[o.Index] = flattenPaths(o.Paths, 0.18)
	}
	r.Clip = map[string][]flatSub{}
	for _, c := range sc.Clips {
		r.Clip[c.ID] = flattenPaths(c.Paths, 0.18)
	}

	var draw func(o *Obj, inherited []string)
	draw = func(o *Obj, inherited []string) {
		clips := inherited
		if o.ClipID != "" {
			clips = append(append([]string{}, inherited...), o.ClipID)
		}
		if len(o.Children) > 0 {
			for _, ch := range o.Children {
				draw(ch, clips)
			}
			return
		}
		r.drawShape(hi, o, clips)
	}
	for _, o := range sc.Objs {
		draw(o, nil)
	}

	r.Img = downsample(hi, W, H, ss)
	return r
}

func (r *RenderResult) clipPass(clips []string, q Pt) bool {
	for _, id := range clips {
		fs := r.Clip[id]
		if fs == nil || !flatContains(fs, q) {
			return false
		}
	}
	return true
}

func (r *RenderResult) drawShape(hi *image.RGBA, o *Obj, clips []string) {
	fs := r.Subs[o.Index]
	if len(fs) == 0 {
		return
	}
	minX, minY, maxX, maxY := bboxOfSubs(fs)
	pad := 1.0
	halfW := 0.0
	if o.Stroke != nil && o.StrokeW > 0 {
		halfW = o.StrokeW / 2
		pad = halfW + 0.75
	}
	ss := r.SS
	x0 := clampInt(int(math.Floor((minX-pad)*float64(ss))), 0, r.hiW)
	x1 := clampInt(int(math.Ceil((maxX+pad)*float64(ss))), 0, r.hiW)
	y0 := clampInt(int(math.Floor((minY-pad)*float64(ss))), 0, r.hiH)
	y1 := clampInt(int(math.Ceil((maxY+pad)*float64(ss))), 0, r.hiH)

	for hy := y0; hy < y1; hy++ {
		qy := (float64(hy) + 0.5) / float64(ss)
		rowBase := hy * r.hiW
		for hx := x0; hx < x1; hx++ {
			q := Pt{(float64(hx) + 0.5) / float64(ss), qy}
			if !r.clipPass(clips, q) {
				continue
			}
			fillHit := o.Fill != nil && flatContains(fs, q)
			strokeHit := halfW > 0 && flatStroke(fs, q, halfW)
			if !fillHit && !strokeHit {
				continue
			}
			var cr, cg, cb, ca float64
			if fillHit {
				c := o.Fill.at(q)
				cr, cg, cb, ca = float64(c.R), float64(c.G), float64(c.B), float64(c.A)/255
			}
			if strokeHit {
				c := o.Stroke.at(q)
				sr, sg, sb := float64(c.R), float64(c.G), float64(c.B)
				sa := float64(c.A) / 255
				// stroke source-over fill at this sample
				oa := sa + ca*(1-sa)
				if oa > 0 {
					cr = (sr*sa + cr*ca*(1-sa)) / oa
					cg = (sg*sa + cg*ca*(1-sa)) / oa
					cb = (sb*sa + cb*ca*(1-sa)) / oa
					ca = oa
				}
			}
			if ca <= 0.001 {
				continue
			}
			idx := rowBase + hx
			off := idx * 4
			dst := hi.Pix[off : off+4]
			da := float64(dst[3]) / 255
			outA := ca + da*(1-ca)
			if outA > 0 {
				dst[0] = byte(clampF((cr*ca+float64(dst[0])*da*(1-ca))/outA + 0.5))
				dst[1] = byte(clampF((cg*ca+float64(dst[1])*da*(1-ca))/outA + 0.5))
				dst[2] = byte(clampF((cb*ca+float64(dst[2])*da*(1-ca))/outA + 0.5))
				dst[3] = byte(clampF(outA*255 + 0.5))
			}
			r.idHi[idx] = int32(o.Index)
		}
	}
}

func downsample(hi *image.RGBA, w, h, ss int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rr, gg, bb, aa int
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					i := ((y*ss+sy)*hi.Stride + (x*ss+sx)*4)
					rr += int(hi.Pix[i])
					gg += int(hi.Pix[i+1])
					bb += int(hi.Pix[i+2])
					aa += int(hi.Pix[i+3])
				}
			}
			n := ss * ss
			o := out.PixOffset(x, y)
			out.Pix[o] = byte(rr / n)
			out.Pix[o+1] = byte(gg / n)
			out.Pix[o+2] = byte(bb / n)
			out.Pix[o+3] = byte(aa / n)
		}
	}
	return out
}

// IDHistogram returns object-index -> sample count for the SS×SS block of px.
func (r *RenderResult) IDHistogram(px, py int) map[int]int {
	m := map[int]int{}
	ss := r.SS
	for sy := 0; sy < ss; sy++ {
		for sx := 0; sx < ss; sx++ {
			hx, hy := px*ss+sx, py*ss+sy
			if hx >= r.hiW || hy >= r.hiH {
				continue
			}
			id := r.idHi[hy*r.hiW+hx]
			if id >= 0 {
				m[int(id)]++
			}
		}
	}
	return m
}

// ModalID returns the most frequent object id in a pixel block, or -1.
func (r *RenderResult) ModalID(px, py int) int {
	best, bestN := -1, 0
	for id, n := range r.IDHistogram(px, py) {
		if n > bestN {
			best, bestN = id, n
		}
	}
	return best
}

// ObjectBBox returns scene bounds for an object (groups include children).
func (r *RenderResult) ObjectBBox(o *Obj) (float64, float64, float64, float64, bool) {
	if o.ClipID != "" {
		if cfs := r.Clip[o.ClipID]; len(cfs) > 0 {
			a, b, c, d := bboxOfSubs(cfs)
			return a, b, c, d, true
		}
	}
	var fs []flatSub
	var gather func(x *Obj)
	gather = func(x *Obj) {
		if len(x.Children) > 0 {
			for _, c := range x.Children {
				gather(c)
			}
		} else if x.Index < len(r.Subs) {
			fs = append(fs, r.Subs[x.Index]...)
		}
	}
	gather(o)
	if len(fs) == 0 {
		return 0, 0, 0, 0, false
	}
	a, b, c, d := bboxOfSubs(fs)
	return a, b, c, d, true
}

func bboxOfSubs(fs []flatSub) (float64, float64, float64, float64) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, f := range fs {
		minX = math.Min(minX, f.minX)
		minY = math.Min(minY, f.minY)
		maxX = math.Max(maxX, f.maxX)
		maxY = math.Max(maxY, f.maxY)
	}
	return minX, minY, maxX, maxY
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
