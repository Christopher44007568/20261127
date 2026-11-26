package demo

import "image"

// Logical scene panel size in device pixels.
const (
	SceneW = 360
	SceneH = 300
)

// StarPath is the canonical self-intersecting pentagram: one closed contour.
// EvenOdd punches the central pentagon while NonZero fills it (wn == 2).
func StarPath() *Path {
	p := &Path{}
	AddPentagram(p, 180, 152, 122)
	return p
}

// NestedPath contains three nesting levels with explicit opposing windings:
// outer rounded rectangle (clockwise), hole rectangle (counter-clockwise),
// inner island diamond (clockwise). Under both rules the ring between outer
// and hole is filled, the rectangle interior is a hole, and the diamond is an
// island.
func NestedPath() *Path {
	p := &Path{}
	AddRoundedRect(p, 36, 28, 324, 272, 30, WindingClockwise)
	AddRect(p, 108, 88, 252, 212, WindingCounterClock)
	AddDiamond(p, 180, 150, 36, 36, WindingClockwise)
	return p
}

// ClipRectNested cuts through the filled outer band on all four sides while
// keeping the hole and island visible. Integer coordinates mean whole-pixel
// clip membership.
var ClipRectNested = image.Rect(96, 70, 276, 234)

// ClipCircleStar is an arbitrary path-based clipping region for the star.
func ClipCircleStar() *Path {
	p := &Path{}
	AddCircle(p, 180, 152, 106, WindingClockwise)
	return p
}

// NamedMask pairs an output file stem with its mask.
type NamedMask struct {
	Name string
	Mask *Mask
}

// BuildAllMasks computes every standalone fill mask used by the demo.
func BuildAllMasks() map[string]*Mask {
	out := map[string]*Mask{}

	star := StarPath()
	starEO := NewMask(SceneW, SceneH)
	FillPath(starEO, star, FillEvenOdd)
	out["mask_star_evenodd"] = starEO

	starNZ := NewMask(SceneW, SceneH)
	FillPath(starNZ, star, FillNonZero)
	out["mask_star_nonzero"] = starNZ

	nested := NestedPath()
	nestEO := NewMask(SceneW, SceneH)
	FillPath(nestEO, nested, FillEvenOdd)
	out["mask_nested_evenodd"] = nestEO

	nestNZ := NewMask(SceneW, SceneH)
	FillPath(nestNZ, nested, FillNonZero)
	out["mask_nested_nonzero"] = nestNZ

	clipRect := NewMask(SceneW, SceneH)
	FillPath(clipRect, nested, FillNonZero)
	ClipRect(clipRect, ClipRectNested)
	out["mask_clipped_rect_nonzero"] = clipRect

	clipCircle := NewMask(SceneW, SceneH)
	FillPath(clipCircle, star, FillEvenOdd)
	ClipPath(clipCircle, ClipCircleStar(), FillNonZero)
	out["mask_clipped_circle_evenodd"] = clipCircle

	return out
}
