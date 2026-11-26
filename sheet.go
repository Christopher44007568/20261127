package main

import "fmt"

const (
	sheetW     = 1312
	sheetH     = 516
	panelY     = 76
	panelX0    = 24
	panelGap   = 32
	panelCount = 3
)

func panelX(index int) int {
	return panelX0 + index*(panelSize+panelGap)
}

func buildSheet(res *ScenarioResult, reference *Surface, independent bool) *Surface {
	sheet := NewSurface(sheetW, sheetH)
	sheet.Fill(colSheet)

	title := scenarioTitle(res)
	subtitle := scenarioSubtitle(res)
	status := scenarioStatus(res, independent)
	accent := colAccentOK
	if res.Bug != BugNone {
		accent = colAccentError
	}

	drawText(sheet, panelX0, 11, 3, title, colInk)
	drawText(sheet, panelX0, 38, 2, subtitle, colMuted)
	sheet.FillRect(panelX0, 55, textWidth(2, subtitle), 2, accent)
	drawStats(sheet, res)

	labels := [3]string{"FRAME 0", "FRAME 1", "FRAME 1 FRESH"}
	for i := 0; i < 2; i++ {
		label := labels[i]
		x := panelX(i)
		labelX := x + (panelSize-textWidth(2, label))/2
		drawText(sheet, labelX, 58, 2, label, RGBA{188, 205, 225, 255})
		sheet.DrawSurface(res.Frames[i], x, panelY)
		sheet.StrokeRect(x-1, panelY-1, panelSize+2, panelSize+2, 1, accent)
	}
	label := labels[2]
	labelX := panelX(2) + (panelSize-textWidth(2, label))/2
	drawText(sheet, labelX, 58, 2, label, RGBA{188, 205, 225, 255})
	sheet.DrawSurface(reference, panelX(2), panelY)
	sheet.StrokeRect(panelX(2)-1, panelY-1, panelSize+2, panelSize+2, 1, colAccentOK)

	statusY := panelY + panelSize + 13
	sheet.FillRect(panelX0, statusY, sheetW-2*panelX0, 26, RGBA{16, 24, 36, 255})
	sheet.FillRect(panelX0, statusY, 4, 26, accent)
	drawText(sheet, panelX0+14, statusY+6, 2, status, accent)
	return sheet
}

func scenarioTitle(res *ScenarioResult) string {
	switch res.Bug {
	case BugEarlyRelease:
		return "EARLY RELEASE / BLACK BLOCKS"
	case BugLateReclaim:
		return "DELAYED RECLAIM / GHOSTING"
	default:
		return "NORMAL / DEFERRED RECLAIM"
	}
}

func scenarioSubtitle(res *ScenarioResult) string {
	switch res.Bug {
	case BugEarlyRelease:
		return "CONSUMER LEASE MISSING / RESOURCE DESTROYED IN FRAME"
	case BugLateReclaim:
		return "EXTRA FRAME IN QUEUE / RETIRED PIXELS REUSED UNCLEARED"
	default:
		return "REFCOUNT + DELAYED QUEUE + FRAME BOUNDARY"
	}
}

func scenarioStatus(res *ScenarioResult, independent bool) string {
	if res.Bug == BugNone {
		if independent {
			return "OK: FRAME 1 MATCHES FRESH REFERENCE"
		}
		return "FRAME 1 DEPENDS ON PREVIOUS FRAME"
	}
	last := res.Diffs[len(res.Diffs)-1]
	if res.Bug == BugEarlyRelease {
		return fmt.Sprintf("MAGENTA = DESTROYED LAYER / BLACK BLOCKS %d", last.BlackPixels)
	}
	return fmt.Sprintf("MAGENTA = STALE FRAME 0 PIXELS / DIFF %d", last.ChangedPixels)
}

func drawStats(sheet *Surface, res *ScenarioResult) {
	s := res.Stats
	line := fmt.Sprintf(
		"ACQ %d  FAIL %d  RETIRE %d  RECLAIM %d  REUSE %d  QUEUE %d",
		s.Acquires, s.FailedAcquires, s.Retired, s.Reclaimed, s.ReusedRetired, res.QueueEnd,
	)
	width := textWidth(1, line)
	drawText(sheet, sheetW-panelX0-width, 15, 1, line, colMuted)
}

func countPixelDiff(a, b *Surface) int {
	if a.W != b.W || a.H != b.H {
		return -1
	}
	count := 0
	for i := range a.Pix {
		if colorDistance(a.Pix[i], b.Pix[i]) > 6 {
			count++
		}
	}
	return count
}

func countExactColor(s *Surface, c RGBA) int {
	count := 0
	for _, p := range s.Pix {
		if p == c {
			count++
		}
	}
	return count
}
