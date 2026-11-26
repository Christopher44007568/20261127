package main

import (
	"math"
	"testing"
)

func TestPchipMonotoneAndExact(t *testing.T) {
	x := []float64{0, 1, 2, 4, 8}
	y := []float64{0.3, 0.25, 0.2, 0.12, 0.1}
	p := NewPchip(x, y)
	for i := range x {
		if got := p.Eval(x[i]); math.Abs(got-y[i]) > 1e-9 {
			t.Fatalf("node %d: got %v want %v", i, got, y[i])
		}
	}
	prev := p.Eval(0)
	for s := 0.01; s <= 8; s += 0.01 {
		v := p.Eval(s)
		if v > prev+1e-12 {
			t.Fatalf("not monotone at %v: %v > %v", s, v, prev)
		}
		if v < 0.1-1e-9 || v > 0.3+1e-9 {
			t.Fatalf("overshoot at %v: %v", s, v)
		}
		prev = v
	}
}

func TestResampleRadiusContinuity(t *testing.T) {
	sk := DemoSkeleton(7, 0.02)
	b := sk.Branches[0]
	r1 := ResampleBranch(b, 0.05, 3)
	r2 := ResampleBranch(b, 0.025, 3)
	// Radius profile must be continuous: bounded slope between stations.
	for i := 1; i < len(r1.R); i++ {
		ds := r1.S[i] - r1.S[i-1]
		if math.Abs(r1.R[i]-r1.R[i-1]) > 1.0*ds+1e-9 {
			t.Fatalf("radius jump at station %d", i)
		}
	}
	// Resampling at half the step must give nearly the same profile.
	// Compare radii at matching arc-length fractions.
	L1 := r1.S[len(r1.S)-1]
	L2 := r2.S[len(r2.S)-1]
	maxDiff := 0.0
	for i := 0; i < len(r1.S); i++ {
		f := r1.S[i] / L1
		s2 := f * L2
		j := 0
		for j+1 < len(r2.S) && r2.S[j+1] < s2 {
			j++
		}
		if j+1 >= len(r2.S) {
			break
		}
		t2 := (s2 - r2.S[j]) / (r2.S[j+1] - r2.S[j])
		v := r2.R[j] + (r2.R[j+1]-r2.R[j])*t2
		if d := math.Abs(v - r1.R[i]); d > maxDiff {
			maxDiff = d
		}
	}
	if maxDiff > 0.02 {
		t.Fatalf("radius profile changed under resampling: max diff %v", maxDiff)
	}
}

func TestParallelTransportFrames(t *testing.T) {
	sk := DemoSkeleton(7, 0.02)
	r := ResampleBranch(sk.Branches[0], 0.05, 3)
	nrm, bin := ParallelTransport(r.P)
	tan := Tangents(r.P)
	for i := range r.P {
		if math.Abs(nrm[i].Len()-1) > 1e-6 || math.Abs(nrm[i].Dot(tan[i])) > 1e-6 {
			t.Fatalf("frame %d not orthonormal", i)
		}
		if i > 0 {
			twist := angleBetween(nrm[i-1], nrm[i])
			if twist > 0.35 {
				t.Fatalf("excessive twist at %d: %v rad", i, twist)
			}
		}
		_ = bin
	}
}

// buildAngledSkeleton makes a root with one child leaving at angleDeg.
func buildAngledSkeleton(angleDeg float64) *Skeleton {
	n := 20
	rootPts := make([]Vec3, n)
	rootR := make([]float64, n)
	for i := 0; i < n; i++ {
		rootPts[i] = Vec3{0, float64(i) * 0.15, 0}
		rootR[i] = 0.25
	}
	root := &Branch{ID: 0, Parent: -1, Points: rootPts, Radii: rootR}
	J := rootPts[12]
	ang := angleDeg * math.Pi / 180
	dir := Vec3{math.Sin(ang), math.Cos(ang), 0}
	m := 12
	chPts := make([]Vec3, m)
	chR := make([]float64, m)
	for i := 0; i < m; i++ {
		chPts[i] = J.Add(dir.Scale(float64(i) * 0.12))
		chR[i] = 0.14
	}
	child := &Branch{ID: 1, Parent: 0, Points: chPts, Radii: chR}
	return &Skeleton{Branches: []*Branch{root, child}}
}

// TestNoOverlapAcrossBranchAngles verifies that for any branch angle the
// two tube surfaces stay separated outside the joint sphere, and the
// child's trimmed end ring lies exactly on the joint sphere.
func TestNoOverlapAcrossBranchAngles(t *testing.T) {
	for _, ang := range []float64{20, 35, 50, 70, 90} {
		sk := buildAngledSkeleton(ang)
		mesh, _, rep := BuildMesh(sk, Params{Step: 0.04, Sides: 16, SmoothIter: 2})
		if len(rep.Joints) != 1 {
			t.Fatalf("angle %v: expected 1 joint", ang)
		}
		j := rep.Joints[0]
		// Vertex-level clearance between different tube components
		// outside the joint sphere.
		vertComp := make([]int, len(mesh.V))
		for i := range vertComp {
			vertComp[i] = -1
		}
		for fi, f := range mesh.F {
			for k := 0; k < 3; k++ {
				if vertComp[f[k]] < 0 {
					vertComp[f[k]] = mesh.Comp[fi]
				}
			}
			// Child end ring lies exactly on the joint sphere: the first
			// `sides` vertices of component 1 are the projected ring.
			for i := 0; i < 16; i++ {
				if vertComp[i] != 1 {
					continue
				}
				if d := math.Abs(mesh.V[i].Dist(j.Pos) - j.SphereR); d > 1e-6 {
					t.Fatalf("angle %v: ring vertex off sphere by %v", ang, d)
				}
			}
		}
		minD := math.Inf(1)
		for a := 0; a < len(mesh.V); a++ {
			if vertComp[a] >= 2 { // skip joint sphere vertices
				continue
			}
			for b := a + 1; b < len(mesh.V); b++ {
				if vertComp[b] >= 2 || vertComp[a] == vertComp[b] {
					continue
				}
				pa, pb := mesh.V[a], mesh.V[b]
				if pa.Dist(j.Pos) < j.SphereR*1.05 || pb.Dist(j.Pos) < j.SphereR*1.05 {
					continue
				}
				if d := pa.Dist(pb); d < minD {
					minD = d
				}
			}
		}
		if minD < 0.005 {
			t.Fatalf("angle %v: tubes nearly touch outside joint (min dist %v)", ang, minD)
		}
		t.Logf("angle %v: min inter-tube vertex distance %.4f", ang, minD)
	}
}

func TestCurvatureClampOnTightBend(t *testing.T) {
	// Hairpin bend: radius of curvature ~0.25, requested radius 0.3.
	n := 40
	pts := make([]Vec3, n)
	rad := make([]float64, n)
	for i := 0; i < n; i++ {
		th := math.Pi * float64(i) / float64(n-1)
		pts[i] = Vec3{0.25 * math.Cos(th), 0.25 * math.Sin(th), 0}
		rad[i] = 0.3
	}
	b := &Branch{ID: 0, Parent: -1, Points: pts, Radii: rad}
	sk := &Skeleton{Branches: []*Branch{b}}
	_, rs, rep := BuildMesh(sk, Params{Step: 0.02, Sides: 12, SmoothIter: 0})
	_ = rs
	br := rep.Branches[0]
	if br.MinCurvRatio < 1.0 {
		t.Fatalf("tube radius exceeds local radius of curvature: rho/R = %v", br.MinCurvRatio)
	}
	if br.RMax > 0.7*0.25+0.02 {
		t.Fatalf("curvature clamp did not engage: RMax = %v", br.RMax)
	}
}
