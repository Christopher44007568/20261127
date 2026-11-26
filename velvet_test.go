package main

import "testing"

func TestVelvetEnergyConstraints(t *testing.T) {
	opts := RenderOptions{Width: 240, Height: 200, AA: 1, Exposure: 3.15}
	camera := makeCamera(V(0.38, 0.16, 5.75), V(0, -0.01, 0), 39)
	material := velvetMaterial()
	views := velvetViews()

	results := make(map[string]RenderResult)
	for _, view := range views {
		results[view.Name] = renderView(camera, view, material, opts)
	}
	front := results["front"].Metrics
	backlit := results["backlit"].Metrics
	rotated := compareFiberRotation(results["front"], results["rotation"])

	if backlit.MaxLinear > material.BacklightBudget+1e-9 {
		t.Fatalf("backlight peak %.3f exceeds budget %.3f", backlit.MaxLinear, material.BacklightBudget)
	}
	if backlit.OverOneFraction > 0 {
		t.Fatalf("backlight produced clipped samples: %.4f", backlit.OverOneFraction)
	}
	if backlit.EdgeInteriorRatio < 4 {
		t.Fatalf("backlight edge/interior ratio too low: %.2f", backlit.EdgeInteriorRatio)
	}
	if backlit.MeanClothLinear < 1.15*front.MeanClothLinear ||
		backlit.MeanClothLinear > 2.5*front.MeanClothLinear {
		t.Fatalf("backlight energy out of range: front %.5f backlit %.5f",
			front.MeanClothLinear, backlit.MeanClothLinear)
	}
	if front.EdgeInteriorRatio > 0.90 {
		t.Fatalf("front edge should remain dark: %.2f", front.EdgeInteriorRatio)
	}
	if rotated.MeanRatio < 0.78 || rotated.MeanRatio > 1.12 {
		t.Fatalf("fiber rotation changed whole-cloth energy too much: %.3f", rotated.MeanRatio)
	}
}
