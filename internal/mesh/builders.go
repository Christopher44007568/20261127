package mesh

import (
	"math"
)

// HolePlateConfig parameterizes a rotationally symmetric flange with a hole.
type HolePlateConfig struct {
	OuterR  float64 // outer radius
	HoleR   float64 // hole radius
	Thick   float64 // plate thickness
	Angular int     // segments around the hole
	Radial  int     // rings from hole to outer edge
}

// HoleHandles names the important loops of BuildHolePlate.
type HoleHandles struct {
	TopRim   []int // top hole rim, angular order
	BotRim   []int
	TopOuter []int
}

// DefaultHolePlate is the standard closed-hole acceptance model.
func DefaultHolePlate() HolePlateConfig {
	return HolePlateConfig{OuterR: 26, HoleR: 8, Thick: 3, Angular: 128, Radial: 10}
}

// BuildHolePlate builds an annular flange with a circular hole and explicit
// hole/outer walls, so both rims are true manifold crease loops.
func BuildHolePlate(cfg HolePlateConfig) (*Mesh, HoleHandles) {
	m := New("hole-plate")
	nA, nR := cfg.Angular, cfg.Radial
	top := make([][]int, nA)
	bot := make([][]int, nA)

	for i := 0; i < nA; i++ {
		theta := 2 * math.Pi * float64(i) / float64(nA)
		cs, sn := math.Cos(theta), math.Sin(theta)
		rOut := cfg.OuterR
		top[i] = make([]int, nR+1)
		bot[i] = make([]int, nR+1)
		for j := 0; j <= nR; j++ {
			f := float64(j) / float64(nR)
			r := cfg.HoleR + (rOut-cfg.HoleR)*f
			x, y := r*cs, r*sn
			top[i][j] = m.AddVertex(Vec3{X: x, Y: y, Z: cfg.Thick * 0.5})
			bot[i][j] = m.AddVertex(Vec3{X: x, Y: y, Z: -cfg.Thick * 0.5})
		}
	}

	h := HoleHandles{}
	for i := 0; i < nA; i++ {
		h.TopRim = append(h.TopRim, top[i][0])
		h.BotRim = append(h.BotRim, bot[i][0])
		h.TopOuter = append(h.TopOuter, top[i][nR])
	}

	// Subdivided wall columns so top and bottom crease vertices are not
	// joined by a single high-curvature edge (keeps feature degree == 2).
	const wallSeg = 4
	inWall := make([][]int, nA)
	outWall := make([][]int, nA)
	for i := 0; i < nA; i++ {
		theta := 2 * math.Pi * float64(i) / float64(nA)
		cs, sn := math.Cos(theta), math.Sin(theta)
		inWall[i] = make([]int, wallSeg+1)
		outWall[i] = make([]int, wallSeg+1)
		inWall[i][0], inWall[i][wallSeg] = top[i][0], bot[i][0]
		outWall[i][0], outWall[i][wallSeg] = top[i][nR], bot[i][nR]
		for w := 1; w < wallSeg; w++ {
			z := cfg.Thick*0.5 - cfg.Thick*float64(w)/wallSeg
			inWall[i][w] = m.AddVertex(Vec3{X: cfg.HoleR * cs, Y: cfg.HoleR * sn, Z: z})
			outWall[i][w] = m.AddVertex(Vec3{X: cfg.OuterR * cs, Y: cfg.OuterR * sn, Z: z})
		}
	}

	next := func(i int) int { return (i + 1) % nA }

	for i := 0; i < nA; i++ {
		i2 := next(i)
		theta := 2 * math.Pi * float64(i) / float64(nA)
		radialOut := Vec3{X: math.Cos(theta), Y: math.Sin(theta), Z: 0}
		radialIn := radialOut.Scale(-1)
		for j := 0; j < nR; j++ {
			// Top skin, desired normal +Z.
			a, b := top[i][j], top[i2][j]
			c, d := top[i2][j+1], top[i][j+1]
			m.AddTriOriented(a, b, c, Vec3{X: 0, Y: 0, Z: 1})
			m.AddTriOriented(a, c, d, Vec3{X: 0, Y: 0, Z: 1})

			// Bottom skin, desired normal -Z.
			a, b = bot[i][j], bot[i2][j]
			c, d = bot[i2][j+1], bot[i][j+1]
			m.AddTriOriented(a, b, c, Vec3{X: 0, Y: 0, Z: -1})
			m.AddTriOriented(a, c, d, Vec3{X: 0, Y: 0, Z: -1})
		}

		// Hole wall, outward normal points toward the hole axis.
		for w := 0; w < wallSeg; w++ {
			a, b := inWall[i][w], inWall[i2][w]
			c, d := inWall[i2][w+1], inWall[i][w+1]
			m.AddTriOriented(a, b, c, radialIn)
			m.AddTriOriented(a, c, d, radialIn)
		}

		// Outer wall, outward normal points away from center.
		for w := 0; w < wallSeg; w++ {
			a, b := outWall[i][w], outWall[i2][w]
			c, d := outWall[i2][w+1], outWall[i][w+1]
			m.AddTriOriented(a, b, c, radialOut)
			m.AddTriOriented(a, c, d, radialOut)
		}
	}

	m.ComputeNormals()
	return m, h
}

// FoldConfig parameterizes the V-folded sheet (open sharp edge).
type FoldConfig struct {
	HalfWidth float64
	Length    float64
	Slope     float64 // dz/d|x|; dihedral normal angle = 2*atan(Slope)
	NX        int     // across (must be even so x=0 is a vertex column)
	NY        int     // along the crease
}

// FoldHandles names the crease column of BuildFold.
type FoldHandles struct {
	Crease []int // vertices along x=0, y ascending
	MidI   int
}

// DefaultFold is a ~60 degree V trough with a long open crease.
func DefaultFold() FoldConfig {
	return FoldConfig{HalfWidth: 12, Length: 30, Slope: math.Tan(30 * math.Pi / 180), NX: 48, NY: 60}
}

// BuildFold builds a rectangular sheet folded along x=0. The crease is an
// open sharp feature with exactly two endpoints on the sheet boundary.
func BuildFold(cfg FoldConfig) (*Mesh, FoldHandles) {
	m := New("open-sharp-fold")
	idx := make([][]int, cfg.NX+1)
	for i := 0; i <= cfg.NX; i++ {
		x := -cfg.HalfWidth + 2*cfg.HalfWidth*float64(i)/float64(cfg.NX)
		idx[i] = make([]int, cfg.NY+1)
		for j := 0; j <= cfg.NY; j++ {
			y := -cfg.Length*0.5 + cfg.Length*float64(j)/float64(cfg.NY)
			z := cfg.Slope * math.Abs(x)
			idx[i][j] = m.AddVertex(Vec3{X: x, Y: y, Z: z})
		}
	}
	mid := cfg.NX / 2
	hh := FoldHandles{MidI: mid}
	for j := 0; j <= cfg.NY; j++ {
		hh.Crease = append(hh.Crease, idx[mid][j])
	}
	up := Vec3{X: 0, Y: 0, Z: 1}
	for i := 0; i < cfg.NX; i++ {
		for j := 0; j < cfg.NY; j++ {
			a := idx[i][j]
			b := idx[i+1][j]
			c := idx[i+1][j+1]
			d := idx[i][j+1]
			m.AddTriOriented(a, b, c, up)
			m.AddTriOriented(a, c, d, up)
		}
	}
	m.ComputeNormals()
	return m, hh
}

// TangentConfig parameterizes a plane tangent to a cylinder.
type TangentConfig struct {
	Radius   float64
	PlaneLen float64 // plane extent away from the seam
	Width    float64 // extent along the cylinder axis (y)
	PhiMax   float64 // cylinder wrap angle
	NPlane   int
	NCyl     int
	NY       int
}

// TangentHandles names the tangent seam row of BuildTangent.
type TangentHandles struct {
	Seam []int // vertices along y at the plane/cylinder seam
}

// DefaultTangent gives a fine tessellation so the discrete dihedral at the
// tangent seam stays far below the normal-angle gate.
func DefaultTangent() TangentConfig {
	return TangentConfig{Radius: 10, PlaneLen: 14, Width: 26, PhiMax: 1.15, NPlane: 28, NCyl: 46, NY: 40}
}

// BuildTangent builds a z=0 plane (x<=0) smoothly tangent to a cylinder of
// radius R centered at (0,R,0). The seam at x=z=0 has matching normals but a
// curvature jump (0 on the plane, ~1/R on the cylinder).
func BuildTangent(cfg TangentConfig) (*Mesh, TangentHandles) {
	m := New("tangent-surfaces")

	type sample struct{ x, z float64 }
	var prof []sample
	// Plane rings from x=-PlaneLen up to x=0 (seam included once).
	for i := 0; i <= cfg.NPlane; i++ {
		x := -cfg.PlaneLen + cfg.PlaneLen*float64(i)/float64(cfg.NPlane)
		prof = append(prof, sample{x, 0})
	}
	// Cylinder rings after the seam: x=R sin(phi), z=R(1-cos phi).
	for k := 1; k <= cfg.NCyl; k++ {
		phi := cfg.PhiMax * float64(k) / float64(cfg.NCyl)
		prof = append(prof, sample{cfg.Radius * math.Sin(phi), cfg.Radius * (1 - math.Cos(phi))})
	}

	ny := cfg.NY
	grid := make([][]int, len(prof))
	for i, s := range prof {
		grid[i] = make([]int, ny+1)
		for j := 0; j <= ny; j++ {
			y := -cfg.Width*0.5 + cfg.Width*float64(j)/float64(ny)
			grid[i][j] = m.AddVertex(Vec3{X: s.x, Y: y, Z: s.z})
		}
	}

	want := Vec3{X: 0, Y: 0, Z: 1}
	for i := 0; i+1 < len(prof); i++ {
		for j := 0; j < ny; j++ {
			a := grid[i][j]
			b := grid[i+1][j]
			c := grid[i+1][j+1]
			d := grid[i][j+1]
			m.AddTriOriented(a, b, c, want)
			m.AddTriOriented(a, c, d, want)
		}
	}
	m.ComputeNormals()
	// Seam is the last plane ring, i = NPlane (prof index NPlane).
	seamI := cfg.NPlane
	th := TangentHandles{}
	for j := 0; j <= cfg.NY; j++ {
		th.Seam = append(th.Seam, grid[seamI][j])
	}
	return m, th
}
