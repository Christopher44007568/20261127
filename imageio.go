package flakepaint

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

type HDRBuffer struct {
	W, H int
	Pix  []Vec3
}

func NewHDRBuffer(w, h int) *HDRBuffer {
	return &HDRBuffer{W: w, H: h, Pix: make([]Vec3, w*h)}
}

func (b *HDRBuffer) Set(x, y int, c Vec3) {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return
	}
	b.Pix[y*b.W+x] = c
}

type SampleShader func(u, v, aspect, pixelAngle float64) Vec3

// RenderRows evaluates a deterministic regular sample grid in parallel. The
// fixed sample offsets are important: output is stable across frames and runs.
func RenderRows(w, h, samples int, shader SampleShader) *HDRBuffer {
	buf := NewHDRBuffer(w, h)
	aspect := float64(w) / float64(h)
	pixelAngle := 1.0 / float64(h)
	if samples < 1 {
		samples = 1
	}
	offsets := sampleOffsets(samples)
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan int, workers*2)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range jobs {
				for x := 0; x < w; x++ {
					var sum Vec3
					for _, o := range offsets {
						u := (float64(x) + o[0]) / float64(w)
						v := (float64(y) + o[1]) / float64(h)
						sum = sum.Add(shader(u, v, aspect, pixelAngle))
					}
					buf.Set(x, y, sum.Mul(1/float64(len(offsets))))
				}
			}
		}()
	}
	for y := 0; y < h; y++ {
		jobs <- y
	}
	close(jobs)
	wg.Wait()
	return buf
}

func sampleOffsets(n int) [][2]float64 {
	switch n {
	case 1:
		return [][2]float64{{0.5, 0.5}}
	case 2:
		return [][2]float64{{0.25, 0.25}, {0.75, 0.75}}
	case 4:
		return [][2]float64{{0.25, 0.25}, {0.75, 0.25}, {0.25, 0.75}, {0.75, 0.75}}
	case 8:
		return [][2]float64{
			{0.25, 0.25}, {0.75, 0.25}, {0.25, 0.75}, {0.75, 0.75},
			{0.125, 0.625}, {0.625, 0.125}, {0.375, 0.875}, {0.875, 0.375},
		}
	default:
		offsets := make([][2]float64, 0, n)
		side := int(math.Ceil(math.Sqrt(float64(n))))
		for y := 0; y < side && len(offsets) < n; y++ {
			for x := 0; x < side && len(offsets) < n; x++ {
				offsets = append(offsets, [2]float64{
					(float64(x) + 0.5) / float64(side),
					(float64(y) + 0.5) / float64(side),
				})
			}
		}
		return offsets
	}
}

func aces(x float64) float64 {
	x = math.Max(0, x)
	return clamp((x*(2.51*x+0.03))/(x*(2.43*x+0.59)+0.14), 0, 1)
}

func toSRGB8(x float64) uint8 {
	x = clamp(x, 0, 1)
	x = aces(x)
	if x <= 0.0031308 {
		x *= 12.92
	} else {
		x = 1.055*math.Pow(x, 1/2.4) - 0.055
	}
	return uint8(math.Round(x * 255))
}

func blurredLuma(src *HDRBuffer, threshold float64) *HDRBuffer {
	sw, sh := src.W, src.H
	dw := (sw + 3) / 4
	dh := (sh + 3) / 4
	small := NewHDRBuffer(dw, dh)
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var sum Vec3
			for oy := 0; oy < 4; oy++ {
				for ox := 0; ox < 4; ox++ {
					sx := int(math.Min(float64(sw-1), float64(x*4+ox)))
					sy := int(math.Min(float64(sh-1), float64(y*4+oy)))
					c := src.Pix[sy*sw+sx]
					sum = sum.Add(c)
				}
			}
			c := sum.Mul(1.0 / 16.0)
			l := c.Luminance()
			if l > threshold {
				small.Pix[y*dw+x] = c.Mul((l - threshold) / (l + 1e-5))
			}
		}
	}
	return gaussianBlur(small, 7)
}

func gaussianBlur(src *HDRBuffer, radius int) *HDRBuffer {
	tmp := NewHDRBuffer(src.W, src.H)
	dst := NewHDRBuffer(src.W, src.H)
	kernel := make([]float64, radius*2+1)
	var norm float64
	for i := -radius; i <= radius; i++ {
		w := math.Exp(-float64(i*i) / (2 * float64(radius*radius) / 5))
		kernel[i+radius] = w
		norm += w
	}
	for i := range kernel {
		kernel[i] /= norm
	}
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			var sum Vec3
			for k := -radius; k <= radius; k++ {
				xx := x + k
				if xx < 0 {
					xx = -xx
				} else if xx >= src.W {
					xx = 2*src.W - xx - 1
				}
				sum = sum.Add(src.Pix[y*src.W+xx].Mul(kernel[k+radius]))
			}
			tmp.Pix[y*src.W+x] = sum
		}
	}
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			var sum Vec3
			for k := -radius; k <= radius; k++ {
				yy := y + k
				if yy < 0 {
					yy = -yy
				} else if yy >= src.H {
					yy = 2*src.H - yy - 1
				}
				sum = sum.Add(tmp.Pix[yy*src.W+x].Mul(kernel[k+radius]))
			}
			dst.Pix[y*src.W+x] = sum
		}
	}
	return dst
}

func sampleBilinear(src *HDRBuffer, fx, fy float64) Vec3 {
	x0 := clampInt(int(math.Floor(fx)), 0, src.W-1)
	y0 := clampInt(int(math.Floor(fy)), 0, src.H-1)
	x1 := int(math.Min(float64(src.W-1), float64(x0+1)))
	y1 := int(math.Min(float64(src.H-1), float64(y0+1)))
	tx := fx - float64(x0)
	ty := fy - float64(y0)
	a := src.Pix[y0*src.W+x0].Lerp(src.Pix[y0*src.W+x1], tx)
	b := src.Pix[y1*src.W+x0].Lerp(src.Pix[y1*src.W+x1], tx)
	return a.Lerp(b, ty)
}

func clampInt(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func scaleRGBA(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if w <= 0 || h <= 0 {
		return dst
	}
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	for y := 0; y < h; y++ {
		sy := (float64(y)+0.5)*float64(sh)/float64(h) - 0.5
		y0 := clampInt(int(math.Floor(sy)), 0, sh-1)
		y1 := clampInt(y0+1, 0, sh-1)
		ty := sy - float64(y0)
		for x := 0; x < w; x++ {
			sx := (float64(x)+0.5)*float64(sw)/float64(w) - 0.5
			x0 := clampInt(int(math.Floor(sx)), 0, sw-1)
			x1 := clampInt(x0+1, 0, sw-1)
			tx := sx - float64(x0)
			c00 := src.RGBAAt(x0, y0)
			c10 := src.RGBAAt(x1, y0)
			c01 := src.RGBAAt(x0, y1)
			c11 := src.RGBAAt(x1, y1)
			a := color.RGBA{
				R: uint8(math.Round(mix(float64(c00.R), float64(c10.R), tx))),
				G: uint8(math.Round(mix(float64(c00.G), float64(c10.G), tx))),
				B: uint8(math.Round(mix(float64(c00.B), float64(c10.B), tx))),
				A: 255,
			}
			b := color.RGBA{
				R: uint8(math.Round(mix(float64(c01.R), float64(c11.R), tx))),
				G: uint8(math.Round(mix(float64(c01.G), float64(c11.G), tx))),
				B: uint8(math.Round(mix(float64(c01.B), float64(c11.B), tx))),
				A: 255,
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(math.Round(mix(float64(a.R), float64(b.R), ty))),
				G: uint8(math.Round(mix(float64(a.G), float64(b.G), ty))),
				B: uint8(math.Round(mix(float64(a.B), float64(b.B), ty))),
				A: 255,
			})
		}
	}
	return dst
}

// ToImage tone-maps the HDR result and adds a restrained, physically stable
// bloom around bright clearcoat reflections.
func (b *HDRBuffer) ToImage(exposure, bloomStrength float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, b.W, b.H))
	var bloom *HDRBuffer
	if bloomStrength > 0 {
		bloom = blurredLuma(b, 0.65/exposure)
	}
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.Pix[y*b.W+x]
			if bloom != nil {
				bx := (float64(x)+0.5)*float64(bloom.W)/float64(b.W) - 0.5
				by := (float64(y)+0.5)*float64(bloom.H)/float64(b.H) - 0.5
				c = c.Add(sampleBilinear(bloom, bx, by).Mul(bloomStrength))
			}
			img.SetRGBA(x, y, color.RGBA{
				R: toSRGB8(c.X * exposure),
				G: toSRGB8(c.Y * exposure),
				B: toSRGB8(c.Z * exposure),
				A: 255,
			})
		}
	}
	return img
}
