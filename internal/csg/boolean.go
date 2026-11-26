package csg

import (
	"sort"

	"meshbool/internal/mesh"
)

type Operation string

const (
	Union        Operation = "union"
	Intersection Operation = "intersection"
	Difference   Operation = "difference"
)

type Result struct {
	Operation Operation
	Mesh      mesh.Mesh
	Interior  mesh.Mesh
	InputA    int
	InputB    int
	Polygons  int
	Coplanar  int
}

// Apply performs a closed-mesh boolean using a BSP. The interior mesh is a
// diagnostic layer: source boundary pieces that lie inside the opposite
// operand. It makes clipped/internal faces visible without leaking them into
// the exported result.
func Apply(a, b mesh.Mesh, operation Operation) Result {
	a = a.OrientOutward()
	b = b.OrientOutward()
	polygonsA := meshPolygons(a, "A")
	polygonsB := meshPolygons(b, "B")

	treeA := NewNode(clonePolygons(polygonsA))
	treeB := NewNode(clonePolygons(polygonsB))
	var interior []Polygon

	switch operation {
	case Union:
		interior = append(
			insideFragments(polygonsA, treeB),
			insideFragments(polygonsB, treeA)...,
		)
		treeA.ClipTo(treeB)
		treeB.ClipTo(treeA)
		treeB.Invert()
		treeB.ClipTo(treeA)
		treeB.Invert()
		treeA.Build(treeB.AllPolygons())
	case Intersection:
		interior = nil
		treeA.Invert()
		treeB.ClipTo(treeA)
		treeB.Invert()
		treeA.ClipTo(treeB)
		treeB.ClipTo(treeA)
		treeA.Build(treeB.AllPolygons())
		treeA.Invert()
	case Difference:
		interior = insideFragments(polygonsA, treeB)
		treeA.Invert()
		treeA.ClipTo(treeB)
		treeB.ClipTo(treeA)
		treeB.Invert()
		treeB.ClipTo(treeA)
		treeB.Invert()
		treeA.Build(treeB.AllPolygons())
		treeA.Invert()
	default:
		panic("unknown boolean operation: " + string(operation))
	}

	resultPolygons := treeA.AllPolygons()
	result := mesh.Mesh{Triangles: polygonTriangles(resultPolygons)}
	result = result.HealTJunctions().UnifyNormals()
	result.SortTriangles()
	interiorMesh := mesh.Mesh{Triangles: polygonTriangles(interior)}
	interiorMesh = interiorMesh.HealTJunctions().UnifyNormals()
	interiorMesh.SortTriangles()

	return Result{
		Operation: operation,
		Mesh:      result,
		Interior:  interiorMesh,
		InputA:    len(a.Triangles),
		InputB:    len(b.Triangles),
		Polygons:  len(resultPolygons),
		Coplanar:  countCoplanar(resultPolygons),
	}
}

func meshPolygons(m mesh.Mesh, source string) []Polygon {
	out := make([]Polygon, 0, len(m.Triangles))
	for _, triangle := range m.Triangles {
		plane, ok := PlaneFromPoints(triangle.A, triangle.B, triangle.C)
		if !ok {
			continue
		}
		out = append(out, Polygon{
			Vertices: []mesh.Vec3{triangle.A, triangle.B, triangle.C},
			Plane:    plane,
			Source:   source,
		})
	}
	return out
}

func polygonTriangles(polygons []Polygon) []mesh.Triangle {
	var out []mesh.Triangle
	for _, polygon := range polygons {
		out = append(out, polygon.Triangles()...)
	}
	return out
}

func clonePolygons(polygons []Polygon) []Polygon {
	out := make([]Polygon, len(polygons))
	for i, polygon := range polygons {
		out[i] = polygon.Clone()
	}
	return out
}

// insideFragments clips source polygons against the inverted solid. Keeping
// the outside of the inverted solid is equivalent to keeping the inside of the
// original solid.
func insideFragments(source []Polygon, solid *Node) []Polygon {
	inverted := solid.Clone()
	inverted.Invert()
	return inverted.ClipPolygons(clonePolygons(source))
}

func countCoplanar(polygons []Polygon) int {
	type planeKey struct {
		NX, NY, NZ int64
		W          int64
	}
	groups := make(map[planeKey]int)
	for _, polygon := range polygons {
		key := planeKey{
			int64(polygon.Plane.Normal.X * 1e7),
			int64(polygon.Plane.Normal.Y * 1e7),
			int64(polygon.Plane.Normal.Z * 1e7),
			int64(polygon.Plane.W * 1e7),
		}
		groups[key]++
	}
	keys := make([]planeKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].NX != keys[j].NX {
			return keys[i].NX < keys[j].NX
		}
		if keys[i].NY != keys[j].NY {
			return keys[i].NY < keys[j].NY
		}
		if keys[i].NZ != keys[j].NZ {
			return keys[i].NZ < keys[j].NZ
		}
		return keys[i].W < keys[j].W
	})
	count := 0
	for _, key := range keys {
		if groups[key] > 1 {
			count += groups[key]
		}
	}
	return count
}
