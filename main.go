package main

import (
	"fmt"
	"math"
	"os"
)

// RotationSweep prints the spectral peak and hue of the film stack as the
// object tilts: the peak must move along the wavelength axis (iridescence),
// which no per-channel RGB phase shift can reproduce.
func RotationSweep(layers []Layer) {
	fmt.Println(" tilt(deg)  cos(theta)  peak(nm)  hue(deg)   sRGB")
	for deg := 0.0; deg <= 75; deg += 15 {
		cosT := math.Cos(deg * math.Pi / 180)
		s := FilmSpectrum(layers, 1.0, 1.50, cosT)
		r, g, b := s.ToSRGB(1.6)
		fmt.Printf(" %8.0f  %9.3f  %8.0f  %8.1f   (%.2f, %.2f, %.2f)\n",
			deg, cosT, s.PeakLambda(), s.Hue(), r, g, b)
	}
}

func run() error {
	layers := BeetleStack()
	lut := NewFilmLUT(layers, 1.0, 1.50, 720)
	outputs := []struct {
		name string
		fn   func(string, *FilmLUT) error
	}{
		{"artifacts/light_directions.png", RenderLightDirections},
		{"artifacts/curvature_sphere.png", RenderCurvatureSphere},
		{"artifacts/hue_stability.png", RenderHueStability},
		{"artifacts/rotation.png", RenderRotation},
	}
	for _, o := range outputs {
		if err := o.fn(o.name, lut); err != nil {
			return fmt.Errorf("%s: %w", o.name, err)
		}
		fmt.Println("wrote", o.name)
	}
	fmt.Println()
	RotationSweep(layers)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
