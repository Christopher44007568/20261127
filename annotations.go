package main

var (
	labelWhite  = hexLinear(0xf4f9ff)
	labelBlue   = hexLinear(0x72e5ff)
	labelRed    = hexLinear(0xff7a68)
	labelViolet = hexLinear(0xd59cff)
)

func header(c *Canvas, title, subtitle string) {
	c.RoundPanel(42, 14, 1010, 104, 20, colInk, 0.66)
	c.Text(title, 66, 24, 4, labelWhite, TextLeft)
	c.Text(subtitle, 68, 75, 2, hexLinear(0xd3e8ff), TextLeft)
}

func infoLines(c *Canvas, x, y int, lines []string, colors []Vec3) {
	panelHeight := 24 + len(lines)*34
	c.RoundPanel(x-18, y-16, x+350, y+panelHeight, 16, colInk, 0.58)
	for i, line := range lines {
		col := labelWhite
		if colors != nil && i < len(colors) {
			col = colors[i]
		}
		c.CirclePixels(Vec2{float64(x), float64(y + i*34 + 10)}, 6, col, 1, false)
		c.Text(line, x+20, y+i*34, 2, col, TextLeft)
	}
}

func AnnotatePrism(c *Canvas) {
	header(c, "TRIANGULAR PRISM", "CONTINUOUS SPECTRUM - REFRACTION + BEER ABSORPTION")
	c.TextWorld("WHITE LIGHT", v2(-6.25, 1.22), 2, labelWhite, TextLeft)
	c.TextWorld("RED: LOWER N", v2(2.05, -2.18), 2, labelRed, TextLeft)
	c.TextWorld("BLUE/VIOLET: HIGHER N", v2(1.55, -3.62), 2, labelBlue, TextLeft)
	c.TextWorld("PRISM BASE", v2(-1.15, -1.92), 2, hexLinear(0x315a86), TextCenter)
	infoLines(c, 1218, 628, []string{
		"N(410NM) > N(680NM)",
		"BLUE BENDS MORE",
		"TOWARD THE BASE",
	}, []Vec3{labelWhite, labelBlue, labelViolet})
}

func AnnotateSphere(c *Canvas) {
	header(c, "GLASS SPHERE", "CHROMATIC FOCUS - BLUE FOCUSES BEFORE RED")
	c.Text("FOCAL ZOOM", 1288, 92, 2, labelBlue, TextLeft)
	infoLines(c, 1218, 594, []string{
		"BLUE FOCUS: LEFT",
		"RED FOCUS: RIGHT",
		"SCREEN: VIOLET OUTER",
	}, []Vec3{labelBlue, labelRed, labelViolet})
	c.TextWorld("PARALLEL WHITE RAYS", v2(-6.25, 2.05), 2, labelWhite, TextLeft)
}

func AnnotatePlate(c *Canvas) {
	header(c, "FLAT GLASS PLATE", "PARALLEL FACES - DIRECTION RESTORED, POSITION DISPERSED")
	c.Text("EDGE ZOOM", 1108, 114, 2, labelBlue, TextLeft)
	infoLines(c, 520, 884, []string{
		"OUTPUT RAYS PARALLEL",
		"BLUE SHIFT IS LARGER",
		"BLUE IS ABOVE RED",
	}, []Vec3{labelWhite, labelBlue, labelViolet})
	c.TextWorld("OBLIQUE WHITE BEAM", v2(-6.25, 1.18), 2, labelWhite, TextLeft)
}
