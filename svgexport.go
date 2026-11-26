package main

import (
	"strconv"
	"strings"
)

// fnum formats a coordinate compactly with at most 2 decimals. The rounded
// value round-trips exactly through ParseSVG.
func fnum(v float64) string {
	r := float64(int64(v*100+sign(v)*0.5)) / 100
	return strconv.FormatFloat(r, 'f', -1, 64)
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func fnums(vs ...float64) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fnum(v)
	}
	return strings.Join(parts, " ")
}

// PathD serializes a path to an SVG path data string.
func PathD(p *Path) string {
	var b strings.Builder
	for _, c := range p.Cmds {
		switch c.Op {
		case 'M':
			b.WriteString("M" + fnum(c.P[0].X) + " " + fnum(c.P[0].Y))
		case 'L':
			b.WriteString("L" + fnum(c.P[0].X) + " " + fnum(c.P[0].Y))
		case 'Q':
			b.WriteString("Q" + fnum(c.P[0].X) + " " + fnum(c.P[0].Y) + " " +
				fnum(c.P[1].X) + " " + fnum(c.P[1].Y))
		case 'C':
			b.WriteString("C" + fnum(c.P[0].X) + " " + fnum(c.P[0].Y) + " " +
				fnum(c.P[1].X) + " " + fnum(c.P[1].Y) + " " +
				fnum(c.P[2].X) + " " + fnum(c.P[2].Y))
		case 'Z':
			b.WriteString("Z")
		}
		b.WriteByte(' ')
	}
	return strings.TrimSpace(b.String())
}

func paintValue(p *Paint) string {
	if p == nil {
		return "none"
	}
	if p.Grad != nil || p.Ref != "" && strings.HasPrefix(p.Ref, "grad-") {
		id := p.Ref
		if p.Grad != nil {
			id = p.Grad.ID
		}
		return "url(#" + id + ")"
	}
	return hexString(p.Solid)
}

// ExportSVG serializes the scene to editable SVG. Every scene object becomes a
// separate identified element; no raster image is embedded.
func ExportSVG(s *Scene) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!-- vectorlab: same scene -> editable SVG -> browser/self rasterization -->` + "\n")
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" `)
	b.WriteString(`width="` + strconv.Itoa(s.W) + `" height="` + strconv.Itoa(s.H) + `" `)
	b.WriteString(`viewBox="0 0 ` + strconv.Itoa(s.W) + ` ` + strconv.Itoa(s.H) + `">` + "\n")
	b.WriteString("  <defs>\n")
	for _, g := range s.Grads {
		writeGradient(&b, g)
	}
	for _, c := range s.Clips {
		b.WriteString(`    <clipPath id="` + c.ID + `" clipPathUnits="userSpaceOnUse">` + "\n")
		for _, p := range c.Paths {
			b.WriteString(`      <path d="` + PathD(p) + `"/>` + "\n")
		}
		b.WriteString("    </clipPath>\n")
	}
	b.WriteString("  </defs>\n")
	for _, o := range s.Objs {
		writeObj(&b, o, 0)
	}
	b.WriteString("</svg>\n")
	return b.String()
}

func writeGradient(b *strings.Builder, g *Gradient) {
	if g.Radial {
		b.WriteString(`    <radialGradient id="` + g.ID + `" gradientUnits="userSpaceOnUse" `)
		b.WriteString(`cx="` + fnum(g.CX) + `" cy="` + fnum(g.CY) + `" r="` + fnum(g.R) + `">` + "\n")
	} else {
		b.WriteString(`    <linearGradient id="` + g.ID + `" gradientUnits="userSpaceOnUse" `)
		b.WriteString(`x1="` + fnum(g.X1) + `" y1="` + fnum(g.Y1) + `" `)
		b.WriteString(`x2="` + fnum(g.X2) + `" y2="` + fnum(g.Y2) + `">` + "\n")
	}
	for _, st := range g.Stops {
		b.WriteString(`      <stop offset="` + fnum(st.Off) + `" stop-color="` + hexString(st.C) + `"`)
		if st.C.A != 255 {
			b.WriteString(` stop-opacity="` + fnum(float64(st.C.A)/255) + `"`)
		}
		b.WriteString(`/>` + "\n")
	}
	if g.Radial {
		b.WriteString("    </radialGradient>\n")
	} else {
		b.WriteString("    </linearGradient>\n")
	}
}

func writeObj(b *strings.Builder, o *Obj, depth int) {
	ind := strings.Repeat("  ", depth+1)
	if o.Tag == "g" || len(o.Children) > 0 {
		b.WriteString(ind + `<g id="` + o.ID + `"`)
		if o.ClipID != "" {
			b.WriteString(` clip-path="url(#` + o.ClipID + `)"`)
		}
		if o.Note != "" {
			b.WriteString(` data-semantic="` + xmlAttr(o.Note) + `"`)
		}
		b.WriteString(">\n")
		for _, c := range o.Children {
			writeObj(b, c, depth+1)
		}
		b.WriteString(ind + "</g>\n")
		return
	}
	for i, p := range o.Paths {
		id := o.ID
		if len(o.Paths) > 1 {
			id = o.ID + "-p" + strconv.Itoa(i+1)
		}
		b.WriteString(ind + `<path id="` + id + `"`)
		if o.Note != "" && i == 0 {
			b.WriteString(` data-semantic="` + xmlAttr(o.Note) + `"`)
		}
		b.WriteString("\n" + ind + `      d="` + PathD(p) + `"`)
		b.WriteString("\n" + ind + `      fill="` + paintValue(o.Fill) + `"`)
		if o.Fill != nil && o.Fill.Solid.A != 255 && o.Fill.Grad == nil {
			b.WriteString(` fill-opacity="` + fnum(float64(o.Fill.Solid.A)/255) + `"`)
		}
		b.WriteString(` stroke="` + paintValue(o.Stroke) + `"`)
		if o.Stroke != nil && o.StrokeW > 0 {
			b.WriteString(` stroke-width="` + fnum(o.StrokeW) + `"`)
			b.WriteString(` stroke-linecap="round" stroke-linejoin="round"`)
		}
		if o.ClipID != "" {
			b.WriteString(` clip-path="url(#` + o.ClipID + `)"`)
		}
		b.WriteString("/>\n")
	}
}

func xmlAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
