package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
)

// Branch is one skeleton polyline with a per-node radius.
// Parent < 0 marks a root branch; a child branch's first point is the
// joint location and must lie on (or very near) its parent's curve.
type Branch struct {
	ID     int       `json:"id"`
	Parent int       `json:"parent"`
	Points []Vec3    `json:"points"`
	Radii  []float64 `json:"radii"`
}

type Skeleton struct {
	Branches []*Branch `json:"branches"`
}

func (v Vec3) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%g,%g,%g]", v.X, v.Y, v.Z)), nil
}

func (v *Vec3) UnmarshalJSON(b []byte) error {
	var a [3]float64
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	v.X, v.Y, v.Z = a[0], a[1], a[2]
	return nil
}

func LoadSkeleton(path string) (*Skeleton, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sk Skeleton
	if err := json.Unmarshal(data, &sk); err != nil {
		return nil, err
	}
	for _, b := range sk.Branches {
		if len(b.Points) != len(b.Radii) {
			return nil, fmt.Errorf("branch %d: %d points but %d radii", b.ID, len(b.Points), len(b.Radii))
		}
		if len(b.Points) < 2 {
			return nil, fmt.Errorf("branch %d: needs at least 2 points", b.ID)
		}
	}
	return &sk, nil
}

func (sk *Skeleton) Save(path string) error {
	data, err := json.MarshalIndent(sk, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (sk *Skeleton) branch(id int) *Branch {
	for _, b := range sk.Branches {
		if b.ID == id {
			return b
		}
	}
	return nil
}

// DemoSkeleton synthesizes a noisy three-branch bifurcating skeleton:
// a gently curved root, one child at ~42 deg, and a grandchild with a
// tight bend that exercises the curvature-based radius clamp.
func DemoSkeleton(seed int64, noise float64) *Skeleton {
	rng := rand.New(rand.NewSource(seed))
	gauss := rng.NormFloat64

	rootPos := func(t float64) Vec3 {
		return Vec3{0.55 * math.Sin(2.4*t+0.3), 3.6 * t, 0.45 * math.Sin(1.9*t+1.2)}
	}
	rootRad := func(t float64) float64 { return 0.30 - 0.10*t + 0.02*math.Sin(5*t) }

	const nRoot = 26
	rootPts := make([]Vec3, nRoot)
	rootR := make([]float64, nRoot)
	for i := 0; i < nRoot; i++ {
		t := float64(i) / (nRoot - 1)
		p := rootPos(t)
		p.X += noise * gauss()
		p.Y += noise * gauss()
		p.Z += noise * gauss()
		rootPts[i] = p
		rootR[i] = math.Max(0.03, rootRad(t)*(1+0.06*gauss()))
	}
	root := &Branch{ID: 0, Parent: -1, Points: rootPts, Radii: rootR}

	// Child A: leaves the root at ~42 degrees.
	idxA := 14
	tA := float64(idxA) / (nRoot - 1)
	JA := rootPts[idxA]
	tanA := rootPos(tA + 0.02).Sub(rootPos(tA - 0.02)).Norm()
	sideA := tanA.Cross(Vec3{0, 0, 1}).Norm()
	angA := 42 * math.Pi / 180
	dirA := tanA.Scale(math.Cos(angA)).Add(sideA.Scale(math.Sin(angA))).Norm()
	bendA := dirA.Cross(Vec3{0, 0, 1}).Norm()
	childA := makeChild(1, 0, JA, dirA, bendA, 1.7, 0.35, 16,
		func(u float64) float64 { return 0.16 - 0.05*u }, noise, gauss)

	// Child B: leaves child A at ~38 degrees, with a tight bend.
	idxB := 9
	cleanA := childPath(JA, dirA, bendA, 1.7, 0.35, 16)
	JB := childA.Points[idxB]
	uB := float64(idxB) / 15
	tanB := cleanA[idxB+1].Sub(cleanA[idxB-1]).Norm()
	sideB := tanB.Cross(Vec3{0, 1, 0}).Norm()
	angB := 38 * math.Pi / 180
	dirB := tanB.Scale(math.Cos(angB)).Add(sideB.Scale(math.Sin(angB))).Norm()
	bendB := dirB.Cross(Vec3{0, 1, 0}).Norm()
	childB := makeChild(2, 1, JB, dirB, bendB, 1.05, 0.55, 13,
		func(u float64) float64 { return 0.105 - 0.035*u }, noise, gauss)
	_ = uB

	return &Skeleton{Branches: []*Branch{root, childA, childB}}
}

func childPath(j, dir, bend Vec3, length, bendMag float64, n int) []Vec3 {
	pts := make([]Vec3, n)
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		pts[i] = j.Add(dir.Scale(length * u)).Add(bend.Scale(bendMag * (1 - math.Cos(math.Pi*u)) * 0.5))
	}
	return pts
}

func makeChild(id, parent int, j, dir, bend Vec3, length, bendMag float64, n int,
	rad func(u float64) float64, noise float64, gauss func() float64) *Branch {
	pts := childPath(j, dir, bend, length, bendMag, n)
	r := make([]float64, n)
	for i := 0; i < n; i++ {
		u := float64(i) / float64(n-1)
		if i > 0 { // keep the joint point exactly on the parent
			pts[i].X += noise * gauss()
			pts[i].Y += noise * gauss()
			pts[i].Z += noise * gauss()
		}
		r[i] = math.Max(0.03, rad(u)*(1+0.06*gauss()))
	}
	return &Branch{ID: id, Parent: parent, Points: pts, Radii: r}
}
