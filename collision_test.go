package main

import "testing"

func TestSweepCollisionDetectsObstacle(t *testing.T) {
	cam := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{0, 5}, Heading: 0, Zoom: 1},
		{Time: 4, Pos: Vec2{20, 5}, Heading: 0, Zoom: 1},
	}}
	ob := []Obstacle{{Name: "block", Rect: Rect{Min: Vec2{8, 0}, Max: Vec2{12, 10}}}}
	events := SweepCollision(cam, ob, 4, 0.01)
	if len(events) != 1 {
		t.Fatalf("got %d collision events, want 1", len(events))
	}
	ev := events[0]
	if ev.Obstacle != "block" || !(ev.EnterTime > 1.5 && ev.EnterTime < 1.9) ||
		!(ev.ExitTime > 2.1 && ev.ExitTime < 2.5) {
		t.Fatalf("unexpected collision window: %+v", ev)
	}
	if ev.MaxDepth <= 0 {
		t.Fatal("max penetration must be positive inside the obstacle")
	}
}

func TestSweepCollisionClearPath(t *testing.T) {
	cam := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{0, 0}, Heading: 0, Zoom: 1},
		{Time: 2, Pos: Vec2{20, 0}, Heading: 0, Zoom: 1},
	}}
	ob := []Obstacle{{Name: "block", Rect: Rect{Min: Vec2{8, 5}, Max: Vec2{12, 10}}}}
	if ev := SweepCollision(cam, ob, 2, 0.01); len(ev) != 0 {
		t.Fatalf("clear path reported collisions: %+v", ev)
	}
}

func TestSweepBounds(t *testing.T) {
	bounds := Rect{Min: Vec2{0, 0}, Max: Vec2{10, 10}}
	cam := CameraPath{Keys: []CameraKey{
		{Time: 0, Pos: Vec2{5, 5}, Heading: 0, Zoom: 1},
		{Time: 2, Pos: Vec2{-5, 5}, Heading: 0, Zoom: 1},
		{Time: 4, Pos: Vec2{5, 5}, Heading: 0, Zoom: 1},
	}}
	events := SweepBounds(cam, bounds, 4, 0.01)
	if len(events) != 1 {
		t.Fatalf("got %d out-of-bounds events, want 1", len(events))
	}
	if events[0].EnterTime <= 0 || events[0].ExitTime >= 4 {
		t.Fatalf("out-of-bounds window should be interior, got %+v", events[0])
	}
}
