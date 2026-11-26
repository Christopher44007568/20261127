package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// Scene is a logically-sized canvas backed by an S-times supersampled RGBA
// buffer. Aliased device-space drawing becomes anti-aliased on downsample.
type Scene struct {
	W, H int
	S    int
	img  *image.RGBA
}

func NewScene(w, h, ss int, bg color.RGBA) *Scene {
	if ss < 1 {
		ss = 1
	}
	dev := image.NewRGBA(image.Rect(0, 0, w*ss, h*ss))
	fillRGBA(dev, bg)
	return &Scene{W: w, H: h, S: ss, img: dev}
}

func fillRGBA(img *image.RGBA, c color.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = 255
	}
}

func (sc *Scene) dx(x float64) float64 { return x * float64(sc.S) }

func (sc *Scene) setDev(x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= sc.img.Rect.Dx() || y >= sc.img.Rect.Dy() {
		return
	}
	i := sc.img.PixOffset(x, y)
	if c.A == 255 {
		sc.img.Pix[i] = c.R
		sc.img.Pix[i+1] = c.G
		sc.img.Pix[i+2] = c.B
		sc.img.Pix[i+3] = 255
		return
	}
	a := float64(c.A) / 255
	ia := 1 - a
	sc.img.Pix[i] = uint8(float64(c.R)*a + float64(sc.img.Pix[i])*ia)
	sc.img.Pix[i+1] = uint8(float64(c.G)*a + float64(sc.img.Pix[i+1])*ia)
	sc.img.Pix[i+2] = uint8(float64(c.B)*a + float64(sc.img.Pix[i+2])*ia)
}

// FillRect fills a logical-coordinate rectangle.
func (sc *Scene) FillRect(x0, y0, x1, y1 float64, c color.RGBA) {
	X0 := int(math.Round(x0 * float64(sc.S)))
	Y0 := int(math.Round(y0 * float64(sc.S)))
	X1 := int(math.Round(x1 * float64(sc.S)))
	Y1 := int(math.Round(y1 * float64(sc.S)))
	w, h := sc.img.Rect.Dx(), sc.img.Rect.Dy()
	if X0 < 0 {
		X0 = 0
	}
	if Y0 < 0 {
		Y0 = 0
	}
	if X1 > w {
		X1 = w
	}
	if Y1 > h {
		Y1 = h
	}
	for y := Y0; y < Y1; y++ {
		for x := X0; x < X1; x++ {
			sc.setDev(x, y, c)
		}
	}
}

// FrameRect strokes a rectangle with the given logical line width.
func (sc *Scene) FrameRect(x0, y0, x1, y1, w float64, c color.RGBA) {
	sc.Line(x0, y0, x1, y0, w, c)
	sc.Line(x1, y0, x1, y1, w, c)
	sc.Line(x1, y1, x0, y1, w, c)
	sc.Line(x0, y1, x0, y0, w, c)
}

// Line draws a thick round-capped segment.
func (sc *Scene) Line(x0, y0, x1, y1, width float64, c color.RGBA) {
	sc.stampLine(Pt{x0, y0}, Pt{x1, y1}, width, c)
}

// Polyline draws a thick round-joined path.
func (sc *Scene) Polyline(pts []Pt, width float64, c color.RGBA) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		sc.stampLine(pts[i], pts[i+1], width, c)
	}
	rad := width * float64(sc.S) / 2
	for _, p := range pts {
		sc.devDisc(Pt{p.X * float64(sc.S), p.Y * float64(sc.S)}, rad, c)
	}
}

func (sc *Scene) stampLine(a, b Pt, width float64, c color.RGBA) {
	S := float64(sc.S)
	ad := Pt{a.X * S, a.Y * S}
	bd := Pt{b.X * S, b.Y * S}
	rad := width * S / 2
	d := ad.dist(bd)
	step := math.Max(rad*0.4, 0.75)
	n := int(math.Ceil(d / step))
	if n < 1 {
		n = 1
	}
	for k := 0; k <= n; k++ {
		t := float64(k) / float64(n)
		sc.devDisc(Pt{ad.X + (bd.X-ad.X)*t, ad.Y + (bd.Y-ad.Y)*t}, rad, c)
	}
}

// Disc fills a circle in logical coordinates.
func (sc *Scene) Disc(cx, cy, r float64, c color.RGBA) {
	S := float64(sc.S)
	sc.devDisc(Pt{cx * S, cy * S}, r*S, c)
}

func (sc *Scene) devDisc(c Pt, r float64, col color.RGBA) {
	w, h := sc.img.Rect.Dx(), sc.img.Rect.Dy()
	x0 := maxI(0, int(math.Floor(c.X-r-0.5)))
	x1 := minI(w-1, int(math.Ceil(c.X+r+0.5)))
	y0 := maxI(0, int(math.Floor(c.Y-r-0.5)))
	y1 := minI(h-1, int(math.Ceil(c.Y+r+0.5)))
	r2 := r * r
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float64(x) + 0.5 - c.X
			dy := float64(y) + 0.5 - c.Y
			if dx*dx+dy*dy <= r2 {
				sc.setDev(x, y, col)
			}
		}
	}
}

// FillOutline fills logical-coordinate contours.
func (sc *Scene) FillOutline(o Outline, c color.RGBA) {
	S := float64(sc.S)
	dev := make([]Contour, len(o.Contours))
	for i, ct := range o.Contours {
		dc := Contour{Closed: ct.Closed, Pts: make([]Pt, len(ct.Pts))}
		for j, p := range ct.Pts {
			dc.Pts[j] = Pt{p.X * S, p.Y * S}
		}
		dev[i] = dc
	}
	b := sc.img.Bounds()
	forEachSpan(b.Dx(), b.Dy(), dev, func(y, x0, x1 int) {
		for x := x0; x <= x1; x++ {
			sc.setDev(x, y, c)
		}
	})
}

// StrokeOutline draws the flattened contour path with a thick round stroke.
func (sc *Scene) StrokeOutline(o Outline, width float64, c color.RGBA) {
	for _, ct := range o.Contours {
		pts := ct.Pts
		sc.Polyline(pts, width, c)
		if ct.Closed && len(pts) > 2 {
			sc.Line(pts[len(pts)-1].X, pts[len(pts)-1].Y, pts[0].X, pts[0].Y, width, c)
		}
	}
}

// Render downsamples the supersampled buffer to a final RGBA image.
func (sc *Scene) Render() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, sc.W, sc.H))
	S := sc.S
	for y := 0; y < sc.H; y++ {
		for x := 0; x < sc.W; x++ {
			var r, g, b, a uint32
			for sy := 0; sy < S; sy++ {
				for sx := 0; sx < S; sx++ {
					i := sc.img.PixOffset(x*S+sx, y*S+sy)
					pr := uint32(sc.img.Pix[i])
					pg := uint32(sc.img.Pix[i+1])
					pb := uint32(sc.img.Pix[i+2])
					pa := uint32(sc.img.Pix[i+3])
					r += pr * pa
					g += pg * pa
					b += pb * pa
					a += pa
				}
			}
			n := uint32(S * S)
			o := out.PixOffset(x, y)
			if a == 0 {
				out.Pix[o+3] = 255
				continue
			}
			out.Pix[o] = uint8(r / a)
			out.Pix[o+1] = uint8(g / a)
			out.Pix[o+2] = uint8(b / a)
			out.Pix[o+3] = 255
			_ = n
		}
	}
	return out
}

// SavePNG renders and writes the scene to path.
func (sc *Scene) SavePNG(path string) error {
	img := sc.Render()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	return enc.Encode(f, img)
}

// SaveImagePNG writes any RGBA image as PNG.
func SaveImagePNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
