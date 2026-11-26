package bezierstroke

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

// Canvas is an opaque RGB canvas with area-coverage compositing.
type Canvas struct {
	W, H int
	Pix  *image.RGBA
}

func NewCanvas(w, h int, background color.RGBA) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		off := y * img.Stride
		for x := 0; x < w; x++ {
			img.Pix[off+x*4+0] = background.R
			img.Pix[off+x*4+1] = background.G
			img.Pix[off+x*4+2] = background.B
			img.Pix[off+x*4+3] = 255
		}
	}
	return &Canvas{W: w, H: h, Pix: img}
}

type coverageEdge struct {
	x0, y0 float64
	x1, y1 float64
	w      int
}

type crossing struct {
	x   float64
	dir int
}

// Coverage computes per-pixel area coverage for the union of polygons. Each
// pixel is sampled on an samples*samples grid, so values range from zero to
// samples-squared. Polygon winding is nonzero and every StrokePolyline
// polygon has the same orientation.
func Coverage(w, h, samples int, polygons []Polygon) []uint16 {
	if samples < 1 {
		samples = 1
	}
	cov := make([]uint16, w*h)
	if w <= 0 || h <= 0 || len(polygons) == 0 {
		return cov
	}

	edges := make([]coverageEdge, 0, len(polygons)*4)
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polygons {
		if len(poly) < 3 {
			continue
		}
		for i := range poly {
			a := poly[i]
			b := poly[(i+1)%len(poly)]
			minX = math.Min(minX, math.Min(a.X, b.X))
			maxX = math.Max(maxX, math.Max(a.X, b.X))
			minY = math.Min(minY, math.Min(a.Y, b.Y))
			maxY = math.Max(maxY, math.Max(a.Y, b.Y))
			if a.Y == b.Y {
				continue
			}
			dir := 1
			if b.Y < a.Y {
				dir = -1
			}
			edges = append(edges, coverageEdge{a.X, a.Y, b.X, b.Y, dir})
		}
	}
	if len(edges) == 0 {
		return cov
	}

	minX = math.Max(minX, 0)
	maxX = math.Min(maxX, float64(w))
	minY = math.Max(minY, 0)
	maxY = math.Min(maxY, float64(h))
	if minX >= maxX || minY >= maxY {
		return cov
	}

	s := float64(samples)
	sy0 := int(math.Ceil(minY*s - 0.5))
	sy1 := int(math.Ceil(maxY*s - 0.5))
	sy0 = clampInt(sy0, 0, h*samples)
	sy1 = clampInt(sy1, 0, h*samples)
	cs := make([]crossing, 0, len(edges))

	for sy := sy0; sy < sy1; sy++ {
		ys := (float64(sy) + 0.5) / s
		cs = cs[:0]
		for _, e := range edges {
			lo, hi := e.y0, e.y1
			if lo > hi {
				lo, hi = hi, lo
			}
			if ys < lo || ys >= hi {
				continue
			}
			cs = append(cs, crossing{
				x:   e.x0 + (ys-e.y0)*(e.x1-e.x0)/(e.y1-e.y0),
				dir: e.w,
			})
		}
		if len(cs) == 0 {
			continue
		}
		sort.Slice(cs, func(i, j int) bool { return cs[i].x < cs[j].x })
		winding := 0
		prev := math.Inf(-1)
		py := sy / samples
		for _, c := range cs {
			if winding != 0 {
				addSpan(cov, w, py, samples, prev, c.x)
			}
			winding += c.dir
			prev = c.x
		}
	}
	return cov
}

func addSpan(cov []uint16, w, y, samples int, x0, x1 float64) {
	x0 = math.Max(x0, 0)
	x1 = math.Min(x1, float64(w))
	if x0 >= x1 {
		return
	}
	s := float64(samples)
	sx0 := int(math.Ceil(x0*s - 0.5))
	sx1 := int(math.Ceil(x1*s - 0.5))
	sx0 = clampInt(sx0, 0, w*samples)
	sx1 = clampInt(sx1, 0, w*samples)
	if sx0 >= sx1 {
		return
	}
	row := cov[y*w : (y+1)*w]
	for sx := sx0; sx < sx1; {
		px := sx / samples
		next := (px + 1) * samples
		if next > sx1 {
			next = sx1
		}
		row[px] += uint16(next - sx)
		sx = next
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// FillCoverage composites over this canvas. Colors are opaque RGB values;
// coverage supplies the antialiasing.
func (c *Canvas) FillCoverage(cov []uint16, samples int, fg color.RGBA) {
	if samples < 1 {
		samples = 1
	}
	maxCov := float64(samples * samples)
	for i, v := range cov {
		if v == 0 {
			continue
		}
		off := i * 4
		// image.RGBA uses premultiplied values. The canvas is fully opaque,
		// so blending opaque colors reduces to a channel-wise lerp.
		coverage := math.Min(float64(v)/maxCov, 1)
		inv := 1 - coverage
		c.Pix.Pix[off+0] = uint8(float64(fg.R)*coverage + float64(c.Pix.Pix[off+0])*inv)
		c.Pix.Pix[off+1] = uint8(float64(fg.G)*coverage + float64(c.Pix.Pix[off+1])*inv)
		c.Pix.Pix[off+2] = uint8(float64(fg.B)*coverage + float64(c.Pix.Pix[off+2])*inv)
		c.Pix.Pix[off+3] = 255
	}
}

func (c *Canvas) FillPolygons(polygons []Polygon, samples int, fg color.RGBA) {
	if len(polygons) == 0 {
		return
	}
	cov := Coverage(c.W, c.H, samples, polygons)
	c.FillCoverage(cov, samples, fg)
}

func (c *Canvas) SavePNG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, c.Pix); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
