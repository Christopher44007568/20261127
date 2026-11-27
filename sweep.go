package main

func RenderSweep(scene *Scene) *Frame {
	radii := []float64{76, 68, 60, 52, 44, 36, 29, 23, 17}
	width := 160 * len(radii)
	height := 260
	frame := NewFrame(width, height)
	drawBackground(frame)
	for i, radius := range radii {
		cx := float64(80 + i*160)
		cy := 138.0
		drawContactShadow(frame, cx, cy+radius*0.90, radius)
		renderSphere(frame, scene, cx, cy, radius, ModeFootprint)
	}
	frame.Bloom(1.65, 0.26, 2)
	return frame
}
