package main

import (
	"math"
	"sort"
)

// Pchip is a monotone (Fritsch-Carlson) cubic Hermite interpolant.
// It is C1 continuous and never overshoots between nodes, which keeps
// the tube radius continuous and bounded under arbitrary resampling.
type Pchip struct {
	x, y, m []float64
}

func NewPchip(x, y []float64) *Pchip {
	n := len(x)
	p := &Pchip{x: append([]float64(nil), x...), y: append([]float64(nil), y...)}
	m := make([]float64, n)
	if n == 1 {
		p.m = m
		return p
	}
	h := make([]float64, n-1)
	d := make([]float64, n-1)
	for i := 0; i < n-1; i++ {
		h[i] = x[i+1] - x[i]
		if h[i] <= 0 {
			h[i] = 1e-12
		}
		d[i] = (y[i+1] - y[i]) / h[i]
	}
	if n == 2 {
		m[0], m[1] = d[0], d[0]
		p.m = m
		return p
	}
	m[0] = pchipEdge(h[0], h[1], d[0], d[1])
	m[n-1] = pchipEdge(h[n-2], h[n-3], d[n-2], d[n-3])
	for i := 1; i < n-1; i++ {
		if d[i-1]*d[i] <= 0 {
			m[i] = 0
		} else {
			w1 := 2*h[i] + h[i-1]
			w2 := h[i] + 2*h[i-1]
			m[i] = (w1 + w2) / (w1/d[i-1] + w2/d[i])
		}
	}
	p.m = m
	return p
}

func pchipEdge(h0, h1, d0, d1 float64) float64 {
	m := ((2*h0+h1)*d0 - h0*d1) / (h0 + h1)
	if m*d0 <= 0 {
		return 0
	}
	if d0*d1 < 0 && math.Abs(m) > 3*math.Abs(d0) {
		return 3 * d0
	}
	return m
}

// Eval interpolates at xq; constant extrapolation outside the range.
func (p *Pchip) Eval(xq float64) float64 {
	n := len(p.x)
	if n == 1 {
		return p.y[0]
	}
	if xq <= p.x[0] {
		return p.y[0]
	}
	if xq >= p.x[n-1] {
		return p.y[n-1]
	}
	i := sort.Search(n-1, func(i int) bool { return p.x[i+1] >= xq })
	if i > n-2 {
		i = n - 2
	}
	h := p.x[i+1] - p.x[i]
	t := (xq - p.x[i]) / h
	t2, t3 := t*t, t*t*t
	y0, y1 := p.y[i], p.y[i+1]
	m0, m1 := p.m[i]*h, p.m[i+1]*h
	return (2*t3-3*t2+1)*y0 + (t3-2*t2+t)*m0 + (-2*t3+3*t2)*y1 + (t3-t2)*m1
}
