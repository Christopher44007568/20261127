package studio

// FloatImage stores premultiplied linear-light RGBA values. Keeping the buffer
// premultiplied is what prevents dark backgrounds from bleeding into soft
// circle and glyph edges during resampling and blending.
type FloatImage struct {
	W, H int
	Pix  []float32
}

func NewImage(w, h int) *FloatImage {
	if w < 0 || h < 0 {
		panic("studio: negative image size")
	}
	return &FloatImage{W: w, H: h, Pix: make([]float32, w*h*4)}
}

func (im *FloatImage) Bounds() (w, h int) {
	return im.W, im.H
}

func (im *FloatImage) offset(x, y int) int {
	return (y*im.W + x) * 4
}

// At returns the stored premultiplied linear value.
func (im *FloatImage) At(x, y int) RGBA {
	i := im.offset(x, y)
	return RGBA{
		R: im.Pix[i],
		G: im.Pix[i+1],
		B: im.Pix[i+2],
		A: im.Pix[i+3],
	}
}

// Set writes a premultiplied linear value verbatim.
func (im *FloatImage) Set(x, y int, p RGBA) {
	i := im.offset(x, y)
	im.Pix[i] = clamp01f(p.R)
	im.Pix[i+1] = clamp01f(p.G)
	im.Pix[i+2] = clamp01f(p.B)
	im.Pix[i+3] = clamp01f(p.A)
}

// SetStraight converts a straight linear colour to premultiplied form.
func (im *FloatImage) SetStraight(x, y int, c RGBA) {
	im.Set(x, y, Premultiply(c))
}

func (im *FloatImage) Clear(c RGBA) {
	p := Premultiply(c)
	for i := 0; i < len(im.Pix); i += 4 {
		im.Pix[i] = p.R
		im.Pix[i+1] = p.G
		im.Pix[i+2] = p.B
		im.Pix[i+3] = p.A
	}
}

func (im *FloatImage) Clone() *FloatImage {
	out := NewImage(im.W, im.H)
	copy(out.Pix, im.Pix)
	return out
}

// OverPixel performs premultiplied source-over for one pixel.
func (im *FloatImage) OverPixel(x, y int, src RGBA) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	i := im.offset(x, y)
	sa := clamp01f(src.A)
	inv := 1 - sa
	im.Pix[i] = clamp01f(src.R + im.Pix[i]*inv)
	im.Pix[i+1] = clamp01f(src.G + im.Pix[i+1]*inv)
	im.Pix[i+2] = clamp01f(src.B + im.Pix[i+2]*inv)
	im.Pix[i+3] = clamp01f(sa + im.Pix[i+3]*inv)
}

// Blend composites src into im using mode. All arithmetic is premultiplied,
// so partial coverage never pulls colour from a fully transparent pixel.
func (im *FloatImage) Blend(src *FloatImage, mode BlendMode) {
	if im.W != src.W || im.H != src.H {
		panic("studio: blend size mismatch")
	}
	for i := 0; i < len(im.Pix); i += 4 {
		d := RGBA{R: im.Pix[i], G: im.Pix[i+1], B: im.Pix[i+2], A: im.Pix[i+3]}
		s := RGBA{R: src.Pix[i], G: src.Pix[i+1], B: src.Pix[i+2], A: src.Pix[i+3]}
		im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = blendPixel(d, s, mode)
	}
}

func blendPixel(dst, src RGBA, mode BlendMode) (float32, float32, float32, float32) {
	sa := clamp01f(src.A)
	da := clamp01f(dst.A)
	if sa <= 0 {
		return dst.R, dst.G, dst.B, dst.A
	}
	sc := Unpremultiply(src)
	dc := Unpremultiply(dst)
	r := blendChannel(sc.R, dc.R, mode)
	g := blendChannel(sc.G, dc.G, mode)
	b := blendChannel(sc.B, dc.B, mode)

	// W3C compositing formula, expressed in premultiplied linear light:
	//   Co = (1-ab)*as*Cs + as*ab*B(Cs,Cb) + (1-as)*ab*Cb
	//   ao = as + ab - as*ab
	outA := sa + da - sa*da
	outR := (1-da)*src.R + sa*da*r + (1-sa)*dst.R
	outG := (1-da)*src.G + sa*da*g + (1-sa)*dst.G
	outB := (1-da)*src.B + sa*da*b + (1-sa)*dst.B
	return clamp01f(outR), clamp01f(outG), clamp01f(outB), clamp01f(outA)
}

func blendChannel(s, d float32, mode BlendMode) float32 {
	switch mode {
	case Multiply:
		return s * d
	case Screen:
		return s + d - s*d
	default:
		return s
	}
}

// ApplyMask multiplies both colour and coverage, equivalent to destination-in
// for a single-channel alpha mask. Hue is unchanged by the operation.
func (im *FloatImage) ApplyMask(mask []float32) {
	if len(mask) != im.W*im.H {
		panic("studio: mask size mismatch")
	}
	for p, m := range mask {
		m = clamp01f(m)
		i := p * 4
		im.Pix[i] *= m
		im.Pix[i+1] *= m
		im.Pix[i+2] *= m
		im.Pix[i+3] *= m
	}
}

// CompositeOver draws src over a copy of dst and returns the result.
func CompositeOver(dst, src *FloatImage) *FloatImage {
	out := dst.Clone()
	out.Blend(src, Normal)
	return out
}

// BlendMode names the supported separable blend functions.
type BlendMode int

const (
	Normal BlendMode = iota
	Multiply
	Screen
)

func (m BlendMode) String() string {
	switch m {
	case Multiply:
		return "multiply"
	case Screen:
		return "screen"
	default:
		return "normal"
	}
}
