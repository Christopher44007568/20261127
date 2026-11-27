package main

import "testing"

func TestBrickGridTilingAlignment(t *testing.T) {
	for y := 0; y < TileH; y++ {
		for x := 0; x < TileW; x++ {
			a := BrickAt(x, y)
			bx := BrickAt(x+TileW, y)
			by := BrickAt(x, y+TileH)
			if bx.Col-a.Col != TileW/BrickW || bx.Lx != a.Lx || bx.Mortar != a.Mortar {
				t.Fatalf("horizontal misalignment at %d,%d: %+v vs %+v", x, y, a, bx)
			}
			if by.Row-a.Row != TileH/BrickH || by.Ly != a.Ly || by.Mortar != a.Mortar {
				t.Fatalf("vertical misalignment at %d,%d: %+v vs %+v", x, y, a, by)
			}
		}
	}
}

func TestTiledFBMIsPeriodic(t *testing.T) {
	for y := 0; y < TileH; y += 7 {
		n0 := TiledFBMOverTile(0, y, 77, 4, 16)
		nx := TiledFBMOverTile(TileW, y, 77, 4, 16)
		ny := TiledFBMOverTile(0, y+TileH, 77, 4, 16)
		if n0 != nx || n0 != ny {
			t.Fatalf("periodic noise not seamless at y=%d: %v %v %v", y, n0, nx, ny)
		}
	}
}

func TestOriginalTilingScoresMaximumRepetition(t *testing.T) {
	tile := BakeTile(1)
	img := TileRepeat(tile, 512, 512)
	hm := RepetitionHeat(img, TileW, TileH, 3, 26)
	if hm.MeanDelta != 0 {
		t.Fatalf("exact tiling must have zero period residual, got %v", hm.MeanDelta)
	}
	if hm.MeanScore != 1 {
		t.Fatalf("exact tiling must score 1, got %v", hm.MeanScore)
	}
}

func TestProcessedWallBreaksRepetition(t *testing.T) {
	tile := BakeTile(1)
	img := ProcessWall(tile, 512, 512, DefaultParams())
	hm := RepetitionHeat(img, TileW, TileH, 6, 26)
	if hm.MeanScore > 0.5 {
		t.Fatalf("processed wall still too repetitive: score %v", hm.MeanScore)
	}

	jO := MeasureJointContrast(TileRepeat(tile, 512, 512))
	jP := MeasureJointContrast(img)
	if jP.StrongFrac < jO.StrongFrac-0.1 {
		t.Fatalf("joint coverage degraded: orig %.2f proc %.2f", jO.StrongFrac, jP.StrongFrac)
	}

	_, _, seamRatio := BoundaryContinuity(img, TileW)
	if seamRatio < 0.7 || seamRatio > 1.3 {
		t.Fatalf("tile boundary discontinuity outside tolerance: ratio %.2f", seamRatio)
	}
}
