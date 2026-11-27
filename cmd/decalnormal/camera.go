package main

// Camera is an orthographic camera whose view direction points down toward
// the heightfield. Orthographic projection means one world unit maps to the
// same pixel count everywhere, which keeps moving decals at a stable scale.
type Camera struct {
	Origin   Vec3
	Right    Vec3
	Up       Vec3
	Forward  Vec3
	ViewH    float64
	ViewW    float64
	WidthPx  int
	HeightPx int
}

func newCamera(origin, lookAt Vec3, viewW, viewH float64, widthPx, heightPx int) Camera {
	forward := lookAt.Sub(origin).Normalized()
	worldUp := v3(0, 0, 1)
	right := forward.Cross(worldUp).Normalized()
	up := right.Cross(forward).Normalized()
	return Camera{
		Origin:   origin,
		Right:    right,
		Up:       up,
		Forward:  forward,
		ViewH:    viewH,
		ViewW:    viewW,
		WidthPx:  widthPx,
		HeightPx: heightPx,
	}
}

// ray returns the ray through pixel (px, py), py growing upward.
func (c Camera) ray(px, py int) (Vec3, Vec3) {
	sx := (float64(px)+0.5)/float64(c.WidthPx) - 0.5
	sy := (float64(py)+0.5)/float64(c.HeightPx) - 0.5
	origin := c.Origin.
		Add(c.Right.Scale(sx * c.ViewW)).
		Add(c.Up.Scale(sy * c.ViewH))
	return origin, c.Forward
}
