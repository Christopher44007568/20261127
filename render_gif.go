package main

import (
	"image"
	"image/color"
	"image/gif"
	"io"
)

var gifPalette = color.Palette{
	color.RGBA{245, 245, 248, 255}, // 0 background
	color.RGBA{30, 30, 40, 255},    // 1 ink / original
	color.RGBA{200, 60, 60, 255},   // 2 compressed
	color.RGBA{180, 180, 190, 255}, // 3 grid
	color.RGBA{20, 160, 90, 255},   // 4 contact marker
}

const (
	gifBlack = 1
	gifRed   = 2
	gifGrid  = 3
	gifGreen = 4
)

type projector struct {
	originX, originY int
	scale            float64
	axisA, axisB     int // world components mapped to canvas x/y
}

func (p projector) project(v Vec3) (int, int) {
	comp := [3]float64{v.X, v.Y, v.Z}
	x := p.originX + int(comp[p.axisA]*p.scale)
	y := p.originY - int(comp[p.axisB]*p.scale)
	return x, y
}

// RenderOverlayGIF draws original vs compressed skeletons in side and top
// projections, sampled at 20 fps so the file stays compact.
func RenderOverlayGIF(w io.Writer, sk *Skeleton, orig, comp *Clip) error {
	worldOrig := sk.WorldClip(orig)
	worldComp := sk.WorldClip(comp)

	const (
		W, H    = 900, 420
		panelH  = 340
		delayCs = 5
		outFPS  = 20
	)
	step := int(synthFPS / outFPS)
	var frames imageList

	scale := 150.0
	side := projector{originX: 160, originY: 300, scale: scale, axisA: 2, axisB: 1}
	top := projector{originX: 600, originY: 220, scale: scale, axisA: 0, axisB: 2}

	for f := 0; f < len(worldOrig); f += step {
		img := image.NewPaletted(image.Rect(0, 0, W, H), gifPalette)
		drawPanelGrid(img, 30, 40, 380, panelH)
		drawPanelGrid(img, 470, 40, 380, panelH)
		setText(img, 40, 18, "SIDE: Z FORWARD, Y UP", gifBlack)
		setText(img, 480, 18, "TOP: X RIGHT, Z FORWARD", gifBlack)

		drawSkeleton(img, sk, worldOrig[f], side, gifBlack, false)
		drawSkeleton(img, sk, worldComp[f], side, gifRed, orig.Frames[f].Key)
		drawSkeleton(img, sk, worldOrig[f], top, gifBlack, false)
		drawSkeleton(img, sk, worldComp[f], top, gifRed, orig.Frames[f].Key)

		frames = append(frames, gifFrame{img: img, delay: delayCs})
	}

	out := &gif.GIF{
		Image: make([]*image.Paletted, len(frames)),
		Delay: make([]int, len(frames)),
	}
	for i, fr := range frames {
		out.Image[i] = fr.img
		out.Delay[i] = fr.delay
	}
	return gif.EncodeAll(w, out)
}

type gifFrame struct {
	img   *image.Paletted
	delay int
}

type imageList []gifFrame

func drawPanelGrid(img *image.Paletted, x, y, w, h int) {
	for gx := x; gx <= x+w; gx += 40 {
		drawLineImg(img, gx, y, gx, y+h, gifGrid)
	}
	for gy := y; gy <= y+h; gy += 40 {
		drawLineImg(img, x, gy, x+w, gy, gifGrid)
	}
}

func drawSkeleton(img *image.Paletted, sk *Skeleton, pose WorldPose, p projector, col uint8, key bool) {
	for j, joint := range sk.Joints {
		if joint.Parent < 0 {
			continue
		}
		x0, y0 := p.project(pose.Positions[joint.Parent])
		x1, y1 := p.project(pose.Positions[j])
		drawLineImg(img, x0, y0, x1, y1, col)
	}
	for j := range sk.Joints {
		x, y := p.project(pose.Positions[j])
		put(img, x, y, col)
		put(img, x+1, y, col)
		put(img, x, y+1, col)
		put(img, x+1, y+1, col)
	}
	if key {
		x, y := p.project(pose.Positions[JRoot])
		for r := 0; r < 6; r++ {
			put(img, x+r, y-8, gifGreen)
		}
	}
}

func drawLineImg(img *image.Paletted, x0, y0, x1, y1 int, col uint8) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		put(img, x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func put(img *image.Paletted, x, y int, col uint8) {
	b := img.Bounds()
	if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
		return
	}
	img.SetColorIndex(x, y, col)
}

func setText(img *image.Paletted, x, y int, s string, col uint8) {
	for i := 0; i < len(s); i++ {
		glyph, ok := font5x7[s[i]]
		if !ok {
			glyph = font5x7[' ']
		}
		for gy := 0; gy < 7; gy++ {
			for gx := 0; gx < 5; gx++ {
				if glyph[gy][gx] == '1' {
					put(img, x+i*6+gx, y+gy, col)
				}
			}
		}
	}
}
