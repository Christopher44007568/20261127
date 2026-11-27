package furnace

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Canvas is a linear HDR RGB buffer.
type Canvas struct {
	W, H int
	Pix  []Vec3
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Pix: make([]Vec3, w*h)}
}

func (c *Canvas) index(x, y int) int { return y*c.W + x }

func (c *Canvas) Set(x, y int, col Vec3) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Pix[c.index(x, y)] = col
}

func (c *Canvas) Add(x, y int, col Vec3) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := c.index(x, y)
	c.Pix[i] = c.Pix[i].Add(col)
}

func (c *Canvas) Fill(col Vec3) {
	for i := range c.Pix {
		c.Pix[i] = col
	}
}

// ToneMap applies a stable Reinhard-ish curve followed by gamma encoding.
func ToneMap(x float64, exposure float64) float64 {
	x = math.Max(0, x*exposure)
	x = x / (1 + x)
	return math.Pow(clamp(x, 0, 1), 1/2.2)
}

func (c *Canvas) RGBA(exposure float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			p := c.Pix[c.index(x, y)]
			r := uint8(ToneMap(p.X, exposure) * 255)
			g := uint8(ToneMap(p.Y, exposure) * 255)
			b := uint8(ToneMap(p.Z, exposure) * 255)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

// SavePNG writes a tone-mapped 8-bit PNG, creating the directory if needed.
func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	return enc.Encode(f, img)
}

// SaveCanvasPNG tone maps and saves an HDR canvas.
func SaveCanvasPNG(path string, c *Canvas, exposure float64) error {
	return SavePNG(path, c.RGBA(exposure))
}

// HeatColor maps t in [0,1] through blue, cyan, green, yellow, and red.
func HeatColor(t float64) color.RGBA {
	t = saturate(t)
	stops := [5][3]float64{
		{0.05, 0.10, 0.45},
		{0.00, 0.65, 0.95},
		{0.10, 0.85, 0.35},
		{1.00, 0.88, 0.10},
		{0.95, 0.12, 0.08},
	}
	q := t * float64(len(stops)-1)
	i := int(q)
	if i >= len(stops)-1 {
		i = len(stops) - 2
	}
	f := q - float64(i)
	r := lerp(stops[i][0], stops[i+1][0], f)
	g := lerp(stops[i][1], stops[i+1][1], f)
	b := lerp(stops[i][2], stops[i+1][2], f)
	return color.RGBA{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
}
