package main

import (
	"math"
	"testing"
)

func TestDispersionAndAbsorption(t *testing.T) {
	if !(RefractiveIndex(420) > RefractiveIndex(550)) {
		t.Fatal("blue must have larger refractive index than green")
	}
	if !(RefractiveIndex(550) > RefractiveIndex(680)) {
		t.Fatal("green must have larger refractive index than red")
	}
	near := BeerTransmission(530, 0.2)
	far := BeerTransmission(530, 2.0)
	if !(near > far && far > 0 && near <= 1) {
		t.Fatalf("Beer-Lambert transmission should decay smoothly: %v %v", near, far)
	}
}

func screenEndpointY(t *testing.T, shape Shape2, lambda, startY, screenX float64) float64 {
	t.Helper()
	world := defaultWorld()
	path := TraceBeam(
		v2(-6.35, startY), v2(1, 0), lambda,
		[]Shape2{shape},
		Rect{Min: world.Min, Max: v2(screenX, world.Max.Y)},
	)
	last := path.Points[len(path.Points)-1]
	if math.Abs(last.X-screenX) > 0.05 {
		t.Fatalf("ray did not reach screen x=%.2f, endpoint=%+v", screenX, last)
	}
	return last.Y
}

func TestPrismDispersionDirection(t *testing.T) {
	prism := NewPolygon([]Vec2{
		v2(-1.15, 2.15),
		v2(-2.75, -1.55),
		v2(0.45, -1.55),
	})
	screenX := 3.42
	blue := screenEndpointY(t, prism, 410, 0.52, screenX)
	red := screenEndpointY(t, prism, 680, 0.52, screenX)
	if !(blue < red-0.12) {
		t.Fatalf("blue should bend farther toward prism base (lower): blue=%.3f red=%.3f", blue, red)
	}
}

func sphereFocalX(lambda, height float64) float64 {
	sphere := Circle2{C: v2(-1.35, 0.22), R: 1.42}
	path := TraceBeam(
		v2(-6.35, 0.22+height), v2(1, 0), lambda,
		[]Shape2{sphere},
		Rect{Min: v2(-7, -50), Max: v2(7, 50)},
	)
	exit := path.Points[len(path.Points)-2]
	end := path.Points[len(path.Points)-1]
	axis := sphere.C.Y
	f := (axis - exit.Y) / (end.Y - exit.Y)
	return exit.X + f*(end.X-exit.X)
}

func TestSphereChromaticFocusOrder(t *testing.T) {
	blue := sphereFocalX(410, 0.28)
	red := sphereFocalX(680, 0.28)
	if !(blue < red-0.015) {
		t.Fatalf("blue focus should be closer to sphere (left): blue=%.3f red=%.3f", blue, red)
	}
}

func TestPlateParallelExitAndLateralOrder(t *testing.T) {
	plate := rotatedRect(v2(-1.25, 0), 1.18, 5.05, 38*math.Pi/180)
	world := defaultWorld()
	screenX := 4.78
	bluePath := TraceBeam(v2(-6.35, -0.1), v2(1, 0), 410, []Shape2{plate}, Rect{Min: world.Min, Max: v2(screenX, world.Max.Y)})
	redPath := TraceBeam(v2(-6.35, -0.1), v2(1, 0), 680, []Shape2{plate}, Rect{Min: world.Min, Max: v2(screenX, world.Max.Y)})
	blueEnd := bluePath.Points[len(bluePath.Points)-1]
	redEnd := redPath.Points[len(redPath.Points)-1]
	if math.Abs(blueEnd.Y-redEnd.Y) < 0.015 {
		t.Fatalf("plate should produce measurable lateral wavelength shift: %.4f", blueEnd.Y-redEnd.Y)
	}
	if !(blueEnd.Y > redEnd.Y) {
		t.Fatalf("for the chosen plate tilt blue should be displaced upward: blue=%.3f red=%.3f", blueEnd.Y, redEnd.Y)
	}
	blueDir := blueEnd.Sub(bluePath.Points[len(bluePath.Points)-2]).Norm()
	redDir := redEnd.Sub(redPath.Points[len(redPath.Points)-2]).Norm()
	if math.Abs(blueDir.Y) > 0.02 || math.Abs(redDir.Y) > 0.02 {
		t.Fatalf("parallel faces must restore direction: blue=%+v red=%+v", blueDir, redDir)
	}
}
