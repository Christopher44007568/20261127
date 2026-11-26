// Command thinfilm renders zero-dependency thin-film interference figures:
// a thickness/angle color ramp, reflectance spectra and a metal-sphere
// comparison. All output is PNG written via the Go standard library.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"thinfilm/internal/render"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory for PNG files")
	filmN := flag.Float64("film-n", 1.5, "dielectric film refractive index")
	rampMax := flag.Float64("ramp-max", 1000, "ramp maximum thickness in nm")
	sphereD := flag.Float64("sphere-d", 500, "coated sphere film thickness in nm")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	jobs := []struct {
		name string
		draw func() *render.Canvas
	}{
		{
			"ramp.png",
			func() *render.Canvas {
				return render.DrawRamp(render.RampConfig{
					Width:        1100,
					Height:       620,
					FilmN:        *filmN,
					ThicknessMax: *rampMax,
					Title:        "THIN FILM COLOR: THICKNESS AND ANGLE",
				})
			},
		},
		{
			"spectrum.png",
			func() *render.Canvas {
				return render.DrawSpectrum(render.SpectrumConfig{
					Width:       1100,
					Height:      640,
					FilmN:       *filmN,
					Thicknesses: []float64{0, 100, 200, 400, 600, 800},
					Title:       "REFLECTANCE SPECTRA (NORMAL INCIDENCE)",
				})
			},
		},
		{
			"spheres.png",
			func() *render.Canvas {
				return render.DrawSpheres(render.SphereConfig{
					Width:         1000,
					Height:        560,
					FilmN:         *filmN,
					FilmThickness: *sphereD,
					Title:         "BARE METAL VS FILM-COATED SPHERE",
				})
			},
		},
	}

	for _, job := range jobs {
		path := filepath.Join(*outDir, job.name)
		if err := writePNG(path, job.draw()); err != nil {
			fatal(err)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

func writePNG(path string, c *render.Canvas) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.EncodePNG(f)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "thinfilm:", err)
	os.Exit(1)
}
