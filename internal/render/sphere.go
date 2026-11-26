package render

import (
	"math"

	"thinfilm/internal/color"
	"thinfilm/internal/thinfilm"
)

// SphereConfig controls the bare-vs-coated metal sphere comparison.
type SphereConfig struct {
	Width, Height int
	FilmN         float64
	FilmThickness float64 // nm; 0 would make both spheres identical
	Title         string
}

type light struct {
	dir     [3]float64
	ambient float64
}

// DrawSpheres renders two shaded spheres side by side on a neutral ground:
// the left one is bare iron, the right one carries the interference film.
// Shading uses the local surface normal, so the film's interference color is
// modulated by the same Lambertian term on both spheres.
func DrawSpheres(cfg SphereConfig) *Canvas {
	c := NewCanvas(cfg.Width, cfg.Height, RGB{238, 238, 234})
	radius := float64(cfg.Height) * 0.30
	cy := cfg.Height/2 + 10
	bareX := cfg.Width / 4
	coatedX := 3 * cfg.Width / 4

	key := light{
		dir:     normalize([3]float64{-0.45, 0.55, 0.75}),
		ambient: 0.28,
	}

	drawGroundShadow(c, bareX, cy, radius)
	drawGroundShadow(c, coatedX, cy, radius)

	bare := thinfilm.NewStack(cfg.FilmN, 0)
	coated := thinfilm.NewStack(cfg.FilmN, cfg.FilmThickness)

	renderSphere(c, bareX, cy, radius, &bare, key)
	renderSphere(c, coatedX, cy, radius, &coated, key)

	// Labels.
	labelBare := "BARE IRON (D=0)"
	labelCoat := "COATED IRON D=" + formatWavelength(int(cfg.FilmThickness)) + "NM"
	c.Text(bareX-TextWidth(labelBare, 2)/2, cy+int(radius)+26, 2, labelBare, RGB{40, 40, 40})
	c.Text(coatedX-TextWidth(labelCoat, 2)/2, cy+int(radius)+26, 2, labelCoat, RGB{40, 40, 40})

	if cfg.Title != "" {
		c.Text((cfg.Width-TextWidth(cfg.Title, 2))/2, 12, 2, cfg.Title, RGB{40, 40, 40})
	}
	return c
}

func renderSphere(c *Canvas, cx, cy int, radius float64,
	stack *thinfilm.Stack, l light) {

	rInt := int(math.Ceil(radius))
	for dy := -rInt; dy <= rInt; dy++ {
		for dx := -rInt; dx <= rInt; dx++ {
			rr2 := float64(dx*dx + dy*dy)
			if rr2 > radius*radius {
				continue
			}
			nz := math.Sqrt(radius*radius - rr2)
			normal := normalize([3]float64{
				float64(dx) / radius,
				float64(dy) / radius,
				nz / radius,
			})

			// Incidence angle of the key light relative to the normal.
			dot := normal[0]*l.dir[0] + normal[1]*l.dir[1] + normal[2]*l.dir[2]
			diffuse := math.Max(0, dot)
			shade := l.ambient + (1-l.ambient)*diffuse

			// Fresnel angle varies across the surface; near the silhouette
			// the local incidence becomes grazing and reflection rises.
			incidence := math.Acos(clamp(normal[2], 0, 1)) * 180 / math.Pi
			spec := func(lambda float64) float64 {
				return stack.Reflectance(lambda, incidence)
			}
			linR, linG, linB := color.RGBFloat(spec)
			col := rgb(
				255*encodeSRGB(linR*shade),
				255*encodeSRGB(linG*shade),
				255*encodeSRGB(linB*shade),
			)
			c.Set(cx+dx, cy+dy, col)
		}
	}
}

func drawGroundShadow(c *Canvas, cx, cy int, radius float64) {
	shadowY := cy + int(radius) + 6
	shadowW := int(radius * 1.1)
	shadowH := int(radius * 0.22)
	for dy := -shadowH; dy <= shadowH; dy++ {
		for dx := -shadowW; dx <= shadowW; dx++ {
			ellipse := float64(dx*dx)/float64(shadowW*shadowW) +
				float64(dy*dy)/float64(shadowH*shadowH)
			if ellipse > 1 {
				continue
			}
			alpha := 0.18 * (1 - ellipse)
			c.Blend(cx+dx, shadowY+dy, RGB{30, 30, 30}, alpha)
		}
	}
}

func encodeSRGB(linear float64) float64 {
	if linear <= 0.0031308 {
		return 12.92 * linear
	}
	return 1.055*math.Pow(linear, 1/2.4) - 0.055
}

func normalize(v [3]float64) [3]float64 {
	length := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	return [3]float64{v[0] / length, v[1] / length, v[2] / length}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
