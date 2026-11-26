package main

import "fmt"

func (f *Font) parseCmap() error {
	t, _, err := f.tableBytes("cmap")
	if err != nil {
		return err
	}
	if len(t) < 4 {
		return fmt.Errorf("cmap too short")
	}
	numTables := int(be16(t, 2))
	type encRec struct {
		platform uint16
		encoding uint16
		offset   int
	}
	var recs []encRec
	for i := 0; i < numTables; i++ {
		o := 4 + i*8
		if o+8 > len(t) {
			break
		}
		recs = append(recs, encRec{be16(t, o), be16(t, o+2), int(be32(t, o+4))})
	}
	// Prefer a full repertoire (format 12) record, then Windows BMP, then Unicode.
	best := -1
	bestScore := -1
	for i, r := range recs {
		if r.offset >= len(t) {
			continue
		}
		format := be16(t, r.offset)
		score := 0
		if format == 12 {
			score = 30
		}
		if format == 4 {
			score = 20
		}
		if r.platform == 3 {
			score += 2
		}
		if r.platform == 0 {
			score += 1
		}
		if r.platform == 3 && r.encoding == 10 {
			score += 3
		}
		if score > bestScore {
			bestScore, best = score, i
		}
	}
	if best < 0 {
		return fmt.Errorf("cmap: no usable encoding record")
	}
	r := recs[best]
	sub := t[r.offset:]
	format := be16(sub, 0)
	m := map[rune]uint16{}
	switch format {
	case 4:
		if err := parseCmapFormat4(sub, m); err != nil {
			return err
		}
	case 12:
		if err := parseCmapFormat12(sub, m); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cmap: unsupported subtable format %d", format)
	}
	f.unicodeToGlyph = m
	return nil
}

func parseCmapFormat4(p []byte, m map[rune]uint16) error {
	if len(p) < 14 {
		return fmt.Errorf("cmap4 short")
	}
	segX2 := int(be16(p, 6))
	segCount := segX2 / 2
	endOff := 14
	startOff := endOff + segX2 + 2
	deltaOff := startOff + segX2
	rangeOff := deltaOff + segX2
	if rangeOff+segX2 > len(p) {
		return fmt.Errorf("cmap4 truncated arrays")
	}
	for s := 0; s < segCount; s++ {
		endCode := be16(p, endOff+s*2)
		startCode := be16(p, startOff+s*2)
		idDelta := uint16(i16(p, deltaOff+s*2))
		idRangeOffPos := rangeOff + s*2
		idRangeOff := be16(p, idRangeOffPos)
		for c := int(startCode); c <= int(endCode) && endCode != 0xFFFF; c++ {
			var gid uint16
			if idRangeOff == 0 {
				gid = uint16(c) + idDelta
			} else {
				glyphPos := idRangeOffPos + int(idRangeOff) + (c-int(startCode))*2
				if glyphPos+2 > len(p) {
					continue
				}
				g := be16(p, glyphPos)
				if g != 0 {
					gid = g + idDelta
				}
			}
			if gid != 0 {
				m[rune(c)] = gid
			}
		}
	}
	return nil
}

func parseCmapFormat12(p []byte, m map[rune]uint16) error {
	if len(p) < 16 {
		return fmt.Errorf("cmap12 short")
	}
	nGroups := int(be32(p, 12))
	if 16+nGroups*12 > len(p) {
		return fmt.Errorf("cmap12 truncated groups")
	}
	for i := 0; i < nGroups; i++ {
		o := 16 + i*12
		start := be32(p, o)
		end := be32(p, o+4)
		startGid := be32(p, o+8)
		// Guard against pathological ranges; BMP + SMP text is all we need.
		if end > 0x10FFFF || start > end {
			continue
		}
		for cp := start; cp <= end; cp++ {
			gid := startGid + (cp - start)
			if gid != 0 && gid < 0xFFFF {
				m[rune(cp)] = uint16(gid)
			}
		}
	}
	return nil
}
