package main

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"os"
)

func gifPalette() color.Palette {
	keys := []RGB{
		hexRGB(0xE6EAF0), colPaper, colPanel, colGrid, colInk, colSlope,
		colGround, colGroundDot, colHipL, colHipR, colTorso, colSkin,
		colFoot, colOK, colWarn, colAmber, colPurple, colGray, colLightGray,
	}
	pal := make(color.Palette, 0, 256)
	for _, k := range keys {
		pal = append(pal, color.RGBA{R: k.R, G: k.G, B: k.B, A: 255})
	}
	for _, c := range palette.Plan9 {
		if len(pal) >= 256 {
			break
		}
		pal = append(pal, c)
	}
	return pal
}

// WriteWalkGIF renders a smoothly looping walk-cycle GIF using only stdlib.
func WriteWalkGIF(path string, cfg GaitConfig, frames int) error {
	anim := &gif.GIF{LoopCount: 0}
	r := image.Rect(0, 0, panelW, panelH)
	pal := gifPalette()
	for i := 0; i < frames; i++ {
		phase := float64(i) / float64(frames)
		t := phase * cfg.CycleT
		cv := NewCanvas(panelW, panelH, hexRGB(0xE6EAF0))
		drawWalkPanelAt(cv, cfg, r, t, fmt.Sprintf("WALK  P=%.2f", phase))

		pm := image.NewPaletted(r, pal)
		draw.Draw(pm, r, cv.img, image.Point{}, draw.Src)
		anim.Image = append(anim.Image, pm)
		anim.Delay = append(anim.Delay, 4) // hundredths of a second -> 25 fps
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, anim)
}
