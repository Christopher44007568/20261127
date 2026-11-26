package holefill

import "fmt"

func DrawTextShadow(cv *Canvas, x, y int, text string, scale int, col RGB) {
	DrawText(cv, x+2, y+2, text, scale, RGB{2, 6, 23})
	DrawText(cv, x, y, text, scale, col)
}

func DrawHoleHUD(cv *Canvas, loops []*BoundaryLoop) {
	DrawTextShadow(cv, 30, 26, "OPEN SCAN: BOUNDARY LOOPS", 3, RGB{248, 250, 252})
	DrawTextShadow(cv, 31, 62, "OUTER CONTOUR IS CLASSIFIED, NOT CAPPED", 2, RGB{148, 163, 184})

	x, y := 28, cv.H-178
	cv.FillRect(x-14, y-16, x+360, y+150, RGB{15, 23, 42})
	cv.FrameRect(x-14, y-16, x+360, y+150, 2, RGB{51, 65, 85})
	entries := []struct {
		col  RGB
		text string
	}{
		{ClassOuter.Color(), "OUTER CONTOUR - LEFT OPEN"},
		{ClassSlit.Color(), "NARROW SLIT"},
		{ClassLNotch.Color(), "L SHAPED NOTCH"},
		{ClassClosePair.Color(), "CLOSE BOUNDARY HOLES"},
		{ClassOrdinaryHole.Color(), "ORDINARY LOCAL HOLE"},
	}
	for i, e := range entries {
		DrawLegendRow(cv, x, y+i*27, e.col, e.text, 2)
	}
	DrawTextShadow(cv, cv.W-330, 28, fmt.Sprintf("%d BOUNDARY LOOPS", len(loops)), 2, RGB{226, 232, 240})
}

func DrawPatchHUD(cv *Canvas, reports []PatchReport, alpha bool) {
	title := "LOCAL PATCH MESH"
	sub := "SHARED BOUNDARY NORMALS"
	if alpha {
		sub = "ALPHA-WRAPPED PATCHES + CONTINUOUS NORMALS"
	}
	DrawTextShadow(cv, 30, 26, title, 3, RGB{248, 250, 252})
	DrawTextShadow(cv, 31, 62, sub, 2, RGB{148, 163, 184})

	x, y := 28, cv.H-126
	cv.FillRect(x-14, y-16, x+390, y+92, RGB{15, 23, 42})
	cv.FrameRect(x-14, y-16, x+390, y+92, 2, RGB{51, 65, 85})
	totalTris := 0
	for _, r := range reports {
		totalTris += r.Triangles
	}
	DrawLegendRow(cv, x, y, ClassSlit.Color(), "SLIT PATCH", 2)
	DrawLegendRow(cv, x, y+27, ClassLNotch.Color(), "L NOTCH PATCH", 2)
	DrawLegendRow(cv, x+190, y, ClassClosePair.Color(), "CLOSE PAIR", 2)
	DrawLegendRow(cv, x+190, y+27, ClassOrdinaryHole.Color(), "ORDINARY", 2)
	DrawTextShadow(cv, cv.W-360, 30, fmt.Sprintf("%d PATCHES  %d FACES", len(reports), totalTris), 2, RGB{226, 232, 240})
}
