package demo

import (
	"image"
	"math"
)

// ClipRect intersects the mask with an integer pixel rectangle. Pixels are
// addressed by their enclosing unit square: r.Min is inclusive and r.Max is
// exclusive, matching image.Rectangle. Clipping is a binary AND, so an edge
// pixel is either fully kept (255) or fully removed (0); there is no coverage
// value that could show up as a half-pixel seam.
func ClipRect(m *Mask, r image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, m.W, m.H))
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if x < r.Min.X || y < r.Min.Y || x >= r.Max.X || y >= r.Max.Y {
				m.Pix[y*m.Stride+x] = 0
			}
		}
	}
}

// ClipByMask intersects m with an arbitrary binary clip mask (for example a
// path clip produced by FillPath). Dimensions must match.
func ClipByMask(m *Mask, clip *Mask) {
	if clip.W != m.W || clip.H != m.H {
		panic("ClipByMask: mismatched mask dimensions")
	}
	for i := range m.Pix {
		if clip.Pix[i] == 0 {
			m.Pix[i] = 0
		}
	}
}

// ClipPath intersects m with the fill of an arbitrary clipping path.
func ClipPath(m *Mask, clipPath *Path, rule FillRule) {
	cm := NewMask(m.W, m.H)
	FillPath(cm, clipPath, rule)
	ClipByMask(m, cm)
}

// SnapClipRectOutward converts a floating-point clip rectangle to whole
// device pixels by expanding it outward (floor of min, ceil of max). This is
// the key anti-seam rule for non-integer clip coordinates: no pixel touched by
// the requested region is downgraded to partial coverage, so the clipped edge
// stays a continuous run of fully covered pixels instead of a half-pixel gap.
func SnapClipRectOutward(x0, y0, x1, y1 float64) image.Rectangle {
	return image.Rect(
		int(math.Floor(x0)),
		int(math.Floor(y0)),
		int(math.Ceil(x1)),
		int(math.Ceil(y1)),
	)
}

// SnapClipRectInward is the conservative variant (ceil min, floor max).
func SnapClipRectInward(x0, y0, x1, y1 float64) image.Rectangle {
	return image.Rect(
		int(math.Ceil(x0)),
		int(math.Ceil(y0)),
		int(math.Floor(x1)),
		int(math.Floor(y1)),
	)
}

// SeamReport describes the integrity of one straight clip edge.
type SeamReport struct {
	Edge          string `json:"edge"`
	Fixed         int    `json:"fixed_coordinate"`
	ExclusiveMax  bool   `json:"exclusive_max"`
	BoundaryPixel int    `json:"boundary_pixel_coordinate"`
	Samples       int    `json:"samples"`
	FullEdge      int    `json:"full_edge_pixels"`
	MissingEdge   int    `json:"missing_edge_pixels"`
	Gaps          int    `json:"sandwiched_zero_gaps"`
	Partial       bool   `json:"has_partial_coverage"`
	OK            bool   `json:"ok"`
}

// CheckVerticalClipSeam inspects a vertical clip boundary at x=edgeX.
// exclusiveMax follows image.Rectangle semantics: false means edgeX is the
// inclusive Min edge and itself the first kept pixel; true means edgeX is the
// exclusive Max edge and edgeX-1 is the last kept pixel. Rows y0..y1 are
// sampled. For every row where the shape is covered on both sides of the
// boundary in the pre-clip reference mask, the clipped mask must keep the
// boundary pixel fully covered (no shaved half pixel) and must not contain a
// zero column sandwiched between covered columns near the seam.
func CheckVerticalClipSeam(clipped, before *Mask, edgeX int, exclusiveMax bool, y0, y1 int) SeamReport {
	bp := edgeX
	if exclusiveMax {
		bp = edgeX - 1
	}
	rep := SeamReport{Edge: "vertical", Fixed: edgeX, ExclusiveMax: exclusiveMax, BoundaryPixel: bp}
	rep.Partial = clipped.HasIntermediateValue()
	for y := y0; y <= y1; y++ {
		if y < 1 || y >= clipped.H-1 || edgeX < 2 || edgeX+2 >= clipped.W {
			continue
		}
		// Shape must cross the seam: reference coverage on both sides and at seam.
		if before.At(edgeX-2, y) == 0 || before.At(edgeX+1, y) == 0 {
			continue
		}
		rep.Samples++
		if clipped.At(bp, y) == 255 {
			rep.FullEdge++
		} else {
			rep.MissingEdge++
		}
		// Sandwiched gap: covered at x-1 and x+1 but zero at x around seam.
		for x := edgeX - 2; x <= edgeX+1; x++ {
			if clipped.At(x-1, y) == 255 && clipped.At(x, y) == 0 && clipped.At(x+1, y) == 255 {
				rep.Gaps++
			}
		}
	}
	rep.OK = rep.MissingEdge == 0 && rep.Gaps == 0 && !rep.Partial && rep.Samples > 0
	return rep
}

// CheckHorizontalClipSeam is the Y-axis counterpart of CheckVerticalClipSeam.
func CheckHorizontalClipSeam(clipped, before *Mask, edgeY int, exclusiveMax bool, x0, x1 int) SeamReport {
	bp := edgeY
	if exclusiveMax {
		bp = edgeY - 1
	}
	rep := SeamReport{Edge: "horizontal", Fixed: edgeY, ExclusiveMax: exclusiveMax, BoundaryPixel: bp}
	rep.Partial = clipped.HasIntermediateValue()
	for x := x0; x <= x1; x++ {
		if x < 1 || x >= clipped.W-1 || edgeY < 2 || edgeY+2 >= clipped.H {
			continue
		}
		if before.At(x, edgeY-2) == 0 || before.At(x, edgeY+1) == 0 {
			continue
		}
		rep.Samples++
		if clipped.At(x, bp) == 255 {
			rep.FullEdge++
		} else {
			rep.MissingEdge++
		}
		for y := edgeY - 2; y <= edgeY+1; y++ {
			if clipped.At(x, y-1) == 255 && clipped.At(x, y) == 0 && clipped.At(x, y+1) == 255 {
				rep.Gaps++
			}
		}
	}
	rep.OK = rep.MissingEdge == 0 && rep.Gaps == 0 && !rep.Partial && rep.Samples > 0
	return rep
}
