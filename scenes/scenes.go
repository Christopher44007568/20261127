// Package scenes builds the small vector test scenes used by the demo:
// a sub-pixel thin horizontal line, a thin 45-degree diagonal, two crossing
// colored paths, and a set of disjoint colored contours.
package scenes

import (
	"coverlab/cover"
	"coverlab/geom"
)

// Palette colors (straight RGB in 0..1).
var (
	Ink   = cover.RGB{R: 0.10, G: 0.11, B: 0.14}
	Red   = cover.RGB{R: 0.83, G: 0.16, B: 0.20}
	Blue  = cover.RGB{R: 0.10, G: 0.34, B: 0.82}
	Green = cover.RGB{R: 0.13, G: 0.60, B: 0.30}
	Amber = cover.RGB{R: 0.92, G: 0.60, B: 0.10}
	Teal  = cover.RGB{R: 0.06, G: 0.55, B: 0.62}
	Pink  = cover.RGB{R: 0.80, G: 0.16, B: 0.62}
)

func stroke(name string, a, b geom.Vec, width float64, color cover.RGB) cover.Layer {
	return cover.Layer{
		Name:  name,
		Poly:  geom.SegmentOutline(a, b, width, width*0.5),
		Color: color,
	}
}

// ThinLineWidth is the deliberately sub-pixel stroke width used by the
// continuity experiment.
const ThinLineWidth = 0.52

// SweepPixel is the fixed pixel whose brightness is tracked while the thin
// line moves by fractions of a pixel.
const SweepPixelX, SweepPixelY = 12, 4

// SweepScene builds the horizontal thin line with its center at y=4.5+dy.
func SweepScene(dy float64) cover.Scene {
	y := 4.5 + dy
	return cover.Scene{
		Name: "thin_sweep",
		W:    26, H: 9,
		Layers: []cover.Layer{
			stroke("thin", geom.Vec{X: 3, Y: y}, geom.Vec{X: 23, Y: y}, ThinLineWidth, Ink),
		},
		Crop: [4]int{8, 2, 17, 7},
	}
}

// ThinH is a single fractional-position thin line used for the gallery.
func ThinH() cover.Scene {
	sc := SweepScene(0.18)
	sc.Name = "thin_h"
	sc.Crop = [4]int{10, 3, 16, 6}
	return sc
}

// Diagonal is a thin 45-degree line, the worst case for regular sampling.
func Diagonal() cover.Scene {
	return cover.Scene{
		Name: "diagonal",
		W:    30, H: 28,
		Layers: []cover.Layer{
			stroke("diag", geom.Vec{X: 4.5, Y: 24.5}, geom.Vec{X: 24.5, Y: 4.5}, 0.72, Ink),
		},
		Crop: [4]int{10, 10, 20, 20},
	}
}

// Cross has two differently colored diagonal paths intersecting.
func Cross() cover.Scene {
	return cover.Scene{
		Name: "cross",
		W:    36, H: 28,
		Layers: []cover.Layer{
			stroke("red_down", geom.Vec{X: 6, Y: 5}, geom.Vec{X: 30, Y: 23}, 0.85, Red),
			stroke("blue_up", geom.Vec{X: 6, Y: 23}, geom.Vec{X: 30, Y: 5}, 0.85, Blue),
		},
		Crop: [4]int{12, 8, 25, 21},
	}
}

// Segregate contains disjoint red/blue/green bars separated by real gaps.
// The middle column between the red and blue bars is a guard column that
// must render as pure background under any true coverage pipeline.
func Segregate() cover.Scene {
	return cover.Scene{
		Name: "segregate",
		W:    13, H: 15,
		Layers: []cover.Layer{
			stroke("red", geom.Vec{X: 3.0, Y: 2.5}, geom.Vec{X: 3.0, Y: 12.5}, 1.0, Red),
			stroke("blue", geom.Vec{X: 5.8, Y: 2.5}, geom.Vec{X: 5.8, Y: 12.5}, 1.0, Blue),
			stroke("green", geom.Vec{X: 9.4, Y: 2.5}, geom.Vec{X: 9.4, Y: 12.5}, 1.0, Green),
		},
		Crop: [4]int{1, 1, 12, 14},
	}
}

// NearDiag is a pair of disjoint, close parallel diagonals. Their geometric
// gap is ~1.5px, small enough for blur to bridge but large enough that true
// coverage never transfers color between the contours.
func NearDiag() cover.Scene {
	return cover.Scene{
		Name: "near_diag",
		W:    24, H: 22,
		Layers: []cover.Layer{
			stroke("pink_a", geom.Vec{X: 4, Y: 17}, geom.Vec{X: 17, Y: 4}, 0.7, Pink),
			stroke("teal_b", geom.Vec{X: 5.7, Y: 18.7}, geom.Vec{X: 18.7, Y: 5.7}, 0.7, Teal),
		},
		Crop: [4]int{6, 6, 18, 18},
	}
}

// GuardPixel is a background pixel strictly between the red and blue bars.
const GuardPixelX, GuardPixelY = 4, 7

// All returns the gallery scenes in display order.
func All() []cover.Scene {
	return []cover.Scene{ThinH(), Diagonal(), Cross(), NearDiag(), Segregate()}
}
