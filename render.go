package skinsss

import (
	"image"
	"image/color"
	"math"
)

// Mode selects which output the renderer produces.
type Mode int

const (
	ModeFront Mode = iota
	ModeSide
	ModeBack
	ModeHeatmap
)

// RenderConfig holds camera, lighting and scene toggles.
type RenderConfig struct {
	Width, Height int
	Samples       int // 1 = no SSAA, 4 = 2x2 supersampling
	SSSEnabled    bool
	CamPos        Vec3
	CamTarget     Vec3
	Focal         float64
}

// DefaultRenderConfig returns the 512px, 2x2 SSAA setup used by the CLI.
func DefaultRenderConfig(sss bool) RenderConfig {
	return RenderConfig{
		Width: 512, Height: 512, Samples: 4,
		SSSEnabled: sss,
		CamPos:     Vec3{0, 0.12, 3.4},
		CamTarget:  Vec3{0, 0, 0},
		Focal:      1.1,
	}
}

// lightDir returns the unit direction *toward* the light for each mode.
func lightDir(mode Mode) Vec3 {
	switch mode {
	case ModeFront:
		// Key light near the camera, slightly above and to the side.
		return Vec3{0.35, 0.45, 1}.Normalize()
	case ModeSide:
		return Vec3{1, 0.15, 0.25}.Normalize()
	case ModeBack:
		// Light sits behind the subject, coaxial with the camera: photons
		// travel through the volume toward the viewer.
		return Vec3{0, 0, -1}
	default:
		return Vec3{0, 0, 1}
	}
}

// trace marches a ray and returns the hit point and whether it intersected.
func trace(scene Scene, origin, dir Vec3) (Vec3, bool) {
	t := 0.0
	const maxDist = 12
	const maxSteps = 96
	for i := 0; i < maxSteps; i++ {
		p := origin.Add(dir.Scale(t))
		d := scene.SDF(p)
		if d < 0.0015 {
			return p, true
		}
		t += d
		if t > maxDist {
			break
		}
	}
	return Vec3{}, false
}

// background renders a dark vertical studio gradient for misses.
func background(dir Vec3) Vec3 {
	g := clamp01(dir.Y*0.6 + 0.55)
	return Vec3{0.05 + 0.04*g, 0.05 + 0.045*g, 0.07 + 0.05*g}
}

// frameBasis builds an orthonormal camera frame from position and target.
func frameBasis(cfg RenderConfig) (forward, right, up Vec3) {
	forward = cfg.CamTarget.Sub(cfg.CamPos).Normalize()
	right = forward.Cross(Vec3{0, 1, 0}).Normalize()
	up = right.Cross(forward).Normalize()
	return
}

// Render produces an 8-bit RGBA image for the requested mode.
func Render(scene Scene, mode Mode, cfg RenderConfig) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	forward, right, up := frameBasis(cfg)
	light := lightDir(mode)

	aspect := float64(cfg.Width) / float64(cfg.Height)
	scale := math.Sqrt(float64(cfg.Samples))

	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			acc := Vec3{}
			for sy := 0; sy < int(scale); sy++ {
				for sx := 0; sx < int(scale); sx++ {
					// Jittered sub-pixel offsets in [-0.5, 0.5].
					px := (float64(x) + (float64(sx)+0.5)/scale - float64(cfg.Width)/2) /
						(float64(cfg.Height) / 2)
					py := -(float64(y) + (float64(sy)+0.5)/scale - float64(cfg.Height)/2) /
						(float64(cfg.Height) / 2)
					dir := forward.Scale(cfg.Focal).
						Add(right.Scale(px * aspect)).
						Add(up.Scale(py)).Normalize()
					acc = acc.Add(samplePixel(scene, cfg.CamPos, dir, light, mode, cfg))
				}
			}
			col := acc.Scale(1 / float64(cfg.Samples)).Clamp01()
			img.SetRGBA(x, y, encodeGamma(col))
		}
	}
	return img
}

func samplePixel(scene Scene, origin, dir, light Vec3, mode Mode, cfg RenderConfig) Vec3 {
	p, hit := trace(scene, origin, dir)
	if !hit {
		return background(dir)
	}
	n := scene.Normal(p)
	view := dir.Scale(-1)
	curv := scene.Curvature(p)

	if mode == ModeHeatmap {
		// Heatmap measures the geometric chord through the volume along the
		// view ray, independent of any lighting setup.
		thick := scene.ChordThickness(p, dir)
		return HeatmapColor(thick / 1.6)
	}

	// Photons travel from the light toward the lit surface: the march starts
	// at the camera-side hit point and moves in the direction of the light.
	photonDir := light
	thick := scene.ChordThickness(p, photonDir)

	in := ShadeInput{
		Normal:     n,
		View:       view,
		Light:      light,
		Thickness:  thick,
		Curvature:  curv,
		SSSEnabled: cfg.SSSEnabled,
	}
	return Shade(in)
}

// encodeGamma applies a 2.2 gamma and converts to 8-bit non-alpha-premultiplied.
func encodeGamma(c Vec3) color.RGBA {
	return color.RGBA{
		R: gammaByte(c.X),
		G: gammaByte(c.Y),
		B: gammaByte(c.Z),
		A: 255,
	}
}

func gammaByte(x float64) uint8 {
	v := math.Pow(clamp01(x), 1/2.2)
	return uint8(clamp(v*255+0.5, 0, 255))
}
