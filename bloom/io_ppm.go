package bloom

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

type PPMImage struct {
	Width  int
	Height int
	MaxVal int
	Pixels []RGB
}

func PPM16Code(v float64) uint16 {
	v = math.Max(0, math.Min(1, v))
	return uint16(math.RoundToEven(v * 65535))
}

// WritePPM16 writes binary P6 with maxval 65535. Input colors are display
// values in [0,1]; write-time clamping is reported by the caller's metrics.
func WritePPM16(path string, im *Image) error {
	if err := im.Check(); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n65535\n", im.Width, im.Height); err != nil {
		return err
	}
	var raw [6]byte
	for _, c := range im.Pixels {
		r := PPM16Code(c.R)
		g := PPM16Code(c.G)
		b := PPM16Code(c.B)
		binary.BigEndian.PutUint16(raw[0:2], r)
		binary.BigEndian.PutUint16(raw[2:4], g)
		binary.BigEndian.PutUint16(raw[4:6], b)
		if _, err := bw.Write(raw[:]); err != nil {
			return err
		}
	}
	return bw.Flush()
}

func ReadPPM(path string) (*PPMImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	magic, err := readToken(br)
	if err != nil {
		return nil, err
	}
	if magic != "P6" && magic != "P3" {
		return nil, fmt.Errorf("unsupported PPM magic %q", magic)
	}
	w, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	h, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	maxVal, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	if maxVal < 1 || maxVal > 65535 {
		return nil, fmt.Errorf("invalid PPM maxval %d", maxVal)
	}
	n, ok := checkedPixelCount(w, h)
	if !ok {
		return nil, fmt.Errorf("invalid PPM dimensions %dx%d", w, h)
	}
	out := &PPMImage{Width: w, Height: h, MaxVal: maxVal, Pixels: make([]RGB, n)}
	if magic == "P3" {
		return readPPMASCII(br, out)
	}
	sampleBytes := 1
	if maxVal > 255 {
		sampleBytes = 2
	}
	raw := make([]byte, n*3*sampleBytes)
	if _, err := io.ReadFull(br, raw); err != nil {
		return nil, fmt.Errorf("read PPM raster: %w", err)
	}
	scale := 1 / float64(maxVal)
	pos := 0
	for i := range out.Pixels {
		values := [3]float64{}
		for c := 0; c < 3; c++ {
			var sample int
			if sampleBytes == 2 {
				sample = int(binary.BigEndian.Uint16(raw[pos : pos+2]))
				pos += 2
			} else {
				sample = int(raw[pos])
				pos++
			}
			if sample > maxVal {
				return nil, fmt.Errorf("PPM sample %d exceeds maxval %d", sample, maxVal)
			}
			values[c] = float64(sample) * scale
		}
		out.Pixels[i] = RGB{values[0], values[1], values[2]}
	}
	return out, nil
}

func readPPMASCII(br *bufio.Reader, out *PPMImage) (*PPMImage, error) {
	scale := 1 / float64(out.MaxVal)
	for i := range out.Pixels {
		values := [3]float64{}
		for c := 0; c < 3; c++ {
			token, err := readToken(br)
			if err != nil {
				return nil, err
			}
			sample, err := strconv.Atoi(token)
			if err != nil || sample < 0 || sample > out.MaxVal {
				return nil, fmt.Errorf("invalid PPM ASCII sample %q", token)
			}
			values[c] = float64(sample) * scale
		}
		out.Pixels[i] = RGB{values[0], values[1], values[2]}
	}
	return out, nil
}
