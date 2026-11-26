package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

const gutter = 4 // 图集每侧 Gutter 纹素数（双线性只需 1，取 4 留余量）

func writePNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func run() error {
	materials := makeMaterials()
	array := NewTextureArray(materials)
	atlas := BuildAtlas(materials, gutter)
	layerOK := CheckLayerIsolation(array)
	gutterOK := CheckAtlasGutter(atlas)
	outDir := "artifacts"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	collage := RenderBallCollage(array, atlas, len(materials))
	if err := writePNG(filepath.Join(outDir, "material_balls.png"), collage); err != nil {
		return err
	}
	boundary, maxDiff := RenderBoundaryTest(array, atlas, len(materials))
	if err := writePNG(filepath.Join(outDir, "boundary_bleed.png"), boundary); err != nil {
		return err
	}
	stats := buildStats(array, atlas, maxDiff, layerOK, gutterOK)
	table := RenderResourceTable(stats, maxDiff)
	if err := os.WriteFile(filepath.Join(outDir, "resource_table.md"), []byte(table), 0o644); err != nil {
		return err
	}
	fmt.Print(table)
	fmt.Printf("层间隔离校验: %v, Gutter 校验: %v\n", layerOK, gutterOK)
	fmt.Println("已生成: artifacts/material_balls.png, artifacts/boundary_bleed.png, artifacts/resource_table.md")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
