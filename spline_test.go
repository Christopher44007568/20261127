package main

import (
	"math"
	"testing"
)

func TestCameraSplinePassesThroughKeys(t *testing.T) {
	cam := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{0, 0}, Heading: 0, Zoom: 1},
		{Time: 1, Pos: Vec2{10, 10}, Heading: math.Pi, Zoom: 2},
		{Time: 2, Pos: Vec2{10, 10}, Heading: 0, Zoom: 1},
	}}
	for _, k := range cam.Keys {
		got := cam.At(k.Time)
		if Dist(got.Pos, k.Pos) > 1e-9 {
			t.Fatalf("spline at %.0fs = %v, want key %v", k.Time, got.Pos, k.Pos)
		}
	}
	// A fully collinear path (with end phantom points) must stay on the line.
	straight := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{0, 0}, Heading: 0, Zoom: 1},
		{Time: 1, Pos: Vec2{10, 0}, Heading: 0, Zoom: 1},
		{Time: 2, Pos: Vec2{20, 0}, Heading: 0, Zoom: 1},
	}}
	mid := straight.At(1.5)
	if mid.Pos.X <= 10 || mid.Pos.X >= 20 || math.Abs(mid.Pos.Y) > 1e-9 {
		t.Fatalf("collinear spline should stay on the segment, got %v", mid.Pos)
	}
	if mid.Zoom != 1 {
		t.Fatalf("constant zoom should hold, got %v", mid.Zoom)
	}
}

func TestAngleLerpShortestArc(t *testing.T) {
	got := AngleLerp(-3, 3, 0.5)
	if math.Abs(got-math.Pi) > 1e-9 {
		t.Fatalf("shortest-arc interpolant = %v, want pi", got)
	}
}

func TestCharacterSamplingAndPauses(t *testing.T) {
	tr := CharacterTrack{Name: "hero", Keys: []CharacterKey{
		{Time: 0, Pos: Vec2{0, 0}},
		{Time: 2, Pos: Vec2{10, 0}},
		{Time: 4, Pos: Vec2{10, 0}},
		{Time: 6, Pos: Vec2{20, 0}},
	}}
	if got := tr.At(1); Dist(got, Vec2{5, 0}) > 1e-9 {
		t.Fatalf("At(1) = %v, want (5,0)", got)
	}
	if got := tr.At(3); Dist(got, Vec2{10, 0}) > 1e-9 {
		t.Fatalf("paused At(3) = %v, want (10,0)", got)
	}
	if got := tr.At(99); Dist(got, Vec2{20, 0}) > 1e-9 {
		t.Fatalf("clamp after end = %v, want (20,0)", got)
	}
	pauses := tr.Pauses()
	if len(pauses) != 1 || pauses[0] != [2]float64{2, 4} {
		t.Fatalf("pauses = %v, want [[2 4]]", pauses)
	}
}

func TestPauseDoesNotFreezeCamera(t *testing.T) {
	tr := CharacterTrack{Keys: []CharacterKey{
		{Time: 0, Pos: Vec2{0, 0}},
		{Time: 2, Pos: Vec2{5, 0}},
		{Time: 4, Pos: Vec2{5, 0}}, // paused 2..4
	}}
	cam := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{0, 0}, Heading: 0, Zoom: 1},
		{Time: 4, Pos: Vec2{40, 0}, Heading: 0, Zoom: 1},
	}}
	if tr.At(2.5) != tr.At(3.5) {
		t.Fatal("character must hold still during pause")
	}
	if cam.At(2.5).Pos == cam.At(3.5).Pos {
		t.Fatal("camera must keep moving while the character pauses")
	}
}
