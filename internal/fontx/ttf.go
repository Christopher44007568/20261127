package fontx

import (
	"encoding/binary"
	"fmt"
	"os"
)

type Font struct {
	data         []byte
	tables       map[string]table
	numGlyphs    int
	loca         []int
	glyf         uint32
	hmtx         table
	numHMetrics  int
	unitsPerEm   int
	cmapSubtable int
}

type table struct {
	offset uint32
	length uint32
}

func LoadFile(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("font file too short")
	}
	f := &Font{data: data, tables: map[string]table{}}
	num := int(binary.BigEndian.Uint16(data[4:6]))
	for i := 0; i < num; i++ {
		off := 12 + i*16
		if off+16 > len(data) {
			return nil, fmt.Errorf("truncated table directory")
		}
		tag := string(data[off : off+4])
		f.tables[tag] = table{
			offset: binary.BigEndian.Uint32(data[off+8 : off+12]),
			length: binary.BigEndian.Uint32(data[off+12 : off+16]),
		}
	}
	head, ok := f.tables["head"]
	if !ok {
		return nil, fmt.Errorf("missing head table")
	}
	maxp, ok := f.tables["maxp"]
	if !ok {
		return nil, fmt.Errorf("missing maxp table")
	}
	if _, ok := f.tables["loca"]; !ok {
		return nil, fmt.Errorf("missing loca table")
	}
	if glyf, ok := f.tables["glyf"]; ok {
		f.glyf = glyf.offset
	}
	if hhea, ok := f.tables["hhea"]; ok && int(hhea.offset)+36 <= len(data) {
		f.numHMetrics = int(binary.BigEndian.Uint16(data[hhea.offset+34 : hhea.offset+36]))
	}
	f.hmtx = f.tables["hmtx"]
	f.unitsPerEm = int(binary.BigEndian.Uint16(data[head.offset+18 : head.offset+20]))
	f.numGlyphs = int(binary.BigEndian.Uint16(data[maxp.offset+4 : maxp.offset+6]))
	if err := f.readLoca(int16(binary.BigEndian.Uint16(data[head.offset+50 : head.offset+52]))); err != nil {
		return nil, err
	}
	if err := f.findCmap(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Font) UnitsPerEm() int { return f.unitsPerEm }

func (f *Font) AdvanceUnits(r rune) int {
	if f.numHMetrics <= 0 {
		return f.unitsPerEm / 2
	}
	gid := f.GlyphID(r)
	metric := gid
	if metric >= f.numHMetrics {
		metric = f.numHMetrics - 1
	}
	off := int(f.hmtx.offset) + metric*4
	if off+2 > len(f.data) {
		return f.unitsPerEm / 2
	}
	return int(binary.BigEndian.Uint16(f.data[off : off+2]))
}

func (f *Font) readLoca(format int16) error {
	t := f.tables["loca"]
	f.loca = make([]int, f.numGlyphs+1)
	if format == 0 {
		for i := 0; i <= f.numGlyphs; i++ {
			off := int(t.offset) + i*2
			f.loca[i] = int(binary.BigEndian.Uint16(f.data[off:off+2])) * 2
		}
	} else {
		for i := 0; i <= f.numGlyphs; i++ {
			off := int(t.offset) + i*4
			f.loca[i] = int(binary.BigEndian.Uint32(f.data[off : off+4]))
		}
	}
	return nil
}

func (f *Font) findCmap() error {
	t, ok := f.tables["cmap"]
	if !ok {
		return fmt.Errorf("missing cmap")
	}
	base := int(t.offset)
	num := int(binary.BigEndian.Uint16(f.data[base+2 : base+4]))
	best := -1
	bestScore := -1
	for i := 0; i < num; i++ {
		rec := base + 4 + i*8
		platform := binary.BigEndian.Uint16(f.data[rec : rec+2])
		encoding := binary.BigEndian.Uint16(f.data[rec+2 : rec+4])
		subOff := int(binary.BigEndian.Uint32(f.data[rec+4 : rec+8]))
		format := binary.BigEndian.Uint16(f.data[base+subOff : base+subOff+2])
		score := 0
		if platform == 3 && encoding == 1 {
			score = 100
		}
		if platform == 0 {
			score = 90
		}
		if format == 12 {
			score += 20
		}
		if format == 4 {
			score += 10
		}
		if score > bestScore {
			bestScore = score
			best = base + subOff
		}
	}
	if best < 0 {
		return fmt.Errorf("no usable cmap subtable")
	}
	f.cmapSubtable = best
	return nil
}

func (f *Font) GlyphID(r rune) int {
	sub := f.cmapSubtable
	format := binary.BigEndian.Uint16(f.data[sub : sub+2])
	if format == 4 && r <= 0xffff {
		return f.glyphIDFormat4(sub, uint16(r))
	}
	if format == 12 {
		return f.glyphIDFormat12(sub, uint32(r))
	}
	return 0
}

func (f *Font) glyphIDFormat4(sub int, r uint16) int {
	segCount := int(binary.BigEndian.Uint16(f.data[sub+6:sub+8])) / 2
	endBase := sub + 14
	startBase := endBase + 2*segCount + 2
	deltaBase := startBase + 2*segCount
	rangeBase := deltaBase + 2*segCount
	for i := 0; i < segCount; i++ {
		end := binary.BigEndian.Uint16(f.data[endBase+2*i : endBase+2*i+2])
		if r > end {
			continue
		}
		start := binary.BigEndian.Uint16(f.data[startBase+2*i : startBase+2*i+2])
		if r < start {
			return 0
		}
		delta := int16(binary.BigEndian.Uint16(f.data[deltaBase+2*i : deltaBase+2*i+2]))
		rangeOff := binary.BigEndian.Uint16(f.data[rangeBase+2*i : rangeBase+2*i+2])
		if rangeOff == 0 {
			return int((int(r) + int(delta)) & 0xffff)
		}
		addr := rangeBase + 2*i + int(rangeOff) + 2*int(r-start)
		gid := int(binary.BigEndian.Uint16(f.data[addr : addr+2]))
		if gid != 0 {
			gid = (gid + int(delta)) & 0xffff
		}
		return gid
	}
	return 0
}

func (f *Font) glyphIDFormat12(sub int, r uint32) int {
	n := int(binary.BigEndian.Uint32(f.data[sub+12 : sub+16]))
	for i := 0; i < n; i++ {
		rec := sub + 16 + i*12
		start := binary.BigEndian.Uint32(f.data[rec : rec+4])
		end := binary.BigEndian.Uint32(f.data[rec+4 : rec+8])
		startGID := binary.BigEndian.Uint32(f.data[rec+8 : rec+12])
		if r >= start && r <= end {
			return int(startGID + r - start)
		}
	}
	return 0
}
