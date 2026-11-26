package main

import "math"

// Obstacle is an axis-aligned solid box in world space.
type Obstacle struct {
	Name string
	Rect Rect
}

// CollisionEvent describes one continuous camera/obstacle overlap found while
// sweeping the camera spline along the shared timeline.
type CollisionEvent struct {
	Obstacle  string  `json:"obstacle"`
	EnterTime float64 `json:"enterTime"`
	ExitTime  float64 `json:"exitTime"`
	MaxDepth  float64 `json:"maxPenetration"`
	AtMaxTime float64 `json:"atMaxTime"`
}

// OutOfBoundsEvent describes a continuous stretch where the camera center
// leaves the playable rectangle.
type OutOfBoundsEvent struct {
	EnterTime float64 `json:"enterTime"`
	ExitTime  float64 `json:"exitTime"`
}

// penetration is how far point p lies inside the obstacle along the shallow
// axis; zero means no overlap. The center of the box gives the maximum.
func penetration(p Vec2, r Rect) float64 {
	if p.X < r.Min.X || p.X > r.Max.X || p.Y < r.Min.Y || p.Y > r.Max.Y {
		return 0
	}
	dx := math.Min(p.X-r.Min.X, r.Max.X-p.X)
	dy := math.Min(p.Y-r.Min.Y, r.Max.Y-p.Y)
	return math.Min(dx, dy)
}

// SweepCollision samples the camera path at fixed time steps and merges
// consecutive overlapping samples into per-obstacle collision events.
func SweepCollision(path CameraPath, obstacles []Obstacle, duration, step float64) []CollisionEvent {
	active := map[string]int{}
	var events []CollisionEvent

	t := 0.0
	for t <= duration+1e-12 {
		pos := path.At(math.Min(t, duration)).Pos
		for _, ob := range obstacles {
			depth := penetration(pos, ob.Rect)
			idx, ok := active[ob.Name]
			if depth > 0 {
				if !ok {
					events = append(events, CollisionEvent{
						Obstacle: ob.Name, EnterTime: t, ExitTime: t, AtMaxTime: t,
					})
					idx = len(events) - 1
					active[ob.Name] = idx
				}
				events[idx].ExitTime = t
				if depth > events[idx].MaxDepth {
					events[idx].MaxDepth = depth
					events[idx].AtMaxTime = t
				}
			} else if ok {
				delete(active, ob.Name)
			}
		}
		t += step
	}
	return events
}

// SweepBounds reports continuous out-of-bounds stretches of the camera.
func SweepBounds(path CameraPath, bounds Rect, duration, step float64) []OutOfBoundsEvent {
	var events []OutOfBoundsEvent
	var cur *OutOfBoundsEvent
	t := 0.0
	for t <= duration+1e-12 {
		pos := path.At(math.Min(t, duration)).Pos
		outside := !bounds.Contains(pos)
		if outside {
			if cur == nil {
				events = append(events, OutOfBoundsEvent{EnterTime: t, ExitTime: t})
				cur = &events[len(events)-1]
			}
			cur.ExitTime = t
		} else {
			cur = nil
		}
		t += step
	}
	return events
}
