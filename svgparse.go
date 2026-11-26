package main

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"
)

// SVGAudit summarizes the reloaded structure. The key invariant is
// HasRasterImage==false with many separate editable vector elements.
type SVGAudit struct {
	Width              int
	Height             int
	PathElements       int
	ShapeElements      int
	GroupElements      int
	ImageElements      int
	GradientElements   int
	ClipPathElements   int
	IdentifiedElements int
	IDs                []string
	ImageHrefs         []string
}

// ParseSVG reloads the exported dialect back into a renderable Scene and also
// returns a structural audit. It never collapses elements into a bitmap.
func ParseSVG(text string) (*Scene, *SVGAudit, error) {
	dec := xml.NewDecoder(strings.NewReader(text))
	audit := &SVGAudit{}
	sc := &Scene{}

	type frame struct{ obj *Obj }
	var stack []*Obj

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			attrs := attrMap(t.Attr)
			switch name {
			case "svg":
				sc.W = int(parseFloatDef(attrs["width"], 960))
				sc.H = int(parseFloatDef(attrs["height"], 600))
			case "linearGradient", "radialGradient":
				g := parseGradient(name == "radialGradient", attrs, dec, audit)
				sc.Grads = append(sc.Grads, g)
				audit.GradientElements++
			case "clipPath":
				c := &Clip{ID: attrs["id"]}
				if err := skipInto(dec, t, func(st xml.StartElement) error {
					if st.Name.Local == "path" {
						if d := attrMap(st.Attr)["d"]; d != "" {
							c.Paths = append(c.Paths, ParsePathD(d))
						}
					}
					return nil
				}); err != nil {
					return nil, nil, err
				}
				sc.Clips = append(sc.Clips, c)
				audit.ClipPathElements++
			case "image":
				audit.ImageElements++
				href := attrs["href"]
				if href == "" {
					href = attrs["xlink:href"]
				}
				if href == "" {
					for k, v := range attrs {
						if strings.HasSuffix(k, ":href") {
							href = v
						}
					}
				}
				audit.ImageHrefs = append(audit.ImageHrefs, truncate(href, 48))
			case "g":
				o := &Obj{ID: attrs["id"], Tag: "g", ClipID: clipRef(attrs), Note: attrs["data-semantic"]}
				if len(stack) == 0 {
					sc.Objs = append(sc.Objs, o)
				} else {
					parent := stack[len(stack)-1]
					parent.Children = append(parent.Children, o)
				}
				stack = append(stack, o)
				audit.GroupElements++
				if o.ID != "" {
					audit.IDs = append(audit.IDs, o.ID)
					audit.IdentifiedElements++
				}
			case "path", "rect", "circle", "ellipse", "polygon", "polyline":
				o := parseShape(name, attrs)
				if len(stack) == 0 {
					sc.Objs = append(sc.Objs, o)
				} else {
					parent := stack[len(stack)-1]
					parent.Children = append(parent.Children, o)
				}
				if name == "path" {
					audit.PathElements++
				} else {
					audit.ShapeElements++
				}
				if o.ID != "" {
					audit.IDs = append(audit.IDs, o.ID)
					audit.IdentifiedElements++
				}
			}
		case xml.EndElement:
			if t.Name.Local == "g" && len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	resolvePaints(sc)
	sc.BG = parseHex("#ffffff")
	sc.indexObjects()
	audit.Width, audit.Height = sc.W, sc.H
	return sc, audit, nil
}

func parseShape(name string, attrs map[string]string) *Obj {
	o := &Obj{ID: attrs["id"], Tag: name, ClipID: clipRef(attrs), Note: attrs["data-semantic"]}
	o.Fill = parsePaint(attrs["fill"], attrs["fill-opacity"])
	o.Stroke = parsePaint(attrs["stroke"], attrs["stroke-opacity"])
	o.StrokeW = parseFloatDef(attrs["stroke-width"], 0)
	switch name {
	case "path":
		o.Paths = []*Path{ParsePathD(attrs["d"])}
	case "rect":
		x, y := pf(attrs["x"]), pf(attrs["y"])
		w, h := pf(attrs["width"]), pf(attrs["height"])
		rx := pf(attrs["rx"])
		if rx > 0 {
			o.Paths = []*Path{RoundRectPath(x, y, w, h, rx)}
		} else {
			o.Paths = []*Path{RectPath(x, y, w, h)}
		}
	case "circle":
		o.Paths = []*Path{EllipsePath(pf(attrs["cx"]), pf(attrs["cy"]), pf(attrs["r"]), pf(attrs["r"]))}
	case "ellipse":
		o.Paths = []*Path{EllipsePath(pf(attrs["cx"]), pf(attrs["cy"]), pf(attrs["rx"]), pf(attrs["ry"]))}
	case "polygon", "polyline":
		o.Paths = []*Path{parsePoints(attrs["points"], name == "polygon")}
	}
	return o
}

func parsePoints(s string, close bool) *Path {
	nums := numberRe.FindAllString(s, -1)
	p := &Path{}
	for i := 0; i+1 < len(nums); i += 2 {
		x, _ := strconv.ParseFloat(nums[i], 64)
		y, _ := strconv.ParseFloat(nums[i+1], 64)
		if i == 0 {
			p.move(x, y)
		} else {
			p.line(x, y)
		}
	}
	if close {
		p.close()
	}
	return p
}

func parseGradient(radial bool, attrs map[string]string, dec *xml.Decoder, audit *SVGAudit) *Gradient {
	g := &Gradient{ID: attrs["id"], Radial: radial}
	if radial {
		g.CX, g.CY, g.R = pf(attrs["cx"]), pf(attrs["cy"]), pf(attrs["r"])
	} else {
		g.X1, g.Y1 = pf(attrs["x1"]), pf(attrs["y1"])
		g.X2, g.Y2 = pf(attrs["x2"]), pf(attrs["y2"])
	}
	_ = skipInto(dec, xml.StartElement{Name: xml.Name{Local: "gradient"}}, func(st xml.StartElement) error {
		if st.Name.Local == "stop" {
			a := attrMap(st.Attr)
			c := parseHex(firstNonEmpty(a["stop-color"], a["style"]))
			if op := a["stop-opacity"]; op != "" {
				c.A = uint8(255 * parseFloatDef(op, 1))
			}
			g.Stops = append(g.Stops, Stop{Off: parseFloatDef(a["offset"], 0), C: c})
		}
		return nil
	})
	return g
}

func parsePaint(v, opacity string) *Paint {
	v = strings.TrimSpace(v)
	if v == "" || v == "none" {
		return nil
	}
	p := &Paint{}
	if strings.HasPrefix(v, "url(") {
		id := strings.TrimPrefix(v, "url(#")
		id = strings.TrimSuffix(id, ")")
		id = strings.Trim(id, `"' `)
		p.Ref = id
		return p
	}
	if strings.HasPrefix(v, "#") {
		p.Solid = parseHex(v)
	} else if strings.HasPrefix(v, "rgb") {
		p.Solid = parseRGBFunc(v)
	} else {
		p.Solid = parseHex("#000000")
	}
	if opacity != "" {
		p.Solid.A = uint8(255 * parseFloatDef(opacity, 1))
	}
	return p
}

func parseRGBFunc(v string) color.RGBA {
	nums := numberRe.FindAllString(v, -1)
	if len(nums) >= 3 {
		r, _ := strconv.Atoi(nums[0])
		g, _ := strconv.Atoi(nums[1])
		b, _ := strconv.Atoi(nums[2])
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}
	return color.RGBA{0, 0, 0, 255}
}

func resolvePaints(sc *Scene) {
	var walk func(os []*Obj)
	walk = func(os []*Obj) {
		for _, o := range os {
			for _, p := range []*Paint{o.Fill, o.Stroke} {
				if p != nil && p.Ref != "" && p.Grad == nil {
					p.Grad = sc.gradByID(p.Ref)
				}
			}
			walk(o.Children)
		}
	}
	walk(sc.Objs)
}

func clipRef(attrs map[string]string) string {
	v := attrs["clip-path"]
	if i := strings.Index(v, "#"); i >= 0 {
		v = v[i+1:]
	}
	v = strings.TrimSuffix(v, ")")
	return strings.TrimSpace(v)
}

func attrMap(as []xml.Attr) map[string]string {
	m := map[string]string{}
	for _, a := range as {
		m[a.Name.Local] = a.Value
	}
	return m
}

func pf(s string) float64 { return parseFloatDef(s, 0) }

func parseFloatDef(s string, def float64) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "px"))
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// skipInto consumes the children of the current element, invoking fn for each
// child start element, until its matching end tag.
func skipInto(dec *xml.Decoder, parent xml.StartElement, fn func(xml.StartElement) error) error {
	depth := 1
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if err := fn(t); err != nil {
				return err
			}
			depth++
		case xml.EndElement:
			depth--
			if depth == 0 {
				if t.Name.Local != parent.Name.Local && parent.Name.Local != "gradient" {
					return fmt.Errorf("unexpected close %s", t.Name.Local)
				}
				return nil
			}
		}
	}
}
