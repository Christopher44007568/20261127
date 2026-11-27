package main

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
)

// WritePPM writes f as a binary P6 PPM. PPM is used because it needs no
// third-party dependencies and can be inspected by most image tools.
func WritePPM(path string, f *Framebuffer) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bw := bufio.NewWriter(file)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", f.W, f.H); err != nil {
		return err
	}
	if _, err := bw.Write(f.Pix); err != nil {
		return err
	}
	return bw.Flush()
}

// WritePNG writes an RGBA PNG preview of f using only the standard library.
func WritePNG(path string, f *Framebuffer) error {
	img := image.NewRGBA(image.Rect(0, 0, f.W, f.H))
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			i := f.index(x, y)
			o := img.PixOffset(x, y)
			img.Pix[o] = f.Pix[i]
			img.Pix[o+1] = f.Pix[i+1]
			img.Pix[o+2] = f.Pix[i+2]
			img.Pix[o+3] = 255
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

// ReadPPM reads a P6 (or plain P3) PPM file. It exists so tests and external
// verification can decode the exact artifacts written by this program.
func ReadPPM(path string) (*Framebuffer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return DecodePPM(file)
}

// DecodePPM parses a minimal PPM (P6 binary or P3 ASCII), skipping comments.
func DecodePPM(r io.Reader) (*Framebuffer, error) {
	br := bufio.NewReader(r)
	magic, err := readToken(br)
	if err != nil {
		return nil, fmt.Errorf("read PPM magic: %w", err)
	}
	if magic != "P6" && magic != "P3" {
		return nil, fmt.Errorf("unsupported PPM magic %q (want P6 or P3)", magic)
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
	if maxVal != 255 {
		return nil, fmt.Errorf("unsupported max value %d (want 255)", maxVal)
	}

	fb := NewFramebuffer(w, h)
	if magic == "P6" {
		// readToken already consumed the single whitespace that terminated
		// maxVal. Skip any additional whitespace, then unread the first
		// raster byte so ReadFull starts exactly at the raster beginning.
		for {
			b, err := br.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("read raster separator: %w", err)
			}
			if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
				continue
			}
			if err := br.UnreadByte(); err != nil {
				return nil, err
			}
			break
		}
		if _, err := io.ReadFull(br, fb.Pix); err != nil {
			return nil, fmt.Errorf("read P6 raster: %w", err)
		}
		return fb, nil
	}

	for i := 0; i < len(fb.Pix); i++ {
		v, err := readIntToken(br)
		if err != nil {
			return nil, fmt.Errorf("read P3 sample %d: %w", i, err)
		}
		fb.Pix[i] = uint8(v)
	}
	return fb, nil
}

func readToken(r *bufio.Reader) (string, error) {
	var tok strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF && tok.Len() > 0 {
				return tok.String(), nil
			}
			return "", err
		}
		if b == '#' {
			if _, err := r.ReadString('\n'); err != nil {
				return "", err
			}
			continue
		}
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			if tok.Len() > 0 {
				return tok.String(), nil
			}
			continue
		}
		tok.WriteByte(b)
	}
}

func readIntToken(r *bufio.Reader) (int, error) {
	tok, err := readToken(r)
	if err != nil {
		return 0, err
	}
	var v int
	if _, err := fmt.Sscanf(tok, "%d", &v); err != nil {
		return 0, fmt.Errorf("parse int token %q: %w", tok, err)
	}
	return v, nil
}
