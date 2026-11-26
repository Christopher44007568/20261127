package render

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Frame is a tightly packed RGB image. Keeping the representation small makes
// byte-for-byte comparison with the uncached reference straightforward.
type Frame struct {
	Width  int
	Height int
	Pix    []byte
}

func NewFrame(width, height int) *Frame {
	return &Frame{
		Width:  width,
		Height: height,
		Pix:    make([]byte, width*height*3),
	}
}

func (f *Frame) Set(x, y int, r, g, b uint8) {
	if x < 0 || x >= f.Width || y < 0 || y >= f.Height {
		return
	}
	i := (y*f.Width + x) * 3
	f.Pix[i+0] = r
	f.Pix[i+1] = g
	f.Pix[i+2] = b
}

func (f *Frame) At(x, y int) (uint8, uint8, uint8) {
	i := (y*f.Width + x) * 3
	return f.Pix[i+0], f.Pix[i+1], f.Pix[i+2]
}

func (f *Frame) Hash() string {
	sum := sha256.Sum256(f.Pix)
	return hex.EncodeToString(sum[:])
}

// Diff describes the pixel difference between two frames.
type Diff struct {
	DifferentPixels int     `json:"different_pixels"`
	TotalPixels     int     `json:"total_pixels"`
	MaxChannelDelta int     `json:"max_channel_delta"`
	MeanAbsDelta    float64 `json:"mean_abs_delta"`
}

func (d Diff) Equal() bool {
	return d.TotalPixels > 0 && d.DifferentPixels == 0
}

func (d Diff) String() string {
	return fmt.Sprintf("%d/%d pixels differ, max channel delta %d, mean abs delta %.4f",
		d.DifferentPixels, d.TotalPixels, d.MaxChannelDelta, d.MeanAbsDelta)
}

func Compare(a, b *Frame) Diff {
	d := Diff{}
	if a == nil || b == nil || a.Width != b.Width || a.Height != b.Height {
		return d
	}
	d.TotalPixels = a.Width * a.Height
	var totalDelta int64
	for i := 0; i < len(a.Pix); i += 3 {
		dr := int(a.Pix[i+0]) - int(b.Pix[i+0])
		dg := int(a.Pix[i+1]) - int(b.Pix[i+1])
		db := int(a.Pix[i+2]) - int(b.Pix[i+2])
		if dr == 0 && dg == 0 && db == 0 {
			continue
		}
		d.DifferentPixels++
		if dr < 0 {
			dr = -dr
		}
		if dg < 0 {
			dg = -dg
		}
		if db < 0 {
			db = -db
		}
		if dr > d.MaxChannelDelta {
			d.MaxChannelDelta = dr
		}
		if dg > d.MaxChannelDelta {
			d.MaxChannelDelta = dg
		}
		if db > d.MaxChannelDelta {
			d.MaxChannelDelta = db
		}
		totalDelta += int64(dr + dg + db)
	}
	if d.TotalPixels > 0 {
		d.MeanAbsDelta = float64(totalDelta) / float64(d.TotalPixels*3)
	}
	return d
}
