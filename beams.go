package main

func DrawScreen(c *Canvas, x, y0, y1 float64) {
	a, b := v2(x, y0), v2(x, y1)
	c.LineWorld(a, b, LineStyle{Color: colInk, Width: 25, Alpha: 0.55})
	c.LineWorld(a, b, LineStyle{Color: colPaper, Width: 17, Alpha: 0.97})
	c.LineWorld(v2(x-0.09, y0), v2(x-0.09, y1), LineStyle{Color: hexLinear(0x8fa4c2), Width: 2, Alpha: 0.8})
}

func segmentInsideGlass(a, b Vec2, shapes []Shape2) bool {
	mid := a.Add(b).Mul(0.5)
	for _, s := range shapes {
		if s.Contains(mid) {
			return true
		}
	}
	return false
}

type BundleResult struct {
	Paths []BeamPath
}

func DrawBundle(
	c *Canvas,
	starts []Vec2,
	dir Vec2,
	lambdas []float64,
	shapes []Shape2,
	screenX float64,
	world Rect,
	dotRadius float64,
) BundleResult {
	clip := Rect{Min: world.Min, Max: v2(screenX+0.02, world.Max.Y)}
	result := BundleResult{}

	for _, start := range starts {
		midLambda := lambdas[len(lambdas)/2]
		whitePath := TraceBeam(start, dir, midLambda, shapes, clip)
		if len(whitePath.Points) >= 2 {
			c.LineWorld(start, whitePath.Points[1], LineStyle{Color: colInk, Width: 7.0, Alpha: 0.18})
			c.LineWorld(start, whitePath.Points[1], LineStyle{Color: v3(0.97, 0.995, 1), Width: 5.3, Alpha: 0.94})
			for i := 1; i < len(whitePath.Points)-1; i++ {
				c.LineWorld(whitePath.Points[i], whitePath.Points[i+1], LineStyle{Color: colInk, Width: 5.2, Alpha: 0.22})
			}
		}

		for _, lambda := range lambdas {
			path := TraceBeam(start, dir, lambda, shapes, clip)
			result.Paths = append(result.Paths, path)
			spectral := WavelengthLinearRGB(lambda)
			for i := 1; i < len(path.Points)-1; i++ {
				a := path.Points[i]
				b := path.Points[i+1]
				col := spectral
				width := 2.65
				if segmentInsideGlass(a, b, shapes) {
					distance := a.Sub(b).Len()
					col = col.Mul(BeerTransmission(lambda, distance*0.82))
					width = 2.35
				}
				c.BeamLineWorld(a, b, width, col, 0.82)
			}

			end := path.Points[len(path.Points)-1]
			if dotRadius > 0 && end.X >= screenX-0.08 && end.X <= screenX+0.08 {
				c.SoftDotWorld(v2(screenX, end.Y), dotRadius, spectral, 0.16)
				c.SoftDotWorld(v2(screenX, end.Y), dotRadius*0.34, spectral, 0.22)
			}
		}
	}
	return result
}

func horizontalStarts(x, centerY, aperture, count float64) []Vec2 {
	out := make([]Vec2, 0, int(count))
	for i := 0; i < int(count); i++ {
		t := 0.0
		if count > 1 {
			t = float64(i) / (count - 1)
		}
		y := centerY - aperture/2 + aperture*t
		out = append(out, v2(x, y))
	}
	return out
}
