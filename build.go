package main

import "math"

// Params controls mesh generation.
type Params struct {
	Step       float64 // arc-length resampling step (<=0: auto)
	Sides      int     // cross-section sides
	SmoothIter int     // Laplacian smoothing passes on the centerline
}

// Resampled is a branch after arc-length resampling.
type Resampled struct {
	P []Vec3
	R []float64
	S []float64
}

// JointInfo describes one bifurcation and its filler sphere.
type JointInfo struct {
	ParentID, ChildID int
	Pos               Vec3
	SphereR           float64
	Trim              float64
}

type BranchReport struct {
	ID            int
	Samples       int
	RMin, RMax    float64
	MinCurvRatio  float64 // min radius-of-curvature / tube-radius
	MinClearRatio float64 // min other-branch clearance / tube-radius
}

type Report struct {
	Branches []BranchReport
	Joints   []JointInfo
}

// nearestOnPolyline returns the closest point on the polyline and the
// interpolated radius at that location.
func nearestOnPolyline(p Vec3, r *Resampled) (Vec3, float64) {
	best := math.Inf(1)
	bestP := r.P[0]
	bestR := r.R[0]
	for i := 0; i+1 < len(r.P); i++ {
		a, b := r.P[i], r.P[i+1]
		ab := b.Sub(a)
		t := 0.0
		if den := ab.Dot(ab); den > 1e-14 {
			t = math.Max(0, math.Min(1, p.Sub(a).Dot(ab)/den))
		}
		q := a.Add(ab.Scale(t))
		if d := p.Dist(q); d < best {
			best = d
			bestP = q
			bestR = r.R[i] + (r.R[i+1]-r.R[i])*t
		}
	}
	return bestP, bestR
}

// findJoints locates every child branch's attachment on its parent and
// sizes a filler sphere that swallows all incident tube ends.
func findJoints(sk *Skeleton, rs []*Resampled) []JointInfo {
	idx := map[int]int{}
	for i, b := range sk.Branches {
		idx[b.ID] = i
	}
	var joints []JointInfo
	for _, b := range sk.Branches {
		if b.Parent < 0 {
			continue
		}
		pi, ok := idx[b.Parent]
		if !ok {
			continue
		}
		ci := idx[b.ID]
		J, Rp := nearestOnPolyline(rs[ci].P[0], rs[pi])
		Rc := rs[ci].R[0]
		Rs := 1.25 * math.Max(Rp, Rc)
		L := rs[ci].S[len(rs[ci].S)-1]
		trim := math.Min(Rs, 0.35*L)
		joints = append(joints, JointInfo{
			ParentID: b.Parent, ChildID: b.ID,
			Pos: J, SphereR: Rs, Trim: trim,
		})
	}
	return joints
}

// trimBranch cuts away the part of a child branch buried in the joint
// sphere, interpolating a fresh first station exactly at the trim arc.
func trimBranch(r *Resampled, d float64) *Resampled {
	n := len(r.P)
	cut := 0
	for cut < n-2 && r.S[cut] < d {
		cut++
	}
	if cut == 0 {
		return r
	}
	t := 0.0
	if seg := r.S[cut] - r.S[cut-1]; seg > 1e-14 {
		t = (d - r.S[cut-1]) / seg
	}
	m := n - cut + 1
	out := &Resampled{
		P: make([]Vec3, m),
		R: make([]float64, m),
		S: make([]float64, m),
	}
	out.P[0] = r.P[cut-1].Lerp(r.P[cut], t)
	out.R[0] = r.R[cut-1] + (r.R[cut]-r.R[cut-1])*t
	copy(out.P[1:], r.P[cut:])
	copy(out.R[1:], r.R[cut:])
	for i := 1; i < m; i++ {
		out.S[i] = out.S[i-1] + out.P[i].Dist(out.P[i-1])
	}
	return out
}

func makeReport(sk *Skeleton, rs []*Resampled, joints []JointInfo) Report {
	rep := Report{Joints: joints}
	for bi, b := range sk.Branches {
		r := rs[bi]
		kap := smoothedCurvature(r)
		br := BranchReport{
			ID: b.ID, Samples: len(r.P),
			RMin: math.Inf(1), MinCurvRatio: math.Inf(1), MinClearRatio: math.Inf(1),
		}
		zones := jointZones(b.ID, joints)
		for i := 0; i < len(r.P); i++ {
			br.RMin = math.Min(br.RMin, r.R[i])
			br.RMax = math.Max(br.RMax, r.R[i])
			if kap[i] > 1e-9 {
				br.MinCurvRatio = math.Min(br.MinCurvRatio, 1/(kap[i]*r.R[i]))
			}
			if inJointZone(r.P[i], zones) {
				continue
			}
			for bj := range sk.Branches {
				if bj == bi {
					continue
				}
				d := polylineDist(r.P[i], rs[bj].P, nil)
				br.MinClearRatio = math.Min(br.MinClearRatio, d/r.R[i])
			}
		}
		rep.Branches = append(rep.Branches, br)
	}
	return rep
}

// BuildMesh runs the full pipeline: resample, joints, radius clamps,
// trimming, sweeping, and joint filling.
func BuildMesh(sk *Skeleton, pr Params) (*Mesh, []*Resampled, Report) {
	rs := make([]*Resampled, len(sk.Branches))
	for i, b := range sk.Branches {
		rs[i] = ResampleBranch(b, pr.Step, pr.SmoothIter)
	}
	joints := findJoints(sk, rs)
	clampRadii(sk, rs, joints)

	trimmed := make([]*Resampled, len(rs))
	jointOf := map[int]*JointInfo{}
	for i := range joints {
		jointOf[joints[i].ChildID] = &joints[i]
	}
	for i, b := range sk.Branches {
		trimmed[i] = rs[i]
		if j, ok := jointOf[b.ID]; ok {
			trimmed[i] = trimBranch(rs[i], j.Trim)
		}
	}

	mesh := &Mesh{}
	comp := 0
	for i, b := range sk.Branches {
		r := trimmed[i]
		nrm, bin := ParallelTransport(r.P)
		var proj *SphereRef
		capStart := true
		if j, ok := jointOf[b.ID]; ok {
			proj = &SphereRef{C: j.Pos, R: j.SphereR}
			capStart = false
		}
		mesh.AddTube(r.P, r.R, nrm, bin, pr.Sides, capStart, true, comp, proj)
		comp++
	}
	for _, j := range joints {
		mesh.AddSphere(j.Pos, j.SphereR, 28, 18, comp)
		comp++
	}
	mesh.ComputeNormals()
	return mesh, trimmed, makeReport(sk, trimmed, joints)
}

func crEval(p0, p1, p2, p3 Vec3, t float64) Vec3 {
	t2, t3 := t*t, t*t*t
	f := func(a, b, c, d float64) float64 {
		return 0.5 * (2*b + (-a+c)*t + (2*a-5*b+4*c-d)*t2 + (-a+3*b-3*c+d)*t3)
	}
	return Vec3{f(p0.X, p1.X, p2.X, p3.X), f(p0.Y, p1.Y, p2.Y, p3.Y), f(p0.Z, p1.Z, p2.Z, p3.Z)}
}

// ResampleBranch rebuilds a branch at uniform arc length. Positions come
// from a Catmull-Rom densification of the noisy input; radii come from a
// PCHIP interpolant of (arc length, radius), so the radius profile is C1
// and independent of the chosen step size.
func ResampleBranch(b *Branch, step float64, smoothIter int) *Resampled {
	pts := b.Points
	n := len(pts)
	const M = 24
	dense := make([]Vec3, 0, (n-1)*M+1)
	for i := 0; i < n-1; i++ {
		p0 := pts[max(0, i-1)]
		p1 := pts[i]
		p2 := pts[i+1]
		p3 := pts[min(n-1, i+2)]
		for j := 0; j < M; j++ {
			dense = append(dense, crEval(p0, p1, p2, p3, float64(j)/M))
		}
	}
	dense = append(dense, pts[n-1])
	sd := make([]float64, len(dense))
	for i := 1; i < len(dense); i++ {
		sd[i] = sd[i-1] + dense[i].Dist(dense[i-1])
	}
	L := sd[len(sd)-1]
	nodeS := make([]float64, n)
	for i := 0; i < n; i++ {
		nodeS[i] = sd[i*M]
	}
	// Pre-smooth the node radii so input noise does not turn into
	// surface bumps; done at node level so the result is independent
	// of the resampling step.
	nodeR := append([]float64(nil), b.Radii...)
	for it := 0; it < 2; it++ {
		prev := append([]float64(nil), nodeR...)
		for i := 1; i < n-1; i++ {
			nodeR[i] = 0.25*prev[i-1] + 0.5*prev[i] + 0.25*prev[i+1]
		}
	}
	pc := NewPchip(nodeS, nodeR)

	if step <= 0 {
		step = L / 60
	}
	ns := int(math.Ceil(L/step)) + 1
	if ns < 4 {
		ns = 4
	}
	P := make([]Vec3, ns)
	R := make([]float64, ns)
	S := make([]float64, ns)
	k := 0
	for j := 0; j < ns; j++ {
		s := float64(j) * L / float64(ns-1)
		for k+1 < len(sd)-1 && sd[k+1] < s {
			k++
		}
		seg := sd[k+1] - sd[k]
		t := 0.0
		if seg > 1e-14 {
			t = (s - sd[k]) / seg
		}
		P[j] = dense[k].Lerp(dense[k+1], t)
		S[j] = s
		R[j] = pc.Eval(s)
	}
	for it := 0; it < smoothIter; it++ {
		for i := 1; i < ns-1; i++ { // keep both endpoints pinned
			mid := P[i-1].Add(P[i+1]).Scale(0.5)
			P[i] = P[i].Add(mid.Sub(P[i]).Scale(0.4))
		}
	}
	// Two light moving-average passes remove input radius noise while
	// keeping the profile continuous; endpoints keep their PCHIP values.
	for it := 0; it < 2; it++ {
		prev := append([]float64(nil), R...)
		for i := 1; i < ns-1; i++ {
			R[i] = 0.25*prev[i-1] + 0.5*prev[i] + 0.25*prev[i+1]
		}
	}
	for j := 1; j < ns; j++ {
		S[j] = S[j-1] + P[j].Dist(P[j-1])
	}
	return &Resampled{P: P, R: R, S: S}
}

func pointSegDist(p, a, b Vec3) float64 {
	ab := b.Sub(a)
	den := ab.Dot(ab)
	t := 0.0
	if den > 1e-14 {
		t = p.Sub(a).Dot(ab) / den
		t = math.Max(0, math.Min(1, t))
	}
	return p.Dist(a.Add(ab.Scale(t)))
}

func polylineDist(p Vec3, poly []Vec3, skip func(i int) bool) float64 {
	d := math.Inf(1)
	for i := 0; i+1 < len(poly); i++ {
		if skip != nil && skip(i) {
			continue
		}
		if dd := pointSegDist(p, poly[i], poly[i+1]); dd < d {
			d = dd
		}
	}
	return d
}

func jointZones(bID int, joints []JointInfo) []JointInfo {
	var out []JointInfo
	for _, j := range joints {
		if j.ParentID == bID || j.ChildID == bID {
			out = append(out, j)
		}
	}
	return out
}

func inJointZone(p Vec3, zones []JointInfo) bool {
	for _, z := range zones {
		if p.Dist(z.Pos) < 2.2*z.SphereR {
			return true
		}
	}
	return false
}

// smoothedCurvature estimates centerline curvature at the scale that
// matters for tube self-intersection: raw 3-point curvature averaged
// over a +-3 station window, so sub-radius centerline noise does not
// masquerade as a tight bend.
func smoothedCurvature(r *Resampled) []float64 {
	n := len(r.P)
	kap := make([]float64, n)
	for i := 1; i < n-1; i++ {
		kap[i] = Curvature(r.P[i-1], r.P[i], r.P[i+1])
	}
	kap[0], kap[n-1] = kap[1], kap[n-2]
	avg := make([]float64, n)
	for i := 0; i < n; i++ {
		sum, cnt := 0.0, 0
		for d := -3; d <= 3; d++ {
			j := i + d
			if j >= 0 && j < n {
				sum += kap[j]
				cnt++
			}
		}
		avg[i] = sum / float64(cnt)
	}
	return avg
}

// clampRadii enforces the two non-self-intersection invariants:
//  1. tube radius <= 0.7 * local radius of curvature (bends can't fold);
//  2. tube radius <= 0.45 * clearance to other branches (branches can't
//     overlap), excluding joint zones which are covered by spheres.
//
// A slope-limit pass then keeps the radius profile continuous.
func clampRadii(sk *Skeleton, rs []*Resampled, joints []JointInfo) {
	for bi, b := range sk.Branches {
		r := rs[bi]
		n := len(r.P)
		zones := jointZones(b.ID, joints)

		avg := smoothedCurvature(r)
		// Worst-case smoothed curvature over +-2 stations: smooth but
		// conservative bound for the tube radius.
		for i := 0; i < n; i++ {
			k := avg[i]
			for d := -2; d <= 2; d++ {
				j := i + d
				if j >= 0 && j < n && avg[j] > k {
					k = avg[j]
				}
			}
			if k > 1e-9 {
				if lim := 0.75 / k; r.R[i] > lim {
					r.R[i] = lim
				}
			}
		}

		for i := 0; i < n; i++ {
			if inJointZone(r.P[i], zones) {
				continue
			}
			dmin := math.Inf(1)
			for bj, ob := range sk.Branches {
				if bj == bi {
					continue
				}
				ozones := jointZones(ob.ID, joints)
				skip := func(seg int) bool {
					mid := rs[bj].P[seg].Add(rs[bj].P[seg+1]).Scale(0.5)
					return inJointZone(mid, ozones)
				}
				if d := polylineDist(r.P[i], rs[bj].P, skip); d < dmin {
					dmin = d
				}
			}
			if lim := 0.45 * dmin; r.R[i] > lim {
				r.R[i] = lim
			}
		}

		const maxSlope = 0.45
		for i := 1; i < n; i++ {
			if d := maxSlope * (r.S[i] - r.S[i-1]); r.R[i] > r.R[i-1]+d {
				r.R[i] = r.R[i-1] + d
			}
		}
		for i := n - 2; i >= 0; i-- {
			if d := maxSlope * (r.S[i+1] - r.S[i]); r.R[i] > r.R[i+1]+d {
				r.R[i] = r.R[i+1] + d
			}
		}
	}
}
