package main

import (
	"io"
	"math"
)

var (
	colBg    = RGBA{245, 245, 248, 255}
	colInk   = RGBA{30, 30, 40, 255}
	colGrid  = RGBA{210, 210, 218, 255}
	colLeft  = RGBA{200, 60, 60, 255}
	colRight = RGBA{40, 90, 200, 255}
	colZero  = RGBA{20, 160, 90, 255}
)

// heatColor maps t in [0,1] from deep blue through cyan/yellow to red.
func heatColor(t float64) RGBA {
	t = clamp(t, 0, 1)
	stops := []struct {
		at  float64
		col RGBA
	}{
		{0.0, RGBA{24, 40, 120, 255}},
		{0.25, RGBA{30, 140, 200, 255}},
		{0.5, RGBA{40, 180, 120, 255}},
		{0.75, RGBA{235, 200, 60, 255}},
		{1.0, RGBA{210, 40, 40, 255}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			a, b := stops[i-1], stops[i]
			u := (t - a.at) / (b.at - a.at)
			return RGBA{
				uint8(lerp(float64(a.col[0]), float64(b.col[0]), u)),
				uint8(lerp(float64(a.col[1]), float64(b.col[1]), u)),
				uint8(lerp(float64(a.col[2]), float64(b.col[2]), u)),
				255,
			}
		}
	}
	return stops[len(stops)-1].col
}

// RenderHeatmap draws frame index on X and joints on Y, colored by
// world-space joint position error in millimetres.
func RenderHeatmap(w io.Writer, sk *Skeleton, heat HeatMatrix) error {
	const (
		leftW   = 110
		topH    = 40
		cellW   = 6
		cellH   = 20
		rightW  = 100
		bottomH = 36
	)
	frames := len(heat.Values)
	W := leftW + frames*cellW + rightW
	H := topH + JointCount*cellH + bottomH
	c := NewCanvas(W, H, colBg)

	var maxErr float64
	for _, row := range heat.Values {
		for _, v := range row {
			if v > maxErr {
				maxErr = v
			}
		}
	}

	c.Text(8, 8, "JOINT POSITION ERROR HEATMAP (MM)", colInk)
	gridX := leftW
	gridY := topH
	for j := 0; j < JointCount; j++ {
		c.Text(4, gridY+j*cellH+7, shortJointName(j), colInk)
	}
	for f := 0; f < frames; f++ {
		for j := 0; j < JointCount; j++ {
			c.FillRect(gridX+f*cellW, gridY+j*cellH, cellW, cellH,
				heatColor(heat.Values[f][j]/math.Max(maxErr, 1e-9)))
		}
	}

	// Color legend on the right, labelled 0..max in millimetres.
	legendX := gridX + frames*cellW + 12
	for y := 0; y < JointCount*cellH; y++ {
		t := 1 - float64(y)/float64(JointCount*cellH)
		c.FillRect(legendX, gridY+y, 18, 1, heatColor(t))
	}
	mm := maxErr * 1000
	c.Text(legendX+24, gridY, formatTick(mm), colInk)
	c.Text(legendX+24, gridY+JointCount*cellH-8, "0.0", colInk)

	// X axis ticks every 0.5 s assuming 60 fps.
	for sec := 0.0; sec <= 2.2; sec += 0.5 {
		f := int(sec * synthFPS)
		if f >= frames {
			continue
		}
		x := gridX + f*cellW
		c.Line(x, gridY, x, gridY+JointCount*cellH, colGrid)
		c.Text(x-6, gridY+JointCount*cellH+10, formatTick(sec), colInk)
	}
	return c.EncodePNG(w)
}

// RenderSlip draws contact-relative ankle drift for both feet over time;
// swing phases are left as background gaps.
func RenderSlip(w io.Writer, slipOrig, slipComp SlipCurve, fps float64) error {
	const (
		W, H    = 1100, 460
		leftW   = 70
		topH    = 50
		rightW  = 30
		bottomH = 60
	)
	c := NewCanvas(W, H, colBg)
	c.Text(8, 10, "FOOT SLIP DURING CONTACT (MM)", colInk)

	plotX := leftW
	plotY := topH
	plotW := W - leftW - rightW
	plotH := H - topH - bottomH

	var maxDrift float64
	for _, v := range append(append([]float64{}, slipComp.Left...), slipComp.Right...) {
		if !math.IsNaN(v) && v > maxDrift {
			maxDrift = v
		}
	}
	maxDrift = math.Max(maxDrift*1.1, 0.005)

	// Grid + y labels at quartiles.
	for g := 0; g <= 4; g++ {
		y := plotY + plotH*g/4
		c.Line(plotX, y, plotX+plotW, y, colGrid)
		val := maxDrift * (1 - float64(g)/4) * 1000
		c.Text(8, y-3, formatTick(val), colInk)
	}

	n := len(slipComp.Left)
	draw := func(vals []float64, col RGBA) {
		var prevX, prevY int
		drawing := false
		for f := 0; f < n; f++ {
			x := plotX + plotW*f/(n-1)
			if math.IsNaN(vals[f]) {
				drawing = false
				continue
			}
			y := plotY + plotH - int(vals[f]/maxDrift*float64(plotH))
			if drawing {
				c.Line(prevX, prevY, x, y, col)
			}
			prevX, prevY, drawing = x, y, true
		}
	}
	draw(slipOrig.Left, colZero)
	draw(slipOrig.Right, colZero)
	draw(slipComp.Left, colLeft)
	draw(slipComp.Right, colRight)

	duration := float64(n) / fps
	for sec := 0.0; sec <= duration+1e-9; sec += 0.5 {
		f := int(sec * fps)
		if f >= n {
			continue
		}
		x := plotX + plotW*f/(n-1)
		c.Text(x-6, plotY+plotH+12, formatTick(sec), colInk)
	}
	c.Text(plotX, H-18, "TIME (S)", colInk)
	c.Text(plotX+120, H-18, "RED L  BLUE R  GREEN ORIG (ZERO)", colInk)
	return c.EncodePNG(w)
}
