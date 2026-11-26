package main

import (
	"image"
	"image/color"
)

// BoardModel feeds the structural diff contact board.
type BoardModel struct {
	RefImg      *image.RGBA
	OwnImg      *image.RGBA
	HeatImg     *image.RGBA
	IDImg       *image.RGBA
	ErrImg      *image.RGBA
	Rep         *DiffReport
	Reload      *SVGAudit
	ReloadDelta float64
	Bad         *SVGAudit
	Checks      []CheckResult
}

// CheckResult is one semantic/structural assertion for the report and board.
type CheckResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

const boardW = 1080

var (
	boardBG    = color.RGBA{11, 18, 32, 255}
	boardPanel = color.RGBA{17, 27, 48, 255}
	inkMain    = color.RGBA{226, 232, 240, 255}
	inkMuted   = color.RGBA{148, 163, 184, 255}
	inkAccent  = color.RGBA{125, 211, 252, 255}
	inkOK      = color.RGBA{52, 211, 153, 255}
	inkWarn    = color.RGBA{251, 191, 36, 255}
	inkBad     = color.RGBA{251, 113, 133, 255}
)

type board struct {
	img *image.RGBA
	y   int
}

// BuildDiffBoard renders diff.png: references, heatmap, ID map, the annotated
// error layer, per-object structural table and reload checks.
func BuildDiffBoard(m *BoardModel) *image.RGBA {
	b := &board{img: newCanvas(boardW, 4200, [3]uint8{11, 18, 32}), y: 0}

	fillRect(b.img, 0, 0, boardW, 92, color.RGBA{15, 23, 42, 255})
	fillRect(b.img, 0, 90, boardW, 2, color.RGBA{56, 189, 248, 255})
	DrawText(b.img, 40, 22, "VECTOR SVG -> RASTER STRUCTURAL DIFF", 3, color.RGBA{240, 249, 255, 255})
	sub := "browser reference | self PNG | heatmap | error layer with original object IDs"
	DrawText(b.img, 42, 64, sub, 2, inkMuted)
	b.y = 112

	b.twoImages(
		"A  BROWSER REFERENCE (HEADLESS CHROME)", m.RefImg,
		"B  SELF PNG (PURE GO RASTERIZER)", m.OwnImg)
	b.twoImages(
		"C  DELTA HEATMAP (GAIN x3)", m.HeatImg,
		"ID SEGMENTATION MAP (EDITABLE OBJECTS)", m.IDImg)

	b.sectionTitle("D  ERROR LAYER - ORIGINAL OBJECT IDS MARKED", inkBad)
	b.paste(m.ErrImg, (boardW-960)/2, b.y)
	strokeRect(b.img, (boardW-960)/2-1, b.y-1, 962, 602, 1, color.RGBA{71, 85, 105, 255})
	b.y += 600 + 34

	b.objectTable(m.Rep)
	b.reloadSection(m)

	return b.crop()
}

func (b *board) sectionTitle(s string, c color.RGBA) {
	DrawText(b.img, 40, b.y, s, 2, c)
	b.y += 30
}

func (b *board) twoImages(t1 string, i1 *image.RGBA, t2 string, i2 *image.RGBA) {
	b.sectionTitle(t1, inkAccent)
	DrawText(b.img, 560, b.y-30, t2, 2, inkAccent)
	thumb1 := resizeBilinear(i1, 480, 300)
	thumb2 := resizeBilinear(i2, 480, 300)
	b.paste(thumb1, 40, b.y)
	b.paste(thumb2, 560, b.y)
	strokeRect(b.img, 39, b.y-1, 482, 302, 1, color.RGBA{51, 65, 85, 255})
	strokeRect(b.img, 559, b.y-1, 482, 302, 1, color.RGBA{51, 65, 85, 255})
	b.y += 300 + 30
}

func (b *board) paste(src *image.RGBA, x, y int) {
	if src == nil {
		return
	}
	for sy := 0; sy < src.Bounds().Dy(); sy++ {
		for sx := 0; sx < src.Bounds().Dx(); sx++ {
			sp := src.PixOffset(sx, sy)
			dp := b.img.PixOffset(x+sx, y+sy)
			b.img.Pix[dp] = src.Pix[sp]
			b.img.Pix[dp+1] = src.Pix[sp+1]
			b.img.Pix[dp+2] = src.Pix[sp+2]
			b.img.Pix[dp+3] = 255
		}
	}
}

func (b *board) crop() *image.RGBA {
	h := b.y + 48
	out := image.NewRGBA(image.Rect(0, 0, boardW, h))
	for y := 0; y < h; y++ {
		for x := 0; x < boardW*4; x++ {
			out.Pix[y*out.Stride+x] = b.img.Pix[y*b.img.Stride+x]
		}
	}
	return out
}

// SaveTitledImage writes an image with a title strip around it.
func SaveTitledImage(path, title string, src *image.RGBA, titleColor color.RGBA) error {
	pad := 24
	head := 52
	W := src.Bounds().Dx() + pad*2
	H := src.Bounds().Dy() + pad*2 + head
	cv := newCanvas(W, H, [3]uint8{11, 18, 32})
	DrawText(cv, pad, 20, title, 2, titleColor)
	y := head + pad - 12
	for sy := 0; sy < src.Bounds().Dy(); sy++ {
		sp := sy * src.Stride
		dp0 := cv.PixOffset(pad, y+sy)
		copy(cv.Pix[dp0:dp0+src.Bounds().Dx()*4], src.Pix[sp:sp+src.Bounds().Dx()*4])
	}
	strokeRect(cv, pad-1, y-1, src.Bounds().Dx()+2, src.Bounds().Dy()+2, 1,
		color.RGBA{51, 65, 85, 255})
	return savePNG(path, cv)
}
