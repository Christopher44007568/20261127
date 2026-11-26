package main

import "math"

type FusionConfig struct {
	VoxelSize   float64
	NX, NY, NZ  int
	Min         Vec3
	Truncation  float64
	MaxWeight   float32
	MinWeight   float32
	RobustDelta float32
	UseRobust   bool
}

type VoxelGrid struct {
	Min        Vec3
	Voxel      float64
	NX, NY, NZ int
	Truncation float64
	Value      []float32
	Weight     []float32
}

func NewVoxelGrid(cfg FusionConfig) *VoxelGrid {
	n := cfg.NX * cfg.NY * cfg.NZ
	return &VoxelGrid{
		Min:        cfg.Min,
		Voxel:      cfg.VoxelSize,
		NX:         cfg.NX,
		NY:         cfg.NY,
		NZ:         cfg.NZ,
		Truncation: cfg.Truncation,
		Value:      make([]float32, n),
		Weight:     make([]float32, n),
	}
}

func (g *VoxelGrid) Index(ix, iy, iz int) int {
	return ix + g.NX*(iy+g.NY*iz)
}

func (g *VoxelGrid) InBounds(ix, iy, iz int) bool {
	return ix >= 0 && ix < g.NX && iy >= 0 && iy < g.NY && iz >= 0 && iz < g.NZ
}

func (g *VoxelGrid) Center(ix, iy, iz int) Vec3 {
	h := g.Voxel
	return V3(
		g.Min.X+(float64(ix)+0.5)*h,
		g.Min.Y+(float64(iy)+0.5)*h,
		g.Min.Z+(float64(iz)+0.5)*h,
	)
}

func (g *VoxelGrid) WorldToIndex(p Vec3) (float64, float64, float64) {
	return (p.X-g.Min.X)/g.Voxel - 0.5,
		(p.Y-g.Min.Y)/g.Voxel - 0.5,
		(p.Z-g.Min.Z)/g.Voxel - 0.5
}

func (g *VoxelGrid) MetricSDF(index int) float64 {
	return float64(g.Value[index]) * g.Truncation
}

// UpdateWeight performs the standard incremental weighted TSDF mean.
// The stored cumulative confidence is explicitly saturated at maxWeight.
func UpdateWeight(oldWeight, oldValue, sample, addWeight, maxWeight float32) (newWeight, newValue float32) {
	if addWeight <= 0 || math.IsNaN(float64(addWeight)) || math.IsInf(float64(addWeight), 0) {
		return oldWeight, oldValue
	}
	if oldWeight <= 0 {
		newWeight = addWeight
		if maxWeight > 0 && newWeight > maxWeight {
			newWeight = maxWeight
		}
		return newWeight, sample
	}
	total := oldWeight + addWeight
	newValue = (oldValue*oldWeight + sample*addWeight) / total
	newWeight = total
	if maxWeight > 0 && newWeight > maxWeight {
		newWeight = maxWeight
	}
	return newWeight, newValue
}

func robustAgreement(oldValue, sample, delta float32) float32 {
	if delta <= 0 {
		return 1
	}
	residual := float32(math.Abs(float64(sample - oldValue)))
	if residual <= delta {
		return 1
	}
	return delta / residual
}

func FuseFrame(grid *VoxelGrid, frame *DepthFrame, cfg FusionConfig) (updated int) {
	maxW := cfg.MaxWeight
	if maxW <= 0 {
		maxW = 20
	}
	minW := cfg.MinWeight
	if minW <= 0 {
		minW = 0.2
	}

	for iz := 0; iz < grid.NZ; iz++ {
		for iy := 0; iy < grid.NY; iy++ {
			for ix := 0; ix < grid.NX; ix++ {
				p := grid.Center(ix, iy, iz)
				u, v, camZ, visible := frame.Fusion.Project(p)
				if !visible {
					continue
				}
				px := clampInt(int(math.Floor(u)), 0, frame.Width-1)
				py := clampInt(int(math.Floor(v)), 0, frame.Height-1)
				di := py*frame.Width + px
				depth := frame.Depth[di]
				if depth <= 0 {
					continue
				}

				metric := depth - camZ
				if n := frame.Normal[di]; n.Len() > 1e-6 {
					surfacePoint := frame.Fusion.UnprojectDepth(px, py, depth)
					planeMetric := n.Dot(p.Sub(surfacePoint))
					if absFloat(planeMetric) < grid.Truncation*1.7 {
						metric = planeMetric
					}
				}
				if metric < -grid.Truncation || metric > grid.Truncation {
					continue
				}
				sample := float32(clampFloat(metric/grid.Truncation, -1, 1))
				weight := frame.ConfidenceAt(px, py)
				if weight <= 0 {
					weight = 1
				}
				id := grid.Index(ix, iy, iz)
				if cfg.UseRobust && grid.Weight[id] >= minW {
					weight *= float64(robustAgreement(grid.Value[id], sample, cfg.RobustDelta))
				}
				w, val := UpdateWeight(grid.Weight[id], grid.Value[id], sample, float32(weight), maxW)
				grid.Weight[id] = w
				grid.Value[id] = val
				updated++
			}
		}
	}
	return updated
}
