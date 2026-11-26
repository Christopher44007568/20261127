package postfx

import (
	"fmt"
	"math"
)

func runScene(ctx *PassContext) error {
	hdr := ctx.Output(ResHDR)
	depth := ctx.Output(ResDepth)
	if hdr == nil || depth == nil {
		return fmt.Errorf("scene outputs are disabled")
	}
	width, height := hdr.Width, hdr.Height
	horizon := 0.58 * float64(height-1)
	sunX := 0.72 * float64(width-1)
	sunY := 0.32 * float64(height-1)
	for y := 0; y < height; y++ {
		py := float64(y)
		for x := 0; x < width; x++ {
			px := float64(x)
			skyT := clamp(py/maxFloat(1, horizon), 0, 1)
			r := mix(0.025, 0.82, skyT)
			gr := mix(0.08, 0.32, skyT)
			b := mix(0.18, 0.26, skyT)
			dx, dy := px-sunX, py-sunY
			radius2 := dx*dx + dy*dy
			sun := 5.5 * math.Exp(-radius2/(2*18*18))
			r += sun
			gr += sun * 0.72
			b += sun * 0.34

			dep := 0.12 + 0.88*math.Abs(py-horizon)/maxFloat(1, horizon)
			if py >= horizon {
				groundT := clamp((py-horizon)/maxFloat(1, float64(height-1)-horizon), 0, 1)
				r = mix(0.042, 0.095, groundT)
				gr = mix(0.060, 0.120, groundT)
				b = mix(0.072, 0.145, groundT)
				gridX := math.Mod(px+float64(int(groundT*12))*23, 34)
				gridY := math.Mod(py+float64(int(px/34))*11, 22)
				if gridX < 1.2 || gridY < 1.2 {
					r, gr, b = r*2.5+0.02, gr*2.4+0.02, b*2.2+0.02
				}
				dep = 0.06 + 0.55*(1-groundT)
			}
			hdr.Set(x, y, r, gr, b)
			depth.Set(x, y, dep, dep, dep)
		}
	}
	return nil
}

func runBloomThreshold(ctx *PassContext) error {
	src := ctx.Input(ResHDR)
	dst := ctx.Output(ResBloomBright)
	if src == nil || dst == nil {
		return fmt.Errorf("bloom threshold inputs/outputs are disabled")
	}
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			r, g, b := src.At(x, y)
			lum := 0.2126*r + 0.7152*g + 0.0722*b
			excess := math.Max(0, lum-0.72) / 1.3
			if excess > 0 {
				dst.Set(x, y, r*excess, g*excess, b*excess)
			}
		}
	}
	return nil
}

func runBloomBlurH(ctx *PassContext) error {
	return blurHorizontal(ctx.Input(ResBloomBright), ctx.Output(ResBloomH), 14)
}

func runBloomBlurV(ctx *PassContext) error {
	return blurVertical(ctx.Input(ResBloomH), ctx.Output(ResBloom), 14)
}

func blurHorizontal(src, dst *Image, radius int) error {
	if src == nil || dst == nil {
		return fmt.Errorf("bloom blur images are disabled")
	}
	weights := gaussianWeights(radius, float64(radius)/3.0)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			r, g, b := 0.0, 0.0, 0.0
			for offset := -radius; offset <= radius; offset++ {
				weight := weights[offset+radius]
				er, eg, eb := src.At(clampInt(x+offset, 0, src.Width-1), y)
				r += er * weight
				g += eg * weight
				b += eb * weight
			}
			dst.Set(x, y, r, g, b)
		}
	}
	return nil
}

func blurVertical(src, dst *Image, radius int) error {
	if src == nil || dst == nil {
		return fmt.Errorf("bloom blur images are disabled")
	}
	weights := gaussianWeights(radius, float64(radius)/3.0)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			r, g, b := 0.0, 0.0, 0.0
			for offset := -radius; offset <= radius; offset++ {
				weight := weights[offset+radius]
				er, eg, eb := src.At(x, clampInt(y+offset, 0, src.Height-1))
				r += er * weight
				g += eg * weight
				b += eb * weight
			}
			dst.Set(x, y, r, g, b)
		}
	}
	return nil
}

func gaussianWeights(radius int, sigma float64) []float64 {
	weights := make([]float64, radius*2+1)
	total := 0.0
	for i := range weights {
		d := float64(i - radius)
		weight := math.Exp(-(d * d) / (2 * sigma * sigma))
		weights[i] = weight
		total += weight
	}
	for i := range weights {
		weights[i] /= total
	}
	return weights
}

func runFogDensity(ctx *PassContext) error {
	depth := ctx.Input(ResDepth)
	dst := ctx.Output(ResFogDensity)
	if depth == nil || dst == nil {
		return fmt.Errorf("fog density inputs/outputs are disabled")
	}
	width, height := dst.Width, dst.Height
	for y := 0; y < height; y++ {
		py := float64(y)
		horizon := 0.62 * float64(height-1)
		band := math.Exp(-math.Abs(py-horizon) / maxFloat(1, 0.08*float64(height)))
		for x := 0; x < width; x++ {
			_, _, rawDepth := depth.At(x, y)
			distance := clamp(rawDepth, 0, 1)
			wisp := 0.5 + 0.5*math.Sin(float64(x)*0.075+float64(y)*0.043)
			density := clamp(0.10*distance+0.34*band*wisp+0.04, 0, 0.55)
			dst.Set(x, y, density, density, density)
		}
	}
	return nil
}

func runFogScatter(ctx *PassContext) error {
	density := ctx.Input(ResFogDensity)
	depth := ctx.Input(ResDepth)
	fog := ctx.Output(ResFog)
	debug := ctx.Output(ResFogDebug)
	if density == nil || depth == nil || (fog == nil && debug == nil) {
		return fmt.Errorf("fog scatter inputs/outputs are disabled")
	}
	width, height := density.Width, density.Height
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			d, _, _ := density.At(x, y)
			_, _, z := depth.At(x, y)
			shaft := 0.55 + 0.45*math.Sin((float64(x)+float64(y)*2)*0.018)
			amount := d * (0.38 + 0.62*clamp(1-z, 0, 1))
			if fog != nil {
				fog.Set(x, y,
					0.12*amount+0.32*shaft*amount,
					0.19*amount+0.46*shaft*amount,
					0.27*amount+0.62*shaft*amount)
			}
			if debug != nil {
				debug.Set(x, y, amount*1.8, amount*0.6, amount*0.18)
			}
		}
	}
	return nil
}

func runUI(ctx *PassContext) error {
	ui := ctx.Output(ResUI)
	debug := ctx.Output(ResUIDebug)
	if ui == nil && debug == nil {
		return fmt.Errorf("UI outputs are disabled")
	}
	if ui == nil {
		for i := range debug.Pix {
			debug.Pix[i] = 0.5
		}
		return nil
	}
	width, height := ui.Width, ui.Height
	drawRect(ui, 14, 14, minInt(width/3, 190), 9, 0.16, 0.84, 1.0)
	drawRect(ui, 14, 28, minInt(width/4, 132), 4, 0.08, 0.42, 0.55)
	drawRect(ui, 14, 38, minInt(width/5, 98), 3, 0.05, 0.24, 0.34)

	panelW, panelH := minInt(176, width/3), minInt(92, height/3)
	panelX, panelY := width-panelW-16, 16
	drawRect(ui, panelX, panelY, panelW, panelH, 0.025, 0.075, 0.11)
	for i := 0; i < 5; i++ {
		drawRect(ui, panelX+10, panelY+12+i*14, panelW-20, 5, 0.10, 0.28, 0.40)
	}
	drawRect(ui, panelX+10, panelY+12, panelW/2-14, 5, 0.30, 0.94, 0.76)

	cx, cy := width/2, height/2
	drawRect(ui, cx-18, cy, 36, 2, 0.78, 0.97, 1.0)
	drawRect(ui, cx, cy-18, 2, 36, 0.78, 0.97, 1.0)
	drawCircle(ui, cx, cy, 10, 0.14, 0.68, 0.82)
	drawCircle(ui, cx, cy, 5, 0.90, 0.98, 1.0)

	drawRect(ui, 16, height-26, minInt(220, width/2), 12, 0.035, 0.11, 0.16)
	drawRect(ui, 20, height-22, minInt(170, width*2/5), 4, 0.24, 0.82, 0.95)
	drawRect(ui, width-20, 16, 2, height-32, 0.10, 0.34, 0.44)
	if debug != nil {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r, g, b := ui.At(x, y)
				lum := 0.2126*r + 0.7152*g + 0.0722*b
				debug.Set(x, y, lum, lum*0.55, lum*0.12)
			}
		}
	}
	return nil
}

func runBeauty(ctx *PassContext) error {
	hdr := ctx.Input(ResHDR)
	bloom := ctx.Input(ResBloom)
	density := ctx.Input(ResFogDensity)
	fog := ctx.Input(ResFog)
	ui := ctx.Input(ResUI)
	dst := ctx.Output(ResBeauty)
	if hdr == nil || bloom == nil || density == nil || fog == nil || ui == nil || dst == nil {
		return fmt.Errorf("beauty inputs/outputs are disabled")
	}
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			r, g, b := hdr.At(x, y)
			br, bg, bb := bloom.At(x, y)
			dr, _, _ := density.At(x, y)
			fr, fg, fb := fog.At(x, y)
			fogMix := clamp(0.18+dr*0.92, 0, 0.62)
			r = mix(r, fr, fogMix) + br*0.58
			g = mix(g, fg, fogMix) + bg*0.58
			b = mix(b, fb, fogMix) + bb*0.58
			ur, ug, ub := ui.At(x, y)
			r = r*(1-0.80*max3(ur, ug, ub)) + ur*0.94
			g = g*(1-0.80*max3(ur, ug, ub)) + ug*0.94
			b = b*(1-0.80*max3(ur, ug, ub)) + ub*0.94
			dst.Set(x, y, r, g, b)
		}
	}
	return nil
}

func runTonemap(ctx *PassContext) error {
	src := ctx.Input(ResBeauty)
	dst := ctx.Output(ResFinal)
	if src == nil || dst == nil {
		return fmt.Errorf("tonemap inputs/outputs are disabled")
	}
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			r, g, b := src.At(x, y)
			dst.Set(x, y, aces(r), aces(g), aces(b))
		}
	}
	return nil
}

func runBloomPreview(ctx *PassContext) error {
	return amplifyCopy(ctx.Input(ResBloom), ctx.Output(ResBloomPreview), 3.2,
		0.10, 0.24, 0.48)
}

func runFogPreview(ctx *PassContext) error {
	density := ctx.Input(ResFogDensity)
	fog := ctx.Input(ResFog)
	dst := ctx.Output(ResFogPreview)
	if density == nil || fog == nil || dst == nil {
		return fmt.Errorf("fog preview inputs/outputs are disabled")
	}
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			d, _, _ := density.At(x, y)
			r, g, b := fog.At(x, y)
			dst.Set(x, y,
				0.025+d*0.24+r*2.2,
				0.055+d*0.34+g*2.2,
				0.080+d*0.44+b*2.2)
		}
	}
	return nil
}

func runUIPreview(ctx *PassContext) error {
	return amplifyCopy(ctx.Input(ResUI), ctx.Output(ResUIPreview), 1.0,
		0.012, 0.028, 0.038)
}

func amplifyCopy(src, dst *Image, gain float64, bgR, bgG, bgB float64) error {
	if src == nil || dst == nil {
		return fmt.Errorf("preview inputs/outputs are disabled")
	}
	for y := 0; y < dst.Height; y++ {
		for x := 0; x < dst.Width; x++ {
			r, g, b := src.At(x, y)
			dst.Set(x, y, bgR+r*gain, bgG+g*gain, bgB+b*gain)
		}
	}
	return nil
}
