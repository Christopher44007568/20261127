package main

import (
	"image"
	"image/color"
	"math"
)

// EffectParams groups strengths for the three anti-repetition layers.
type EffectParams struct {
	Seed uint64

	// Per-brick deterministic tint (local color variation).
	BrickHue   float64
	BrickSat   float64
	BrickLight float64

	// Slow color drift over whole wall (distant-view de-perioding).
	MacroLight float64
	MacroHue   float64

	// Amplitude in pixels of the low-frequency vector offset field.
	// It only warps color/dye coordinates, never brick UVs or masks.
	OffsetAmp float64

	MottleLight float64
	DyeAlpha    float64
	Grain       float64
}

func DefaultParams() EffectParams {
	return EffectParams{
		Seed:        20261127,
		BrickHue:    0.013,
		BrickSat:    0.060,
		BrickLight:  0.075,
		MacroLight:  0.085,
		MacroHue:    0.014,
		OffsetAmp:   22,
		MottleLight: 0.050,
		DyeAlpha:    1.0,
		Grain:       0.016,
	}
}

type pigment struct {
	r, g, b      float64
	scale        float64 // world-space feature size in pixels (inverse freq)
	lo, hi       float64 // concentration ramp thresholds
	seedSalt     int64
	alpha        float64
	absorbBrick  float64
	absorbMortar float64
}

func pigments() []pigment {
	return []pigment{
		{r: 0.14, g: 0.125, b: 0.11, scale: 430, lo: 0.50, hi: 0.74, seedSalt: 11, alpha: 0.55, absorbBrick: 1.0, absorbMortar: 0.55}, // damp soot
		{r: 0.52, g: 0.25, b: 0.10, scale: 300, lo: 0.55, hi: 0.80, seedSalt: 23, alpha: 0.34, absorbBrick: 0.9, absorbMortar: 0.45},  // rust / ochre runoff
		{r: 0.80, g: 0.76, b: 0.62, scale: 230, lo: 0.58, hi: 0.84, seedSalt: 37, alpha: 0.26, absorbBrick: 0.55, absorbMortar: 0.8},  // lime bloom
		{r: 0.20, g: 0.27, b: 0.15, scale: 360, lo: 0.60, hi: 0.85, seedSalt: 51, alpha: 0.30, absorbBrick: 0.8, absorbMortar: 0.7},   // moss tint
	}
}

// ProcessWall applies the anti-repetition treatment to a naive tiling.
//
// Structural lock: the source tile is always sampled at the unshifted
// modulo coordinate (x%TileW, y%TileH). No rotation, translation or warp
// ever touches brick UVs, so mortar joints, tile-border seams and wall
// corners remain exactly where the tiling put them.
func ProcessWall(tile *image.RGBA, w, h int, p EffectParams) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	dyeList := pigments()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			base := tile.RGBAAt(x%TileW, y%TileH)
			c := rgba8ToRGB(base.R, base.G, base.B)
			info := BrickAt(x, y)

			// --- low-frequency offset field -------------------------
			// Two decorrelated very-smooth noises form a vector field.
			// It offsets color-field coordinates only.
			nx := FBM(float64(x)/250, float64(y)/250, int64(p.Seed)+1, 3)
			ny := FBM(float64(x)/250+31.7, float64(y)/250-17.3, int64(p.Seed)+2, 3)
			wx := float64(x) + (nx-0.5)*2*p.OffsetAmp
			wy := float64(y) + (ny-0.5)*2*p.OffsetAmp*0.7

			macro := FBM(float64(x)/340, float64(y)/340, int64(p.Seed)+3, 3)
			macro2 := FBM(float64(x)/210+90, float64(y)/210, int64(p.Seed)+4, 3)
			mottle := FBM(wx/70, wy/70, int64(p.Seed)+5, 3)

			// --- local color variation ------------------------------
			if info.Mortar {
				// Mortar gets no per-brick jumps: joints stay uniform
				// and complete; only continuous macro drift touches it.
				c = shiftHSL(c,
					(macro-0.5)*p.MacroHue*0.5,
					(macro2-0.5)*0.04,
					(macro-0.5)*p.MacroLight+(mottle-0.5)*p.MottleLight*0.4)
			} else {
				bseed := splitmix64(p.Seed ^
					uint64(uint32(info.Col))*0x9E3779B1 ^
					uint64(uint32(info.Row))*0x85EBCA77)

				// Re-phase hue/saturation per world brick while keeping
				// the baked luminance field (clay mottle, bevel, AO and
				// grain), so every cloned instance becomes a distinct
				// but physically plausible brick.
				_, _, bakedL := rgbToHSL(c)
				u := frand(bseed, 10)
				var hue, sat, lShift float64
				switch {
				case u < 0.10: // pale buff / yellow stock
					hue = 0.075 + frand(bseed, 11)*0.035
					sat = 0.30 + frand(bseed, 12)*0.20
					lShift = 0.05 + frand(bseed, 13)*0.05
				case u < 0.22: // dark clinker / purple-brown
					hue = math.Mod(0.97+frand(bseed, 11)*0.06, 1)
					sat = 0.16 + frand(bseed, 12)*0.22
					lShift = -0.10 + frand(bseed, 13)*0.04
				default: // red / orange / brown terracotta
					hue = 0.004 + frand(bseed, 11)*0.075
					sat = 0.40 + frand(bseed, 12)*0.24
					lShift = (frand(bseed, 13) - 0.5) * 0.10
				}
				light := bakedL*0.88 + lShift +
					(frand(bseed, 14)-0.5)*p.BrickLight +
					(macro-0.5)*p.MacroLight*1.6 +
					(mottle-0.5)*p.MottleLight*1.6 +
					(hashFloat(int64(x), int64(y), int64(p.Seed)+9)-0.5)*p.Grain*2
				c = hslToRGB(hue+(macro-0.5)*p.MacroHue,
					clamp01(sat+(macro2-0.5)*p.BrickSat),
					clamp01(light))
			}

			// --- boundary-continuous dye blending -------------------
			// Concentrations are world-space FBM fields sampled through
			// the offset vector field, hence continuous across every
			// 256px tile boundary (no per-tile dye state at all).
			c = blendDyes(c, dyeList, wx, wy, info, p)

			r, g, b, a := rgbToRGBA8(c)
			out.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	return out
}

func blendDyes(c RGB, list []pigment, wx, wy float64, info BrickInfo, p EffectParams) RGB {
	var wSum, rSum, gSum, bSum, aSum float64
	absorb := 1.0
	if info.Mortar {
		absorb = 0.6
	} else {
		bseed := splitmix64(p.Seed ^
			uint64(uint32(info.Col))*0x27D4EB2F ^
			uint64(uint32(info.Row))*0x165667B1)
		absorb = 0.7 + frand(bseed, 40)*0.55
	}

	for _, pg := range list {
		n := FBM(wx/pg.scale, wy/pg.scale, int64(p.Seed)+pg.seedSalt, 4)
		conc := smoothStep01((n - pg.lo) / (pg.hi - pg.lo))
		if conc <= 0 {
			continue
		}
		alpha := conc * pg.alpha * p.DyeAlpha
		if info.Mortar {
			alpha *= pg.absorbMortar
		} else {
			alpha *= pg.absorbBrick * absorb
		}
		// Weighted pigment mixing where patches overlap.
		w := alpha
		wSum += w
		rSum += pg.r * w
		gSum += pg.g * w
		bSum += pg.b * w
		aSum += alpha
	}
	if wSum <= 1e-6 {
		return c
	}
	dyed := RGB{rSum / wSum, gSum / wSum, bSum / wSum}
	alpha := math.Min(0.72, aSum)
	// Stains darken as well as tint, so use a soft multiply component.
	mixed := mixRGB(c, dyed, alpha*0.82)
	dark := mulRGB(c, 1-alpha*0.35)
	return mixRGB(mixed, dark, 0.35)
}
