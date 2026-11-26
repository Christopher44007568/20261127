package stroke

import "math"

type dashInterval struct {
	start float64
	end   float64
}

type dashPiece struct {
	pts    []Point
	closed bool
}

func hasDash(pattern []float64) bool {
	var sum float64
	for _, v := range pattern {
		if v > 0 && isFinite(v) {
			sum += v
		}
	}
	return sum > 1e-12
}

func expandDashPattern(in []float64) []float64 {
	if len(in) == 0 {
		return nil
	}
	out := make([]float64, len(in))
	copy(out, in)
	for i, v := range out {
		if !isFinite(v) || v < 0 {
			out[i] = 0
		}
	}
	if len(out)%2 == 1 {
		out = append(out, out...)
	}
	if !hasDash(out) {
		return nil
	}
	return out
}

func styleAllowsPoint(style Style) bool {
	if style.Cap == CapButt {
		return false
	}
	if !hasDash(style.Dashes) {
		return true
	}
	return dashPointOn(expandDashPattern(style.Dashes), style.DashOffset)
}

func dashPointOn(pattern []float64, offset float64) bool {
	if len(pattern) == 0 {
		return true
	}
	period := dashPeriod(pattern)
	if period <= 0 {
		return false
	}
	phase := math.Mod(offset, period)
	if phase < 0 {
		phase += period
	}
	for idx := 0; idx < len(pattern); idx++ {
		l := pattern[idx]
		if l <= 0 {
			if phase == 0 && idx%2 == 0 {
				return true
			}
			continue
		}
		if phase < l-1e-12 {
			return idx%2 == 0
		}
		phase -= l
	}
	return false
}

func dashPeriod(pattern []float64) float64 {
	var total float64
	for _, v := range pattern {
		total += v
	}
	return total
}

func dashPolyline(pts []Point, closed bool, pattern []float64, offset float64) []dashPiece {
	metric := newPathMetric(pts, closed)
	if metric.total <= 1e-12 {
		if dashPointOn(pattern, offset) {
			return []dashPiece{{pts: []Point{pts[0]}}}
		}
		return nil
	}
	intervals := dashOnIntervals(metric.total, pattern, offset)
	if len(intervals) == 0 {
		return nil
	}
	pieces := make([]dashPiece, 0, len(intervals))
	for _, iv := range intervals {
		piece := metric.extract(iv.start, iv.end)
		if closed && len(piece) > 1 && pointsEqual(piece[0], piece[len(piece)-1]) {
			piece = piece[:len(piece)-1]
		}
		if len(piece) > 0 {
			pieces = append(pieces, dashPiece{pts: piece})
		}
	}
	if closed && len(intervals) == 1 &&
		intervals[0].start <= 1e-12 && intervals[0].end >= metric.total-1e-12 {
		pieces[0].closed = true
		return pieces
	}
	if closed && len(pieces) >= 2 {
		first, last := intervals[0], intervals[len(intervals)-1]
		if first.start <= 1e-12 && last.end >= metric.total-1e-12 {
			merged := append([]Point(nil), pieces[len(pieces)-1].pts...)
			merged = append(merged, pieces[0].pts[1:]...)
			pieces = append([]dashPiece{{pts: merged}}, pieces[1:len(pieces)-1]...)
		}
	}
	return pieces
}

func dashOnIntervals(length float64, pattern []float64, offset float64) []dashInterval {
	period := dashPeriod(pattern)
	if period <= 0 || length <= 0 {
		return nil
	}
	idx, local := locatePattern(pattern, period, offset)
	if local == 0 && pattern[idx] == 0 {
		idx = (idx + 1) % len(pattern)
	}
	pos := 0.0
	var out []dashInterval
	guard := 0
	for pos < length-1e-12 && guard < 1000000 {
		guard++
		l := pattern[idx]
		if l <= 0 {
			if idx%2 == 0 && pos > 0 {
				out = appendOnInterval(out, dashInterval{start: pos, end: pos})
			}
			idx = (idx + 1) % len(pattern)
			local = 0
			continue
		}
		remain := l - local
		take := math.Min(remain, length-pos)
		if idx%2 == 0 && take > 1e-12 {
			out = appendOnInterval(out, dashInterval{start: pos, end: pos + take})
		}
		pos += take
		local += take
		if local >= l-1e-12 {
			local = 0
			idx = (idx + 1) % len(pattern)
		}
	}
	return out
}

func appendOnInterval(out []dashInterval, iv dashInterval) []dashInterval {
	if len(out) > 0 {
		last := &out[len(out)-1]
		if iv.start <= last.end+1e-12 {
			if iv.end > last.end {
				last.end = iv.end
			}
			return out
		}
	}
	return append(out, iv)
}

func locatePattern(pattern []float64, period, offset float64) (int, float64) {
	phase := math.Mod(offset, period)
	if phase < 0 {
		phase += period
	}
	for idx := 0; idx < len(pattern); idx++ {
		l := pattern[idx]
		if l <= 0 {
			continue
		}
		if phase < l-1e-12 {
			return idx, phase
		}
		phase -= l
	}
	return 0, 0
}

type pathMetric struct {
	pts    []Point
	cum    []float64
	total  float64
	closed bool
}

func newPathMetric(pts []Point, closed bool) pathMetric {
	m := pathMetric{pts: pts, closed: closed}
	n := len(pts)
	if n == 0 {
		return m
	}
	if closed {
		m.cum = make([]float64, n+1)
		for i := 0; i < n; i++ {
			m.cum[i+1] = m.cum[i] + pointDistance(pts[i], pts[(i+1)%n])
		}
	} else {
		m.cum = make([]float64, n)
		for i := 1; i < n; i++ {
			m.cum[i] = m.cum[i-1] + pointDistance(pts[i-1], pts[i])
		}
	}
	m.total = m.cum[len(m.cum)-1]
	return m
}

func (m pathMetric) pointAt(d float64) Point {
	n := len(m.pts)
	if n == 0 {
		return Point{}
	}
	if d <= 0 {
		return m.pts[0]
	}
	if d >= m.total {
		if m.closed {
			return m.pts[0]
		}
		return m.pts[n-1]
	}
	lo, hi := 0, len(m.cum)-1
	for lo+1 < hi {
		mid := (lo + hi) / 2
		if m.cum[mid] <= d {
			lo = mid
		} else {
			hi = mid
		}
	}
	segLen := m.cum[lo+1] - m.cum[lo]
	if segLen <= 1e-15 {
		return m.pts[lo%n]
	}
	t := (d - m.cum[lo]) / segLen
	return add(m.pts[lo%n], scale(sub(m.pts[(lo+1)%n], m.pts[lo%n]), t))
}

func (m pathMetric) extract(a, b float64) []Point {
	if b < a {
		a, b = b, a
	}
	start := m.pointAt(a)
	if b-a <= 1e-12 {
		return []Point{start}
	}
	out := []Point{start}
	n := len(m.pts)
	limit := n
	if m.closed {
		limit = n + 1
	}
	for i := 1; i < limit; i++ {
		cum := m.cum[i]
		if cum <= a+1e-12 || cum >= b-1e-12 {
			continue
		}
		p := m.pts[i%n]
		if len(out) == 0 || !pointsEqual(out[len(out)-1], p) {
			out = append(out, p)
		}
	}
	end := m.pointAt(b)
	if len(out) == 0 || !pointsEqual(out[len(out)-1], end) {
		out = append(out, end)
	}
	return out
}
