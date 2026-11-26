package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// sampleNormal returns the posed normal at image pixel (px, py). The mesh UV
// maps directly onto the image, so the nearest grid vertex is used.
func sampleNormal(mesh *Mesh, width, height, px, py int) Vec3 {
	u := (float64(px) + 0.5) / float64(width)
	v := (float64(py) + 0.5) / float64(height)

	i := int(math.Round(u * float64(gridU-1)))
	j := int(math.Round(v * float64(gridV-1)))
	i = int(clampScalar(float64(i), 0, gridU-1))
	j = int(clampScalar(float64(j), 0, gridV-1))
	return mesh.Normals[i*gridV+j]
}

func toByte(v float64) uint8 {
	return uint8(clampScalar(math.Round(v), 0, 255))
}

// WriteNormalDiffMap renders the per-pixel difference between posed and base
// normals. Two files are produced per call:
//   - name_normal_diff_rgb.png: signed (dx, dy, dz) remapped to 0..255, so a
//     flat gray image means the pose keeps every base normal.
//   - name_normal_diff_mag.png: red-hot heatmap of the angular change.
func WriteNormalDiffMap(dir, name string, base, posed *Mesh, size int) (rgbPath, magPath string, err error) {
	rgb := image.NewRGBA(image.Rect(0, 0, size, size))
	mag := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			baseN := sampleNormal(base, size, size, x, y)
			posedN := sampleNormal(posed, size, size, x, y)
			diff := posedN.Sub(baseN)

			rgb.Set(x, y, color.RGBA{
				R: toByte((diff.X + 1) * 127.5),
				G: toByte((diff.Y + 1) * 127.5),
				B: toByte((diff.Z + 1) * 127.5),
				A: 255,
			})

			angle := math.Acos(clampScalar(baseN.Dot(posedN), -1, 1))
			heat := clampScalar(angle/(math.Pi/4), 0, 1) // saturate at 45 degrees
			mag.Set(x, y, color.RGBA{
				R: toByte(255 * heat),
				G: toByte(60 * heat * heat),
				B: toByte(40 * (1 - heat)),
				A: 255,
			})
		}
	}

	rgbPath = filepath.Join(dir, name+"_normal_diff_rgb.png")
	if err = encodePNG(rgbPath, rgb); err != nil {
		return "", "", err
	}
	magPath = filepath.Join(dir, name+"_normal_diff_mag.png")
	if err = encodePNG(magPath, mag); err != nil {
		return "", "", err
	}
	return rgbPath, magPath, nil
}

func encodePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
