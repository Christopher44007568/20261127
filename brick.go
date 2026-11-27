package main

// Global wall/tile layout. The source tile is square and seamlessly
// tileable: TileW is an integer number of brick pitches and TileH is an
// even number of courses, which matches the running-bond row parity.
const (
	TileW = 256
	TileH = 256

	BrickW = 64 // horizontal pitch, including the vertical joint strip
	BrickH = 32 // vertical pitch, including the horizontal joint strip
	Joint  = 6  // mortar strip thickness in pixels

	OutW = 1024
	OutH = 1024
)

// BrickInfo describes one output/tile pixel in the running-bond grid.
type BrickInfo struct {
	Row int // world course index
	Col int // world brick-column index

	// Lx/Ly are continuous coordinates inside the pitch cell. For a
	// brick straddling the tile's left/right wrap edge the two fragments
	// share one Col and Lx runs continuously across the wrap.
	Lx, Ly int

	Mortar bool

	// FaceEdge is the distance (px) to the nearest brick-face edge;
	// zero on mortar. Used for bevel/AO shading only.
	FaceEdge float64
}

// BrickAt maps integer pixel coordinates to running-bond geometry.
// It is the single source of truth for mortar positions, used by both
// tile baking and the anti-repetition processor so structural joints
// can never drift between stages.
func BrickAt(x, y int) BrickInfo {
	row := floorDiv(y, BrickH)
	ly := y - row*BrickH

	shift := 0
	if row&1 == 1 {
		shift = BrickW / 2
	}
	cx := x + shift
	col := floorDiv(cx, BrickW)
	lx := cx - col*BrickW

	info := BrickInfo{Row: row, Col: col, Lx: lx, Ly: ly}

	if lx < Joint || ly < Joint {
		info.Mortar = true
		return info
	}

	dLeft := lx - Joint
	dRight := BrickW - 1 - lx
	dTop := ly - Joint
	dBottom := BrickH - 1 - ly
	info.FaceEdge = float64(min4(dLeft, dRight, dTop, dBottom))
	return info
}

func floorDiv(a, b int) int {
	q := a / b
	r := a % b
	if r != 0 && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func min4(a, b, c, d int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	if d < m {
		m = d
	}
	return m
}

// TileKey returns the wrap-stable identity of a brick inside the source
// tile. Edge-straddling fragments map to the same key.
func TileKey(info BrickInfo) (int, int) {
	kc := info.Col % (TileW / BrickW)
	if kc < 0 {
		kc += TileW / BrickW
	}
	kr := info.Row % (TileH / BrickH)
	if kr < 0 {
		kr += TileH / BrickH
	}
	return kc, kr
}
