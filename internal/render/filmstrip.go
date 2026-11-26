package render

import (
	"fmt"

	"bipedblend/internal/biped"
)

const (
	filmWidth   = 1744
	filmLeft    = 168
	filmCellW   = 126
	filmCellH   = 196
	filmTop     = 82
	filmRowStep = 222
	filmMeterPx = 76
)

func FilmstripSVG(strips []biped.FilmStrip) string {
	height := filmTop + len(strips)*filmRowStep + 24
	s := newSVG(filmWidth, height)
	s.text(32, 38, "二足动画混合：三种速度的连续帧", `font-size="26" font-weight="800" fill="`+colorInk+`"`)
	s.text(32, 64, "所有 Clip 使用同一归一化相位；蓝/橙为左右脚，绿色足底阴影表示接触锚定。", `font-size="15" fill="`+colorMuted+`"`)

	for i, strip := range strips {
		rowY := float64(filmTop + i*filmRowStep)
		drawFilmRow(s, strip, i, rowY)
	}
	return s.end()
}

func drawFilmRow(s *svgBuilder, strip biped.FilmStrip, index int, rowY float64) {
	s.rect(20, rowY-12, filmWidth-40, filmCellH+24, 18,
		`fill="`+colorPanel+`" stroke="`+colorGrid+`" stroke-width="1.5"`)

	labelColor := colorIdle
	speedLabel := "0.0 m/s"
	if strip.Speed >= 3 {
		labelColor = colorRun
		speedLabel = fmt.Sprintf("%.1f m/s", strip.Speed)
	} else if strip.Speed > 0.2 {
		labelColor = colorWalk
		speedLabel = fmt.Sprintf("%.1f m/s", strip.Speed)
	}
	s.circle(42, rowY+54, 8, `fill="`+labelColor+`"`)
	s.text(58, rowY+63, strip.Label, `font-size="28" font-weight="800" fill="`+colorInk+`"`)
	s.text(34, rowY+96, speedLabel, `font-size="18" font-weight="700" fill="`+labelColor+`"`)
	s.text(34, rowY+124, fmt.Sprintf("%d 帧序列", len(strip.Frames)), `font-size="14" fill="`+colorMuted+`"`)
	s.text(34, rowY+146, fmt.Sprintf("每 %d 帧取样", strip.FrameStep), `font-size="14" fill="`+colorMuted+`"`)

	for j, frame := range strip.Frames {
		x := float64(filmLeft + j*filmCellW)
		drawFilmCell(s, frame, x, rowY)
	}
}

func drawFilmCell(s *svgBuilder, frame biped.Frame, x, y float64) {
	cellX := x + 3
	cellY := y
	cellW := float64(filmCellW) - 6
	groundY := y + 166
	s.rect(cellX, cellY, cellW, filmCellH, 12, `fill="#fbfdff" stroke="`+colorGrid+`" stroke-width="1"`)
	s.text(x+12, y+22, fmt.Sprintf("f%03d", frame.Index), `font-size="12" font-weight="700" fill="`+colorMuted+`"`)
	if len(frame.Triggered) > 0 {
		s.circle(x+float64(filmCellW)-17, y+17, 4.5, `fill="`+colorSpeed+`"`)
	}
	s.line(cellX+10, groundY, cellX+cellW-10, groundY, `stroke="#94a3b8" stroke-width="2" stroke-dasharray="5 5"`)

	centerX := x + filmCellW/2
	fmt.Fprintf(&s.Builder, `<g transform="translate(%s,%s) scale(%d,%d)">`,
		num(centerX), num(groundY), filmMeterPx, -filmMeterPx)
	writeSkeleton(s, frame.Skeleton)
	s.WriteString(`</g>`)

	s.text(centerX-18, y+188, fmt.Sprintf("φ %.2f", frame.Phase), `font-size="11" fill="`+colorMuted+`"`)
}

func writeSkeleton(s *svgBuilder, sk biped.Skeleton) {
	if sk.LeftFoot.Contact {
		s.ellipseM(sk.LeftFoot.Target.RelX, 0.012, 0.18, 0.026, `fill="`+colorLeft+`" opacity="0.16" stroke="none"`)
	}
	if sk.RightFoot.Contact {
		s.ellipseM(sk.RightFoot.Target.RelX, 0.018, 0.18, 0.026, `fill="`+colorRight+`" opacity="0.16" stroke="none"`)
	}

	writeLimb(s, sk.HipRight, sk.KneeRight, sk.AnkleRight, colorRight, 7, 0.72)
	writePathM(s, []biped.Vec2{sk.ShoulderRight, sk.ElbowRight, sk.HandRight}, colorRight, 5, 0.55)
	writeLimb(s, sk.HipLeft, sk.KneeLeft, sk.AnkleLeft, colorLeft, 7.5, 1)
	writePathM(s, []biped.Vec2{sk.ShoulderLeft, sk.ElbowLeft, sk.HandLeft}, colorLeft, 5.5, 0.9)

	writePathM(s, []biped.Vec2{sk.Pelvis, sk.Chest, sk.Neck}, colorInk, 9, 1)
	s.circleM(sk.Head.X, sk.Head.Y, sk.HeadRadius, `fill="#e2e8f0" stroke="`+colorInk+`" stroke-width="3" vector-effect="non-scaling-stroke"`)

	writeFoot(s, sk.RightFoot, colorRight, 0.78)
	writeFoot(s, sk.LeftFoot, colorLeft, 1)
	writeJoint(s, sk.Pelvis, 0.045, colorInk)
	writeJoint(s, sk.KneeLeft, 0.035, colorLeft)
	writeJoint(s, sk.KneeRight, 0.032, colorRight)
}

func writeLimb(s *svgBuilder, a, knee, b biped.Vec2, color string, width float64, opacity float64) {
	writePathM(s, []biped.Vec2{a, knee, b}, color, width, opacity)
}

func writePathM(s *svgBuilder, points []biped.Vec2, color string, width, opacity float64) {
	if len(points) < 2 {
		return
	}
	d := fmt.Sprintf("M %s %s", num(points[0].X), num(points[0].Y))
	for _, p := range points[1:] {
		d += fmt.Sprintf(" L %s %s", num(p.X), num(p.Y))
	}
	attrs := lineAttrs(color, width, fmt.Sprintf(`opacity="%.2f"`, opacity))
	s.path(d, attrs)
}

func writeFoot(s *svgBuilder, foot biped.FootPose, color string, opacity float64) {
	mid := biped.Vec2{X: (foot.Heel.X + foot.Toe.X) / 2, Y: (foot.Heel.Y + foot.Toe.Y) / 2}
	writePathM(s, []biped.Vec2{foot.Ankle, mid}, color, 4.5, opacity)
	writePathM(s, []biped.Vec2{foot.Heel, foot.Toe}, color, 7, opacity)
}

func writeJoint(s *svgBuilder, p biped.Vec2, radius float64, color string) {
	s.circleM(p.X, p.Y, radius, `fill="#ffffff" stroke="`+color+`" stroke-width="2.5" vector-effect="non-scaling-stroke"`)
}

func (s *svgBuilder) circleM(cx, cy, r float64, attrs string) {
	fmt.Fprintf(&s.Builder, `<circle cx="%s" cy="%s" r="%s" %s/>`, num(cx), num(cy), num(r), attrs)
}

func (s *svgBuilder) ellipseM(cx, cy, rx, ry float64, attrs string) {
	fmt.Fprintf(&s.Builder, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s" %s/>`, num(cx), num(cy), num(rx), num(ry), attrs)
}
