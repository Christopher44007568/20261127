package main

import "testing"

func TestShadingDoesNotMutateGeometryOrSilhouette(t *testing.T) {
	cam := defaultCamera()
	g := renderGBuffer(cam, 128, 80, 72, 58)
	depth := append([]float64(nil), g.Depth...)
	pos := append([]Vec3(nil), g.Pos...)
	valid := append([]bool(nil), g.Valid...)
	height := append([]float64(nil), g.GeoH...)

	for _, mode := range []ShadeMode{
		ShadeHeight,
		ShadeNaive,
		ShadeMaterialMud,
		ShadeMaterialMoss,
		ShadeMaterialStone,
		ShadeReflectionWet,
		ShadeReflectionDiff,
	} {
		_ = shadeImage(cam, g, mode)
	}

	for i := range depth {
		if g.Depth[i] != depth[i] || g.Pos[i] != pos[i] || g.Valid[i] != valid[i] || g.GeoH[i] != height[i] {
			t.Fatalf("shading mutated geometry at pixel %d", i)
		}
	}
}

func TestWaterlineBandIsLocal(t *testing.T) {
	atLine := wetnessAt(0.4, -0.3, waterLevel)
	above := wetnessAt(0.4, -0.3, waterLevel+0.22)
	below := wetnessAt(0.4, -0.3, waterLevel-0.12)
	if atLine < 0.65 {
		t.Fatalf("waterline wetness too low: %v", atLine)
	}
	if above > 0.05 {
		t.Fatalf("wetness leaks too far above the line: %v", above)
	}
	if below < 0.65 {
		t.Fatalf("submerged surface should stay wet: %v", below)
	}
}
