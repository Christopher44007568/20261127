package scenes

import (
	"math"

	"meshbool/internal/csg"
	"meshbool/internal/mesh"
)

type Case struct {
	ID          string
	Number      string
	Title       string
	English     string
	Description string
	Names       [2]string
	A           mesh.Mesh
	B           mesh.Mesh
	Expected    map[csg.Operation]float64
	Tolerance   float64
	Tags        []string
}

func All() []Case {
	return []Case{boxes(), spherePlane(), coplanarSheets()}
}

func boxes() Case {
	a := mesh.Box(
		mesh.Vec3{X: -1.15, Y: -1, Z: -1},
		mesh.Vec3{X: 0.45, Y: 1, Z: 1},
	)
	b := mesh.Box(
		mesh.Vec3{X: -0.45, Y: -0.45, Z: -0.45},
		mesh.Vec3{X: 1.25, Y: 1.25, Z: 1.25},
	)
	intersection := 0.9 * 1.45 * 1.45
	volumeA := 1.6 * 2 * 2
	volumeB := 1.7 * 1.7 * 1.7
	return Case{
		ID:          "boxes",
		Number:      "01",
		Title:       "双盒相交",
		English:     "INTERSECTING BOXES",
		Description: "两组轴对齐盒体产生规则截面，用于核对体积守恒与六个方向的裁剪片。",
		Names:       [2]string{"BOX A · 1.6 × 2 × 2", "BOX B · 1.7³"},
		A:           a,
		B:           b,
		Expected: map[csg.Operation]float64{
			csg.Union:        volumeA + volumeB - intersection,
			csg.Intersection: intersection,
			csg.Difference:   volumeA - intersection,
		},
		Tolerance: 2e-5,
		Tags:      []string{"闭合流形", "规则截面", "体积守恒"},
	}
}

func spherePlane() Case {
	sphere := mesh.Sphere(mesh.Vec3{}, 1, 32, 16)
	cutHeight := 0.15
	cutter := mesh.Box(
		mesh.Vec3{X: -2, Y: -2, Z: -2},
		mesh.Vec3{X: 2, Y: cutHeight, Z: 2},
	)
	sphereVolume := 4 * math.Pi / 3
	upperCap := math.Pi * (2.0/3.0 - cutHeight + cutHeight*cutHeight*cutHeight/3.0)
	lowerPart := sphereVolume - upperCap
	boxVolume := 4 * (2 + cutHeight) * 4
	return Case{
		ID:          "sphere-plane",
		Number:      "02",
		Title:       "球切平面",
		English:     "SPHERE / PLANE CUT",
		Description: "球面与半空间盒相交，斜向相机直接检查切割圆、封口面与面片法线。",
		Names:       [2]string{"SPHERE · R=1", "HALF-SPACE · y=0.15"},
		A:           sphere,
		B:           cutter,
		Expected: map[csg.Operation]float64{
			csg.Union:        boxVolume + upperCap,
			csg.Intersection: lowerPart,
			csg.Difference:   upperCap,
		},
		Tolerance: 0.075,
		Tags:      []string{"曲面分裂", "封口面", "法线重定向"},
	}
}

func coplanarSheets() Case {
	a := mesh.Box(
		mesh.Vec3{X: -1.28, Y: -0.6, Z: -0.11},
		mesh.Vec3{X: 0.32, Y: 0.6, Z: 0.11},
	)
	b := mesh.Box(
		mesh.Vec3{X: -0.32, Y: -0.6, Z: -0.11},
		mesh.Vec3{X: 1.28, Y: 0.6, Z: 0.11},
	)
	lengthA := 1.6
	height := 1.2
	thickness := 0.22
	overlapX := 0.64
	volumeA := lengthA * height * thickness
	overlap := overlapX * height * thickness
	return Case{
		ID:          "coplanar-sheets",
		Number:      "03",
		Title:       "共边薄片",
		English:     "COPLANAR THIN SHEETS",
		Description: "两片薄板共享上下共面区域；保留单层表面，避免重叠面叠色与接缝空洞。",
		Names:       [2]string{"SHEET A · 1.6 × 1.2 × .22", "SHEET B · 1.6 × 1.2 × .22"},
		A:           a,
		B:           b,
		Expected: map[csg.Operation]float64{
			csg.Union:        volumeA*2 - overlap,
			csg.Intersection: overlap,
			csg.Difference:   volumeA - overlap,
		},
		Tolerance: 2e-5,
		Tags:      []string{"共面路由", "重复面剔除", "薄壁稳定"},
	}
}
