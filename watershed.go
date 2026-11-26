package main

// Watershed performs seeded geodesic watershed flooding on the grid graph.
// Four-neighbor propagation is intentional: every label predecessor edge is a
// real cell adjacency, so the initial basins cannot connect diagonally across
// a hole.
func Watershed(g *Grid, seeds []Seed, feature []float64, opts SegmentOptions) (*DistanceField, error) {
	distanceOptions := DistanceOptions{
		EightNeighbors:  false,
		VerticalScale:   opts.VerticalScale,
		FeatureDiscount: opts.FeatureDiscount,
	}
	return MultiSourceDistance(g, seeds, feature, distanceOptions)
}
