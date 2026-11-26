package main

// UnitsPerEm is the design-unit em size used by the procedural glyph.
const UnitsPerEm = 1000.0

// SampleGlyph returns a self-contained outline with one true counter and a
// narrow open notch between the legs. The sharp apex and counter apex make
// uncorrected normal offsets prone to long miters and crossings.
func SampleGlyph() Outline {
	outer := Contour{
		{X: 300, Y: 700},
		{X: 535, Y: 40},
		{X: 425, Y: 40},
		{X: 328, Y: 252},
		{X: 272, Y: 252},
		{X: 175, Y: 40},
		{X: 65, Y: 40},
	}
	counter := Contour{
		{X: 300, Y: 575},
		{X: 238, Y: 330},
		{X: 362, Y: 330},
	}
	return Outline{Outer: outer, Holes: []Contour{counter}}.Canonical()
}

// CornerTrap returns a tight hairpin stroke centerline used in the
// self-intersection diagnostic panel.
func CornerTrap() Contour {
	return Contour{
		{X: 40, Y: 260},
		{X: 180, Y: 260},
		{X: 250, Y: 70},
		{X: 320, Y: 260},
		{X: 460, Y: 260},
	}
}
