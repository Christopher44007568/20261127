package main

import (
	"image"
	"image/color"
	"math"

	"image/draw"
)

// ObjStat is the per-original-object structural diff record.
type ObjStat struct {
	ID             string     `json:"id"`
	Tag            string     `json:"tag"`
	Note           string     `json:"note,omitempty"`
	Parent         string     `json:"parent,omitempty"`
	BBox           [4]float64 `json:"bbox"`
	CoveragePixels int        `json:"coveragePixels"`
	ErrorPixels    int        `json:"errorPixels"`
	ErrorRatio     float64    `json:"errorRatio"`
	MeanDelta      float64    `json:"meanDelta"`
	MaxDelta       float64    `json:"maxDelta"`
	Status         string     `json:"status"`
}

// DiffReport is the full structural/pixel diff result.
type DiffReport struct {
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	AlignX         int       `json:"alignX"`
	AlignY         int       `json:"alignY"`
	Threshold      float64   `json:"threshold"`
	TotalPixels    int       `json:"totalPixels"`
	ComparedPixels int       `json:"comparedPixels"`
	ErrorPixels    int       `json:"errorPixels"`
	ErrorFraction  float64   `json:"errorFraction"`
	MeanDelta      float64   `json:"meanDelta"`
	MaxDelta       float64   `json:"maxDelta"`
	Objects        []ObjStat `json:"objects"`
}

type pixelDiff struct {
	delta   []float64 // per-pixel RMS color distance
	errMask []bool
	w, h    int
	report  DiffReport
}

func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	return out
}

// alignRef picks the integer shift in [-2,2] that minimizes mean difference,
// protecting attribution from off-by-one viewport offsets.
func alignRef(ref, own *image.RGBA) (int, int) {
	bw, bh := own.Bounds().Dx(), own.Bounds().Dy()
	bestX, bestY, bestSad := 0, 0, math.Inf(1)
	for oy := -2; oy <= 2; oy++ {
		for ox := -2; ox <= 2; ox++ {
			var sad float64
			n := 0
			for y := 4; y < bh-4; y += 3 {
				ry := y + oy
				if ry < 0 || ry >= ref.Bounds().Dy() {
					continue
				}
				for x := 4; x < bw-4; x += 3 {
					rx := x + ox
					if rx < 0 || rx >= ref.Bounds().Dx() {
						continue
					}
					sad += pixelDist(ref, rx, ry, own, x, y)
					n++
				}
			}
			if n > 0 {
				sad /= float64(n)
			}
			if sad < bestSad {
				bestSad, bestX, bestY = sad, ox, oy
			}
		}
	}
	return bestX, bestY
}

func pixelDist(a *image.RGBA, ax, ay int, b *image.RGBA, bx, by int) float64 {
	ia := a.PixOffset(ax, ay)
	ib := b.PixOffset(bx, by)
	dr := float64(int(a.Pix[ia]) - int(b.Pix[ib]))
	dg := float64(int(a.Pix[ia+1]) - int(b.Pix[ib+1]))
	db := float64(int(a.Pix[ia+2]) - int(b.Pix[ib+2]))
	return math.Sqrt((dr*dr + dg*dg + db*db) / 3)
}

// ComputeDiff compares browser reference vs own render and attributes every
// mismatching pixel to its original object ID via the ID buffer.
func ComputeDiff(refImg image.Image, rr *RenderResult, threshold float64) (*pixelDiff, *DiffReport) {
	own := rr.Img
	W, H := rr.Scene.W, rr.Scene.H
	ref0 := toRGBA(refImg)
	ref := ref0
	if ref0.Bounds().Dx() != W || ref0.Bounds().Dy() != H {
		ref = resizeBilinear(ref0, W, H)
	}
	ax, ay := alignRef(ref, own)

	d := &pixelDiff{w: W, h: H, delta: make([]float64, W*H), errMask: make([]bool, W*H)}
	rep := &DiffReport{Width: W, Height: H, AlignX: ax, AlignY: ay, Threshold: threshold, TotalPixels: W * H}

	nObj := len(rr.Scene.All)
	type acc struct {
		cov, errPx int
		sumDelta   float64
		maxDelta   float64
	}
	accs := make([]acc, nObj)

	var sumAll float64
	compN := 0
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			rx, ry := x+ax, y+ay
			if rx < 0 || rx >= W || ry < 0 || ry >= H {
				continue
			}
			dv := pixelDist(ref, rx, ry, own, x, y)
			d.delta[i] = dv
			sumAll += dv
			compN++
			if dv > rep.MaxDelta {
				rep.MaxDelta = dv
			}
			isErr := dv > threshold
			d.errMask[i] = isErr
			if isErr {
				rep.ErrorPixels++
			}
			id := rr.ModalID(x, y)
			if id >= 0 && id < nObj {
				a := &accs[id]
				a.cov++
				a.sumDelta += dv
				if dv > a.maxDelta {
					a.maxDelta = dv
				}
				if isErr {
					a.errPx++
				}
			}
		}
	}
	rep.ComparedPixels = compN
	if compN > 0 {
		rep.MeanDelta = sumAll / float64(compN)
	}
	rep.ErrorFraction = float64(rep.ErrorPixels) / float64(W*H)

	for idx, o := range rr.Scene.All {
		a := accs[idx]
		st := ObjStat{
			ID: o.ID, Tag: o.Tag, Note: o.Note,
			CoveragePixels: a.cov, ErrorPixels: a.errPx,
			MaxDelta: round2(a.maxDelta),
		}
		if p := o.Parent; p != nil {
			st.Parent = p.ID
		}
		if a.cov > 0 {
			st.ErrorRatio = float64(a.errPx) / float64(a.cov)
			st.MeanDelta = round2(a.sumDelta / float64(a.cov))
			st.ErrorRatio = round4(st.ErrorRatio)
		}
		if x0, y0, x1, y1, ok := rr.ObjectBBox(o); ok {
			st.BBox = [4]float64{round1(x0), round1(y0), round1(x1), round1(y1)}
		}
		switch {
		case st.ErrorRatio >= 0.06 || st.ErrorPixels >= 400:
			st.Status = "error"
		case st.ErrorRatio >= 0.012 || st.ErrorPixels >= 80:
			st.Status = "minor"
		default:
			st.Status = "match"
		}
		rep.Objects = append(rep.Objects, st)
	}
	d.report = *rep
	return d, rep
}

// Heatmap renders the amplified delta with an iron colormap.
func (d *pixelDiff) Heatmap(gain float64) *image.RGBA {
	img := newCanvas(d.w, d.h, [3]uint8{10, 14, 26})
	for i, dv := range d.delta {
		t := math.Min(1, dv*gain/255)
		c := heatColor(t)
		o := i * 4
		img.Pix[o] = c.R
		img.Pix[o+1] = c.G
		img.Pix[o+2] = c.B
		img.Pix[o+3] = 255
	}
	return img
}

func heatColor(t float64) color.RGBA {
	type stop struct {
		at float64
		c  [3]float64
	}
	stops := []stop{
		{0, [3]float64{16, 20, 38}},
		{0.25, [3]float64{37, 99, 235}},
		{0.5, [3]float64{6, 182, 212}},
		{0.75, [3]float64{250, 204, 21}},
		{1, [3]float64{225, 29, 72}},
	}
	for i := 0; i+1 < len(stops); i++ {
		if t >= stops[i].at && t <= stops[i+1].at {
			k := (t - stops[i].at) / (stops[i+1].at - stops[i].at)
			return color.RGBA{
				R: uint8(stops[i].c[0] + (stops[i+1].c[0]-stops[i].c[0])*k),
				G: uint8(stops[i].c[1] + (stops[i+1].c[1]-stops[i].c[1])*k),
				B: uint8(stops[i].c[2] + (stops[i+1].c[2]-stops[i].c[2])*k),
				A: 255,
			}
		}
	}
	return color.RGBA{225, 29, 72, 255}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
