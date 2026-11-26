package ppm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"shadowcache/internal/render"
)

// WriteBinary writes a standard binary PPM (P6) with an 8-bit RGB payload.
func WriteBinary(path string, frame *render.Frame) error {
	if frame == nil || frame.Width <= 0 || frame.Height <= 0 {
		return fmt.Errorf("invalid frame")
	}
	if len(frame.Pix) != frame.Width*frame.Height*3 {
		return fmt.Errorf("frame buffer has %d bytes, want %d", len(frame.Pix), frame.Width*frame.Height*3)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 64*1024)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", frame.Width, frame.Height); err != nil {
		return err
	}
	if _, err := w.Write(frame.Pix); err != nil {
		return err
	}
	return w.Flush()
}
