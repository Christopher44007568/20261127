package main

type BoundarySegment struct {
	X1     float64 `json:"x1"`
	Y1     float64 `json:"y1"`
	X2     float64 `json:"x2"`
	Y2     float64 `json:"y2"`
	LabelA int     `json:"label_a"`
	LabelB int     `json:"label_b"`
}

// ExtractBoundarySegments returns shared open-cell edges where labels differ.
// Edges adjacent only to holes are not region boundaries.
func ExtractBoundarySegments(g *Grid, labels []int) []BoundarySegment {
	segments := make([]BoundarySegment, 0, 256)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				continue
			}
			if x+1 < g.W && !g.Blocked[id+1] {
				right := id + 1
				if labels[id] != labels[right] {
					segments = append(segments, BoundarySegment{
						X1: float64(x + 1), Y1: float64(y),
						X2: float64(x + 1), Y2: float64(y + 1),
						LabelA: labels[id], LabelB: labels[right],
					})
				}
			}
			if y+1 < g.H && !g.Blocked[id+g.W] {
				down := id + g.W
				if labels[id] != labels[down] {
					segments = append(segments, BoundarySegment{
						X1: float64(x), Y1: float64(y + 1),
						X2: float64(x + 1), Y2: float64(y + 1),
						LabelA: labels[id], LabelB: labels[down],
					})
				}
			}
		}
	}
	return segments
}

func BoundaryFeatureScore(g *Grid, labels []int, curvature *CurvatureField) (int, float64) {
	count := 0
	sum := 0.0
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				continue
			}
			if x+1 < g.W && !g.Blocked[id+1] && labels[id] != labels[id+1] {
				count++
				sum += orientedEdgeFeature(g, curvature, x, y, x+1, y)
			}
			if y+1 < g.H && !g.Blocked[id+g.W] && labels[id] != labels[id+g.W] {
				count++
				sum += orientedEdgeFeature(g, curvature, x, y, x, y+1)
			}
		}
	}
	if count == 0 {
		return 0, 0
	}
	return count, sum / float64(count)
}
