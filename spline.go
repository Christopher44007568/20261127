package main

import (
	"math"
	"sort"
)

// CameraKey is one keyframe of the spline camera.
type CameraKey struct {
	Time    float64
	Pos     Vec2
	Heading float64 // radians
	Zoom    float64
}

// CameraPath is a Catmull-Rom spline through camera keyframes. Position is
// interpolated with the spline while heading and zoom use shortest-arc/linear
// interpolation on the same timeline.
type CameraPath struct {
	Keys []CameraKey
}

// At evaluates the camera at time t (seconds), clamping outside the key range.
func (c CameraPath) At(t float64) CameraKey {
	n := len(c.Keys)
	if n == 1 {
		k := c.Keys[0]
		k.Time = t
		return k
	}
	if t <= c.Keys[0].Time {
		k := c.Keys[0]
		k.Time = t
		return k
	}
	if t >= c.Keys[n-1].Time {
		k := c.Keys[n-1]
		k.Time = t
		return k
	}
	// Segment [i, i+1] is the one whose time range contains t.
	i := sort.Search(n-1, func(j int) bool { return t <= c.Keys[j+1].Time })
	if i >= n-1 {
		i = n - 2
	}
	p0 := c.Keys[max(i-1, 0)].Pos
	p1 := c.Keys[i].Pos
	p2 := c.Keys[i+1].Pos
	p3 := c.Keys[min(i+2, n-1)].Pos
	span := c.Keys[i+1].Time - c.Keys[i].Time
	u := (t - c.Keys[i].Time) / span

	return CameraKey{
		Time:    t,
		Pos:     centripetalCatmullRom(p0, p1, p2, p3, u),
		Heading: AngleLerp(c.Keys[i].Heading, c.Keys[i+1].Heading, u),
		Zoom:    Lerp(c.Keys[i].Zoom, c.Keys[i+1].Zoom, u),
	}
}

// centripetalCatmullRom interpolates position with the centripetal
// parameterization (alpha=0.5). Unlike the uniform variant it never
// overshoots collinear control points, which keeps a straight camera move
// exactly straight. Zero-length parameter spans (coincident control points,
// including the duplicated end phantom points) collapse to their anchor so
// no division by zero or NaN can occur.
func centripetalCatmullRom(p0, p1, p2, p3 Vec2, u float64) Vec2 {
	const alpha = 0.5
	d := math.Pow(Dist(p1, p2), alpha)
	if d == 0 {
		return p1
	}
	t := u * d

	// First level: tangents between neighboring anchor pairs. A repeated
	// phantom point makes that outer span collapse onto the real anchor.
	d01 := math.Pow(Dist(p0, p1), alpha)
	var a1 Vec2
	if d01 == 0 {
		a1 = p1
	} else {
		v := Clamp((t+d01)/d01, 0, 1) // t in [-d01, 0] relative to p1
		a1 = LerpVec2(p0, p1, v)
	}
	a2 := p1.Scale(1 - t/d).Add(p2.Scale(t / d))
	d23 := math.Pow(Dist(p2, p3), alpha)
	var a3 Vec2
	if d23 == 0 {
		a3 = p2
	} else {
		v := Clamp(t/d23, 0, 1) // t in [0, d23] relative to p2
		a3 = LerpVec2(p2, p3, v)
	}

	// Second level spans the anchor interval [p1, p2].
	b1 := a1.Scale(1 - t/d).Add(a2.Scale(t / d))
	b2 := a2.Scale(1 - t/d).Add(a3.Scale(t / d))
	return b1.Scale(1 - t/d).Add(b2.Scale(t / d))
}

// CharacterKey is one keyframe of a character track. Equal positions on
// adjacent keys describe a deliberate pause on the shared timeline.
type CharacterKey struct {
	Time float64
	Pos  Vec2
}

// CharacterTrack is sampled piecewise-linearly. When the character pauses the
// camera timeline is untouched, so the camera keeps moving independently.
type CharacterTrack struct {
	Name string
	Keys []CharacterKey
}

// At samples the character position at time t, holding outside the key range.
func (tr CharacterTrack) At(t float64) Vec2 {
	keys := tr.Keys
	n := len(keys)
	if n == 1 || t <= keys[0].Time {
		return keys[0].Pos
	}
	if t >= keys[n-1].Time {
		return keys[n-1].Pos
	}
	// search over segment starts: first key whose time reaches t.
	i := sort.Search(n-1, func(j int) bool { return t <= keys[j+1].Time })
	if i >= n-1 {
		i = n - 2
	}
	span := keys[i+1].Time - keys[i].Time
	u := 0.0
	if span > 0 {
		u = (t - keys[i].Time) / span
	}
	return LerpVec2(keys[i].Pos, keys[i+1].Pos, u)
}

// Pauses reports the time intervals on which the character is stationary
// (zero-length segments of positive duration), merged when adjacent.
func (tr CharacterTrack) Pauses() [][2]float64 {
	var out [][2]float64
	for i := 0; i+1 < len(tr.Keys); i++ {
		if Dist(tr.Keys[i].Pos, tr.Keys[i+1].Pos) < 1e-12 &&
			tr.Keys[i+1].Time > tr.Keys[i].Time {
			seg := [2]float64{tr.Keys[i].Time, tr.Keys[i+1].Time}
			if m := len(out); m > 0 && math.Abs(out[m-1][1]-seg[0]) < 1e-12 {
				out[m-1][1] = seg[1]
			} else {
				out = append(out, seg)
			}
		}
	}
	return out
}
