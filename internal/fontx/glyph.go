package fontx

import (
	"encoding/binary"
	"fmt"
	"math"

	"glyphsdf/internal/geom"
)

type OutlinePoint struct {
	X, Y    float64
	OnCurve bool
}

type Outline struct {
	Contours [][]OutlinePoint
}

func (f *Font) GlyphOutline(r rune) (Outline, error) {
	gid := f.GlyphID(r)
	if gid == 0 {
		return Outline{}, fmt.Errorf("missing glyph for %q", r)
	}
	return f.readOutline(gid, 0)
}

func (f *Font) readOutline(gid, depth int) (Outline, error) {
	if depth > 8 || gid < 0 || gid >= f.numGlyphs {
		return Outline{}, fmt.Errorf("bad composite glyph")
	}
	start := int(f.glyf) + f.loca[gid]
	end := int(f.glyf) + f.loca[gid+1]
	if end <= start {
		return Outline{}, nil
	}
	d := f.data[start:end]
	if len(d) < 10 {
		return Outline{}, fmt.Errorf("truncated glyph %d", gid)
	}
	nContours := int16(binary.BigEndian.Uint16(d[0:2]))
	if nContours > 0 {
		return parseSimple(d, int(nContours))
	}
	if nContours != -1 {
		return Outline{}, nil
	}
	return f.parseComposite(d, depth)
}

func parseSimple(d []byte, n int) (Outline, error) {
	if len(d) < 10+2*n {
		return Outline{}, fmt.Errorf("truncated contour endpoints")
	}
	ends := make([]int, n)
	total := 0
	for i := 0; i < n; i++ {
		ends[i] = int(binary.BigEndian.Uint16(d[10+2*i : 12+2*i]))
		total = ends[i] + 1
	}
	p := 10 + 2*n
	insLen := int(binary.BigEndian.Uint16(d[p : p+2]))
	p += 2 + insLen

	flags := make([]byte, 0, total)
	for len(flags) < total {
		if p >= len(d) {
			return Outline{}, fmt.Errorf("truncated flags")
		}
		flag := d[p]
		p++
		flags = append(flags, flag)
		if flag&0x08 != 0 {
			if p >= len(d) {
				return Outline{}, fmt.Errorf("truncated flag repeat")
			}
			repeat := int(d[p])
			p++
			for i := 0; i < repeat && len(flags) < total; i++ {
				flags = append(flags, flag)
			}
		}
	}
	if len(flags) > total {
		flags = flags[:total]
	}

	xs, err := decodeCoords(d, &p, flags, 0x02, 0x10)
	if err != nil {
		return Outline{}, err
	}
	ys, err := decodeCoords(d, &p, flags, 0x04, 0x20)
	if err != nil {
		return Outline{}, err
	}
	points := make([]OutlinePoint, total)
	for i := range points {
		points[i] = OutlinePoint{X: xs[i], Y: ys[i], OnCurve: flags[i]&0x01 != 0}
	}
	out := Outline{}
	first := 0
	for _, end := range ends {
		if end >= total {
			return Outline{}, fmt.Errorf("bad contour endpoint")
		}
		out.Contours = append(out.Contours, points[first:end+1])
		first = end + 1
	}
	return out, nil
}

func decodeCoords(d []byte, p *int, flags []byte, shortBit, sameBit byte) ([]float64, error) {
	coords := make([]float64, len(flags))
	value := 0
	for i, flag := range flags {
		switch {
		case flag&shortBit != 0:
			if *p >= len(d) {
				return nil, fmt.Errorf("truncated short coordinate")
			}
			delta := int(d[*p])
			*p++
			if flag&sameBit == 0 {
				delta = -delta
			}
			value += delta
		case flag&sameBit != 0:
			// Same as the previous coordinate: zero delta.
		default:
			if *p+2 > len(d) {
				return nil, fmt.Errorf("truncated coordinate")
			}
			value += int(int16(binary.BigEndian.Uint16(d[*p : *p+2])))
			*p += 2
		}
		coords[i] = float64(value)
	}
	return coords, nil
}

func (f *Font) parseComposite(d []byte, depth int) (Outline, error) {
	out := Outline{}
	p := 10
	for {
		if p+4 > len(d) {
			return out, fmt.Errorf("truncated composite header")
		}
		flags := binary.BigEndian.Uint16(d[p : p+2])
		gid := int(binary.BigEndian.Uint16(d[p+2 : p+4]))
		p += 4
		var dx, dy float64
		if flags&0x0001 != 0 {
			dx = float64(int16(binary.BigEndian.Uint16(d[p : p+2])))
			dy = float64(int16(binary.BigEndian.Uint16(d[p+2 : p+4])))
			p += 4
		} else {
			dx = float64(int(int8(d[p])))
			dy = float64(int(int8(d[p+1])))
			p += 2
		}
		xx, xy, yx, yy := 1.0, 0.0, 0.0, 1.0
		if flags&0x0008 != 0 {
			s := f2dot14(binary.BigEndian.Uint16(d[p : p+2]))
			p += 2
			xx, yy = s, s
		} else if flags&0x0040 != 0 {
			xx = f2dot14(binary.BigEndian.Uint16(d[p : p+2]))
			yy = f2dot14(binary.BigEndian.Uint16(d[p+2 : p+4]))
			p += 4
		} else if flags&0x0080 != 0 {
			xx = f2dot14(binary.BigEndian.Uint16(d[p : p+2]))
			yx = f2dot14(binary.BigEndian.Uint16(d[p+2 : p+4]))
			xy = f2dot14(binary.BigEndian.Uint16(d[p+4 : p+6]))
			yy = f2dot14(binary.BigEndian.Uint16(d[p+6 : p+8]))
			p += 8
		}
		sub, err := f.readOutline(gid, depth+1)
		if err != nil {
			return out, err
		}
		for _, c := range sub.Contours {
			nc := make([]OutlinePoint, len(c))
			for i, pt := range c {
				nc[i] = OutlinePoint{
					X:       xx*pt.X + yx*pt.Y + dx,
					Y:       xy*pt.X + yy*pt.Y + dy,
					OnCurve: pt.OnCurve,
				}
			}
			out.Contours = append(out.Contours, nc)
		}
		if flags&0x0020 == 0 {
			break
		}
	}
	return out, nil
}

func f2dot14(v uint16) float64 {
	return float64(int16(v)) / 16384.0
}

// Flatten converts TrueType on/off-curve points into tolerance-flattened
// closed polylines. Coordinate transforms should be applied first.
func (o Outline) Flatten(tolerance float64) geom.Shape {
	shape := geom.Shape{}
	for _, pts := range o.Contours {
		if len(pts) < 2 {
			continue
		}
		flat := flattenContour(pts, tolerance)
		if len(flat) >= 3 {
			shape.Contours = append(shape.Contours, geom.Contour{Points: flat})
		}
	}
	return shape
}

func flattenContour(pts []OutlinePoint, tol float64) []geom.V {
	n := len(pts)
	// Rotate so an on-curve point is first. If all points are off-curve,
	// insert implied midpoints and use the midpoint between last and first.
	start := 0
	for start < n && !pts[start].OnCurve {
		start++
	}
	if start == n {
		mid := geom.V{
			X: (pts[n-1].X + pts[0].X) * 0.5,
			Y: (pts[n-1].Y + pts[0].Y) * 0.5,
		}
		out := []geom.V{mid}
		cur := mid
		for i := 0; i < n; i++ {
			p := geom.V{X: pts[i].X, Y: pts[i].Y}
			next := geom.V{X: pts[(i+1)%n].X, Y: pts[(i+1)%n].Y}
			m := p.Add(next.Sub(p).Mul(0.5))
			geom.FlattenQuad(cur, p, m, tol, &out)
			cur = m
		}
		return out
	}

	ordered := make([]OutlinePoint, n)
	for i := 0; i < n; i++ {
		ordered[i] = pts[(start+i)%n]
	}
	out := []geom.V{{X: ordered[0].X, Y: ordered[0].Y}}
	cur := out[0]
	i := 1
	for i < n {
		if ordered[i].OnCurve {
			end := geom.V{X: ordered[i].X, Y: ordered[i].Y}
			out = append(out, end)
			cur = end
			i++
			continue
		}
		control := geom.V{X: ordered[i].X, Y: ordered[i].Y}
		i++
		var end geom.V
		if i < n {
			end = geom.V{X: ordered[i].X, Y: ordered[i].Y}
			if !ordered[i].OnCurve {
				end = geom.V{
					X: (control.X + end.X) * 0.5,
					Y: (control.Y + end.Y) * 0.5,
				}
			} else {
				i++
			}
		} else {
			// Last point is off-curve; the cyclic first point is the end.
			end = out[0]
		}
		geom.FlattenQuad(cur, control, end, tol, &out)
		cur = end
	}
	// Close to the first point if the last command ended at it.
	if len(out) > 1 && out[len(out)-1].Dist(out[0]) < tol*0.5 {
		out = out[:len(out)-1]
	}
	return out
}

func GlyphShapeFromSystem(r rune) (geom.Shape, string, error) {
	paths := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf",
	}
	for _, p := range paths {
		f, err := LoadFile(p)
		if err != nil {
			continue
		}
		o, err := f.GlyphOutline(r)
		if err != nil {
			continue
		}
		scale := 1.0
		if f.unitsPerEm > 0 {
			scale = float64(f.unitsPerEm) / 1000.0
		}
		shape := o.Flatten(math.Max(0.75, scale))
		if !shape.Empty() {
			return shape, p, nil
		}
	}
	return geom.Shape{}, "", fmt.Errorf("no system glyph")
}
