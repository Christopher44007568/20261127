package main

import (
	"image"
	"math"
)

type ErrorStats struct {
	RMSE         float64
	MAE          float64
	MaxLumaError float64
	PSNR         float64
}

func compareImages(img, ref *image.RGBA, fps *FootprintCache) ErrorStats {
	if len(fps.Pixels) == 0 {
		return ErrorStats{}
	}
	var sumSq, sumAbs, maxErr float64
	for _, p := range fps.Pixels {
		a := getRGB(img, p.X, p.Y).Luma()
		b := getRGB(ref, p.X, p.Y).Luma()
		d := a - b
		sumSq += d * d
		sumAbs += math.Abs(d)
		if math.Abs(d) > maxErr {
			maxErr = math.Abs(d)
		}
	}
	n := float64(len(fps.Pixels))
	rmse := math.Sqrt(sumSq / n)
	psnr := 0.0
	if rmse > 1e-12 {
		psnr = 20 * math.Log10(1/rmse)
	}
	return ErrorStats{
		RMSE:         rmse,
		MAE:          sumAbs / n,
		MaxLumaError: maxErr,
		PSNR:         psnr,
	}
}

func addDirectionalMetrics(stats *RenderStats, img *image.RGBA, fps *FootprintCache, scene *Scene) {
	var longSum, crossSum float64
	var count int
	for _, p := range fps.Pixels {
		if p.X < 1 || p.X >= scene.Width-1 || p.Y < 1 || p.Y >= scene.Height-1 {
			continue
		}
		if _, ok := fps.At[(p.Y-1)*scene.Width+p.X]; !ok {
			continue
		}
		if _, ok := fps.At[(p.Y+1)*scene.Width+p.X]; !ok {
			continue
		}
		ratio := p.FP.Major / math.Max(p.FP.Minor, 1e-6)
		if ratio < 1.25 {
			continue
		}
		gx := (getRGB(img, p.X+1, p.Y).Luma() - getRGB(img, p.X-1, p.Y).Luma()) * 0.5
		gy := (getRGB(img, p.X, p.Y+1).Luma() - getRGB(img, p.X, p.Y-1).Luma()) * 0.5
		g := Vec2{gx, gy}
		majorScreen, okMajor := fpScreenDirection(p.FP, p.FP.AxisMajor)
		minorScreen, okMinor := fpScreenDirection(p.FP, p.FP.AxisMinor)
		if !okMajor || !okMinor {
			continue
		}
		longSum += math.Abs(g.Dot(majorScreen))
		crossSum += math.Abs(g.Dot(minorScreen))
		count++
	}
	if count == 0 {
		return
	}
	stats.LongAxisVariation = longSum / float64(count)
	stats.CrossAxisEnergy = crossSum / float64(count)
}

func fpScreenDirection(fp Footprint, texDir Vec2) (Vec2, bool) {
	det := fp.Dx.X*fp.Dy.Y - fp.Dx.Y*fp.Dy.X
	if math.Abs(det) < 1e-12 {
		return Vec2{}, false
	}
	// J = [Dx Dy] maps screen offsets to texture offsets.  The inverse
	// maps a texture-space footprint axis back to its screen direction.
	sx := (texDir.X*fp.Dy.Y - texDir.Y*fp.Dy.X) / det
	sy := (fp.Dx.X*texDir.Y - fp.Dx.Y*texDir.X) / det
	return Vec2{sx, sy}.Unit(), true
}
