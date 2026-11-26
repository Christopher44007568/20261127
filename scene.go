package main

type ViewSpec struct {
	Azimuth  float64
	Radius   float64
	EyeZ     float64
	ErrScale float64
}

type Scene struct {
	ID       string
	Title    string
	Subtitle string
	Geometry Geometry
	Target   Vec3
	SliceZ   float64
	Views    []ViewSpec
	Seed     int64
}

func symmetricPoseError(c Camera, i int, scale float64) (Vec3, float64, float64) {
	sign := 1.0
	if i%2 == 1 {
		sign = -1.0
	}
	offset := c.Right.Mul(scale * 0.82 * sign).
		Add(c.Forward.Mul(scale * 0.42 * sign)).
		Add(V3(0, 0, scale*0.16*float64((i%3)-1)))
	yaw := degToRad(0.28 * sign)
	pitch := degToRad(0.10 * float64((i%2)*2-1))
	return offset, yaw, pitch
}

func (s Scene) Frames(width, height int) []*DepthFrame {
	frames := make([]*DepthFrame, 0, len(s.Views))
	for i, view := range s.Views {
		trueCam := OrbitCamera(
			s.ID+"-true",
			s.Target,
			view.Radius,
			view.Azimuth,
			view.EyeZ,
			43,
			width,
			height,
		)
		scale := view.ErrScale
		if scale == 0 {
			scale = 0.014
		}
		offset, yaw, pitch := symmetricPoseError(trueCam, i, scale)
		fusionCam := trueCam.WithPoseError(offset, yaw, pitch)
		fusionCam.Name = s.ID + "-fusion"
		label := "V" + string(rune('1'+i))
		frames = append(frames, RenderDepthFrame(
			i, label, s.Geometry, trueCam, fusionCam,
			s.Seed+int64(i)*1009,
		))
	}
	return frames
}

func ThinSheetScene() Scene {
	return Scene{
		ID:       "thin_sheet",
		Title:    "Thin sheet",
		Subtitle: "45 mm plate; fine 20 mm voxels",
		Target:   V3(0, 0, 0.575),
		SliceZ:   0.56,
		Seed:     2101,
		Geometry: Geometry{
			{
				Name:   "sheet",
				Center: V3(0, 0, 0.575),
				Half:   V3(0.62, 0.0225, 0.575),
				Color:  RGB{0.35, 0.78, 0.95},
			},
		},
		Views: []ViewSpec{
			{Azimuth: -58, Radius: 2.55, EyeZ: 0.34, ErrScale: 0.013},
			{Azimuth: -8, Radius: 2.50, EyeZ: 0.38, ErrScale: 0.016},
			{Azimuth: 44, Radius: 2.62, EyeZ: 0.32, ErrScale: 0.012},
			{Azimuth: 101, Radius: 2.58, EyeZ: 0.36, ErrScale: 0.015},
			{Azimuth: 166, Radius: 2.52, EyeZ: 0.34, ErrScale: 0.014},
			{Azimuth: 224, Radius: 2.60, EyeZ: 0.39, ErrScale: 0.017},
			{Azimuth: 286, Radius: 2.56, EyeZ: 0.33, ErrScale: 0.013},
			{Azimuth: 332, Radius: 2.54, EyeZ: 0.37, ErrScale: 0.015},
		},
	}
}

func ConcaveCornerScene() Scene {
	return Scene{
		ID:       "concave_corner",
		Title:    "Concave corner",
		Subtitle: "90 degree re-entrant L profile",
		Target:   V3(0.02, -0.02, 0.575),
		SliceZ:   0.56,
		Seed:     2202,
		Geometry: Geometry{
			{
				Name:   "back_wall",
				Center: V3(0.10, 0.48, 0.575),
				Half:   V3(0.55, 0.06, 0.575),
				Color:  RGB{0.98, 0.68, 0.28},
			},
			{
				Name:   "left_wall",
				Center: V3(-0.45, -0.02, 0.575),
				Half:   V3(0.06, 0.50, 0.575),
				Color:  RGB{0.55, 0.86, 0.55},
			},
		},
		Views: []ViewSpec{
			{Azimuth: -72, Radius: 2.62, EyeZ: 0.36, ErrScale: 0.014},
			{Azimuth: -42, Radius: 2.52, EyeZ: 0.40, ErrScale: 0.017},
			{Azimuth: -12, Radius: 2.48, EyeZ: 0.34, ErrScale: 0.012},
			{Azimuth: 20, Radius: 2.52, EyeZ: 0.39, ErrScale: 0.016},
			{Azimuth: 52, Radius: 2.58, EyeZ: 0.35, ErrScale: 0.014},
			{Azimuth: 88, Radius: 2.66, EyeZ: 0.37, ErrScale: 0.015},
			{Azimuth: 178, Radius: 2.62, EyeZ: 0.34, ErrScale: 0.013},
			{Azimuth: 268, Radius: 2.66, EyeZ: 0.38, ErrScale: 0.016},
		},
	}
}

func WideBaselineScene() Scene {
	return Scene{
		ID:       "wide_baseline",
		Title:    "Wide baseline",
		Subtitle: "Six sparse orbit views with occlusion",
		Target:   V3(-0.05, 0.02, 0.52),
		SliceZ:   0.52,
		Seed:     2303,
		Geometry: Geometry{
			{
				Name:   "front_block",
				Center: V3(-0.18, -0.28, 0.45),
				Half:   V3(0.28, 0.23, 0.45),
				Color:  RGB{0.95, 0.42, 0.36},
			},
			{
				Name:   "rear_tower",
				Center: V3(0.43, 0.31, 0.55),
				Half:   V3(0.25, 0.22, 0.55),
				Color:  RGB{0.42, 0.70, 0.98},
			},
			{
				Name:   "left_block",
				Center: V3(-0.64, 0.20, 0.34),
				Half:   V3(0.22, 0.30, 0.34),
				Color:  RGB{0.70, 0.55, 0.95},
			},
		},
		Views: []ViewSpec{
			{Azimuth: -72, Radius: 2.95, EyeZ: 0.86, ErrScale: 0.015},
			{Azimuth: -12, Radius: 2.88, EyeZ: 0.78, ErrScale: 0.018},
			{Azimuth: 48, Radius: 2.98, EyeZ: 0.92, ErrScale: 0.014},
			{Azimuth: 108, Radius: 2.92, EyeZ: 0.82, ErrScale: 0.017},
			{Azimuth: 168, Radius: 3.00, EyeZ: 0.90, ErrScale: 0.015},
			{Azimuth: 228, Radius: 2.94, EyeZ: 0.80, ErrScale: 0.016},
		},
	}
}

func AllScenes() []Scene {
	return []Scene{ThinSheetScene(), ConcaveCornerScene(), WideBaselineScene()}
}
