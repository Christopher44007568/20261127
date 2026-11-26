package main

import (
	"image/color"
)

func (b *board) objectTable(rep *DiffReport) {
	b.sectionTitle("E  PER-OBJECT STRUCTURAL DELTA (ORIGINAL IDS)", inkAccent)

	headers := []struct {
		x    int
		text string
	}{
		{64, "OBJECT ID"}, {356, "ERRPX"}, {452, "ERROR RATIO"},
		{700, "MEAN/MAX"}, {900, "STATUS"},
	}
	for _, h := range headers {
		DrawText(b.img, h.x, b.y, h.text, 1, inkMuted)
	}
	b.y += 20
	fillRect(b.img, 40, b.y, boardW-80, 2, color.RGBA{51, 65, 85, 255})
	b.y += 10

	statusCol := map[string]color.RGBA{
		"error": inkBad, "minor": inkWarn, "match": inkOK,
	}
	for i, st := range rep.Objects {
		ry := b.y
		if i%2 == 0 {
			fillRect(b.img, 40, ry-4, boardW-80, 28, boardPanel)
		}
		col := statusCol[st.Status]
		fillRect(b.img, 46, ry+4, 12, 12, col)

		id := st.ID
		if st.Parent != "" {
			id = "  " + id
		}
		DrawText(b.img, 64, ry+2, id, 2, inkMain)
		DrawText(b.img, 356, ry+3, itoa(st.ErrorPixels), 2, inkMain)

		barX, barW := 452, 200
		fillRect(b.img, barX, ry+6, barW, 10, color.RGBA{30, 41, 59, 255})
		frac := st.ErrorRatio / 0.08
		if frac > 1 {
			frac = 1
		}
		fillRect(b.img, barX, ry+6, int(float64(barW)*frac), 10, col)
		pct := fnum(st.ErrorRatio*100) + "%"
		DrawText(b.img, barX+barW+8, ry+3, pct, 1, inkMuted)

		md := fnum(st.MeanDelta) + "/" + fnum(st.MaxDelta)
		DrawText(b.img, 700, ry+3, md, 2, inkMain)
		DrawText(b.img, 900, ry+3, st.Status, 2, col)
		b.y += 30
	}
	b.y += 18
}

func (b *board) reloadSection(m *BoardModel) {
	b.sectionTitle("F  SVG RELOAD AUDIT - PATHS STAY EDITABLE, NEVER ONE BITMAP", inkAccent)

	drawCheck := func(y int, c CheckResult) int {
		mark := "[OK] "
		col := inkOK
		if !c.Pass {
			mark = "[!!] "
			col = inkBad
		}
		DrawText(b.img, 44, y, mark+c.Name, 2, col)
		DrawText(b.img, 44, y+22, "       "+c.Detail, 1, inkMuted)
		return y + 42
	}
	y := b.y
	for _, c := range m.Checks {
		y = drawCheck(y, c)
	}
	b.y = y + 8

	if m.Reload != nil {
		a := m.Reload
		line := "reloaded: ids=" + itoa(a.IdentifiedElements) +
			" path=" + itoa(a.PathElements) +
			" shape=" + itoa(a.ShapeElements) +
			" g=" + itoa(a.GroupElements) +
			" gradients=" + itoa(a.GradientElements) +
			" clipPaths=" + itoa(a.ClipPathElements) +
			" <image>=" + itoa(a.ImageElements)
		DrawText(b.img, 44, b.y, line, 2, inkMain)
		b.y += 30
		line2 := "own.png vs reloaded.png mean delta=" + fnum(m.ReloadDelta) +
			" (round-trip geometry stays vector)"
		DrawText(b.img, 44, b.y, line2, 1, inkMuted)
		b.y += 26
	}
	if m.Bad != nil {
		line := "flattened_bad.svg: image=" + itoa(m.Bad.ImageElements) +
			" path=" + itoa(m.Bad.PathElements) +
			" -> REJECTED (one merged bitmap)"
		DrawText(b.img, 44, b.y, line, 2, inkWarn)
		b.y += 30
	}
}
