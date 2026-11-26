package main

import (
	"unicode"
)

// LayoutStyle parameterizes outline-driven line layout.
type LayoutStyle struct {
	Ppem         float64
	TrackingEm   float64
	LineHeightEm float64 // minimum baseline-to-baseline step, in em
	MinInkGapEm  float64 // minimum gap between adjacent lines' ink extents
	MaxWidth     float64
	Bold         bool
}

// PlacedGlyph is one rune positioned in layout-box coordinates.
type PlacedGlyph struct {
	Char         string  `json:"char"`
	Code         rune    `json:"code"`
	FontTag      string  `json:"font"`
	PenX         float64 `json:"penX"`
	BaselineY    float64 `json:"baselineY"`
	Advance      float64 `json:"advance"`
	InkMinX      float64 `json:"inkMinX"`
	InkMinY      float64 `json:"inkMinY"`
	InkMaxX      float64 `json:"inkMaxX"`
	InkMaxY      float64 `json:"inkMaxY"`
	BoldRadiusEm float64 `json:"boldRadiusEm,omitempty"`
}

// LaidLine is one visual line after wrapping.
type LaidLine struct {
	Index     int           `json:"index"`
	Text      string        `json:"text"`
	Glyphs    []PlacedGlyph `json:"glyphs"`
	BaselineY float64       `json:"baselineY"`
	Ascent    float64       `json:"inkAscent"`
	Descent   float64       `json:"inkDescent"`
	Width     float64       `json:"width"`
}

// Layout is the complete geometric result.
type Layout struct {
	Lines      []LaidLine  `json:"lines"`
	Width      float64     `json:"width"`
	Height     float64     `json:"height"`
	MinPairGap float64     `json:"minPairGap"`
	PairGaps   []float64   `json:"pairGaps"`
	Overlap    bool        `json:"overlap"`
	Style      LayoutStyle `json:"style"`
}

type layAtom struct {
	rs         []rune
	glue       bool
	breakAfter bool
}

func isCJKClass(r rune) bool {
	if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
		return true
	}
	switch {
	case r >= 0x3000 && r <= 0x303F: // CJK punctuation
		return true
	case r >= 0xFF00 && r <= 0xFFEF: // fullwidth forms
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	}
	return false
}

// cjkTrailingPunct includes closing punctuation that should stay attached.
func cjkTrailingPunct(r rune) bool {
	switch r {
	case '，', '。', '、', '；', '：', '！', '？', '”', '’', '）', '】', '》', '…', '—':
		return true
	}
	return false
}

func tokenizeParagraph(s string) []layAtom {
	runes := []rune(s)
	var atoms []layAtom
	i := 0
	for i < len(runes) {
		r := runes[i]
		switch {
		case r == ' ' || r == '\t':
			j := i
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
				j++
			}
			atoms = append(atoms, layAtom{rs: runes[i:j], glue: true})
			i = j
		case isCJKClass(r):
			j := i + 1
			for j < len(runes) && cjkTrailingPunct(runes[j]) {
				j++
			}
			atoms = append(atoms, layAtom{rs: runes[i:j], breakAfter: true})
			i = j
		default:
			j := i
			for j < len(runes) {
				c := runes[j]
				if c == ' ' || c == '\t' || isCJKClass(c) {
					break
				}
				j++
			}
			atoms = append(atoms, layAtom{rs: runes[i:j]})
			i = j
		}
	}
	return atoms
}

func (fs *FontSet) runesWidth(rs []rune, st LayoutStyle) float64 {
	track := st.TrackingEm * st.Ppem
	w := 0.0
	for _, r := range rs {
		w += fs.RuneAdvance(r, st.Ppem) + track
	}
	if n := len(rs); n > 0 {
		w -= track
	}
	return w
}

// Layout performs wrapping and outline-metric vertical rhythm.
func (fs *FontSet) Layout(text string, st LayoutStyle) *Layout {
	if st.LineHeightEm <= 0 {
		st.LineHeightEm = 1.5
	}
	if st.MaxWidth <= 0 {
		st.MaxWidth = 1e9
	}
	track := st.TrackingEm * st.Ppem
	minGap := st.MinInkGapEm * st.Ppem

	var visualLines [][]rune
	for _, para := range splitLines(text) {
		atoms := tokenizeParagraph(para)
		var cur []rune
		curWidth := 0.0
		flush := func() {
			if len(cur) > 0 {
				visualLines = append(visualLines, cur)
				cur = nil
				curWidth = 0
			}
		}
		for ai := 0; ai < len(atoms); ai++ {
			a := atoms[ai]
			if a.glue && len(cur) == 0 {
				continue // drop leading whitespace
			}
			aw := fs.runesWidth(a.rs, st)
			addW := aw
			if len(cur) > 0 {
				addW += track // tracking across the atom boundary
			}
			if st.MaxWidth < 1e8 && curWidth+addW > st.MaxWidth && len(cur) > 0 {
				// Wrap: prefer the last break-after atom; our atoms already
				// encode legal CJK break points, so break before this atom.
				flush()
				if a.glue {
					continue // whitespace hangs at line end / dropped at start
				}
				cur = append([]rune(nil), a.rs...)
				curWidth = aw
				// Hard split if a single unbreakable word is too wide.
				for curWidth > st.MaxWidth && len(cur) > 1 {
					visualLines = append(visualLines, cur[:len(cur)-1])
					cur = []rune{cur[len(cur)-1]}
					curWidth = fs.runesWidth(cur, st)
				}
				continue
			}
			if len(cur) > 0 {
				curWidth += track
			}
			cur = append(cur, a.rs...)
			curWidth += aw
		}
		flush()
	}
	if len(visualLines) == 0 {
		visualLines = [][]rune{{}}
	}

	lay := &Layout{Style: st, Width: st.MaxWidth}
	for li, rs := range visualLines {
		line := LaidLine{Index: li, Text: string(rs)}
		pen := 0.0
		for _, r := range rs {
			f, g := fs.Glyph(r)
			adv := float64(g.Advance) * st.Ppem / float64(f.UnitsPerEm)
			ink := fs.RuneInk(r, st.Ppem, st.Bold)
			pr := PlacedGlyph{
				Char: string(r), Code: r,
				FontTag: fontTag(f),
				PenX:    pen, Advance: adv,
				InkMinX: pen + ink.MinX, InkMinY: ink.MinY,
				InkMaxX: pen + ink.MaxX, InkMaxY: ink.MaxY,
			}
			if st.Bold {
				pr.BoldRadiusEm = fs.BoldRadiusEm(r)
			}
			line.Glyphs = append(line.Glyphs, pr)
			if !isSpaceRune(r) {
				if -ink.MinY > line.Ascent {
					line.Ascent = -ink.MinY
				}
				if ink.MaxY > line.Descent {
					line.Descent = ink.MaxY
				}
			}
			pen += adv + track
		}
		if len(rs) > 0 {
			line.Width = pen - track
		}
		lay.Lines = append(lay.Lines, line)
	}

	// Vertical stacking driven by real ink extents.
	y := 0.0
	prevBaseline := 0.0
	for i := range lay.Lines {
		ln := &lay.Lines[i]
		if i == 0 {
			ln.BaselineY = ln.Ascent
			y = ln.BaselineY + ln.Descent
		} else {
			prev := &lay.Lines[i-1]
			need := prev.Descent + minGap + ln.Ascent
			step := maxF(st.LineHeightEm*st.Ppem, need)
			ln.BaselineY = prevBaseline + step
			gap := step - prev.Descent - ln.Ascent
			lay.PairGaps = append(lay.PairGaps, gap)
			if gap < minGap-1e-6 {
				lay.Overlap = true
			}
			y = ln.BaselineY + ln.Descent
		}
		for gi := range ln.Glyphs {
			gp := &ln.Glyphs[gi]
			gp.BaselineY = ln.BaselineY
			// Ink Y was baseline-relative; make it layout-box absolute.
			gp.InkMinY += ln.BaselineY
			gp.InkMaxY += ln.BaselineY
		}
		prevBaseline = ln.BaselineY
	}
	lay.Height = y
	lay.MinPairGap = 1e18
	for _, g := range lay.PairGaps {
		if g < lay.MinPairGap {
			lay.MinPairGap = g
		}
	}
	if len(lay.PairGaps) == 0 {
		lay.MinPairGap = 0
	}
	return lay
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	out = append(out, cur)
	return out
}

func isSpaceRune(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func fontTag(f *Font) string {
	if f == nil {
		return "?"
	}
	if containsStr(f.Path, "Droid") {
		return "DroidSansFallback"
	}
	if containsStr(f.Path, "DejaVuSansMono") {
		return "DejaVuSansMono"
	}
	if containsStr(f.Path, "DejaVuSans") {
		return "DejaVuSans"
	}
	return f.Path
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (indexStr(s, sub) >= 0)
}

func indexStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
