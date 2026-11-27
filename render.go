package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	imageW = 1440
	imageH = 860
	panelY = 76
	panelH = 650
	panelW = 684
	leftX  = 20
	rightX = 736
)

type Rect struct {
	X, Y, W, H int
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

type Camera struct {
	Eye    Vec3
	Target Vec3
	F      Vec3
	R      Vec3
	U      Vec3
	Focal  float64
	CX     float64
	CY     float64
}

type Projected struct {
	X, Y float64
	Z    float64
	W    float64
}

func NewCamera(rect Rect) Camera {
	eye := V(2.35, 1.55, 2.65).Normalize().Mul(3.35)
	target := V(0, -0.015, 0)
	f := target.Sub(eye).Normalize()
	worldUp := V(0, 1, 0)
	r := f.Cross(worldUp).Normalize()
	u := r.Cross(f).Normalize()
	fovy := 35 * math.Pi / 180
	focal := float64(rect.H) / (2 * math.Tan(fovy/2))
	return Camera{
		Eye: eye, Target: target, F: f, R: r, U: u,
		Focal: focal,
		CX:    float64(rect.X) + float64(rect.W)/2,
		CY:    float64(rect.Y) + float64(rect.H)/2,
	}
}

func (c Camera) Project(p Vec3) (Projected, bool) {
	rel := p.Sub(c.Eye)
	z := rel.Dot(c.F)
	if z <= 0.02 {
		return Projected{}, false
	}
	x := rel.Dot(c.R)
	y := rel.Dot(c.U)
	return Projected{
		X: c.CX + c.Focal*x/z,
		Y: c.CY - c.Focal*y/z,
		Z: z,
		W: 1 / z,
	}, true
}

func RenderAll(outDir string, results []*SceneResult) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, res := range results {
		img := image.NewRGBA(image.Rect(0, 0, imageW, imageH))
		fillImage(img, color.RGBA{10, 13, 19, 255})

		left := Rect{X: leftX, Y: panelY, W: panelW, H: panelH}
		right := Rect{X: rightX, Y: panelY, W: panelW, H: panelH}
		drawPanelChrome(img, left, "BASELINE: MASS-POINT / LINEAR SPLIT", "rounded sharp features", color.RGBA{58, 38, 34, 255})
		drawPanelChrome(img, right, "NORMAL-QEF + ADAPTIVE CRACK STITCH", "edge-aware quad split", color.RGBA{24, 54, 52, 255})

		lv := viewport(left)
		cam := NewCamera(lv)
		ldepth := make([]float64, lv.W*lv.H)
		renderMesh(img, lv, cam, res.BaselineMesh, ldepth, true)
		drawReferenceFeatures(img, lv, cam, res, ldepth, false)
		drawZoomInset(img, left, res, res.BaselineMesh, true)
		drawPanelHUD(img, left, res, true)

		rv := viewport(right)
		rcam := NewCamera(rv)
		rdepth := make([]float64, rv.W*rv.H)
		renderMesh(img, rv, rcam, res.SharpMesh, rdepth, false)
		drawReferenceFeatures(img, rv, rcam, res, rdepth, false)
		drawZoomInset(img, right, res, res.SharpMesh, false)
		drawMarkers(img, rv, rcam, res, rdepth)
		drawPanelHUD(img, right, res, false)

		drawGlobalHeader(img, res)
		drawFooter(img, res)

		path := filepath.Join(outDir, res.Scene.ID+".png")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func fillImage(img *image.RGBA, c color.Color) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, c)
		}
	}
}

func drawPanelChrome(img *image.RGBA, r Rect, title, subtitle string, accent color.RGBA) {
	fillRect(img, r, color.RGBA{17, 22, 31, 255})
	strokeRect(img, r, color.RGBA{54, 65, 82, 255})
	fillRect(img, Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: 34}, accent)
	drawText(img, r.X+14, r.Y+11, title, 2, color.RGBA{235, 241, 250, 255})
	drawText(img, r.X+14, r.Y+42, subtitle, 1, color.RGBA{157, 170, 189, 255})
}

func fillRect(img *image.RGBA, r Rect, c color.RGBA) {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func strokeRect(img *image.RGBA, r Rect, c color.RGBA) {
	for x := r.X; x < r.X+r.W; x++ {
		img.SetRGBA(x, r.Y, c)
		img.SetRGBA(x, r.Y+r.H-1, c)
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		img.SetRGBA(r.X, y, c)
		img.SetRGBA(r.X+r.W-1, y, c)
	}
}
