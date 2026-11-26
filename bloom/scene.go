package bloom

import (
	"fmt"
	"math"
)

// HDRScene builds a deterministic test image that combines a log exposure
// ramp, low-light gradients, color patches, and very bright emitters. It makes
// format banding and range handling visible without any external asset.
func HDRScene(width, height int) (*Image, error) {
	if width < 32 || height < 32 {
		return nil, fmt.Errorf("scene needs at least 32x32 pixels, got %dx%d", width, height)
	}
	im, err := NewImage(width, height)
	if err != nil {
		return nil, err
	}
	emitters := []struct {
		x, y, sigma, intensity float64
		color                  RGB
	}{
		{0.18, 0.23, 0.035, 36, RGB{1.0, 0.72, 0.42}},
		{0.52, 0.20, 0.028, 12, RGB{0.42, 0.72, 1.0}},
		{0.72, 0.53, 0.018, 220, RGB{1.0, 0.95, 0.84}},
		{0.90, 0.22, 0.009, 150000, RGB{1.0, 0.86, 0.66}},
	}
	for y := 0; y < height; y++ {
		v := (float64(y) + 0.5) / float64(height)
		for x := 0; x < width; x++ {
			u := (float64(x) + 0.5) / float64(width)
			var c RGB
			switch {
			case v < 0.25:
				lum := math.Exp2(-8 + 14*u)
				c = RGB{lum, lum, lum}
			case v < 0.40:
				lum := math.Exp2(-8 + 14*u)
				tint := 0.5 + 0.5*math.Sin(u*math.Pi*2)
				c = RGB{lum, lum * (0.72 + 0.22*tint), lum * (0.38 + 0.48*(1-tint))}
			case v < 0.62:
				idx := int(u * 5)
				if idx > 4 {
					idx = 4
				}
				levels := [5]float64{0.20, 0.65, 2.0, 8.0, 64.0}
				local := u*5 - float64(idx)
				lum := levels[idx] * (0.7 + 0.6*local)
				switch idx {
				case 0:
					c = RGB{lum, lum * 0.18, lum * 0.12}
				case 1:
					c = RGB{lum * 0.12, lum, lum * 0.18}
				case 2:
					c = RGB{lum * 0.14, lum * 0.28, lum}
				case 3:
					c = RGB{lum, lum * 0.72, lum * 0.12}
				default:
					c = RGB{lum, lum, lum}
				}
			default:
				low := 0.0006 + 0.075*u
				ripple := 1 + 0.12*math.Sin((v-0.62)*31+u*9)
				lum := low * ripple
				c = RGB{lum, lum * 0.94, lum * 1.06}
			}
			for _, e := range emitters {
				dx := u - e.x
				dy := (v - e.y) * 1.7
				glow := math.Exp(-(dx*dx + dy*dy) / (2 * e.sigma * e.sigma))
				c = c.Add(e.color.Mul(e.intensity * glow))
			}
			hotDX := (u - emitters[3].x) * float64(width)
			hotDY := (v - emitters[3].y) * float64(height)
			hotRadius := math.Max(1.2, float64(width)*0.003)
			if hotDX*hotDX+hotDY*hotDY <= hotRadius*hotRadius {
				c = emitters[3].color.Mul(emitters[3].intensity)
			}
			im.Set(x, y, c)
		}
	}
	return im, nil
}

// BuildBloomGraph wires one bloom chain. Node dependencies are ordered so that
// composite input 0 is the original scene and input 1 is bloom.
func BuildBloomGraph(width, height int) (*Graph, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid graph dimensions %dx%d", width, height)
	}
	source := NewSourceNode(func(_, _ int) (*Image, error) {
		return HDRScene(width, height)
	})
	threshold, err := NewThresholdNode(0.85, 0.20)
	if err != nil {
		return nil, err
	}
	blurH, err := NewBlurNode(BlurHorizontal, 16, 5.0)
	if err != nil {
		return nil, err
	}
	blurV, err := NewBlurNode(BlurVertical, 16, 5.0)
	if err != nil {
		return nil, err
	}

	g := NewGraph("composite")
	if err := g.AddNode("scene", source); err != nil {
		return nil, err
	}
	if err := g.AddNode("threshold", threshold, "scene"); err != nil {
		return nil, err
	}
	if err := g.AddNode("bloom-h", blurH, "threshold"); err != nil {
		return nil, err
	}
	if err := g.AddNode("bloom-v", blurV, "bloom-h"); err != nil {
		return nil, err
	}
	if err := g.AddNode("bloom-scale", NewScaleNode(0.55), "bloom-v"); err != nil {
		return nil, err
	}
	if err := g.AddNode("composite", NewCompositeNode(1.0), "scene", "bloom-scale"); err != nil {
		return nil, err
	}
	return g, nil
}
