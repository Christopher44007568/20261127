package main

// Region is a list of dirty rectangles. Rectangles may overlap; call
// Merged to obtain disjoint repaint rectangles.
type Region struct {
	Rects []Rect
}

// Add appends a rectangle; empty rectangles are ignored.
func (r *Region) Add(rc Rect) {
	if !rc.Empty() {
		r.Rects = append(r.Rects, rc)
	}
}

// AddChange records both the previous and current bounds of a changing
// object. Including the old bounds is the essential "do not miss the old
// dirty area" rule: it guarantees trails are restored/repainted.
func (r *Region) AddChange(oldBounds, newBounds Rect) {
	r.Add(oldBounds)
	r.Add(newBounds)
}

// Merged returns pairwise disjoint conservative repaint rectangles.
func (r *Region) Merged() []Rect {
	return MergeRects(r.Rects)
}

// RawPixels sums the unmerged rectangle areas (may double-count overlaps).
func (r *Region) RawPixels() int {
	n := 0
	for _, rc := range r.Rects {
		n += rc.Pixels()
	}
	return n
}

// DirtyTracker remembers each object's last bounds and version, and produces
// a dirty region covering old+new bounds whenever an object changed.
type DirtyTracker struct {
	bounds   map[string]Rect
	versions map[string]int
}

func NewDirtyTracker() *DirtyTracker {
	return &DirtyTracker{bounds: map[string]Rect{}, versions: map[string]int{}}
}

// Snapshot records current state without producing a region. Use it after a
// full initial paint so the first drag frame only dirties real changes.
func (t *DirtyTracker) Snapshot(sc *Scene) {
	for _, o := range sc.Objects {
		t.bounds[o.ID()] = o.Bounds()
		t.versions[o.ID()] = o.Version()
	}
}

// Collect returns the dirty region for all changes since the previous
// Collect/Snapshot and then remembers the new state.
func (t *DirtyTracker) Collect(sc *Scene) Region {
	var reg Region
	for _, o := range sc.Objects {
		id := o.ID()
		b := o.Bounds()
		v := o.Version()
		oldB, hadB := t.bounds[id]
		oldV, hadV := t.versions[id]

		switch {
		case !hadB:
			// Brand new object: just its current bounds are needed.
			reg.Add(b)
		case !hadV || v != oldV || b != oldB:
			// Content changed or bounds moved. Always cover the old bounds
			// as well as the new ones, otherwise pixels at the previous
			// position keep showing the old graphic ("dirty trail").
			reg.AddChange(oldB, b)
		}

		t.bounds[id] = b
		t.versions[id] = v
	}
	return reg
}
