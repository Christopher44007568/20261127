package main

import "testing"

func TestHudNeverEntersVelocityBuffer(t *testing.T) {
	sk := NewSkeleton()
	mesh := BuildHumanoid(sk)
	pose := Animate(10, frameCount)
	prev := Animate(9, frameCount)
	f := RenderFrame(mesh, sk, pose, prev)
	DrawHud(f)

	hudCount := 0
	for i := range f.Color {
		if f.Hud[i] {
			hudCount++
			if f.VelX[i] != 0 || f.VelY[i] != 0 {
				t.Fatalf("HUD pixel %d carries velocity (%v,%v)", i, f.VelX[i], f.VelY[i])
			}
		}
		if !f.Cover[i] && !f.Hud[i] && (f.VelX[i] != 0 || f.VelY[i] != 0) {
			t.Fatalf("background pixel %d carries velocity", i)
		}
	}
	if hudCount == 0 {
		t.Fatal("test scene produced no HUD pixels")
	}
}

func TestPerVertexVelocityExceedsRootOnWavingHand(t *testing.T) {
	sk := NewSkeleton()
	mesh := BuildHumanoid(sk)
	x0, y0, x1, y1 := ViewW/2, ViewH/4, ViewW, ViewH
	dominant := 0
	for f := 1; f < frameCount; f++ {
		skinned := RenderFrame(mesh, sk, Animate(f, frameCount), Animate(f-1, frameCount))
		root := RenderRootBaseline(mesh, sk, Animate(f, frameCount), Animate(f-1, frameCount))
		if RegionSpeed(skinned, x0, y0, x1, y1) > 2*RegionSpeed(root, x0, y0, x1, y1) {
			dominant++
		}
	}
	if dominant < frameCount/3 {
		t.Fatalf("only %d frames show joint-local velocity dominating root velocity", dominant)
	}
}
