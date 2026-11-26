package qem

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type RGB struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

func rgbf(r, g, b float64) RGB {
	clamp := func(x float64) uint8 {
		if x <= 0 {
			return 0
		}
		if x >= 255 {
			return 255
		}
		return uint8(x + 0.5)
	}
	return RGB{clamp(r), clamp(g), clamp(b)}
}

var (
	ColorBackground = RGB{18, 22, 29}
	ColorWire       = RGB{8, 11, 16}
	ColorHighlight  = RGB{255, 43, 170}
	ColorHighlight2 = RGB{255, 196, 61}
)

// PPM is an 8-bit binary P6 image.
type PPM struct {
	Width  int   `json:"width"`
	Height int   `json:"height"`
	Pix    []RGB `json:"-"`
}

func NewPPM(w, h int, bg RGB) *PPM {
	p := &PPM{Width: w, Height: h, Pix: make([]RGB, w*h)}
	p.Clear(bg)
	return p
}

func (p *PPM) Clear(c RGB) {
	for i := range p.Pix {
		p.Pix[i] = c
	}
}

func (p *PPM) index(x, y int) int { return y*p.Width + x }

func (p *PPM) InBounds(x, y int) bool {
	return x >= 0 && x < p.Width && y >= 0 && y < p.Height
}

func (p *PPM) Set(x, y int, c RGB) {
	if p.InBounds(x, y) {
		p.Pix[p.index(x, y)] = c
	}
}

func (p *PPM) Get(x, y int) RGB {
	if !p.InBounds(x, y) {
		return RGB{}
	}
	return p.Pix[p.index(x, y)]
}

func (p *PPM) Blend(x, y int, c RGB, alpha float64) {
	if !p.InBounds(x, y) || alpha <= 0 {
		return
	}
	if alpha >= 1 {
		p.Set(x, y, c)
		return
	}
	d := p.Pix[p.index(x, y)]
	out := RGB{
		R: uint8(float64(d.R)*(1-alpha) + float64(c.R)*alpha + 0.5),
		G: uint8(float64(d.G)*(1-alpha) + float64(c.G)*alpha + 0.5),
		B: uint8(float64(d.B)*(1-alpha) + float64(c.B)*alpha + 0.5),
	}
	p.Pix[p.index(x, y)] = out
}

func (p *PPM) WriteTo(w io.Writer) (int64, error) {
	bw := bufio.NewWriter(w)
	header := fmt.Sprintf("P6\n%d %d\n255\n", p.Width, p.Height)
	n, err := bw.WriteString(header)
	if err != nil {
		return int64(n), err
	}
	for i := range p.Pix {
		_, err := bw.Write([]byte{p.Pix[i].R, p.Pix[i].G, p.Pix[i].B})
		if err != nil {
			return int64(n), err
		}
		n += 3
	}
	if err := bw.Flush(); err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

func SavePPM(path string, img *PPM) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = img.WriteTo(f)
	return err
}

// ContactSheet tiles equally sized images in column-major rows.
func ContactSheet(images []*PPM, cols int, bg RGB) *PPM {
	if len(images) == 0 {
		return NewPPM(1, 1, bg)
	}
	if cols < 1 {
		cols = 1
	}
	rows := (len(images) + cols - 1) / cols
	tileW, tileH := images[0].Width, images[0].Height
	sheet := NewPPM(cols*tileW, rows*tileH, bg)
	for idx, img := range images {
		if img.Width != tileW || img.Height != tileH {
			continue
		}
		cx := (idx % cols) * tileW
		cy := (idx / cols) * tileH
		for y := 0; y < tileH; y++ {
			for x := 0; x < tileW; x++ {
				sheet.Pix[(cy+y)*sheet.Width+cx+x] = img.Pix[y*tileW+x]
			}
		}
	}
	return sheet
}
