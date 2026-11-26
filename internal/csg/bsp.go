package csg

// Node is a BSP tree whose front side is outside the solid and whose back
// side is inside. Leaf front regions are empty; leaf back regions are solid.
type Node struct {
	Plane    *Plane
	Front    *Node
	Back     *Node
	Polygons []Polygon
}

func NewNode(polygons []Polygon) *Node {
	node := &Node{}
	node.Build(polygons)
	return node
}

func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}
	out := &Node{Polygons: make([]Polygon, len(n.Polygons))}
	for i, polygon := range n.Polygons {
		out.Polygons[i] = polygon.Clone()
	}
	if n.Plane != nil {
		plane := *n.Plane
		out.Plane = &plane
	}
	out.Front = n.Front.Clone()
	out.Back = n.Back.Clone()
	return out
}

func (n *Node) Invert() {
	if n == nil {
		return
	}
	for i := range n.Polygons {
		n.Polygons[i] = n.Polygons[i].Flip()
	}
	if n.Plane != nil {
		plane := n.Plane.Flip()
		n.Plane = &plane
	}
	n.Front, n.Back = n.Back, n.Front
	n.Front.Invert()
	n.Back.Invert()
}

func (n *Node) ClipPolygons(polygons []Polygon) []Polygon {
	if n == nil || n.Plane == nil {
		return append([]Polygon(nil), polygons...)
	}
	var frontPolygons, backPolygons []Polygon
	for _, polygon := range polygons {
		splitPolygon(polygon, *n.Plane,
			&frontPolygons, &backPolygons,
			&frontPolygons, &backPolygons)
	}
	if n.Front != nil {
		frontPolygons = n.Front.ClipPolygons(frontPolygons)
	}
	if n.Back != nil {
		backPolygons = n.Back.ClipPolygons(backPolygons)
	} else {
		backPolygons = nil
	}
	return append(frontPolygons, backPolygons...)
}

func (n *Node) ClipTo(other *Node) {
	if n == nil {
		return
	}
	n.Polygons = other.ClipPolygons(n.Polygons)
	n.Front.ClipTo(other)
	n.Back.ClipTo(other)
}

func (n *Node) AllPolygons() []Polygon {
	if n == nil {
		return nil
	}
	out := make([]Polygon, 0, len(n.Polygons))
	for _, polygon := range n.Polygons {
		out = append(out, polygon.Clone())
	}
	out = append(out, n.Front.AllPolygons()...)
	out = append(out, n.Back.AllPolygons()...)
	return out
}

func (n *Node) Build(polygons []Polygon) {
	if len(polygons) == 0 {
		return
	}
	if n.Plane == nil {
		plane := polygons[0].Plane
		n.Plane = &plane
	}
	var frontPolygons, backPolygons []Polygon
	for _, polygon := range polygons {
		splitPolygon(polygon, *n.Plane,
			&n.Polygons, &n.Polygons,
			&frontPolygons, &backPolygons)
	}
	if len(frontPolygons) > 0 {
		if n.Front == nil {
			n.Front = &Node{}
		}
		n.Front.Build(frontPolygons)
	}
	if len(backPolygons) > 0 {
		if n.Back == nil {
			n.Back = &Node{}
		}
		n.Back.Build(backPolygons)
	}
}
