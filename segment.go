package main

import "fmt"

type Result struct {
	Grid              *Grid              `json:"-"`
	Options           SegmentOptions     `json:"options"`
	Seeds             []Seed             `json:"seeds"`
	AutoAddedSeeds    int                `json:"auto_added_seeds"`
	Feature           *CurvatureField    `json:"-"`
	Distance          []float64          `json:"-"`
	Labels            []int              `json:"labels"`
	SeedDistances     [][]float64        `json:"-"`
	Flips             int                `json:"relaxation_flips"`
	Energy            float64            `json:"energy"`
	RepairedCells     int                `json:"repaired_cells"`
	Connectivity      ConnectivityReport `json:"connectivity"`
	BoundaryEdgeCount int                `json:"boundary_edge_count"`
	BoundaryFeature   float64            `json:"boundary_feature_score"`
}

func Segment(g *Grid, requestedSeeds []Seed, optionValue *SegmentOptions) (*Result, error) {
	if g == nil {
		g = NewSyntheticGrid(192, 144)
	}
	opts := DefaultSegmentOptions()
	if optionValue != nil {
		opts = *optionValue
	}
	opts = opts.withDefaults()

	feature := ComputeCurvature(g)
	seeds := append([]Seed(nil), requestedSeeds...)
	autoAdded := 0
	var err error
	if len(seeds) == 0 {
		seeds, err = FarthestSeeds(g, opts.SeedCount)
		if err != nil {
			return nil, err
		}
	} else if opts.StrictSeeds {
		seeds, err = ValidateSeeds(g, seeds)
		if err != nil {
			return nil, err
		}
	} else {
		seeds, autoAdded, err = EnsureSeedCoverage(g, seeds)
		if err != nil {
			return nil, err
		}
	}

	geoOpts := DistanceOptions{
		EightNeighbors:  false,
		VerticalScale:   opts.VerticalScale,
		FeatureDiscount: opts.FeatureDiscount,
	}
	watershed, err := Watershed(g, seeds, feature.Feature, opts)
	if err != nil {
		return nil, fmt.Errorf("watershed distance field: %w", err)
	}
	labels := append([]int(nil), watershed.Label...)
	if _, repairErr := EnsureConnected(g, labels, seeds); repairErr != nil {
		return nil, repairErr
	}

	seedDistances := make([][]float64, len(seeds))
	for i, seed := range seeds {
		d, distErr := SingleSourceDistance(g, seed, feature.Feature, geoOpts)
		if distErr != nil {
			return nil, distErr
		}
		seedDistances[i] = d
	}

	flips, energy := RelaxBoundaries(g, labels, seeds, seedDistances, feature, opts)
	smoothOpts := opts
	smoothOpts.FeatureWeight = 0
	smoothOpts.GeoWeight = 0
	smoothOpts.RelaxIterations = opts.RelaxIterations/3 + 6
	smoothFlips, _ := RelaxBoundaries(g, labels, seeds, seedDistances, feature, smoothOpts)
	flips += smoothFlips
	repaired, err := EnsureConnected(g, labels, seeds)
	if err != nil {
		return nil, err
	}
	report := CheckConnectivity(g, labels, seeds)
	if !report.Connected {
		return nil, fmt.Errorf("connectivity guarantee failed: %v", report.Problems)
	}
	energy = LabelEnergy(g, labels, seedDistances, feature, opts)

	heatOpts := geoOpts
	heatOpts.EightNeighbors = !opts.FourNeighborHeatmap
	heatField, err := MultiSourceDistance(g, seeds, feature.Feature, heatOpts)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Grid:              g,
		Options:           opts,
		Seeds:             seeds,
		AutoAddedSeeds:    autoAdded,
		Feature:           feature,
		Distance:          heatField.Dist,
		Labels:            labels,
		SeedDistances:     seedDistances,
		Flips:             flips,
		Energy:            energy,
		RepairedCells:     repaired,
		Connectivity:      report,
		BoundaryEdgeCount: CountBoundaryEdges(g, labels),
	}
	_, result.BoundaryFeature = BoundaryFeatureScore(g, labels, feature)
	return result, nil
}

func CountBoundaryEdges(g *Grid, labels []int) int {
	count := 0
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				continue
			}
			if x+1 < g.W && !g.Blocked[id+1] && labels[id] != labels[id+1] {
				count++
			}
			if y+1 < g.H && !g.Blocked[id+g.W] && labels[id] != labels[id+g.W] {
				count++
			}
		}
	}
	return count
}
