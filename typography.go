package main

import (
	"fmt"
	"image/color"
	"os"
)

// FontSet resolves Han and Latin runs to bundled system TrueType fonts.
type FontSet struct {
	Latin     *Font
	Han       *Font
	Mono      *Font
	gcache    map[fontRune]*Glyph
	boldCache map[rune]BoldReport
}

type fontRune struct {
	f *Font
	r rune
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// LoadFontSet discovers the required TTF files without installing anything.
func LoadFontSet() (*FontSet, error) {
	latinPath := firstExisting(
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
	)
	hanPath := firstExisting(
		"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	)
	monoPath := firstExisting(
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
	)
	if latinPath == "" {
		return nil, fmt.Errorf("no Latin TrueType font found under /usr/share/fonts")
	}
	lf, err := LoadFont(latinPath)
	if err != nil {
		return nil, err
	}
	fs := &FontSet{Latin: lf, gcache: map[fontRune]*Glyph{}}
	if hanPath != "" {
		if hf, err := LoadFont(hanPath); err == nil {
			fs.Han = hf
		}
	}
	if monoPath != "" {
		if mf, err := LoadFont(monoPath); err == nil {
			fs.Mono = mf
		}
	}
	return fs, nil
}

// FontFor picks the Han fallback when the Latin font lacks the rune.
func (fs *FontSet) FontFor(r rune) *Font {
	if fs.Latin.HasRune(r) {
		return fs.Latin
	}
	if fs.Han != nil && fs.Han.HasRune(r) {
		return fs.Han
	}
	return fs.Latin
}

func (fs *FontSet) Glyph(r rune) (*Font, *Glyph) {
	f := fs.FontFor(r)
	key := fontRune{f, r}
	if g, ok := fs.gcache[key]; ok {
		return f, g
	}
	g, err := f.LoadGlyph(r)
	if err != nil {
		g = &Glyph{Index: 0, Advance: f.UnitsPerEm / 2}
	}
	fs.gcache[key] = g
	return f, g
}

// OutlineFor builds a flattened outline at the requested pixel size.
func (fs *FontSet) OutlineFor(r rune, ppem, tol float64) (*Font, *Glyph, Outline) {
	f, g := fs.Glyph(r)
	if tol <= 0 {
		tol = 0.3
	}
	return f, g, GlyphOutline(g, f.UnitsPerEm, ppem, tol)
}

// TextStyle configures DrawText / Measure.
type TextStyle struct {
	Ppem         float64
	TrackingEm   float64
	Color        color.RGBA
	Bold         bool
	BoldTargetEm float64 // target embolden radius in em; 0 => default
}

func (fs *FontSet) trackingPx(st TextStyle) float64 {
	return st.TrackingEm * st.Ppem
}

// RuneAdvance returns pen advance in logical px (without tracking).
func (fs *FontSet) RuneAdvance(r rune, ppem float64) float64 {
	f, g := fs.Glyph(r)
	return float64(g.Advance) * ppem / float64(f.UnitsPerEm)
}

// Measure returns the typographic advance width of s including tracking.
func (fs *FontSet) Measure(s string, st TextStyle) float64 {
	track := fs.trackingPx(st)
	w := 0.0
	n := 0
	for _, r := range s {
		w += fs.RuneAdvance(r, st.Ppem) + track
		n++
	}
	if n > 0 {
		w -= track // no tracking after the final rune
	}
	return w
}

// DrawRune paints one rune with its pen origin at (x, baseline).
// Returns the advance width (without tracking).
func (sc *Scene) DrawRune(fs *FontSet, r rune, x, baseline float64, st TextStyle) float64 {
	f, g := fs.Glyph(r)
	rad := 0.0
	if st.Bold {
		rad = fs.BoldRadiusEm(r) * st.Ppem
	}
	sc.DrawRuneRadius(fs, r, x, baseline, st, rad)
	return float64(g.Advance) * st.Ppem / float64(f.UnitsPerEm)
}

// DrawRuneRadius paints one rune with an explicit embolden radius (px).
func (sc *Scene) DrawRuneRadius(fs *FontSet, r rune, x, baseline float64, st TextStyle, radPx float64) {
	_, _, o := fs.OutlineFor(r, st.Ppem, 0.25)
	translated := o.Translate(x, baseline)
	sc.FillOutline(translated, st.Color)
	if radPx > 0 {
		sc.StrokeOutline(translated, 2*radPx, st.Color)
	}
}

// DrawText draws s starting at pen x with the given baseline, returning end x.
func (sc *Scene) DrawText(s string, x, baseline float64, fs *FontSet, st TextStyle) float64 {
	track := fs.trackingPx(st)
	for _, r := range s {
		adv := sc.DrawRune(fs, r, x, baseline, st)
		x += adv + track
	}
	return x - track
}

// RuneInk returns tight ink bounds in pen-local coordinates (baseline y=0).
func (fs *FontSet) RuneInk(r rune, ppem float64, bold bool) Rect {
	f, g := fs.Glyph(r)
	s := ppem / float64(f.UnitsPerEm)
	b := Rect{
		MinX: g.Bounds.MinX * s,
		MinY: -g.Bounds.MaxY * s,
		MaxX: g.Bounds.MaxX * s,
		MaxY: -g.Bounds.MinY * s,
	}
	if g.Bounds.Empty() || (b.MinX == 0 && b.MaxX == 0 && b.MinY == 0 && b.MaxY == 0) {
		// Whitespace / empty glyph: fall back to font metrics line.
		b = Rect{MinX: 0, MinY: float64(f.TypoAscender) * s, MaxX: fs.RuneAdvance(r, ppem), MaxY: float64(f.TypoDescender) * s}
		// That fallback has MinY positive; fix sign: ascender above baseline => MinY negative.
		b.MinY = -float64(f.TypoAscender) * s
		b.MaxY = -float64(f.TypoDescender) * s
	}
	if bold {
		rad := fs.BoldRadiusEm(r) * ppem
		b.MinX -= rad
		b.MaxX += rad
		b.MinY -= rad
		b.MaxY += rad
	}
	return b
}
