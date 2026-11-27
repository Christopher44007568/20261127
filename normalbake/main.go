package main

import (
	"fmt"
	"log"
	"time"
)

func stats(name string, r *BakeResult) {
	var cov, mir, occ, zero int
	for i := 0; i < r.W*r.H; i++ {
		if r.Covered[i] {
			cov++
		}
		if r.Mirror[i] {
			mir++
		}
		if r.Occl[i] {
			occ++
		}
		if r.Zero[i] {
			zero++
		}
	}
	fmt.Printf("%s: covered=%d mirrored=%d occluded=%d zero-hit=%d\n", name, cov, mir, occ, zero)
}

func main() {
	start := time.Now()

	low := genTorus(96, 48, 1.0, 0.5, dimpleDisp, true)
	high := genTorus(256, 128, 1.0, 0.5, func(u, v float64) float64 {
		return dimpleDisp(u, v) + detailDisp(u, v)
	}, false)
	fmt.Printf("low: %d tris, high: %d tris\n", len(low.Tris), len(high.Tris))

	if err := writeOBJ("low.obj", low); err != nil {
		log.Fatal(err)
	}
	if err := writeOBJ("high.obj", high); err != nil {
		log.Fatal(err)
	}

	lowBVH := NewBVH(low)
	highBVH := NewBVH(high)

	const res = 512
	const cage = 0.12
	const maxDist = 0.25

	nearest := Bake(low, high, lowBVH, highBVH, res, MethodNearest, cage, maxDist)
	stats("nearest", nearest)
	caged := Bake(low, high, lowBVH, highBVH, res, MethodCage, cage, maxDist)
	stats("cage", caged)

	must := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	must(SaveNormalMap(nearest, "normalmap_nearest.png"))
	must(SaveNormalMap(caged, "normalmap_cage.png"))
	must(SaveDiag(nearest, "diag_nearest.png"))
	must(SaveDiag(caged, "diag_cage.png"))
	must(SaveNormalSphere(nearest, "normal_sphere.png", 512))
	must(SaveCompare(low, high, nearest, "compare.png", 512))

	fmt.Printf("done in %s\n", time.Since(start).Round(time.Millisecond))
}
