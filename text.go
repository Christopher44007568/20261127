package main

import (
	"image"
	"math"
)

type vectorGlyph struct {
	advance float64
	width   float64
	paths   [][]PointF
	closed  []bool
}

func ellipseArc(cx, cy, rx, ry, startDeg, endDeg float64, segments int) []PointF {
	points := make([]PointF, 0, segments+1)
	start := startDeg * math.Pi / 180
	end := endDeg * math.Pi / 180
	for i := 0; i <= segments; i++ {
		t := float64(i) / float64(segments)
		angle := start + (end-start)*t
		points = append(points, PointF{
			X: cx + math.Cos(angle)*rx,
			Y: cy + math.Sin(angle)*ry,
		})
	}
	return points
}

func vectorGlyphs() map[rune]vectorGlyph {
	o := ellipseArc(40, 50, 29, 38, 0, 360, 56)
	c := ellipseArc(40, 50, 29, 38, 48, 312, 48)
	gArc := ellipseArc(40, 50, 29, 38, 48, 312, 48)
	pBowlRight := ellipseArc(39, 36, 25, 22, -90, 90, 28)

	return map[rune]vectorGlyph{
		'A': {
			advance: 72, width: 12,
			paths: [][]PointF{
				{{X: 14, Y: 86}, {X: 38, Y: 14}, {X: 63, Y: 86}},
				{{X: 24, Y: 59}, {X: 53, Y: 59}},
			},
		},
		'D': {
			advance: 74, width: 12,
			paths: [][]PointF{
				{
					{X: 22, Y: 14}, {X: 44, Y: 14}, {X: 61, Y: 29},
					{X: 61, Y: 71}, {X: 44, Y: 86}, {X: 22, Y: 86},
				},
			},
			closed: []bool{true},
		},
		'E': {
			advance: 68, width: 12,
			paths: [][]PointF{
				{
					{X: 62, Y: 14}, {X: 22, Y: 14},
					{X: 22, Y: 86}, {X: 62, Y: 86},
				},
				{{X: 22, Y: 50}, {X: 54, Y: 50}},
			},
		},
		'O': {advance: 80, width: 14, paths: [][]PointF{o}, closed: []bool{true}},
		'C': {advance: 76, width: 14, paths: [][]PointF{c}, closed: []bool{false}},
		'G': {
			advance: 80, width: 14,
			paths: [][]PointF{
				gArc,
				{{X: 40, Y: 50}, {X: 69, Y: 50}},
				{{X: 69, Y: 50}, {X: 69, Y: 66}},
			},
			closed: []bool{false, false, false},
		},
		'L': {
			advance: 72, width: 14,
			paths: [][]PointF{
				{{X: 25, Y: 14}, {X: 25, Y: 86}, {X: 68, Y: 86}},
			},
			closed: []bool{false},
		},
		'I': {
			advance: 56, width: 14,
			paths: [][]PointF{
				{{X: 28, Y: 14}, {X: 28, Y: 86}},
				{{X: 16, Y: 14}, {X: 40, Y: 14}},
				{{X: 16, Y: 86}, {X: 40, Y: 86}},
			},
			closed: []bool{false, false, false},
		},
		'N': {
			advance: 74, width: 12,
			paths: [][]PointF{
				{{X: 22, Y: 86}, {X: 22, Y: 14}, {X: 60, Y: 86}, {X: 60, Y: 14}},
			},
		},
		'P': {
			advance: 76, width: 14,
			paths: [][]PointF{
				{{X: 24, Y: 86}, {X: 24, Y: 14}},
				append([]PointF{{X: 24, Y: 15}}, pBowlRight...),
				{{X: 63, Y: 57}, {X: 24, Y: 57}},
			},
			closed: []bool{false, false, false},
		},
		'R': {
			advance: 74, width: 12,
			paths: [][]PointF{
				{
					{X: 22, Y: 86}, {X: 22, Y: 14}, {X: 47, Y: 14},
					{X: 61, Y: 24}, {X: 61, Y: 40}, {X: 47, Y: 52}, {X: 22, Y: 52},
				},
				{{X: 43, Y: 52}, {X: 64, Y: 86}},
			},
		},
		'S': {
			advance: 70, width: 12,
			paths: [][]PointF{
				{
					{X: 60, Y: 19}, {X: 29, Y: 19}, {X: 21, Y: 28},
					{X: 21, Y: 41}, {X: 31, Y: 50}, {X: 51, Y: 50},
					{X: 61, Y: 59}, {X: 61, Y: 72}, {X: 53, Y: 81}, {X: 22, Y: 81},
				},
			},
		},
		'T': {
			advance: 68, width: 12,
			paths: [][]PointF{
				{{X: 13, Y: 14}, {X: 65, Y: 14}},
				{{X: 39, Y: 14}, {X: 39, Y: 86}},
			},
		},
		'V': {
			advance: 72, width: 12,
			paths: [][]PointF{
				{{X: 15, Y: 14}, {X: 39, Y: 86}, {X: 63, Y: 14}},
			},
		},
		'X': {
			advance: 70, width: 12,
			paths: [][]PointF{
				{{X: 17, Y: 14}, {X: 61, Y: 86}},
				{{X: 61, Y: 14}, {X: 17, Y: 86}},
			},
		},
	}
}

func MeasureVectorText(text string, em float64) float64 {
	glyphs := vectorGlyphs()
	units := 0.0
	for i, r := range text {
		glyph, ok := glyphs[r]
		if !ok {
			units += 72
		} else {
			units += glyph.advance
		}
		if i < len(text)-1 {
			units += 4
		}
	}
	return units * em / 100
}

func TextMask(bounds image.Rectangle, text string, left, top, em float64, samples int) *Mask {
	glyphs := vectorGlyphs()
	scale := em / 100
	cursor := left
	var paths []StrokePath

	for _, r := range text {
		glyph, ok := glyphs[r]
		if !ok {
			cursor += 76 * scale
			continue
		}
		for pathIndex, source := range glyph.paths {
			transformed := make([]PointF, len(source))
			for i, p := range source {
				transformed[i] = PointF{
					X: cursor + p.X*scale,
					Y: top + p.Y*scale,
				}
			}
			closed := false
			if pathIndex < len(glyph.closed) {
				closed = glyph.closed[pathIndex]
			}
			paths = append(paths, StrokePath{
				Points: transformed,
				Width:  glyph.width * scale,
				Closed: closed,
			})
		}
		cursor += (glyph.advance + 4) * scale
	}
	return StrokeMask(bounds, paths, samples)
}
