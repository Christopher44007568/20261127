package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Font is a minimal TrueType outline reader (no external dependencies).
// It only understands the tables needed for outline extraction and metrics.
type Font struct {
	Path string
	data []byte

	tables map[string]tableRecord

	UnitsPerEm    int
	TypoAscender  int // hhea ascender, font units (y up)
	TypoDescender int // hhea descender, negative or zero
	TypoLineGap   int

	numGlyphs      int
	numHMetrics    int
	locaFormat     int16
	glyfOff        int
	locaOff        int
	hmtxOff        int
	hmtxLen        int
	advanceWidths  []uint16
	leftSideBear   []int16
	glyphLSBExtra  []int16
	unicodeToGlyph map[rune]uint16
}

type tableRecord struct {
	offset int
	length int
}

// RawPoint is one point as stored in a glyf simple/contour component.
type RawPoint struct {
	X, Y  int
	OnCur bool
}

// Glyph is one decoded glyph: metrics plus raw contours in font units (y up).
type Glyph struct {
	Index    uint16
	Advance  int
	LSB      int
	Bounds   Rect
	Contours [][]RawPoint
}

// Rect is an axis aligned bounds in an arbitrary coordinate space.
type Rect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

func (r Rect) Empty() bool {
	return r.MaxX <= r.MinX || r.MaxY <= r.MinY
}

func (r Rect) Width() float64  { return r.MaxX - r.MinX }
func (r Rect) Height() float64 { return r.MaxY - r.MinY }

func (r Rect) ContainsX(x float64) bool { return x >= r.MinX && x <= r.MaxX }

func be16(b []byte, o int) uint16 { return binary.BigEndian.Uint16(b[o : o+2]) }
func be32(b []byte, o int) uint32 { return binary.BigEndian.Uint32(b[o : o+4]) }
func i16(b []byte, o int) int16   { return int16(be16(b, o)) }

// LoadFont reads and parses the table directory and metric/cmap tables.
func LoadFont(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("%s: not a font file", path)
	}
	f := &Font{Path: path, data: data, tables: map[string]tableRecord{}}
	sfnt := be32(data, 0)
	if sfnt != 0x00010000 && sfnt != 0x74727565 && sfnt != 0x4F54544F {
		return nil, fmt.Errorf("%s: unsupported sfnt version %08x", path, sfnt)
	}
	n := int(be16(data, 4))
	for i := 0; i < n; i++ {
		off := 12 + i*16
		if off+16 > len(data) {
			return nil, fmt.Errorf("%s: truncated table record", path)
		}
		tag := string(data[off : off+4])
		rec := tableRecord{offset: int(be32(data, off+8)), length: int(be32(data, off+12))}
		f.tables[tag] = rec
	}
	if err := f.parseHead(); err != nil {
		return nil, err
	}
	if err := f.parseHhea(); err != nil {
		return nil, err
	}
	if err := f.parseMaxp(); err != nil {
		return nil, err
	}
	if err := f.parseHmtx(); err != nil {
		return nil, err
	}
	if err := f.parseCmap(); err != nil {
		return nil, err
	}
	if glyf, ok := f.tables["glyf"]; ok {
		f.glyfOff = glyf.offset
	}
	if loca, ok := f.tables["loca"]; ok {
		f.locaOff = loca.offset
	}
	return f, nil
}

func (f *Font) tableBytes(tag string) ([]byte, tableRecord, error) {
	rec, ok := f.tables[tag]
	if !ok {
		return nil, rec, fmt.Errorf("%s: missing %s table", f.Path, tag)
	}
	return f.data[rec.offset : rec.offset+rec.length], rec, nil
}

func (f *Font) parseHead() error {
	t, _, err := f.tableBytes("head")
	if err != nil {
		return err
	}
	if len(t) < 54 {
		return fmt.Errorf("head table too short")
	}
	f.UnitsPerEm = int(be16(t, 18))
	f.locaFormat = i16(t, 50)
	return nil
}

func (f *Font) parseHhea() error {
	t, _, err := f.tableBytes("hhea")
	if err != nil {
		return err
	}
	if len(t) < 36 {
		return fmt.Errorf("hhea table too short")
	}
	f.TypoAscender = int(i16(t, 4))
	f.TypoDescender = int(i16(t, 6))
	f.TypoLineGap = int(i16(t, 8))
	f.numHMetrics = int(be16(t, 34))
	return nil
}

func (f *Font) parseMaxp() error {
	t, _, err := f.tableBytes("maxp")
	if err != nil {
		return err
	}
	if len(t) < 6 {
		return fmt.Errorf("maxp table too short")
	}
	f.numGlyphs = int(be16(t, 4))
	return nil
}

func (f *Font) parseHmtx() error {
	rec, ok := f.tables["hmtx"]
	if !ok {
		return fmt.Errorf("missing hmtx table")
	}
	f.hmtxOff, f.hmtxLen = rec.offset, rec.length
	d := f.data[rec.offset : rec.offset+rec.length]
	f.advanceWidths = make([]uint16, f.numHMetrics)
	f.leftSideBear = make([]int16, f.numHMetrics)
	for i := 0; i < f.numHMetrics && i*4+4 <= len(d); i++ {
		f.advanceWidths[i] = be16(d, i*4)
		f.leftSideBear[i] = i16(d, i*4+2)
	}
	extra := f.numGlyphs - f.numHMetrics
	if extra > 0 {
		f.glyphLSBExtra = make([]int16, extra)
		start := f.numHMetrics * 4
		for i := 0; i < extra && start+i*2+2 <= len(d); i++ {
			f.glyphLSBExtra[i] = i16(d, start+i*2)
		}
	}
	return nil
}

// GlyphIndex maps a rune through cmap; returns 0 (.notdef) when absent.
func (f *Font) GlyphIndex(r rune) uint16 {
	if g, ok := f.unicodeToGlyph[r]; ok {
		return g
	}
	return 0
}

func (f *Font) HasRune(r rune) bool {
	_, ok := f.unicodeToGlyph[r]
	return ok
}

// Advance returns the horizontal advance in font units.
func (f *Font) Advance(gid uint16) int {
	if int(gid) < f.numHMetrics {
		return int(f.advanceWidths[gid])
	}
	if f.numHMetrics > 0 {
		return int(f.advanceWidths[f.numHMetrics-1])
	}
	return f.UnitsPerEm / 2
}

func (f *Font) lsb(gid uint16) int {
	if int(gid) < f.numHMetrics {
		return int(f.leftSideBear[gid])
	}
	idx := int(gid) - f.numHMetrics
	if idx >= 0 && idx < len(f.glyphLSBExtra) {
		return int(f.glyphLSBExtra[idx])
	}
	return 0
}
