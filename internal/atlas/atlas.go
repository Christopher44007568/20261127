package atlas

import (
	"math"

	"glyphsdf/internal/field"
	"glyphsdf/internal/fontx"
	"glyphsdf/internal/geom"
)

type Glyph struct {
	Rune     rune
	Field    *field.Field
	Advance  float64
	Baseline float64
	Pad      float64
}

type Atlas struct {
	Font     *fontx.Font
	Em       float64
	Pad      float64
	Baseline float64
	Height   int
	cache    map[rune]*Glyph
}

func New(fontPath string, em, pad float64) (*Atlas, error) {
	f, err := fontx.LoadFile(fontPath)
	if err != nil {
		return nil, err
	}
	a := &Atlas{
		Font:     f,
		Em:       em,
		Pad:      pad,
		Baseline: pad + 0.96*em,
		Height:   int(math.Ceil(2*pad + 1.28*em)),
		cache:    map[rune]*Glyph{},
	}
	return a, nil
}

func (a *Atlas) Get(r rune) (*Glyph, error) {
	if g, ok := a.cache[r]; ok {
		return g, nil
	}
	outline, err := a.Font.GlyphOutline(r)
	if err != nil {
		return nil, err
	}
	upem := float64(a.Font.UnitsPerEm())
	scale := a.Em / upem
	toleranceFontUnits := 0.028 / scale
	fontShape := outline.Flatten(toleranceFontUnits)
	advance := float64(a.Font.AdvanceUnits(r)) * scale
	w := int(math.Ceil(advance + 2*a.Pad))
	if w < 8 {
		w = 8
	}
	h := a.Height
	texShape := geom.MapShape(fontShape, scale, -scale, geom.V{X: a.Pad, Y: a.Baseline})
	rng := math.Max(3.2, a.Pad*0.78)
	f := field.Generate(texShape, field.Options{
		W: w, H: h, Range: rng, CornerAngle: 0.43, ClampDistance: true,
	})
	field.CorrectSigns(f, 7, 5)
	g := &Glyph{Rune: r, Field: f, Advance: advance, Baseline: a.Baseline, Pad: a.Pad}
	a.cache[r] = g
	return g, nil
}

func (a *Atlas) LineWidth(text string, fontSize float64) float64 {
	scale := fontSize / a.Em
	width := 0.0
	for _, r := range text {
		g, err := a.Get(r)
		if err != nil {
			width += fontSize * 0.6
			continue
		}
		width += g.Advance * scale
	}
	return width
}
