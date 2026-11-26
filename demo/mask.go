package demo

import "image"

// FillRule selects how overlapping/nested contours decide coverage.
type FillRule int

const (
	// FillEvenOdd applies parity: a point is inside after an odd number of
	// ray crossings, regardless of contour direction.
	FillEvenOdd FillRule = iota
	// FillNonZero applies the signed winding number: a point is inside when
	// the summed signed crossings is not zero.
	FillNonZero
)

func (r FillRule) String() string {
	switch r {
	case FillEvenOdd:
		return "evenodd"
	case FillNonZero:
		return "nonzero"
	default:
		return "unknown"
	}
}

// Mask is a binary coverage mask. Pix[x+y*Stride] is either 0 or 255; the
// pipeline deliberately has no fractional coverage so clipped seams can never
// contain a half-covered pixel.
type Mask struct {
	W, H   int
	Stride int
	Pix    []byte
}

// NewMask creates a cleared mask.
func NewMask(w, h int) *Mask {
	return &Mask{W: w, H: h, Stride: w, Pix: make([]byte, w*h)}
}

// Bounds returns the mask rectangle.
func (m *Mask) Bounds() image.Rectangle { return image.Rect(0, 0, m.W, m.H) }

// At returns 0 or 255 for a pixel.
func (m *Mask) At(x, y int) byte {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return 0
	}
	return m.Pix[y*m.Stride+x]
}

// Set writes a coverage value.
func (m *Mask) Set(x, y int, v byte) {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return
	}
	m.Pix[y*m.Stride+x] = v
}

// Clone returns an independent copy.
func (m *Mask) Clone() *Mask {
	c := NewMask(m.W, m.H)
	c.Stride = m.Stride
	c.Pix = append([]byte(nil), m.Pix...)
	return c
}

// Count returns the number of covered pixels.
func (m *Mask) Count() int {
	n := 0
	for _, v := range m.Pix {
		if v != 0 {
			n++
		}
	}
	return n
}

// HasIntermediateValue reports whether any pixel value lies strictly between
// 0 and 255. The binary pipeline always returns false; the helper exists so
// tests and reports can prove the "no half-pixel seam" invariant.
func (m *Mask) HasIntermediateValue() bool {
	for _, v := range m.Pix {
		if v != 0 && v != 255 {
			return true
		}
	}
	return false
}

// FillPath rasterizes every subpath of path into m using rule. All subpaths
// are evaluated together, which is the root-cause fix for "main contour
// covers the hole": hole contours contribute zero/even winding in the same
// classification instead of being painted over afterwards.
func FillPath(m *Mask, path *Path, rule FillRule) {
	loops := path.Flatten(0.35)
	for y := 0; y < m.H; y++ {
		py := float64(y) + 0.5
		for x := 0; x < m.W; x++ {
			px := float64(x) + 0.5
			inside, _ := ClassifyLoops(loops, px, py)
			if rule == FillEvenOdd {
				if inside.parity {
					m.Pix[y*m.Stride+x] = 255
				}
			} else {
				if inside.winding != 0 {
					m.Pix[y*m.Stride+x] = 255
				}
			}
		}
	}
}

// classification holds both fill-rule decisions for a sample point.
type classification struct {
	parity  bool
	winding int
}

// ClassifyPoint classifies a point against a path under both rules.
func ClassifyPoint(path *Path, x, y float64) (evenOdd bool, winding int) {
	c, _ := ClassifyLoops(path.Flatten(0.2), x, y)
	return c.parity, c.winding
}

// ClassifyLoops runs the half-open ray crossing / winding-number algorithm.
// The horizontal ray starts at the point and goes to +X. The straddle tests
// (<= low, > high) give each non-vertex scanline a stable result; callers
// sample at pixel centers (x.5, y.5) while demo geometry uses integer vertex
// coordinates, so samples never land exactly on vertices.
func ClassifyLoops(loops [][]Vec2, px, py float64) (classification, bool) {
	var c classification
	onEdge := false
	for _, loop := range loops {
		n := len(loop)
		if n < 3 {
			continue
		}
		for i := 0; i < n-1; i++ {
			a, b := loop[i], loop[i+1]
			cross := crossValue(a, b, px, py)
			if a.Y <= py && b.Y > py {
				// Upward straddle: crossing lies right of P iff cross > 0.
				if cross > 0 {
					c.winding++
					c.parity = !c.parity
				} else if cross == 0 {
					onEdge = true
				}
			} else if a.Y > py && b.Y <= py {
				// Downward straddle: crossing lies right of P iff cross < 0.
				if cross < 0 {
					c.winding--
					c.parity = !c.parity
				} else if cross == 0 {
					onEdge = true
				}
			}
		}
	}
	return c, onEdge
}

// crossValue is (B-A) x (P-A); its sign tells which side of directed edge A->B
// the point P lies on.
func crossValue(a, b Vec2, px, py float64) float64 {
	return (b.X-a.X)*(py-a.Y) - (px-a.X)*(b.Y-a.Y)
}

// FillPathInRect is a convenience that fills path then intersects with a
// pixel-aligned clip rectangle.
func FillPathInRect(m *Mask, path *Path, rule FillRule, clip image.Rectangle) {
	FillPath(m, path, rule)
	ClipRect(m, clip)
}
