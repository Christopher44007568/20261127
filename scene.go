package main

const panelSize = 400

var (
	colSheet       = rgb(7, 11, 18)
	colInk         = rgb(224, 232, 244)
	colMuted       = rgb(120, 145, 176)
	colAccentOK    = rgb(74, 222, 128)
	colAccentError = colMagenta
)

type signalBar struct {
	X, Y, W, H int
	Color      RGBA
}

func signalBars(frame int) []signalBar {
	if frame%2 == 0 {
		return []signalBar{
			{X: 54, Y: 92, W: 118, H: 58, Color: RGBA{246, 158, 67, 238}},
			{X: 208, Y: 164, W: 132, H: 54, Color: RGBA{45, 212, 191, 238}},
			{X: 92, Y: 266, W: 212, H: 42, Color: RGBA{163, 230, 53, 238}},
		}
	}
	return []signalBar{
		{X: 126, Y: 68, W: 118, H: 58, Color: RGBA{246, 158, 67, 238}},
		{X: 48, Y: 154, W: 132, H: 54, Color: RGBA{45, 212, 191, 238}},
		{X: 156, Y: 244, W: 212, H: 42, Color: RGBA{163, 230, 53, 238}},
	}
}

func paintBackdrop(s *Surface) {
	top := [3]int{8, 13, 23}
	bot := [3]int{20, 34, 54}
	for y := 0; y < s.H; y++ {
		t := y * 100 / maxInt(s.H-1, 1)
		c := rgb(
			uint8(top[0]+(bot[0]-top[0])*t/100),
			uint8(top[1]+(bot[1]-top[1])*t/100),
			uint8(top[2]+(bot[2]-top[2])*t/100),
		)
		for x := 0; x < s.W; x++ {
			s.Pix[y*s.W+x] = c
		}
	}
	grid := RGBA{76, 112, 154, 28}
	for x := 20; x < s.W; x += 40 {
		s.DrawLine(x, 0, x, s.H, grid)
	}
	for y := 20; y < s.H; y += 40 {
		s.DrawLine(0, y, s.W, y, grid)
	}
	halo := RGBA{52, 134, 160, 32}
	for i := 0; i < 20; i++ {
		s.DrawLine(0, 342+i, s.W, 342+i, halo)
	}
}

func drawSignalBar(s *Surface, b signalBar) {
	shadow := RGBA{0, 0, 0, 72}
	s.FillRect(b.X+5, b.Y+5, b.W, b.H, shadow)
	s.FillRect(b.X, b.Y, b.W, b.H, b.Color)
	s.StrokeRect(b.X, b.Y, b.W, b.H, 2, RGBA{255, 255, 255, 48})

	x0, y0 := maxInt(b.X+3, 0), maxInt(b.Y+3, 0)
	x1, y1 := minInt(b.X+b.W-3, s.W), minInt(b.Y+b.H-3, s.H)
	stripe := RGBA{255, 255, 255, 54}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if (x+y)%18 < 3 {
				s.Set(x, y, blendPixel(s.At(x, y), stripe))
			}
		}
	}
	tab := RGBA{255, 255, 255, 214}
	s.FillRect(b.X+7, b.Y+7, 16, 4, tab)
	s.FillRect(b.X+7, b.Y+15, 8, 3, RGBA{255, 255, 255, 130})
}

func drawSignalLayerOnto(layer *Surface, frame int) {
	for _, b := range signalBars(frame) {
		drawSignalBar(layer, b)
	}
}

func rasterSignalLayer(frame int, width, height int) *Surface {
	layer := NewSurface(width, height)
	drawSignalLayerOnto(layer, frame)
	return layer
}

func paintPanelChrome(s *Surface, frame int) {
	s.StrokeRect(0, 0, s.W, s.H, 2, RGBA{70, 99, 134, 255})
	corner := RGBA{110, 160, 210, 220}
	for i := 0; i < 18; i++ {
		s.Set(8+i, 8, corner)
		s.Set(8, 8+i, corner)
		s.Set(s.W-9-i, 8, corner)
		s.Set(s.W-9, 8+i, corner)
		s.Set(8+i, s.H-9, corner)
		s.Set(8, s.H-9-i, corner)
		s.Set(s.W-9-i, s.H-9, corner)
		s.Set(s.W-9, s.H-9-i, corner)
	}
	for x := 24; x < s.W-24; x += 32 {
		s.FillRect(x, 354, 12, 2, RGBA{93, 132, 170, 160})
	}
	drawText(s, s.W-32, 14, 2, string(rune('0'+frame)), RGBA{160, 188, 218, 190})
}

func renderIdealFrame(frame int) *Surface {
	s := NewSurface(panelSize, panelSize)
	paintBackdrop(s)
	rasterSignalLayer(frame, panelSize, panelSize).BlendOver(s)
	paintPanelChrome(s, frame)
	return s
}
