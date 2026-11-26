package main

import "fmt"

func (f *Font) glyphOffset(gid uint16) (int, int, error) {
	if f.locaOff == 0 || f.glyfOff == 0 {
		return 0, 0, fmt.Errorf("loca/glyf tables unavailable")
	}
	var o0, o1 int
	if f.locaFormat == 0 {
		o0 = int(be16(f.data, f.locaOff+int(gid)*2)) * 2
		o1 = int(be16(f.data, f.locaOff+int(gid+1)*2)) * 2
	} else {
		o0 = int(be32(f.data, f.locaOff+int(gid)*4))
		o1 = int(be32(f.data, f.locaOff+int(gid+1)*4))
	}
	return f.glyfOff + o0, o1 - o0, nil
}

// LoadGlyph decodes one glyph, recursively expanding composite glyphs.
func (f *Font) LoadGlyph(r rune) (*Glyph, error) {
	return f.LoadGlyphIndex(f.GlyphIndex(r))
}

func (f *Font) LoadGlyphIndex(gid uint16) (*Glyph, error) {
	g := &Glyph{Index: gid, Advance: f.Advance(gid), LSB: f.lsb(gid)}
	off, length, err := f.glyphOffset(gid)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		// Space-like glyph: metrics only.
		g.Bounds = Rect{}
		return g, nil
	}
	if off+length > len(f.data) || length < 10 {
		return nil, fmt.Errorf("glyph %d out of range", gid)
	}
	d := f.data[off : off+length]
	numContours := int(i16(d, 0))
	g.Bounds = Rect{
		MinX: float64(i16(d, 2)),
		MinY: float64(i16(d, 4)),
		MaxX: float64(i16(d, 6)),
		MaxY: float64(i16(d, 8)),
	}
	switch {
	case numContours > 0:
		cs, err := parseSimpleContours(d, numContours)
		if err != nil {
			return nil, err
		}
		g.Contours = cs
	case numContours == -1:
		cs, err := f.parseComposite(d, map[uint16]bool{gid: true})
		if err != nil {
			return nil, err
		}
		g.Contours = cs
	default:
		// numContours == 0: empty glyph.
	}
	return g, nil
}

type flagReader struct {
	d     []byte
	pos   int
	flags []byte
	rep   int
	curFl byte
}

func (r *flagReader) readFlag() byte {
	if r.rep > 0 {
		r.rep--
		return r.curFl
	}
	fl := r.d[r.pos]
	r.pos++
	r.curFl = fl
	if fl&0x08 != 0 {
		r.rep = int(r.d[r.pos])
		r.pos++
	}
	return fl
}

func parseSimpleContours(d []byte, numContours int) ([][]RawPoint, error) {
	endPts := make([]int, numContours)
	for i := 0; i < numContours; i++ {
		endPts[i] = int(be16(d, 10+i*2))
	}
	p := 10 + numContours*2
	insLen := int(be16(d, p))
	p += 2 + insLen
	numPoints := endPts[numContours-1] + 1

	fr := &flagReader{d: d, pos: p}
	allFlags := make([]byte, numPoints)
	for i := 0; i < numPoints; i++ {
		allFlags[i] = fr.readFlag()
	}
	p = fr.pos

	xs := make([]int, numPoints)
	x := 0
	for i := 0; i < numPoints; i++ {
		fl := allFlags[i]
		switch {
		case fl&0x02 != 0:
			v := int(d[p])
			p++
			if fl&0x10 == 0 {
				v = -v
			}
			x += v
		case fl&0x10 == 0:
			v := int(i16(d, p))
			p += 2
			x += v
		}
		xs[i] = x
	}
	ys := make([]int, numPoints)
	y := 0
	for i := 0; i < numPoints; i++ {
		fl := allFlags[i]
		switch {
		case fl&0x04 != 0:
			v := int(d[p])
			p++
			if fl&0x20 == 0 {
				v = -v
			}
			y += v
		case fl&0x20 == 0:
			v := int(i16(d, p))
			p += 2
			y += v
		}
		ys[i] = y
	}

	contours := make([][]RawPoint, 0, numContours)
	start := 0
	for _, end := range endPts {
		c := make([]RawPoint, 0, end-start+1)
		for i := start; i <= end; i++ {
			c = append(c, RawPoint{X: xs[i], Y: ys[i], OnCur: allFlags[i]&0x01 != 0})
		}
		contours = append(contours, c)
		start = end + 1
	}
	return contours, nil
}

// affine maps component local coordinates into parent glyph coordinates.
type affine struct{ a, b, c, d, tx, ty float64 }

func (m affine) apply(x, y float64) (float64, float64) {
	return m.a*x + m.c*y + m.tx, m.b*x + m.d*y + m.ty
}

func f2dot14(v int16) float64 { return float64(v) / 16384.0 }

func (f *Font) parseComposite(d []byte, seen map[uint16]bool) ([][]RawPoint, error) {
	var out [][]RawPoint
	p := 10
	for {
		flags := be16(d, p)
		gid := be16(d, p+2)
		p += 4
		words := flags&0x0001 != 0
		xyVals := flags&0x0002 != 0
		var a1, a2 int
		if words {
			a1 = int(i16(d, p))
			a2 = int(i16(d, p+2))
			p += 4
		} else {
			a1 = int(int8(d[p]))
			a2 = int(int8(d[p+1]))
			p += 2
		}
		m := affine{a: 1, b: 0, c: 0, d: 1}
		if xyVals {
			m.tx, m.ty = float64(a1), float64(a2)
		}
		switch {
		case flags&0x0008 != 0: // WE_HAVE_A_SCALE
			s := f2dot14(i16(d, p))
			p += 2
			m.a, m.d = s, s
		case flags&0x0040 != 0: // WE_HAVE_AN_X_AND_Y_SCALE
			m.a = f2dot14(i16(d, p))
			m.d = f2dot14(i16(d, p+2))
			p += 4
		case flags&0x0080 != 0: // WE_HAVE_A_TWO_BY_TWO
			m.a = f2dot14(i16(d, p))
			m.b = f2dot14(i16(d, p+2))
			m.c = f2dot14(i16(d, p+4))
			m.d = f2dot14(i16(d, p+6))
			p += 8
		}
		if !seen[gid] {
			seen[gid] = true
			off, length, err := f.glyphOffset(gid)
			if err == nil && length > 0 && off+length <= len(f.data) {
				cd := f.data[off : off+length]
				nc := int(i16(cd, 0))
				var cs [][]RawPoint
				if nc > 0 {
					cs, err = parseSimpleContours(cd, nc)
				} else if nc == -1 {
					cs, err = f.parseComposite(cd, seen)
				}
				if err == nil {
					for _, c := range cs {
						rc := make([]RawPoint, len(c))
						for i, pt := range c {
							x, y := m.apply(float64(pt.X), float64(pt.Y))
							rc[i] = RawPoint{X: int(x + 0.5), Y: int(y + 0.5), OnCur: pt.OnCur}
						}
						out = append(out, rc)
					}
				}
			}
		}
		if flags&0x0020 == 0 { // MORE_COMPONENTS
			break
		}
	}
	return out, nil
}
