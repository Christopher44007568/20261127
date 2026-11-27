package flakepaint

import (
	"bytes"
	"image/png"
	"math"
	"testing"
)

func TestFarRenderDeterministic(t *testing.T) {
	a := RenderFarSphere(120, 76, 1, 0x20261127).ToImage(0.95, 0)
	b := RenderFarSphere(120, 76, 1, 0x20261127).ToImage(0.95, 0)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("far sphere render changed between identical frames")
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, a); err != nil {
		t.Fatal(err)
	}
	if encoded.Len() == 0 {
		t.Fatal("PNG encoder produced an empty image")
	}
}

func TestFarHighlightFormsContinuousRun(t *testing.T) {
	buf := RenderFarSphere(240, 152, 2, 0x20261127)
	maxLuma := 0.0
	for _, c := range buf.Pix {
		maxLuma = math.Max(maxLuma, c.Luminance())
	}
	if maxLuma <= 0.1 {
		t.Fatalf("expected a bright specular band, max luma %.4f", maxLuma)
	}
	threshold := maxLuma * 0.28
	seen := make([]bool, buf.W*buf.H)
	largest := 0
	stack := make([][2]int, 0, 1024)
	for y := 0; y < buf.H; y++ {
		for x := 0; x < buf.W; x++ {
			idx := y*buf.W + x
			if seen[idx] || buf.Pix[idx].Luminance() < threshold {
				continue
			}
			seen[idx] = true
			stack = append(stack[:0], [2]int{x, y})
			size := 0
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size++
				for oy := -1; oy <= 1; oy++ {
					for ox := -1; ox <= 1; ox++ {
						xx, yy := p[0]+ox, p[1]+oy
						if xx < 0 || yy < 0 || xx >= buf.W || yy >= buf.H {
							continue
						}
						ni := yy*buf.W + xx
						if seen[ni] || buf.Pix[ni].Luminance() < threshold {
							continue
						}
						seen[ni] = true
						stack = append(stack, [2]int{xx, yy})
					}
				}
			}
			if size > largest {
				largest = size
			}
		}
	}
	if largest < buf.W*3/2 {
		t.Fatalf("bright band is not continuous: largest component=%d px", largest)
	}
}
