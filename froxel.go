package main

import (
	"math"
)

type GridConfig struct {
	TilesX int
	TilesY int
	Slices int
}

func DefaultGridConfig() GridConfig {
	return GridConfig{TilesX: 16, TilesY: 9, Slices: 14}
}

type FallbackReason int

const (
	NoFallback FallbackReason = iota
	WideFOVFallback
	DepthOutsideFallback
	DegenerateCellFallback
)

func (r FallbackReason) String() string {
	switch r {
	case WideFOVFallback:
		return "wide-fov"
	case DepthOutsideFallback:
		return "depth-outside-grid"
	case DegenerateCellFallback:
		return "degenerate-cell"
	default:
		return "none"
	}
}

type FroxelCell struct {
	Bounds   AABB
	Lights   []int
	Count    int
	Fallback bool
}

type FroxelStats struct {
	Lights        int
	Cells         int
	FallbackCells int
	Entries       int
	MaxCount      int
	MinCount      int
	WideFOV       bool
	HorizontalFOV float64
}

type FroxelGrid struct {
	Config    GridConfig
	Camera    *Camera
	Near      float64
	Far       float64
	Cells     []FroxelCell
	AllLights []int
	Stats     FroxelStats
	wideFOV   bool
}

func BuildFroxelGrid(scene *Scene, camera *Camera, cfg GridConfig) *FroxelGrid {
	cfg.TilesX = max(cfg.TilesX, 1)
	cfg.TilesY = max(cfg.TilesY, 1)
	cfg.Slices = max(cfg.Slices, 1)
	g := &FroxelGrid{
		Config: cfg,
		Camera: camera,
		Near:   camera.Near,
		Far:    camera.Far,
	}
	g.Cells = make([]FroxelCell, cfg.TilesX*cfg.TilesY*cfg.Slices)
	g.AllLights = make([]int, len(scene.Lights))
	for i := range g.AllLights {
		g.AllLights[i] = i
	}
	g.wideFOV = camera.FovY > 120*math.Pi/180 || camera.HorizontalFOV() > 150*math.Pi/180
	g.Stats = FroxelStats{
		Lights:        len(scene.Lights),
		Cells:         len(g.Cells),
		MinCount:      len(scene.Lights),
		WideFOV:       g.wideFOV,
		HorizontalFOV: camera.HorizontalFOV() * 180 / math.Pi,
	}
	viewCenters := make([]Vec3, len(scene.Lights))
	radii := make([]float64, len(scene.Lights))
	for i, light := range scene.Lights {
		center, radius := light.InfluenceSphere()
		viewCenters[i] = camera.View(center)
		radii[i] = radius
	}

	for tz := 0; tz < cfg.Slices; tz++ {
		z0, z1 := g.depthSliceBounds(tz)
		for ty := 0; ty < cfg.TilesY; ty++ {
			for tx := 0; tx < cfg.TilesX; tx++ {
				index := g.cellIndex(tx, ty, tz)
				bounds := g.clusterBounds(tx, ty, z0, z1)
				cell := &g.Cells[index]
				cell.Bounds = bounds
				if g.wideFOV || bounds.Empty() || math.IsNaN(bounds.Min.X) {
					cell.Fallback = true
					cell.Count = len(scene.Lights)
					g.Stats.FallbackCells++
				} else {
					for i := range scene.Lights {
						if sphereIntersectsAABB(viewCenters[i], radii[i], bounds) {
							cell.Lights = append(cell.Lights, i)
						}
					}
					cell.Count = len(cell.Lights)
					g.Stats.Entries += cell.Count
				}
				g.Stats.MaxCount = max(g.Stats.MaxCount, cell.Count)
				g.Stats.MinCount = min(g.Stats.MinCount, cell.Count)
			}
		}
	}
	return g
}

func (g *FroxelGrid) cellIndex(tx, ty, tz int) int {
	return (tz*g.Config.TilesY+ty)*g.Config.TilesX + tx
}

func (g *FroxelGrid) depthSliceBounds(tz int) (float64, float64) {
	d := float64(g.Config.Slices)
	z0 := g.Near * math.Pow(g.Far/g.Near, float64(tz)/d)
	z1 := g.Near * math.Pow(g.Far/g.Near, float64(tz+1)/d)
	return z0, z1
}

func (g *FroxelGrid) depthSliceFor(viewZ float64) (int, bool) {
	if math.IsNaN(viewZ) || viewZ < g.Near || viewZ > g.Far {
		return 0, false
	}
	u := math.Log(viewZ/g.Near) / math.Log(g.Far/g.Near)
	tz := int(u * float64(g.Config.Slices))
	if tz < 0 {
		tz = 0
	}
	if tz >= g.Config.Slices {
		tz = g.Config.Slices - 1
	}
	return tz, true
}

func (g *FroxelGrid) clusterBounds(tx, ty int, z0, z1 float64) AABB {
	cam := g.Camera
	tanHalf := math.Tan(cam.FovY * 0.5)
	ndcX0 := -1 + 2*float64(tx)/float64(g.Config.TilesX)
	ndcX1 := -1 + 2*float64(tx+1)/float64(g.Config.TilesX)
	ndcY0 := 1 - 2*float64(ty)/float64(g.Config.TilesY)
	ndcY1 := 1 - 2*float64(ty+1)/float64(g.Config.TilesY)
	kx0 := ndcX0 * tanHalf * cam.Aspect
	kx1 := ndcX1 * tanHalf * cam.Aspect
	ky0 := ndcY0 * tanHalf
	ky1 := ndcY1 * tanHalf
	result := emptyAABB()
	for _, z := range [2]float64{z0, z1} {
		for _, x := range [2]float64{kx0, kx1} {
			for _, y := range [2]float64{ky0, ky1} {
				p := V(x*z, y*z, z)
				result = result.Union(AABB{Min: p, Max: p})
			}
		}
	}
	return result
}

func (g *FroxelGrid) Query(px, py, width, height, viewZ float64) ([]int, int, FallbackReason) {
	if g.wideFOV {
		return g.AllLights, -1, WideFOVFallback
	}
	tx := clampInt(int(px/width*float64(g.Config.TilesX)), 0, g.Config.TilesX-1)
	ty := clampInt(int(py/height*float64(g.Config.TilesY)), 0, g.Config.TilesY-1)
	tz, ok := g.depthSliceFor(viewZ)
	if !ok {
		return g.AllLights, -1, DepthOutsideFallback
	}
	index := g.cellIndex(tx, ty, tz)
	cell := &g.Cells[index]
	if cell.Fallback {
		return g.AllLights, index, DegenerateCellFallback
	}
	return cell.Lights, index, NoFallback
}

func (g *FroxelGrid) InspectCell(tx, ty, tz int) (FroxelCell, bool) {
	if tx < 0 || tx >= g.Config.TilesX || ty < 0 || ty >= g.Config.TilesY || tz < 0 || tz >= g.Config.Slices {
		return FroxelCell{}, false
	}
	return g.Cells[g.cellIndex(tx, ty, tz)], true
}
