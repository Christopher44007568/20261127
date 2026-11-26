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

// WritePFM writes a little-endian, 32-bit RGB Portable Float Map. PFM rows
// are stored bottom-to-top as required by the format.
func WritePFM(path string, im *Image) error {
	if err := im.Check(); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "PF\n%d %d\n-1.0\n", im.Width, im.Height); err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	var buf [4]byte
	for y := im.Height - 1; y >= 0; y-- {
		for x := 0; x < im.Width; x++ {
			c := im.At(x, y)
			for _, v := range []float64{c.R, c.G, c.B} {
				binary.LittleEndian.PutUint32(buf[:], math.Float32bits(float32(v)))
				if _, err := bw.Write(buf[:]); err != nil {
					return err
				}
			}
		}
	}
	return bw.Flush()
}

func ReadPFM(path string) (*Image, error) {
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
	if magic != "PF" && magic != "Pf" {
		return nil, fmt.Errorf("invalid PFM magic %q", magic)
	}
	w, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	h, err := readIntToken(br)
	if err != nil {
		return nil, err
	}
	scale, err := readFloatToken(br)
	if err != nil {
		return nil, err
	}
	im, err := NewImage(w, h)
	if err != nil {
		return nil, err
	}
	channels := 3
	if magic == "Pf" {
		channels = 1
	}
	littleEndian := scale < 0
	var raw [4]byte
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			values := [3]float64{0, 0, 1}
			for c := 0; c < channels; c++ {
				if _, err := io.ReadFull(br, raw[:]); err != nil {
					return nil, fmt.Errorf("read PFM pixel: %w", err)
				}
				var bits uint32
				if littleEndian {
					bits = binary.LittleEndian.Uint32(raw[:])
				} else {
					bits = binary.BigEndian.Uint32(raw[:])
				}
				values[c] = float64(math.Float32frombits(bits))
			}
			if channels == 1 {
				values[1], values[2] = values[0], values[0]
			}
			im.Set(x, y, RGB{values[0], values[1], values[2]})
		}
	}
	return im, nil
}

func readToken(br *bufio.Reader) (string, error) {
	for {
		b, err := br.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '#' {
			for b != '\n' {
				b, err = br.ReadByte()
				if err != nil {
					return "", err
				}
			}
			continue
		}
		if b <= ' ' {
			continue
		}
		token := []byte{b}
		for {
			b, err = br.ReadByte()
			if err == io.EOF {
				return string(token), nil
			}
			if err != nil {
				return "", err
			}
			if b <= ' ' {
				if b == '\r' {
					if next, err := br.Peek(1); err == nil && next[0] == '\n' {
						_, _ = br.ReadByte()
					}
				}
				return string(token), nil
			}
			token = append(token, b)
		}
	}
}

func readIntToken(br *bufio.Reader) (int, error) {
	token, err := readToken(br)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("invalid integer token %q: %w", token, err)
	}
	return v, nil
}

func readFloatToken(br *bufio.Reader) (float64, error) {
	token, err := readToken(br)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float token %q: %w", token, err)
	}
	return v, nil
}
