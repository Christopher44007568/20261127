package main

import "math"

func cliffHeight(x, y float64) float64 {
	broad := worldFBM2(x*0.34+4.1, y*0.31-2.2, 5, 11) - 0.5
	mid := worldFBM2(x*0.95-7.0, y*0.82+3.0, 4, 29) - 0.5
	fine := worldFBM2(x*2.1, y*2.0, 3, 47) - 0.5
	strataWarp := worldFBM2(x*0.22+8, y*0.18, 3, 53) * 3.2
	strata := math.Tanh(2.6*math.Sin(y*2.7+strataWarp)) * 0.15
	bulge := 0.82 * math.Exp(-((x-1.7)*(x-1.7)/1.45 + (y+0.7)*(y+0.7)/3.1))
	notch := -0.42 * math.Exp(-((x+1.25)*(x+1.25)/0.48 + (y-1.55)*(y-1.55)/0.82))
	return broad*1.22 + mid*0.40 + fine*0.11 + strata + bulge + notch
}

func caveHeight(x, y float64) float64 {
	arch := 3.75 - 0.078*x*x
	broad := worldFBM2(x*0.43+2.0, y*0.38-5.0, 5, 67) - 0.5
	mid := worldFBM2(x*1.15, y*0.95+9.0, 4, 79) - 0.5
	ribs := 0.18 * math.Sin(y*2.25+worldFBM2(x*0.28, y*0.22, 3, 83)*4.0)
	return arch + broad*1.05 + mid*0.34 + ribs
}

func undersideHeight(x, y float64) float64 {
	broad := worldFBM2(x*0.48-3.0, y*0.52+6.0, 5, 101) - 0.5
	mid := worldFBM2(x*1.25+4.0, y*1.1, 4, 103) - 0.5
	strata := math.Tanh(3.2*math.Sin(x*1.65+y*0.42+worldFBM2(x*0.2, y*0.2, 3, 107)*2.5)) * 0.13
	pockets := -0.34 * math.Exp(-((x+1.4)*(x+1.4)/1.5 + (y-0.4)*(y-0.4)/1.1))
	return broad*0.72 + mid*0.28 + strata + pockets
}

func MirrorMeshX(mesh Mesh) Mesh {
	out := Mesh{Indices: append([]int(nil), mesh.Indices...)}
	out.Vertices = make([]Vertex, len(mesh.Vertices))
	for i, vertex := range mesh.Vertices {
		out.Vertices[i] = Vertex{
			Position: V(-vertex.Position.X, vertex.Position.Y, vertex.Position.Z),
			Normal:   V(-vertex.Normal.X, vertex.Normal.Y, vertex.Normal.Z),
		}
	}
	return out
}

func materialWith(texture Texture, worldScale, hardness float64) Config {
	cfg := DefaultConfig(texture)
	if worldScale > 0 {
		cfg.WorldScale = worldScale
	}
	if hardness > 0 {
		cfg.Hardness = hardness
	}
	return cfg
}

func BuildScenes(texture Texture, worldScale, hardness float64) []Scene {
	material := materialWith(texture, worldScale, hardness)
	outdoorTop := srgb255(139, 163, 181)
	outdoorBottom := srgb255(211, 201, 177)
	caveTop := srgb255(15, 18, 22)
	caveBottom := srgb255(35, 30, 25)
	skyTop := srgb255(119, 148, 174)
	skyBottom := srgb255(214, 207, 188)

	cliff := TransformMesh(
		NewGridMesh(8.6, 10.8, 104, 132, cliffHeight),
		Transform{
			Translation: V(0, 0.05, 0),
			Rotation:    V(-0.20, 0.30, -0.045),
			Scale:       V(1, 1, 1),
		},
	)

	cave := TransformMesh(
		NewGridMesh(12.5, 15.0, 120, 142, caveHeight),
		Transform{
			Translation: V(0, 0, -1.8),
			Rotation:    V(-math.Pi/2+0.035, 0, 0),
			Scale:       V(1, 1, 1),
		},
	)
	underside := TransformMesh(
		NewGridMesh(12.0, 11.4, 112, 104, undersideHeight),
		Transform{
			Translation: V(0, 3.42, -2.15),
			Rotation:    V(math.Pi/2+0.13, 0.07, 0.075),
			Scale:       V(1, 1, 1),
		},
	)

	mirrorSource := TransformMesh(
		NewGridMesh(4.35, 7.7, 76, 112, cliffHeight),
		Transform{
			Translation: V(-2.55, 0.0, 0),
			Rotation:    V(-0.13, 0.36, 0.035),
			Scale:       V(1, 1, 1),
		},
	)
	mirrorInstance := MirrorMeshX(mirrorSource)

	return []Scene{
		{
			Name:       "rock_wall",
			Camera:     NewCamera(V(1.55, 0.42, 16.1), V(0, 0.0, 0), V(0, 1, 0), 43),
			Meshes:     []Mesh{cliff},
			Material:   material,
			Background: outdoorTop,
			FloorFog:   outdoorBottom,
		},
		{
			Name:       "cave_ceiling",
			Camera:     NewCamera(V(0.15, 0.9, 6.25), V(0, 4.35, -3.2), V(0, 1, 0), 74),
			Meshes:     []Mesh{cave},
			Material:   material,
			Background: caveTop,
			FloorFog:   caveBottom,
		},
		{
			Name:       "underside",
			Camera:     NewCamera(V(0.72, 0.62, 4.85), V(0, 4.75, -0.55), V(0, 1, 0), 66),
			Meshes:     []Mesh{underside},
			Material:   material,
			Background: skyTop,
			FloorFog:   skyBottom,
		},
		{
			Name:       "mirrored_instance",
			Camera:     NewCamera(V(0, 0.35, 13.2), V(0, 0.05, 0), V(0, 1, 0), 45),
			Meshes:     []Mesh{mirrorSource, mirrorInstance},
			Material:   material,
			Background: outdoorTop,
			FloorFog:   outdoorBottom,
		},
	}
}
