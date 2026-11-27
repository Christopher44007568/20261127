package main

import "math"

type Vertex struct {
	Position Vec3
	Normal   Vec3
}

type Mesh struct {
	Vertices []Vertex
	Indices  []int
}

type Transform struct {
	Translation Vec3
	Rotation    Vec3
	Scale       Vec3
}

func DefaultTransform() Transform {
	return Transform{Scale: V(1, 1, 1)}
}

func worldHash(x, y, seed int) float64 {
	return float64(hashInt(x, y, seed)&0xffffff) / float64(0xffffff)
}

func worldNoise2(x, y float64, seed int) float64 {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	tx := x - math.Floor(x)
	ty := y - math.Floor(y)
	sx := tx * tx * (3 - 2*tx)
	sy := ty * ty * (3 - 2*ty)
	n00 := worldHash(x0, y0, seed)
	n10 := worldHash(x0+1, y0, seed)
	n01 := worldHash(x0, y0+1, seed)
	n11 := worldHash(x0+1, y0+1, seed)
	return lerpFloat(lerpFloat(n00, n10, sx), lerpFloat(n01, n11, sx), sy)
}

func worldFBM2(x, y float64, octaves, seed int) float64 {
	sum := 0.0
	amp := 0.5
	norm := 0.0
	freq := 1.0
	for octave := 0; octave < octaves; octave++ {
		sum += amp * worldNoise2(x*freq, y*freq, seed+octave*131)
		norm += amp
		amp *= 0.5
		freq *= 2.03
	}
	return sum / norm
}

type HeightFunc func(x, y float64) float64

func NewGridMesh(width, height float64, segX, segY int, heightAt HeightFunc) Mesh {
	if segX < 1 {
		segX = 1
	}
	if segY < 1 {
		segY = 1
	}
	mesh := Mesh{Vertices: make([]Vertex, 0, (segX+1)*(segY+1))}
	eps := 0.035

	for iy := 0; iy <= segY; iy++ {
		y := -height/2 + height*float64(iy)/float64(segY)
		for ix := 0; ix <= segX; ix++ {
			x := -width/2 + width*float64(ix)/float64(segX)
			z := 0.0
			dx := 0.0
			dy := 0.0
			if heightAt != nil {
				z = heightAt(x, y)
				dx = (heightAt(x+eps, y) - heightAt(x-eps, y)) / (2 * eps)
				dy = (heightAt(x, y+eps) - heightAt(x, y-eps)) / (2 * eps)
			}
			normal := V(-dx, -dy, 1).Normalize()
			mesh.Vertices = append(mesh.Vertices, Vertex{
				Position: V(x, y, z),
				Normal:   normal,
			})
		}
	}

	row := segX + 1
	for iy := 0; iy < segY; iy++ {
		for ix := 0; ix < segX; ix++ {
			a := iy*row + ix
			b := a + 1
			c := (iy+1)*row + ix
			d := c + 1
			mesh.Indices = append(mesh.Indices, a, b, c, b, d, c)
		}
	}
	return mesh
}

func rotateX(p Vec3, angle float64) Vec3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return V(p.X, p.Y*c-p.Z*s, p.Y*s+p.Z*c)
}

func rotateY(p Vec3, angle float64) Vec3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return V(p.X*c+p.Z*s, p.Y, -p.X*s+p.Z*c)
}

func rotateZ(p Vec3, angle float64) Vec3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return V(p.X*c-p.Y*s, p.X*s+p.Y*c, p.Z)
}

func rotateEuler(p Vec3, r Vec3) Vec3 {
	p = rotateX(p, r.X)
	p = rotateY(p, r.Y)
	p = rotateZ(p, r.Z)
	return p
}

func TransformMesh(mesh Mesh, tr Transform) Mesh {
	if tr.Scale.X == 0 {
		tr.Scale.X = 1
	}
	if tr.Scale.Y == 0 {
		tr.Scale.Y = 1
	}
	if tr.Scale.Z == 0 {
		tr.Scale.Z = 1
	}

	out := Mesh{Indices: append([]int(nil), mesh.Indices...)}
	out.Vertices = make([]Vertex, len(mesh.Vertices))
	for i, vertex := range mesh.Vertices {
		scaled := vertex.Position.Mul(tr.Scale)
		position := rotateEuler(scaled, tr.Rotation).Add(tr.Translation)

		// Inverse-transpose of a local scale, followed by rotation. This also
		// handles negative/mirrored scale components.
		normal := V(
			vertex.Normal.X/tr.Scale.X,
			vertex.Normal.Y/tr.Scale.Y,
			vertex.Normal.Z/tr.Scale.Z,
		)
		normal = rotateEuler(normal, tr.Rotation).Normalize()
		out.Vertices[i] = Vertex{Position: position, Normal: normal}
	}
	return out
}
