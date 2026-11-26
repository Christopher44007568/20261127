package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func sceneBounds(sk *Skeleton) (Vec3, float64) {
	lo := Vec3{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := Vec3{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, b := range sk.Branches {
		for _, p := range b.Points {
			lo.X, lo.Y, lo.Z = math.Min(lo.X, p.X), math.Min(lo.Y, p.Y), math.Min(lo.Z, p.Z)
			hi.X, hi.Y, hi.Z = math.Max(hi.X, p.X), math.Max(hi.Y, p.Y), math.Max(hi.Z, p.Z)
		}
	}
	c := lo.Add(hi).Scale(0.5)
	return c, lo.Dist(hi) / 2
}

func meanRadius(sk *Skeleton) float64 {
	sum, n := 0.0, 0
	for _, b := range sk.Branches {
		for _, r := range b.Radii {
			sum += r
			n++
		}
	}
	if n == 0 {
		return 0.1
	}
	return sum / float64(n)
}

func main() {
	input := flag.String("input", "", "skeleton JSON file (empty: use built-in demo)")
	outDir := flag.String("out", "out", "output directory")
	step := flag.Float64("step", 0, "resampling step (0: auto = mean radius / 5)")
	sides := flag.Int("sides", 28, "cross-section sides")
	smooth := flag.Int("smooth", 10, "centerline smoothing iterations")
	noise := flag.Float64("noise", 0.02, "demo noise sigma")
	seed := flag.Int64("seed", 7, "demo random seed")
	size := flag.Int("size", 900, "image size in pixels")
	az := flag.Float64("az", 38, "camera azimuth (deg)")
	el := flag.Float64("el", 14, "camera elevation (deg)")
	flag.Parse()

	var sk *Skeleton
	var err error
	if *input == "" {
		sk = DemoSkeleton(*seed, *noise)
	} else {
		sk, err = LoadSkeleton(*input)
		if err != nil {
			fmt.Fprintln(os.Stderr, "load:", err)
			os.Exit(1)
		}
	}
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *input == "" {
		p := filepath.Join(*outDir, "demo_skeleton.json")
		if err := sk.Save(p); err == nil {
			fmt.Println("wrote", p)
		}
	}

	st := *step
	if st <= 0 {
		st = meanRadius(sk) / 5
	}
	pr := Params{Step: st, Sides: *sides, SmoothIter: *smooth}
	mesh, rs, rep := BuildMesh(sk, pr)
	crack := mesh.CrackValues(rep.Joints)

	center, radius := sceneBounds(sk)
	cam := FitCamera(center, radius, *size, *size, *az, *el)

	outputs := []struct {
		name string
		r    *Raster
	}{
		{"skeleton_overlay.png", RenderOverlay(mesh, cam, sk, rs, rep.Joints)},
		{"tube_solid.png", RenderSolid(mesh, cam)},
		{"normal_crack_heatmap.png", RenderHeat(mesh, cam, crack, 0.96)},
	}
	for _, o := range outputs {
		p := filepath.Join(*outDir, o.name)
		if err := o.r.SavePNG(p); err != nil {
			fmt.Fprintln(os.Stderr, "save:", err)
			os.Exit(1)
		}
		fmt.Println("wrote", p)
	}
	objPath := filepath.Join(*outDir, "tube.obj")
	if err := mesh.SaveOBJ(objPath); err == nil {
		fmt.Println("wrote", objPath)
	}

	fmt.Printf("\nmesh: %d vertices, %d triangles, %d joints\n",
		len(mesh.V), len(mesh.F), len(rep.Joints))
	for _, b := range rep.Branches {
		fmt.Printf("branch %d: %d samples, radius [%.3f, %.3f], min rho/R = %.2f, min clearance/R = %.2f\n",
			b.ID, b.Samples, b.RMin, b.RMax, b.MinCurvRatio, b.MinClearRatio)
	}
	for _, j := range rep.Joints {
		fmt.Printf("joint parent=%d child=%d: sphere R = %.3f, trim = %.3f\n",
			j.ParentID, j.ChildID, j.SphereR, j.Trim)
	}
	maxCrack := 0.0
	for _, c := range crack {
		maxCrack = math.Max(maxCrack, c)
	}
	fmt.Printf("max normal crack: %.1f deg (heatmap scale 0-55 deg)\n", maxCrack*180/math.Pi)
}
