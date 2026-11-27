package main

import (
	"image"
	"image/png"
	"os"
)

var (
	colorSteel    = Vec3{0.56, 0.57, 0.58}
	colorAluminum = Vec3{0.91, 0.92, 0.92}
	colorSkyTop   = Vec3{0.18, 0.21, 0.27}
	colorSkyBot   = Vec3{0.08, 0.09, 0.13}
	colorPanel    = [3]float64{0.16, 0.19, 0.24}
	colorInk      = [3]float64{0.86, 0.90, 0.95}
	colorFrame    = [3]float64{0.30, 0.42, 0.55}
	colorCyan     = [3]float64{0.28, 0.78, 0.95}
	colorDark     = [3]float64{0.07, 0.09, 0.12}
)

func makeEnv(sunDir Vec3, boxCenter Vec3, boxU Vec3, halfU, halfV float64) *Env {
	return &Env{
		Top:    colorSkyTop,
		Bottom: colorSkyBot,
		Sun: &DirectionalLight{
			Dir:        sunDir.Normalize(),
			Color:      Vec3{1.0, 0.97, 0.92},
			Intensity:  1.7,
			AngularRad: 0.035,
			Samples:    4,
		},
		Box: &SoftBox{
			Center:    boxCenter.Normalize(),
			AxisU:     boxU.Normalize(),
			AxisV:     cross(boxCenter.Normalize(), boxU.Normalize()).Normalize(),
			HalfU:     halfU,
			HalfV:     halfV,
			Color:     Vec3{0.8, 0.86, 1.0},
			Intensity: 0.9,
			Samples:   4,
		},
	}
}

func writePNG(path string, c *Canvas) error {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for i := 0; i < c.W*c.H; i++ {
		img.Pix[i*4+0] = c.Pixels[i*3+0]
		img.Pix[i*4+1] = c.Pixels[i*3+1]
		img.Pix[i*4+2] = c.Pixels[i*3+2]
		img.Pix[i*4+3] = 255
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg := RenderConfig{
		SamplesPerPixel: 4,
		SunSamples:      16,
		BoxSamples:      16,
	}
	lut := buildEnergyLUT()

	type job struct {
		path string
		draw func(*EnergyLUT, RenderConfig) *Canvas
	}
	jobs := []job{
		{"artifacts/sphere_reference.png", buildSphereScene},
		{"artifacts/brushed_plate.png", buildPlateScene},
		{"artifacts/roughness_ladder.png", buildLadderScene},
	}
	for _, j := range jobs {
		canvas := j.draw(lut, cfg)
		if err := writePNG(j.path, canvas); err != nil {
			return err
		}
	}
	return nil
}
