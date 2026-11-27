package main

import (
	"fmt"
	"image"
)

var font5x7 = map[rune][7]uint8{
	'A': {14, 17, 17, 31, 17, 17, 17},
	'B': {30, 17, 17, 30, 17, 17, 30},
	'C': {15, 16, 16, 16, 16, 16, 15},
	'D': {30, 17, 17, 17, 17, 17, 30},
	'E': {31, 16, 16, 30, 16, 16, 31},
	'F': {31, 16, 16, 30, 16, 16, 16},
	'G': {15, 16, 16, 23, 17, 17, 15},
	'H': {17, 17, 17, 31, 17, 17, 17},
	'I': {31, 4, 4, 4, 4, 4, 31},
	'J': {7, 2, 2, 2, 18, 18, 12},
	'K': {17, 18, 20, 24, 20, 18, 17},
	'L': {16, 16, 16, 16, 16, 16, 31},
	'M': {17, 27, 21, 21, 17, 17, 17},
	'N': {17, 25, 21, 19, 17, 17, 17},
	'O': {14, 17, 17, 17, 17, 17, 14},
	'P': {30, 17, 17, 30, 16, 16, 16},
	'Q': {14, 17, 17, 17, 21, 18, 13},
	'R': {30, 17, 17, 30, 20, 18, 17},
	'S': {15, 16, 16, 14, 1, 1, 30},
	'T': {31, 4, 4, 4, 4, 4, 4},
	'U': {17, 17, 17, 17, 17, 17, 14},
	'V': {17, 17, 17, 17, 17, 10, 4},
	'W': {17, 17, 17, 21, 21, 21, 10},
	'X': {17, 17, 10, 4, 10, 17, 17},
	'Y': {17, 17, 10, 4, 4, 4, 4},
	'Z': {31, 1, 2, 4, 8, 16, 31},
	'0': {14, 17, 19, 21, 25, 17, 14},
	'1': {4, 12, 4, 4, 4, 4, 14},
	'2': {14, 17, 1, 2, 4, 8, 31},
	'3': {30, 1, 1, 14, 1, 1, 30},
	'4': {2, 6, 10, 18, 31, 2, 2},
	'5': {31, 16, 16, 30, 1, 1, 30},
	'6': {14, 16, 16, 30, 17, 17, 14},
	'7': {31, 1, 2, 4, 8, 8, 8},
	'8': {14, 17, 17, 14, 17, 17, 14},
	'9': {14, 17, 17, 15, 1, 1, 14},
	'.': {0, 0, 0, 0, 0, 12, 12},
	'-': {0, 0, 0, 31, 0, 0, 0},
	'/': {1, 2, 2, 4, 8, 8, 16},
	'%': {25, 26, 2, 4, 8, 11, 19},
	':': {0, 12, 12, 0, 12, 12, 0},
	'+': {0, 4, 4, 31, 4, 4, 0},
}

func textWidth(text string, scale int) int {
	if len(text) == 0 {
		return 0
	}
	return len(text)*6*scale - scale
}

func drawText(img *image.RGBA, x, y int, text string, scale int, c RGB, alpha float32) {
	cursor := x
	for _, ch := range text {
		glyph, ok := font5x7[ch]
		if !ok {
			cursor += 6 * scale
			continue
		}
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if glyph[row]&(1<<uint(4-col)) == 0 {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						coverPixel(img, cursor+col*scale+sx, y+row*scale+sy, c, alpha)
					}
				}
			}
		}
		cursor += 6 * scale
	}
}

func drawInterface(img *image.RGBA, scene Scene, stats RenderStats, panel wallPanel) {
	title := "STONE WALL / PARALLAX MAPPING"
	subtitle := "HEIGHT GRADIENT  /  BOUNDARY FALLBACK  /  LAYER BUDGET"
	drawText(img, 80, 24, title, 2, RGB{0.80, 0.77, 0.70}, 0.96)
	drawText(img, 80, 54, subtitle, 1, RGB{0.37, 0.40, 0.39}, 0.92)

	sceneTag := ""
	sceneTitle := ""
	switch scene.Kind {
	case ViewFront:
		sceneTag = "VIEW 01 / FRONT"
		sceneTitle = "FRONT VIEW  /  BUDGET 48"
	case ViewGrazing:
		sceneTag = "VIEW 02 / GRAZING"
		sceneTitle = "GRAZING VIEW  /  BUDGET 64"
	default:
		sceneTag = "VIEW 03 / FALLBACK"
		sceneTitle = "ERROR FALLBACK  /  BUDGET 7"
	}
	drawText(img, imageWidth-80-textWidth(sceneTag, 1), 26, sceneTag, 1,
		RGB{0.72, 0.55, 0.30}, 0.95)
	drawRectOutline(img, 80, 88, panel.x1, 88, RGB{0.31, 0.28, 0.22}, 0.72)
	drawText(img, 80, panel.y1+16, sceneTitle, 2, RGB{0.76, 0.70, 0.58}, 0.96)

	hitPct := 100 * float64(stats.Hit) / float64(stats.Pixels)
	edgePct := 100 * float64(stats.BoundaryFallback) / float64(stats.Pixels)
	budgetPct := 100 * float64(stats.BudgetFallback) / float64(stats.Pixels)
	statsLine := fmt.Sprintf("HIT %05.2f%%   EDGE %05.2f%%   BUDGET %05.2f%%   AVG LAYER %.1f",
		hitPct, edgePct, budgetPct, stats.AverageLayers)
	drawText(img, 80, panel.y1+48, statsLine, 1, RGB{0.48, 0.49, 0.44}, 0.94)

	note := "GROOVE OCCLUSION FOLLOWS VIEW DIRECTION"
	if scene.Kind == ViewFallback {
		note = "AMBER = LAST VALID  /  BLUE = HIT  /  DIM = UV EDGE"
	}
	noteX := panel.x1 - textWidth(note, 1)
	drawText(img, noteX, panel.y1+48, note, 1, RGB{0.55, 0.42, 0.27}, 0.90)
}
