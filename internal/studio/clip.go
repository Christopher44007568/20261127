package studio

import "math"

// ClipStack is an intersection stack of coverage masks. Pushing a mask never
// mutates the caller's slice, so a stack can safely share the scene group mask.
type ClipStack struct {
	W, H    int
	current []float32
	saved   [][]float32
}

func NewClipStack(w, h int) *ClipStack {
	c := &ClipStack{W: w, H: h}
	c.current = make([]float32, w*h)
	for i := range c.current {
		c.current[i] = 1
	}
	return c
}

func (c *ClipStack) Depth() int {
	return len(c.saved)
}

func (c *ClipStack) Current() []float32 {
	return c.current
}

func (c *ClipStack) Push(mask []float32) {
	if len(mask) != c.W*c.H {
		panic("studio: clip mask size mismatch")
	}
	next := make([]float32, len(c.current))
	for i := range next {
		next[i] = c.current[i] * clamp01f(mask[i])
	}
	c.saved = append(c.saved, c.current)
	c.current = next
}

func (c *ClipStack) PushRect(x, y, w, h float32) {
	c.Push(RectMask(c.W, c.H, x, y, w, h))
}

func (c *ClipStack) PushCircle(cx, cy, radius float32) {
	c.Push(CircleMask(c.W, c.H, cx, cy, radius))
}

func (c *ClipStack) Pop() {
	n := len(c.saved)
	if n == 0 {
		panic("studio: clip stack underflow")
	}
	c.current = c.saved[n-1]
	c.saved = c.saved[:n-1]
}

// Apply multiplies a layer's premultiplied RGBA by the current clip coverage.
func (c *ClipStack) Apply(im *FloatImage) {
	if im.W != c.W || im.H != c.H {
		panic("studio: clip image size mismatch")
	}
	im.ApplyMask(c.current)
}

// RectMask returns hard-edged rectangular coverage. Scene clips use the
// anti-aliased shape masks; this primitive is mainly for frame bounds/tests.
func RectMask(w, h int, x, y, rw, rh float32) []float32 {
	m := make([]float32, w*h)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			if float32(px) >= x && float32(px) < x+rw &&
				float32(py) >= y && float32(py) < y+rh {
				m[py*w+px] = 1
			}
		}
	}
	return m
}

// CircleMask uses a one-pixel-wide analytic footprint. The result is a true
// coverage mask rather than a binary edge.
func CircleMask(w, h int, cx, cy, radius float32) []float32 {
	m := make([]float32, w*h)
	r2 := (radius + 0.5) * (radius + 0.5)
	ri2 := (radius - 0.5) * (radius - 0.5)
	for py := 0; py < h; py++ {
		dy := float32(py) + 0.5 - cy
		for px := 0; px < w; px++ {
			dx := float32(px) + 0.5 - cx
			d2 := dx*dx + dy*dy
			var cov float32
			switch {
			case d2 <= ri2:
				cov = 1
			case d2 < r2:
				// Approximates the chord area in the boundary pixel.
				cov = clamp01f(radius + 0.5 - sqrt32(d2))
			}
			m[py*w+px] = cov
		}
	}
	return m
}

// UnionMask combines shape coverages. Max is intentional: it keeps shared
// glyph strokes from becoming denser where two strokes meet.
func UnionMask(dst, src []float32) {
	if len(dst) != len(src) {
		panic("studio: mask union size mismatch")
	}
	for i := range dst {
		if src[i] > dst[i] {
			dst[i] = src[i]
		}
	}
}

func MultiplyMask(a, b []float32) []float32 {
	if len(a) != len(b) {
		panic("studio: mask multiply size mismatch")
	}
	out := make([]float32, len(a))
	for i := range out {
		out[i] = clamp01f(a[i]) * clamp01f(b[i])
	}
	return out
}

func sqrt32(v float32) float32 {
	if v <= 0 {
		return 0
	}
	return float32(math.Sqrt(float64(v)))
}
