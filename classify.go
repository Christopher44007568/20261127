package holefill

import (
	"fmt"
	"math"
)

type HoleClass int

const (
	ClassUnknown HoleClass = iota
	ClassOuter
	ClassOrdinaryHole
	ClassSlit
	ClassLNotch
	ClassClosePair
)

func (c HoleClass) String() string {
	switch c {
	case ClassOuter:
		return "outer-contour"
	case ClassOrdinaryHole:
		return "ordinary-hole"
	case ClassSlit:
		return "narrow-slit"
	case ClassLNotch:
		return "l-notch"
	case ClassClosePair:
		return "close-boundary-hole"
	default:
		return "unclassified"
	}
}

func (c HoleClass) Color() RGB {
	switch c {
	case ClassOuter:
		return RGB{226, 232, 240}
	case ClassSlit:
		return RGB{34, 211, 238}
	case ClassLNotch:
		return RGB{251, 146, 60}
	case ClassClosePair:
		return RGB{244, 114, 182}
	case ClassOrdinaryHole:
		return RGB{163, 230, 53}
	default:
		return RGB{148, 163, 184}
	}
}

type LoopMetrics struct {
	SignedArea    float64
	Area          float64
	HullArea      float64
	Convexity     float64
	Perimeter     float64
	Length        float64
	Width         float64
	Aspect        float64
	MinSelfGap    float64
	ReflexCorners int
	RightCorners  int
	MinPairGap    float64
	MinOuterGap   float64
	PairID        int
	Reason        string
}

type BoundaryLoop struct {
	ID       int
	Verts    []int
	Centroid Vec3
	Normal   Vec3
	U        Vec3
	V        Vec3
	Pts      []Vec2
	Metrics  LoopMetrics
	Class    HoleClass
}

func (l *BoundaryLoop) Radius() float64 {
	return math.Sqrt(math.Max(l.Metrics.Area, 1e-9) / math.Pi)
}

func (l *BoundaryLoop) Describe() string {
	m := l.Metrics
	return fmt.Sprintf(
		"loop=%d class=%s area=%.3f aspect=%.2f convexity=%.2f selfGap=%.3f pairGap=%.3f outerGap=%.3f reflex=%d pair=%d reason=%s",
		l.ID, l.Class, m.Area, m.Aspect, m.Convexity, m.MinSelfGap,
		m.MinPairGap, m.MinOuterGap, m.ReflexCorners, m.PairID, m.Reason,
	)
}
