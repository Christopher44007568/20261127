package main

import (
	"image"
	"math"
)

func mixRGB(a, b RGB, t float64) RGB {
	return RGB{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

func statusColor(s ReachStatus, normal RGB) RGB {
	switch s {
	case StatusOverReach:
		return colWarn
	case StatusTooClose, StatusLimit:
		return colAmber
	default:
		return normal
	}
}

// followView builds a camera that keeps the pelvis at a fixed panel spot.
func followView(r image.Rectangle, f Frame, scale float64) View {
	return View{
		CX:     float64(r.Min.X) + 0.40*float64(r.Dx()),
		CY:     float64(r.Min.Y) + 0.60*float64(r.Dy()),
		Center: f.Pelvis.sub(vec(0, 0.12)),
		Scale:  scale,
	}
}

func (v View) unpx(x, y int) Vec {
	return vec(
		v.Center.X+(float64(x)-v.CX)/v.Scale,
		v.Center.Y-(float64(y)-v.CY)/v.Scale,
	)
}

// drawGround paints the panel background, metric grid, slope and hatching.
func drawGround(cv *Canvas, r image.Rectangle, v View, slope Slope) {
	cv.fillRect(r.Min.X, r.Min.Y, r.Dx(), r.Dy(), colPanel)

	// Fine world-aligned metric grid.
	const gridStep = 0.25
	w0 := v.unpx(r.Min.X, r.Max.Y)
	w1 := v.unpx(r.Max.X, r.Min.Y)
	for gx := math.Ceil(w0.X/gridStep) * gridStep; gx <= w1.X; gx += gridStep {
		xA, yA := v.px(vec(gx, w0.Y))
		xB, yB := v.px(vec(gx, w1.Y))
		cv.line(xA, yA, xB, yB, 1, colGrid)
	}
	for gy := math.Ceil(w0.Y/gridStep) * gridStep; gy <= w1.Y; gy += gridStep {
		xA, yA := v.px(vec(w0.X, gy))
		xB, yB := v.px(vec(w1.X, gy))
		cv.line(xA, yA, xB, yB, 1, colGrid)
	}

	// Earth fill below the infinite slope line.
	var poly []image.Point
	for px := r.Min.X; px <= r.Max.X; px += 10 {
		w := v.unpx(px, 0)
		gy := math.Tan(slope.Ang) * w.X
		sx, sy := v.px(vec(w.X, gy))
		poly = append(poly, image.Pt(sx, sy))
	}
	poly = append(poly, image.Pt(r.Max.X, r.Max.Y))
	poly = append(poly, image.Pt(r.Min.X, r.Max.Y))
	cv.fillPolygon(poly, colGround)

	// Surface line.
	l := v.unpx(r.Min.X, 0)
	rr := v.unpx(r.Max.X, 0)
	ly := math.Tan(slope.Ang) * l.X
	ry := math.Tan(slope.Ang) * rr.X
	wc := wcanvas{cv: cv, v: v}
	wc.line(vec(l.X, ly), vec(rr.X, ry), 0.022, colSlope)

	// Ground hatching ticks at fixed along-slope coordinates (slide with world).
	sMin := (w0.X - slope.Origin.X) / slope.Uphill.X
	sMax := (w1.X - slope.Origin.X) / slope.Uphill.X
	for s := math.Floor(sMin/0.16)*0.16 - 0.16; s <= sMax+0.16; s += 0.16 {
		p := slope.Point(s)
		wc.line(p, p.sub(slope.Normal.mul(0.07)), 0.012, colGroundDot)
	}
}

type charOpts struct {
	mono          *RGB // if non-nil, draw a ghost/monochrome version
	thin          bool
	showTargets   bool
	compactTarget bool
}

// drawCharacter renders the full biped. All IK points come from Frame.
func drawCharacter(cv *Canvas, cfg GaitConfig, f Frame, v View, opt charOpts) {
	wc := wcanvas{cv: cv, v: v}

	upTorso := slopeTorsoDir(cfg)
	pelvis := f.Pelvis
	neck := pelvis.add(upTorso.mul(0.56))
	headC := neck.add(upTorso.mul(0.135))
	shoulder := pelvis.add(upTorso.mul(0.47))

	baseL := colHipL
	baseR := colHipR
	if opt.mono != nil {
		baseL, baseR = *opt.mono, *opt.mono
	}

	// Back arm (simple FK, counter-swings with the opposite leg).
	drawArm(cv, v, shoulder, upTorso, f.Feet[1].Pose.HipFlex, opt)

	// Far leg first, then near leg.
	drawLeg(cv, cfg, v, f.Feet[0], baseL, opt)
	drawLeg(cv, cfg, v, f.Feet[1], baseR, opt)

	if opt.mono == nil {
		// Pelvis and torso.
		wc.circle(pelvis, 0.10, colTorso)
		wc.line(pelvis, neck, 0.17, colTorso)
		// Front arm.
		drawArm(cv, v, shoulder, upTorso, f.Feet[0].Pose.HipFlex, opt)
		// Head with side eye.
		wc.circle(headC, 0.115, colSkin)
		hx, hy := v.px(headC)
		cv.ring(hx, hy, 0.115*v.Scale, 2, colInk)
		eye := headC.add(upTorso.rotate(-math.Pi / 2).mul(0.075)).add(upTorso.mul(0.02))
		wc.circle(eye, 0.018, colInk)
	} else {
		wc.line(pelvis, neck, 0.10, *opt.mono)
		wc.circle(headC, 0.08, *opt.mono)
	}
}

func slopeTorsoDir(cfg GaitConfig) Vec {
	// Near-upright with a slight uphill/forward lean: the slope normal points
	// up-back (104 deg), so rotate clockwise toward world vertical/forward.
	return cfg.Slope.Normal.rotate(rad(-15))
}

func drawArm(cv *Canvas, v View, shoulder, upTorso Vec, oppositeHipFlex float64, opt charOpts) {
	wc := wcanvas{cv: cv, v: v}
	downAng := upTorso.angle() + math.Pi
	swing := clampF((oppositeHipFlex-rad(20))*1.5, -0.6, 0.6)
	upperAng := downAng + swing
	upperLen, foreLen := 0.27, 0.25
	elbow := shoulder.add(vec(math.Cos(upperAng), math.Sin(upperAng)).mul(upperLen))
	foreAng := upperAng + 0.42
	hand := elbow.add(vec(math.Cos(foreAng), math.Sin(foreAng)).mul(foreLen))
	col := colTorso
	width := 0.055
	if opt.mono != nil {
		col = *opt.mono
		width = 0.03
	}
	wc.line(shoulder, elbow, width, col)
	wc.line(elbow, hand, width*0.9, col)
	if opt.mono == nil {
		wc.circle(hand, 0.04, colSkin)
	}
}

func drawLeg(cv *Canvas, cfg GaitConfig, v View, fs FootSample, base RGB, opt charOpts) {
	wc := wcanvas{cv: cv, v: v}
	pose := fs.Pose

	legCol := base
	thighW, shinW := 0.078, 0.064
	if opt.mono != nil {
		legCol = *opt.mono
		thighW, shinW = 0.035, 0.03
	} else if fs.Index == 0 {
		legCol = mixRGB(base, colPaper, 0.35) // far leg hazed back
	}
	if opt.mono == nil {
		legCol = statusColor(pose.Status, legCol)
	}

	wc.line(pose.Hip, pose.Knee, thighW, legCol)
	wc.line(pose.Knee, pose.Ankle, shinW, legCol)

	if opt.mono == nil {
		wc.circle(pose.Knee, 0.052, legCol)
		kx, ky := v.px(pose.Knee)
		cv.ring(kx, ky, 0.052*v.Scale, 2, colInk)
		wc.circle(pose.Knee, 0.02, colPaper)
	}

	// Sole / foot.
	footCol := colFoot
	if opt.mono != nil {
		footCol = *opt.mono
	} else if pose.Status != StatusOK {
		footCol = statusColor(pose.Status, colFoot)
	}
	wc.line(fs.Heel, fs.Toe, 0.055, footCol)
	if opt.mono == nil {
		wc.circle(pose.Ankle, 0.04, footCol)
	}

	// Stance lock marker: green contact glow + fixed ground diamond.
	if opt.mono == nil && fs.Stance {
		wc.line(fs.Heel, fs.Toe, 0.085, colOK)
		wc.line(fs.Heel, fs.Toe, 0.05, colFoot)
		mid := fs.Heel.add(fs.Toe).mul(0.5)
		groundMid := vec(mid.X, math.Tan(cfg.Slope.Ang)*mid.X)
		drawDiamond(cv, v, groundMid, 0.055, colOK)
	}

	if opt.showTargets && opt.mono == nil {
		drawTargetHandle(cv, v, fs, opt.compactTarget)
	}
}

func drawDiamond(cv *Canvas, v View, p Vec, rM float64, col RGB) {
	x, y := v.px(p)
	r := int(rM * v.Scale)
	pts := []image.Point{{X: x, Y: y - r}, {X: x + r, Y: y}, {X: x, Y: y + r}, {X: x - r, Y: y}}
	cv.fillPolygon(pts, col)
}

// drawTargetHandle draws the commanded IK target and the error vector.
func drawTargetHandle(cv *Canvas, v View, fs FootSample, compact bool) {
	pose := fs.Pose
	tx, ty := v.px(pose.CmdTarget)
	bad := pose.Status == StatusOverReach
	warn := pose.Status == StatusTooClose || pose.Status == StatusLimit
	col := colTarget
	if bad {
		col = colWarn
	} else if warn {
		col = colAmber
	}

	if pose.EndError > 1e-6 {
		wc := wcanvas{cv: cv, v: v}
		wc.dashed(pose.CmdTarget, pose.Ankle, 0.012, 0.05, 0.035, col)
	}

	if compact {
		cv.ring(tx, ty, 8, 2, colPurple)
		return
	}

	// Crosshair target.
	cv.ring(tx, ty, 9, 2.5, col)
	cv.line(tx-6, ty, tx+6, ty, 2, col)
	cv.line(tx, ty-6, tx, ty+6, 2, col)
	if bad {
		cv.line(tx-8, ty-8, tx+8, ty+8, 4, colWarn)
		cv.line(tx-8, ty+8, tx+8, ty-8, 4, colWarn)
	}
}
