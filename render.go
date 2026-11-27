package main

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type Scene struct {
	Name       string
	Camera     Camera
	Meshes     []Mesh
	Material   Config
	Background Vec3
	FloorFog   Vec3
}

type RenderOptions struct {
	Width  int
	Height int
	AA     int
}

func rockShade(scene Scene, tri Triplanar) ShadeFunc {
	keyDirection := V(-0.42, 0.82, 0.38).Normalize()
	fillDirection := V(0.72, 0.22, 0.56).Normalize()
	bounceDirection := V(0.28, -0.62, 0.58).Normalize()
	skyAmbient := srgb255(126, 143, 158).Scale(0.96)
	groundAmbient := srgb255(96, 77, 58).Scale(0.88)
	keyColor := V(1.0, 0.91, 0.76)
	fillColor := V(0.48, 0.61, 0.78)
	bounceColor := V(0.72, 0.55, 0.38)

	return func(ctx ShadeContext) Vec3 {
		world := ctx.World
		normal := ctx.Normal
		albedo := tri.SampleGrad(world, normal, ctx.DWorldDX, ctx.DWorldDY)
		view := scene.Camera.Eye.Sub(world).Normalize()
		faceNormal := normal
		if faceNormal.Dot(view) < 0 {
			faceNormal = faceNormal.Scale(-1)
		}

		hemisphere := faceNormal.Y*0.5 + 0.5
		ambient := lerpVec(groundAmbient, skyAmbient, clamp01(hemisphere))
		ndl := math.Max(faceNormal.Dot(keyDirection), 0)
		wrapped := math.Max((faceNormal.Dot(keyDirection)+0.18)/1.18, 0)
		diffuseLight := ambient.Add(keyColor.Scale(0.92*ndl + 0.22*wrapped))

		ndf := math.Max(faceNormal.Dot(fillDirection), 0)
		diffuseLight = diffuseLight.Add(fillColor.Scale(0.20 * ndf))
		bounce := math.Max(faceNormal.Dot(bounceDirection), 0)
		diffuseLight = diffuseLight.Add(bounceColor.Scale(0.62 * bounce))
		headLight := math.Max(faceNormal.Dot(view), 0)
		diffuseLight = diffuseLight.Add(V(0.34, 0.36, 0.38).Scale(0.42 * headLight))

		halfVector := keyDirection.Add(view).Normalize()
		specular := math.Pow(math.Max(faceNormal.Dot(halfVector), 0), 34)
		specularColor := V(1.0, 0.88, 0.68).Scale(0.16 * specular)
		fresnel := math.Pow(1-math.Max(faceNormal.Dot(view), 0), 3)
		rim := fillColor.Scale(0.16 * fresnel)

		radiance := albedo.Mul(diffuseLight).Add(specularColor).Add(rim.Mul(albedo.Add(V(0.10, 0.12, 0.14))))

		distance := world.Sub(scene.Camera.Eye).Length()
		fog := smoothstep(15, 27, distance)
		return lerpVec(radiance, scene.Background.Scale(0.55), fog*0.72)
	}
}

func renderLinear(scene Scene, width, height int) *Frame {
	frame := NewFrame(width, height)
	frame.Clear(scene.Background, scene.FloorFog)
	tri := Triplanar{Config: scene.Material}.WithDefaults()
	shade := rockShade(scene, tri)
	for _, mesh := range scene.Meshes {
		DrawMesh(frame, scene.Camera, mesh, shade)
	}
	return frame
}

func downsampleFrame(large *Frame, width, height int) *Frame {
	if large.Width == width && large.Height == height {
		return large
	}
	factorX := large.Width / width
	factorY := large.Height / height
	if factorX < 1 {
		factorX = 1
	}
	if factorY < 1 {
		factorY = 1
	}
	out := NewFrame(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := V(0, 0, 0)
			depth := 0.0
			count := 0
			for sy := y * factorY; sy < (y+1)*factorY; sy++ {
				for sx := x * factorX; sx < (x+1)*factorX; sx++ {
					i := large.index(sx, sy)
					sum = sum.Add(large.Color[i])
					if large.Depth[i] > depth {
						depth = large.Depth[i]
					}
					count++
				}
			}
			out.Color[out.index(x, y)] = sum.Scale(1 / float64(count))
			out.Depth[out.index(x, y)] = depth
		}
	}
	return out
}

func linearToSRGB(c float64) uint8 {
	c = clamp01(c)
	if c <= 0.0031308 {
		c *= 12.92
	} else {
		c = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return uint8(math.Round(clamp01(c) * 255))
}

func FrameToRGBA(frame *Frame, exposure float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for i, radiance := range frame.Color {
		mapped := V(
			reinardTonemap(radiance.X*exposure),
			reinardTonemap(radiance.Y*exposure),
			reinardTonemap(radiance.Z*exposure),
		)
		img.Pix[i*4+0] = linearToSRGB(mapped.X)
		img.Pix[i*4+1] = linearToSRGB(mapped.Y)
		img.Pix[i*4+2] = linearToSRGB(mapped.Z)
		img.Pix[i*4+3] = 255
	}
	return img
}

func reinardTonemap(x float64) float64 {
	x = math.Max(x, 0)
	return x / (1 + x)
}

func RenderFrame(scene Scene, opts RenderOptions) *Frame {
	if opts.Width <= 0 {
		opts.Width = 640
	}
	if opts.Height <= 0 {
		opts.Height = 480
	}
	if opts.AA <= 0 {
		opts.AA = 1
	}
	large := renderLinear(scene, opts.Width*opts.AA, opts.Height*opts.AA)
	return downsampleFrame(large, opts.Width, opts.Height)
}

func RenderImage(scene Scene, opts RenderOptions) *image.RGBA {
	frame := RenderFrame(scene, opts)
	return FrameToRGBA(frame, 1.34)
}

func WritePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	return encoder.Encode(file, img)
}
