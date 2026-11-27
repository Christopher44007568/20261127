package main

import (
	"fmt"
	"math"
	"sort"
)

type ErrorBin struct {
	Depth          int
	H              float64
	Count          int
	MaxSDF         float64
	SumSDF         float64
	MaxFeatureDist float64
	SumFeatureDist float64
	BaselineMax    float64
	BaselineSum    float64
}

func (b ErrorBin) MeanSDF() float64 {
	if b.Count == 0 {
		return 0
	}
	return b.SumSDF / float64(b.Count)
}

func (b ErrorBin) MeanFeatureDist() float64 {
	if b.Count == 0 {
		return 0
	}
	return b.SumFeatureDist / float64(b.Count)
}

type Marker struct {
	Pos         Vec3
	H           float64
	Depth       int
	Err         float64
	BaselineErr float64
	FeatureDist float64
	Label       string
	Inner       bool
}

type SceneResult struct {
	Scene         Scene
	Oct           *Octree
	Top           *Topology
	SharpMesh     *Mesh
	BaselineMesh  *Mesh
	UsedIDs       []int
	LeafCount     int
	Histogram     map[int]int
	VertexCount   int
	MaxSDF        float64
	BaselineMax   float64
	SharpBins     []ErrorBin
	AllBins       []ErrorBin
	Markers       []Marker
	WorstMarker   *Marker
	NaiveBoundary int
	RawBoundary   int
	FinalBoundary int
}

func BuildMetrics(scene Scene, o *Octree, top *Topology) *SceneResult {
	sharp := NewMesh(o, top, true)
	baseline := NewMesh(o, top, false)
	usedSet := map[int]bool{}
	for _, tri := range top.Tris {
		for _, id := range tri.V {
			usedSet[id] = true
		}
	}
	used := make([]int, 0, len(usedSet))
	for id := range usedSet {
		used = append(used, id)
	}
	sort.Ints(used)

	res := &SceneResult{
		Scene:        scene,
		Oct:          o,
		Top:          top,
		SharpMesh:    sharp,
		BaselineMesh: baseline,
		UsedIDs:      used,
		LeafCount:    len(o.Leaves),
		Histogram:    o.DepthHistogram(),
		VertexCount:  len(used),
	}

	sharpBinMap := map[int]*ErrorBin{}
	allBinMap := map[int]*ErrorBin{}
	var worst Marker
	worstErr := -1.0

	for _, id := range used {
		n := o.Leaves[id]
		p := n.Sharp
		err := math.Abs(scene.SDF(p))
		baseErr := math.Abs(scene.SDF(n.Baseline))
		fd := scene.FeatureDistance(p)
		h := n.H
		res.MaxSDF = math.Max(res.MaxSDF, err)
		res.BaselineMax = math.Max(res.BaselineMax, baseErr)

		all := binFor(allBinMap, n.Depth, h)
		all.Count++
		all.SumSDF += err
		all.MaxSDF = math.Max(all.MaxSDF, err)

		isSharp := scene.Kind != KindSphere && fd <= 0.75*h
		if isSharp {
			bin := binFor(sharpBinMap, n.Depth, h)
			bin.Count++
			bin.SumSDF += err
			bin.MaxSDF = math.Max(bin.MaxSDF, err)
			bin.SumFeatureDist += fd
			bin.MaxFeatureDist = math.Max(bin.MaxFeatureDist, fd)
			bin.BaselineSum += baseErr
			bin.BaselineMax = math.Max(bin.BaselineMax, baseErr)

			if err > worstErr {
				worstErr = err
				worst = Marker{
					Pos: p, H: h, Depth: n.Depth, Err: err,
					BaselineErr: baseErr, FeatureDist: fd,
					Label: "worst sharp",
				}
			}
		}
	}

	res.SharpBins = finalizeBins(sharpBinMap)
	res.AllBins = finalizeBins(allBinMap)
	if worstErr >= 0 {
		res.WorstMarker = &worst
	}
	res.Markers = buildCornerMarkers(scene, o, usedSet)
	return res
}

func binFor(m map[int]*ErrorBin, depth int, h float64) *ErrorBin {
	b, ok := m[depth]
	if !ok {
		b = &ErrorBin{Depth: depth, H: h}
		m[depth] = b
	}
	return b
}

func finalizeBins(m map[int]*ErrorBin) []ErrorBin {
	out := make([]ErrorBin, 0, len(m))
	for _, b := range m {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Depth < out[j].Depth })
	return out
}

func buildCornerMarkers(scene Scene, o *Octree, used map[int]bool) []Marker {
	if scene.Kind == KindSphere {
		return nil
	}
	type target struct {
		p     Vec3
		inner bool
		name  string
	}
	var targets []target
	signs := []float64{-1, 1}
	for _, x := range signs {
		for _, y := range signs {
			for _, z := range signs {
				b := scene.Half
				p := V(x*b.X, y*b.Y, z*b.Z)
				targets = append(targets, target{p: p, name: "outer corner"})
				if scene.Kind == KindShell {
					in := b.Sub(V(scene.Thickness, scene.Thickness, scene.Thickness))
					pi := V(x*in.X, y*in.Y, z*in.Z)
					targets = append(targets, target{p: pi, inner: true, name: "inner corner"})
				}
			}
		}
	}

	var markers []Marker
	for _, t := range targets {
		bestID := -1
		bestD := math.Inf(1)
		for id := range used {
			d := o.Leaves[id].Sharp.Sub(t.p).Norm2()
			if d < bestD {
				bestD = d
				bestID = id
			}
		}
		if bestID < 0 {
			continue
		}
		n := o.Leaves[bestID]
		err := math.Abs(scene.SDF(n.Sharp))
		baseErr := math.Abs(scene.SDF(n.Baseline))
		markers = append(markers, Marker{
			Pos: n.Sharp, H: n.H, Depth: n.Depth, Err: err,
			BaselineErr: baseErr,
			FeatureDist: scene.FeatureDistance(n.Sharp),
			Label:       fmt.Sprintf("h=%.4f e=%.5f", n.H, err),
			Inner:       t.inner,
		})
	}
	return markers
}
