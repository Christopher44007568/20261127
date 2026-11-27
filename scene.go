package main

// Object is a drawable scene participant. Bounds must conservatively cover
// every pixel Draw can touch; Version must increase on any visual change.
type Object interface {
	ID() string
	Bounds() Rect
	Version() int
	Draw(f *Framebuffer)
}

// Scene is the ordered (back-to-front) collection of objects plus the
// immutable static background buffer.
type Scene struct {
	Width   int
	Height  int
	Bg      *Framebuffer
	Objects []Object
}

func (s *Scene) add(o Object) { s.Objects = append(s.Objects, o) }

func canvasBounds(w, h int) Rect { return Rect{0, 0, w, h} }

// PaintScene paints background then every object, all clipped to clip.
// Full repaint calls this with the whole canvas; dirty repaint calls it once
// per merged dirty rectangle. Running the identical draw code in the same
// z-order is what guarantees pixel-identical results.
func PaintScene(fb *Framebuffer, sc *Scene, clip Rect) {
	savedClip := fb.clip
	fb.SetClip(clip)
	defer fb.SetClip(savedClip)

	// Restore the static background first: this erases whatever the dirty
	// rectangle used to contain (the "old dirty area" cleanup).
	effective := fb.Clip()
	fb.CopyRect(sc.Bg, effective)

	for _, o := range sc.Objects {
		// Cheap rejection: objects whose conservative bounds do not touch
		// this repaint rectangle cannot contribute any pixels to it.
		if o.Bounds().Intersect(effective).Empty() {
			continue
		}
		o.Draw(fb)
	}
}

// RenderFull performs the reference full-canvas repaint.
func RenderFull(sc *Scene) *Framebuffer {
	fb := NewFramebuffer(sc.Width, sc.Height)
	PaintScene(fb, sc, canvasBounds(sc.Width, sc.Height))
	return fb
}

type shapeKind int

const (
	shapeRect shapeKind = iota
	shapeDisk
	shapeDiamond
)

// ShapeObject is a draggable graphic with an optional cached coordinate label.
type ShapeObject struct {
	id      string
	kind    shapeKind
	cx, cy  int
	w, h    int // rect size
	rad     int // disk/diamond radius
	fill    Color
	border  Color
	borderW int

	label    *CachedText
	padX     int
	padY     int
	panelBg  Color
	panelBd  Color
	labelGap int

	ver    int
	bounds Rect
}

// NewShape constructs a draggable shape with a cached caption.
func NewShape(id string, kind shapeKind, cx, cy int, fill Color, font *Font, caption string) *ShapeObject {
	s := &ShapeObject{
		id:       id,
		kind:     kind,
		cx:       cx,
		cy:       cy,
		w:        132,
		h:        88,
		rad:      46,
		fill:     fill,
		border:   Color{R: 245, G: 248, B: 255},
		borderW:  4,
		label:    NewCachedText(font, 2, Color{R: 235, G: 240, B: 255}),
		padX:     9,
		padY:     6,
		panelBg:  Color{R: 16, G: 26, B: 52},
		panelBd:  Color{R: 96, G: 124, B: 184},
		labelGap: 10,
	}
	s.label.SetText(caption)
	s.bounds = s.computeBounds()
	return s
}

func (s *ShapeObject) ID() string         { return s.id }
func (s *ShapeObject) Bounds() Rect       { return s.bounds }
func (s *ShapeObject) Version() int       { return s.ver }
func (s *ShapeObject) Center() (int, int) { return s.cx, s.cy }

// SetCenter moves the shape and updates its cached caption. The object
// version increments only when something visible changed.
func (s *ShapeObject) SetCenter(cx, cy int, caption string) {
	changed := false
	if s.cx != cx || s.cy != cy {
		s.cx, s.cy = cx, cy
		changed = true
	}
	if s.label.SetText(caption) {
		changed = true
	}
	if changed {
		s.ver++
		s.bounds = s.computeBounds()
	}
}

func (s *ShapeObject) shapeBounds() Rect {
	expand := s.borderW + 1
	switch s.kind {
	case shapeRect:
		return RectWH(s.cx-s.w/2, s.cy-s.h/2, s.w, s.h).Inflate(expand)
	default:
		r := s.rad + s.borderW
		return Rect{X0: s.cx - r - 1, Y0: s.cy - r - 1, X1: s.cx + r + 2, Y1: s.cy + r + 2}
	}
}

func (s *ShapeObject) labelBounds() Rect {
	tw, th := s.label.Size()
	if tw == 0 {
		return Rect{}
	}
	pw := tw + 2*s.padX
	ph := th + 2*s.padY
	sh := s.shapeBounds()
	x0 := s.cx - pw/2
	y0 := sh.Y1 + s.labelGap
	return RectWH(x0, y0, pw, ph)
}

func (s *ShapeObject) computeBounds() Rect {
	b := s.shapeBounds().Union(s.labelBounds()).Inflate(1)
	return b
}

func (s *ShapeObject) Draw(f *Framebuffer) {
	s.drawShape(f)

	panel := s.labelBounds()
	if !panel.Empty() {
		f.FillRect(panel, s.panelBg)
		f.StrokeRect(panel, 2, s.panelBd)
		if sp := s.label.Surface(); sp != nil {
			_, th := s.label.Size()
			f.Blit(sp, panel.X0+s.padX, panel.Y0+(panel.Height()-th)/2)
		}
	}
}

func (s *ShapeObject) drawShape(f *Framebuffer) {
	switch s.kind {
	case shapeRect:
		outer := RectWH(s.cx-s.w/2, s.cy-s.h/2, s.w, s.h)
		f.FillRect(outer.Inflate(s.borderW), s.border)
		f.FillRect(outer, s.fill)
	case shapeDisk:
		f.FillDisk(s.cx, s.cy, s.rad+s.borderW, s.border)
		f.FillDisk(s.cx, s.cy, s.rad, s.fill)
	case shapeDiamond:
		fillDiamond(f, s.cx, s.cy, s.rad+s.borderW, s.border)
		fillDiamond(f, s.cx, s.cy, s.rad, s.fill)
	}
}

func fillDiamond(f *Framebuffer, cx, cy, rad int, c Color) {
	bounds := Rect{cx - rad, cy - rad, cx + rad + 1, cy + rad + 1}.Intersect(f.clip)
	if bounds.Empty() {
		return
	}
	for y := bounds.Y0; y < bounds.Y1; y++ {
		dy := y - cy
		if dy < 0 {
			dy = -dy
		}
		for x := bounds.X0; x < bounds.X1; x++ {
			dx := x - cx
			if dx < 0 {
				dx = -dx
			}
			if dx+dy <= rad {
				f.SetPixel(x, y, c)
			}
		}
	}
}

// LabelObject is a panel containing a cached text line, used for the status
// readout and HUD. AlignRight treats X as the right edge anchor.
type LabelObject struct {
	id         string
	x, y       int
	alignRight bool
	text       *CachedText
	padX       int
	padY       int
	bg         Color
	border     Color
	ver        int
	bounds     Rect
}

func NewLabel(id string, x, y int, alignRight bool, font *Font, scale int, fg Color, bg, border Color, initial string) *LabelObject {
	l := &LabelObject{
		id:         id,
		x:          x,
		y:          y,
		alignRight: alignRight,
		text:       NewCachedText(font, scale, fg),
		padX:       12,
		padY:       9,
		bg:         bg,
		border:     border,
	}
	l.text.SetText(initial)
	l.bounds = l.computeBounds()
	return l
}

func (l *LabelObject) ID() string   { return l.id }
func (l *LabelObject) Bounds() Rect { return l.bounds }
func (l *LabelObject) Version() int { return l.ver }

func (l *LabelObject) SetText(s string) {
	if l.text.SetText(s) {
		l.ver++
		l.bounds = l.computeBounds()
	}
}

func (l *LabelObject) computeBounds() Rect {
	tw, th := l.text.Size()
	w := tw + 2*l.padX
	h := th + 2*l.padY
	x0 := l.x
	if l.alignRight {
		x0 = l.x - w
	}
	return RectWH(x0, l.y, w, h).Inflate(1)
}

func (l *LabelObject) Draw(f *Framebuffer) {
	panel := l.bounds
	if panel.Empty() {
		return
	}
	f.FillRect(panel, l.bg)
	f.StrokeRect(panel, 2, l.border)
	if sp := l.text.Surface(); sp != nil {
		tw, th := l.text.Size()
		tx := panel.X0 + (panel.Width()-tw)/2
		ty := panel.Y0 + (panel.Height()-th)/2
		f.Blit(sp, tx, ty)
	}
}
