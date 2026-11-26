package demo

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
)

// Run generates every artifact into outDir and returns the verification report.
func Run(outDir string) (*Report, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	pal := DefaultPalette()
	rep := &Report{OK: true}
	masks := BuildAllMasks()

	// ---- standalone grayscale fill masks ----
	for name, m := range masks {
		file := name + ".png"
		if err := WriteMaskPNG(filepath.Join(outDir, file), m); err != nil {
			return nil, err
		}
		rep.Outputs = append(rep.Outputs, file)
	}

	// ---- color scene renderings (one unified mask paint per scene) ----
	paperNested := NewPaper(SceneW, SceneH, pal)
	sceneNestedEO := cloneNRGBA(paperNested)
	PaintMask(sceneNestedEO, masks["mask_nested_evenodd"], pal.NestedEO)
	paintIsland(sceneNestedEO, pal.Island, nil)

	sceneNestedNZ := cloneNRGBA(paperNested)
	PaintMask(sceneNestedNZ, masks["mask_nested_nonzero"], pal.NestedNZ)
	paintIsland(sceneNestedNZ, pal.Island, nil)

	// Hole-compositing proof uses the pristine paper vs painted NZ scene.
	VerifyHoleCompositing(paperNested, sceneNestedNZ, masks["mask_nested_nonzero"], rep)

	sceneStarEO := NewPaper(SceneW, SceneH, pal)
	PaintMask(sceneStarEO, masks["mask_star_evenodd"], pal.EvenOdd)

	sceneStarNZ := NewPaper(SceneW, SceneH, pal)
	PaintMask(sceneStarNZ, masks["mask_star_nonzero"], pal.NonZero)

	sceneClipRect := NewPaper(SceneW, SceneH, pal)
	PaintMask(sceneClipRect, masks["mask_clipped_rect_nonzero"], pal.NestedNZ)
	clipIsland := islandMask()
	ClipRect(clipIsland, ClipRectNested)
	PaintMask(sceneClipRect, clipIsland, pal.Island)

	sceneClipCircle := NewPaper(SceneW, SceneH, pal)
	PaintMask(sceneClipCircle, masks["mask_clipped_circle_evenodd"], pal.EvenOdd)

	// ---- verification ----
	VerifyMasks(masks, rep)
	beforeClip := NewMask(SceneW, SceneH)
	FillPath(beforeClip, NestedPath(), FillNonZero)
	VerifyClipSeams(beforeClip, masks["mask_clipped_rect_nonzero"], rep)
	VerifySnap(rep)

	// ---- main composite contact sheet ----
	sheet := buildCompositeSheet(pal, []panelSpec{
		{title: "1 STAR - EVENODD", scene: sceneStarEO},
		{title: "2 STAR - NONZERO", scene: sceneStarNZ},
		{title: "3 NESTED - EVENODD", scene: sceneNestedEO},
		{title: "4 NESTED - NONZERO", scene: sceneNestedNZ},
		{title: "5 RECT CLIP - NESTED NZ", scene: sceneClipRect, markRect: &ClipRectNested},
		{title: "6 CIRCLE CLIP - STAR EO", scene: sceneClipCircle, markCircle: true},
	}, rep.OK)
	if err := WritePNG(filepath.Join(outDir, "composite.png"), sheet); err != nil {
		return nil, err
	}
	rep.Outputs = append(rep.Outputs, "composite.png")

	// ---- individual boundary zooms ----
	zoomJobs := buildZoomJobs(pal, masks, sceneNestedNZ, sceneClipRect)
	for _, z := range zoomJobs {
		img := ZoomImage(z.src, z.opt)
		file := z.name + ".png"
		if err := WritePNG(filepath.Join(outDir, file), img); err != nil {
			return nil, err
		}
		rep.Outputs = append(rep.Outputs, file)
	}

	// ---- zoom contact sheet ----
	zoomSheet := buildZoomSheet(pal, zoomJobs)
	if err := WritePNG(filepath.Join(outDir, "boundary_zooms.png"), zoomSheet); err != nil {
		return nil, err
	}
	rep.Outputs = append(rep.Outputs, "boundary_zooms.png")

	rep.Outputs = append(rep.Outputs, "report.json")
	if err := WriteReport(filepath.Join(outDir, "report.json"), rep); err != nil {
		return nil, err
	}
	return rep, nil
}

func islandMask() *Mask {
	m := NewMask(SceneW, SceneH)
	p := &Path{}
	AddDiamond(p, 180, 150, 36, 36, WindingClockwise)
	FillPath(m, p, FillNonZero)
	return m
}

func paintIsland(dst *image.NRGBA, c color.NRGBA, clip *image.Rectangle) {
	m := islandMask()
	if clip != nil {
		ClipRect(m, *clip)
	}
	PaintMask(dst, m, c)
}

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Rect)
	copy(dst.Pix, src.Pix)
	dst.Stride = src.Stride
	return dst
}

type panelSpec struct {
	title      string
	scene      *image.NRGBA
	markRect   *image.Rectangle
	markCircle bool
}

const (
	sheetMargin = 28
	sheetGapX   = 22
	sheetGapY   = 24
	panelLabelH = 30
	sheetHeader = 64
	sheetFooter = 34
)

func buildCompositeSheet(pal Palette, panels []panelSpec, allOK bool) *image.NRGBA {
	cardW := SceneW + 2
	cardH := panelLabelH + SceneH + 1
	cols := 3
	rows := (len(panels) + cols - 1) / cols
	w := sheetMargin*2 + cols*cardW + (cols-1)*sheetGapX
	h := sheetHeader + rows*cardH + (rows-1)*sheetGapY + sheetFooter + sheetMargin
	sheet := image.NewNRGBA(image.Rect(0, 0, w, h))
	fillRect(sheet, sheet.Bounds(), color.NRGBA{238, 241, 247, 255})

	DrawText(sheet, "FILL RULES: EVEN-ODD vs NON-ZERO", sheetMargin, 16, 3, pal.Ink)
	sub := "SAME SELF-INTERSECTING PATH - CLIP REGIONS - NESTED SUBPATHS"
	DrawText(sheet, sub, sheetMargin, 48, 2, color.NRGBA{74, 82, 100, 255})

	status := "ALL PIXEL CHECKS PASSED"
	statusC := pal.Good
	if !allOK {
		status = "CHECKS FAILED - SEE report.json"
		statusC = pal.Bad
	}
	tw := TextWidth(status, 2)
	DrawText(sheet, status, w-sheetMargin-tw, 22, 2, statusC)

	for i, pc := range panels {
		col := i % cols
		row := i / cols
		x := sheetMargin + col*(cardW+sheetGapX)
		y := sheetHeader + row*(cardH+sheetGapY)
		r := image.Rect(x, y, x+cardW, y+cardH)
		fillRect(sheet, r, color.NRGBA{255, 255, 255, 255})
		RectOutline(sheet, r, color.NRGBA{190, 198, 214, 255})
		labelR := image.Rect(x+1, y+1, x+cardW-1, y+panelLabelH)
		fillRect(sheet, labelR, pal.PanelHead)
		DrawText(sheet, pc.title, x+10, y+8, 2, pal.Ink)

		sx, sy := x+1, y+panelLabelH
		draw.Draw(sheet, image.Rect(sx, sy, sx+SceneW, sy+SceneH), pc.scene,
			image.Pt(0, 0), draw.Src)
		RectOutline(sheet, image.Rect(sx, sy, sx+SceneW, sy+SceneH),
			color.NRGBA{170, 178, 196, 255})

		if pc.markRect != nil {
			rr := *pc.markRect
			shifted := image.Rect(rr.Min.X+sx, rr.Min.Y+sy, rr.Max.X+sx, rr.Max.Y+sy)
			RectOutline(sheet, shifted, pal.ClipMark)
		}
		if pc.markCircle {
			drawCircleOutline(sheet, sx+180, sy+152, 106, pal.ClipMark)
		}
	}

	foot := "MASKS ARE BINARY 0/255 - HOLES STAY PAPER - CLIP EDGES KEEP WHOLE PIXELS"
	DrawText(sheet, foot, sheetMargin, h-26, 2, color.NRGBA{90, 98, 118, 255})
	return sheet
}

func fillRect(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	r = r.Intersect(dst.Bounds())
	for y := r.Min.Y; y < r.Max.Y; y++ {
		i0 := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			i := i0 + (x-r.Min.X)*4
			dst.Pix[i+0] = c.R
			dst.Pix[i+1] = c.G
			dst.Pix[i+2] = c.B
			dst.Pix[i+3] = c.A
		}
	}
}

func drawCircleOutline(dst *image.NRGBA, cx, cy, rad int, c color.NRGBA) {
	steps := rad * 8
	for i := 0; i <= steps; i++ {
		a := float64(i) / float64(steps) * 2 * math.Pi
		x := cx + int(float64(rad)*math.Cos(a)+0.5)
		y := cy + int(float64(rad)*math.Sin(a)+0.5)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx*dx+dy*dy <= 2 {
					blendPixel(dst, x+dx, y+dy, c)
				}
			}
		}
	}
}

// zoomJob is one named boundary enlargement.
type zoomJob struct {
	name  string
	title string
	src   image.Image
	opt   ZoomOptions
}

func buildZoomJobs(pal Palette, masks map[string]*Mask, nestedNZ, clipRectScene *image.NRGBA) []zoomJob {
	// 72x72 crop straddling the horizontal top edge (y~114) of the pentagram's
	// central pentagon hole: EvenOdd shows fill/hole boundary while NonZero
	// stays solid, making the rule difference visible at the boundary.
	core := image.Rect(144, 84, 216, 156)
	holeCrop := image.Rect(84, 114, 156, 186)
	clipCrop := image.Rect(72, 114, 144, 186)
	grid := color.NRGBA{96, 104, 128, 120}
	return []zoomJob{
		{
			name:  "zoom_star_core_evenodd",
			title: "STAR HOLE EDGE: EVENODD",
			src:   MaskImage(masks["mask_star_evenodd"]),
			opt:   ZoomOptions{Crop: core, Scale: 8, Grid: true, GridC: grid},
		},
		{
			name:  "zoom_star_core_nonzero",
			title: "SAME EDGE: NONZERO FILLED",
			src:   MaskImage(masks["mask_star_nonzero"]),
			opt:   ZoomOptions{Crop: core, Scale: 8, Grid: true, GridC: grid},
		},
		{
			name:  "zoom_hole_edge",
			title: "NESTED: FILL HOLE ISLAND",
			src:   nestedNZ,
			opt:   ZoomOptions{Crop: holeCrop, Scale: 8, Grid: true, GridC: grid},
		},
		{
			name:  "zoom_clip_edge",
			title: "CLIP SEAM: NO HALF PIXEL GAP",
			src:   MaskImage(masks["mask_clipped_rect_nonzero"]),
			opt:   ZoomOptions{Crop: clipCrop, Scale: 8, Grid: true, GridC: grid},
		},
	}
}

func buildZoomSheet(pal Palette, jobs []zoomJob) *image.NRGBA {
	const (
		pad     = 14
		titleH  = 28
		scale   = 6
		cols    = 2
		margin  = 28
		gapX    = 22
		gapY    = 22
		headerH = 58
	)
	// All crops are 72x72.
	zoomSize := 72 * scale
	cardW := zoomSize + pad*2
	cardH := titleH + zoomSize + pad
	rows := (len(jobs) + cols - 1) / cols
	w := margin*2 + cols*cardW + (cols-1)*gapX
	h := headerH + rows*cardH + (rows-1)*gapY + margin
	sheet := image.NewNRGBA(image.Rect(0, 0, w, h))
	fillRect(sheet, sheet.Bounds(), color.NRGBA{238, 241, 247, 255})
	DrawText(sheet, "BOUNDARY ENLARGEMENTS - NEAREST NEIGHBOUR + PIXEL GRID",
		margin, 16, 2, pal.Ink)

	for i, j := range jobs {
		col := i % cols
		row := i / cols
		x := margin + col*(cardW+gapX)
		y := headerH + row*(cardH+gapY)
		card := image.Rect(x, y, x+cardW, y+cardH)
		fillRect(sheet, card, color.NRGBA{255, 255, 255, 255})
		RectOutline(sheet, card, color.NRGBA{190, 198, 214, 255})
		DrawText(sheet, j.title, x+pad, y+7, 2, pal.Ink)

		zopt := j.opt
		zopt.Scale = scale
		zoom := ZoomImage(j.src, zopt)
		zx, zy := x+pad, y+titleH
		draw.Draw(sheet, image.Rect(zx, zy, zx+zoomSize, zy+zoomSize), zoom,
			image.Pt(0, 0), draw.Src)
		RectOutline(sheet, image.Rect(zx, zy, zx+zoomSize, zy+zoomSize),
			color.NRGBA{150, 158, 176, 255})
	}
	return sheet
}
