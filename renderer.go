package main

import "image/color"

// Camera is a smoothed orthographic camera. It follows the pelvis only with
// damping, so during fast turns the hip/CoP never teleport on screen, and the
// weight point genuinely drifts in screen space rather than being pinned.
type Camera struct {
	FocusX, FocusY float64
	Initialised    bool
}

const (
	CanvasW    = 1000
	CanvasH    = 760
	PanelY0    = 40
	PanelH     = 470
	ChartY0    = 525
	ChartH     = 210
	PixelsPerM = 165
	CamDamping = 0.12
)

func (cam *Camera) follow(focus Vec3) {
	if !cam.Initialised {
		cam.FocusX, cam.FocusY = focus.X, 0
		cam.Initialised = true
		return
	}
	cam.FocusX += (focus.X - cam.FocusX) * CamDamping
	// Keep the lateral focus fixed at zero: the camera tracks travel distance
	// but does not rotate, so the character stays in side view through turns
	// and the CoP still shifts on screen via its real Y/X screen projection.
}

// Screen projects a world point into a panel. View direction looks along
// (-X, -Y) at a 30 deg azimuth with a slight downward tilt.
func (cam *Camera) Screen(p Vec3, panelCenterX int) (int, int) {
	const depth = 0.35 // foreshortened lateral/heading depth
	sx := (p.X - cam.FocusX) + (p.Y-cam.FocusY)*depth
	// Screen Y grows downward; world up (+Z) therefore subtracts directly.
	sy := (p.Y-cam.FocusY)*depth - p.Z
	x := panelCenterX + int(sx*PixelsPerM)
	y := PanelY0 + 360 + int(sy*PixelsPerM)
	return x, y
}

// Renderer keeps per-panel camera and accumulated sole trails.
type Renderer struct {
	SrcCam, TgtCam Camera
	SrcTrails      [2][]Vec3
	TgtTrails      [2][]Vec3
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

var bonePairs = [][2]int{
	{JRoot, JPelvis}, {JPelvis, JSpine}, {JSpine, JNeck}, {JNeck, JHead},
	{JSpine, JLShoulder}, {JLShoulder, JLElbow}, {JLElbow, JLHand},
	{JSpine, JRShoulder}, {JRShoulder, JRElbow}, {JRElbow, JRHand},
	{JPelvis, JLHip}, {JLHip, JLKnee}, {JLKnee, JLAnkle}, {JLAnkle, JLToe},
	{JPelvis, JRHip}, {JRHip, JRKnee}, {JRKnee, JRAnkle}, {JRAnkle, JRToe},
}

func (r *Renderer) renderFrame(frames []Frame, f int) *FrameBuffer {
	fb := NewFrameBuffer(CanvasW, CanvasH)
	fb.Fill(colBg)

	leftCenter, rightCenter := CanvasW/4, CanvasW*3/4
	r.SrcCam.follow(frames[f].Source.Pos[JPelvis])
	r.TgtCam.follow(frames[f].Target.Pos[JPelvis])

	drawPanel(fb, 0, leftCenter, "SOURCE A  (SHORT LEGS)", colSrc)
	drawPanel(fb, CanvasW/2, rightCenter, "TARGET B  (LONG LEGS, RETARGETED)", colTgt)

	r.drawScene(fb, &r.SrcCam, leftCenter, frames[f].Source, f, true)
	r.drawScene(fb, &r.TgtCam, rightCenter, frames[f].Target, f, false)

	drawHeader(fb, f, frames[f].TurnRate)
	drawKneeChart(fb, frames, f)
	return fb
}

func drawPanel(fb *FrameBuffer, x0, centerX int, title string, accent color.Color) {
	fb.FillRect(x0+8, PanelY0, x0+CanvasW/2-8, PanelY0+PanelH, colPanel)
	fb.FillRect(x0+8, PanelY0, x0+CanvasW/2-8, PanelY0+3, accent)
	fb.DrawText(x0+20, PanelY0+12, 1, title, colText)
	_ = centerX
}
