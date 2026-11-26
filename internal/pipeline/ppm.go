package pipeline

import (
	"fmt"
	"io"
)

// WritePPM writes a binary P6 image. PPM was chosen so the deliverables need
// no image codec or external tool beyond the Go standard library.
func WritePPM(w io.Writer, frame Frame) error {
	if frame.Width <= 0 || frame.Height <= 0 {
		return fmt.Errorf("invalid frame size %dx%d", frame.Width, frame.Height)
	}
	if len(frame.RGB) != frame.Width*frame.Height*3 {
		return fmt.Errorf("RGB length %d does not match %dx%d", len(frame.RGB), frame.Width, frame.Height)
	}
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", frame.Width, frame.Height); err != nil {
		return err
	}
	_, err := w.Write(frame.RGB)
	return err
}
